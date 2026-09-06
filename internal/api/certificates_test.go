package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// certFlatResult renders the flat node stream PAN-OS returns for the
// key-free certificate xpath: a self-closing <entry name=...> followed by
// that entry's own children, repeated per certificate. Values are synthetic;
// the SHAPE is what was captured from a PA-440 on 11.2.10-h8.
func certFlatResult(entries ...string) string {
	return `<response status="success" code="19"><result total-count="3" count="3">` +
		strings.Join(entries, "") + `</result></response>`
}

func certEntry(name, subject, issuer, algorithm string, notAfter time.Time) string {
	return fmt.Sprintf(`<entry name=%q/>`+
		`<subject-hash>aaaa1111</subject-hash>`+
		`<issuer-hash>bbbb2222</issuer-hash>`+
		`<not-valid-before>Nov 10 22:51:48 2024 GMT</not-valid-before>`+
		`<issuer>%s</issuer>`+
		`<not-valid-after>%s</not-valid-after>`+
		`<common-name>example.com</common-name>`+
		`<expiry-epoch>%d</expiry-epoch>`+
		`<ca>no</ca>`+
		`<subject>%s</subject>`+
		`<algorithm>%s</algorithm>`,
		name, issuer, notAfter.UTC().Format("Jan _2 15:04:05 2006 MST"),
		notAfter.Unix(), subject, algorithm)
}

// TestGetCertificates_NeverRequestsKeyMaterial is the regression guard for
// the rule that pyre must never pull a private key off the device. Every
// certificate xpath must exclude both key elements at the source, so the
// material never reaches the wire, the response buffer, or the debug preview.
func TestGetCertificates_NeverRequestsKeyMaterial(t *testing.T) {
	var mu sync.Mutex
	var xpaths []string
	var types []string

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		mu.Lock()
		xpaths = append(xpaths, r.Form.Get("xpath"))
		types = append(types, r.Form.Get("type"))
		mu.Unlock()
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, `<response status="success" code="7"><result/></response>`)
	})

	if _, err := c.GetCertificates(context.Background(), ""); err != nil {
		t.Fatalf("GetCertificates: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(xpaths) == 0 {
		t.Fatal("no request was made")
	}
	for i, xp := range xpaths {
		if types[i] != "config" {
			t.Errorf("request %d: type = %q, want %q (the op command is rejected by PAN-OS)", i, types[i], "config")
		}
		if xp == "" {
			t.Errorf("request %d: empty xpath", i)
			continue
		}
		if !strings.Contains(xp, "not(self::private-key") {
			t.Errorf("request %d: xpath does not exclude private-key at the source:\n  %s", i, xp)
		}
		if !strings.Contains(xp, "public-key") {
			t.Errorf("request %d: xpath does not exclude public-key at the source:\n  %s", i, xp)
		}
	}
}

func TestGetCertificates_ParsesFlatEntryStream(t *testing.T) {
	now := time.Now()
	body := certFlatResult(
		certEntry("edge-cert", "/CN=edge.example.com", "/CN=Example CA", "EC", now.Add(400*24*time.Hour)),
		certEntry("soon-cert", "/CN=soon.example.com", "/CN=Example CA", "RSA", now.Add(5*24*time.Hour)),
		certEntry("dead-cert", "/CN=dead.example.com", "/CN=Example CA", "RSA", now.Add(-3*24*time.Hour)),
	)
	first := true
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		if first {
			first = false
			fmt.Fprint(w, body)
			return
		}
		fmt.Fprint(w, `<response status="success" code="7"><result/></response>`)
	})

	certs, err := c.GetCertificates(context.Background(), "")
	if err != nil {
		t.Fatalf("GetCertificates: %v", err)
	}
	if len(certs) != 3 {
		t.Fatalf("got %d certs, want 3: %+v", len(certs), certs)
	}

	want := []struct {
		name, status, algorithm string
	}{
		{"edge-cert", "valid", "EC"},
		{"soon-cert", "expiring", "RSA"},
		{"dead-cert", "expired", "RSA"},
	}
	for i, w := range want {
		if certs[i].Name != w.name {
			t.Errorf("cert %d: Name = %q, want %q", i, certs[i].Name, w.name)
		}
		if certs[i].Status != w.status {
			t.Errorf("cert %d (%s): Status = %q, want %q (DaysLeft=%d)", i, w.name, certs[i].Status, w.status, certs[i].DaysLeft)
		}
		if certs[i].Algorithm != w.algorithm {
			t.Errorf("cert %d (%s): Algorithm = %q, want %q", i, w.name, certs[i].Algorithm, w.algorithm)
		}
	}
	// Fields must bind to the entry they follow, not leak across entries.
	if certs[0].Subject != "/CN=edge.example.com" {
		t.Errorf("Subject = %q, want %q", certs[0].Subject, "/CN=edge.example.com")
	}
	if certs[2].Subject != "/CN=dead.example.com" {
		t.Errorf("cert 2 Subject = %q, want %q", certs[2].Subject, "/CN=dead.example.com")
	}
}

// A field arriving before any <entry> has no certificate to bind to. It must
// be dropped rather than attached to whatever entry happens to come next.
func TestGetCertificates_DropsFieldsBeforeFirstEntry(t *testing.T) {
	body := `<response status="success" code="19"><result>` +
		`<subject>/CN=orphan.example.com</subject>` +
		`<entry name="real-cert"/>` +
		`<subject>/CN=real.example.com</subject>` +
		`</result></response>`
	first := true
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		if first {
			first = false
			fmt.Fprint(w, body)
			return
		}
		fmt.Fprint(w, `<response status="success" code="7"><result/></response>`)
	})

	certs, err := c.GetCertificates(context.Background(), "")
	if err != nil {
		t.Fatalf("GetCertificates: %v", err)
	}
	if len(certs) != 1 {
		t.Fatalf("got %d certs, want 1: %+v", len(certs), certs)
	}
	if certs[0].Subject != "/CN=real.example.com" {
		t.Errorf("Subject = %q, want %q (orphan field leaked)", certs[0].Subject, "/CN=real.example.com")
	}
}

// PAN-OS answers a missing node with success + code 7 (see XMLResponse.NodeAbsent).
// A firewall with no vsys certificate store must read as "none", not an error.
func TestGetCertificates_AbsentNodeIsNotAnError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, `<response status="success" code="7"><result/></response>`)
	})

	certs, err := c.GetCertificates(context.Background(), "")
	if err != nil {
		t.Fatalf("GetCertificates: %v", err)
	}
	if len(certs) != 0 {
		t.Fatalf("got %d certs, want 0", len(certs))
	}
}
