package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestGetCertificates_SanitizesSubjectAndIssuer(t *testing.T) {
	// Raw ESC (0x1B) is illegal in XML 1.0 and would be rejected by the parser
	// before sanitization; DEL (0x7F) is a legal XML character that the sanitizer
	// strips, demonstrating the sanitizer is wired in to the fetcher.
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, `<response status="success"><result>`+
			`<certificate><entry name="test-cert">`+
			`<subject>CN=`+"\x7f"+"evil"+`</subject>`+
			`<issuer>O=`+"bold"+"\x7f"+"issuer"+`</issuer>`+
			`<not-valid-before>Jan 01 00:00:00 2025 GMT</not-valid-before>`+
			`<not-valid-after>Jan 01 00:00:00 2027 GMT</not-valid-after>`+
			`<serial-number>DEADBEEF</serial-number>`+
			`<algorithm>RSA</algorithm>`+
			`</entry></certificate>`+
			`</result></response>`)
	})

	certs, err := c.GetCertificates(context.Background(), "")
	if err != nil {
		t.Fatalf("GetCertificates: %v", err)
	}
	if len(certs) != 1 {
		t.Fatalf("expected 1 cert, got %d", len(certs))
	}
	if certs[0].Subject != "CN=evil" {
		t.Errorf("Subject = %q, want %q", certs[0].Subject, "CN=evil")
	}
	if certs[0].Issuer != "O=boldissuer" {
		t.Errorf("Issuer = %q, want %q", certs[0].Issuer, "O=boldissuer")
	}
}

func TestGetDiskUsage_SanitizesFields(t *testing.T) {
	// PAN-OS embeds df -h output as text in the XML result. The XML 1.0
	// parser already rejects most C0 controls, but DEL (0x7f) survives and
	// any future move to CDATA-wrapped output could carry ESC sequences.
	// Verify we scrub control bytes before populating DiskUsage fields.
	rawDF := "Filesystem      Size  Used Avail Use% Mounted on\n" +
		"/dev/sda\x7f1  10\x7fG   2\x7fG   8\x7fG  20%  /va\x7fr\n"

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprintf(w, `<response status="success"><result>%s</result></response>`, rawDF)
	})

	got, err := c.GetDiskUsage(context.Background(), "")
	if err != nil {
		t.Fatalf("GetDiskUsage err: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected at least one DiskUsage entry, got 0")
	}
	const bad = "\x1b\x07\x7f"
	for _, du := range got {
		if strings.ContainsAny(du.Filesystem, bad) {
			t.Errorf("Filesystem contains control byte: %q", du.Filesystem)
		}
		if strings.ContainsAny(du.MountPoint, bad) {
			t.Errorf("MountPoint contains control byte: %q", du.MountPoint)
		}
		if strings.ContainsAny(du.Size, bad) {
			t.Errorf("Size contains control byte: %q", du.Size)
		}
		if strings.ContainsAny(du.Used, bad) {
			t.Errorf("Used contains control byte: %q", du.Used)
		}
		if strings.ContainsAny(du.Available, bad) {
			t.Errorf("Available contains control byte: %q", du.Available)
		}
	}
}
