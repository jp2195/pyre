package api_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jp2195/pyre/internal/api"
)

// Security policies are the reason this tool exists, and their fetcher was
// one of roughly fifteen that never sanitized anything: the sanitize call was
// something each fetcher had to remember, and most did not. A rule name is
// attacker-influenced in the cases that matter -- a compromised firewall, a
// Panorama pushing rules from elsewhere, or a MITM when --insecure is in use.
//
// Dangerous characters are written as escapes so the test can be reviewed.
func TestGetSecurityPolicies_SanitizesUntrustedFields(t *testing.T) {
	const (
		csi  = "\u009b"
		rlo  = "\u202e"
		zwsp = "\u200b"
	)
	body := `<response status="success"><result><rules>` +
		`<entry name="allow-` + csi + `31mweb` + rlo + `x">` +
		`<description>desc` + zwsp + `text</description>` +
		`<action>allow</action>` +
		`<to><member>untrust</member></to>` +
		`<from><member>trust</member></from>` +
		`</entry></rules></result></response>`

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		if r.URL.Query().Get("type") == "config" {
			_, _ = io.WriteString(w, body)
			return
		}
		_, _ = io.WriteString(w, `<response status="success"><result></result></response>`)
	}))
	defer srv.Close()

	client, err := api.NewClient(strings.TrimPrefix(srv.URL, "https://"), "k", api.ClientOptions{Insecure: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	rules, err := client.GetSecurityPolicies(context.Background(), "")
	if err != nil {
		t.Fatalf("GetSecurityPolicies: %v", err)
	}
	if len(rules) == 0 {
		t.Fatal("expected at least one rule")
	}

	for _, r := range rules {
		if strings.ContainsRune(r.Name, 0x009b) {
			t.Errorf("rule name still carries a C1 CSI: %q", r.Name)
		}
		if strings.ContainsRune(r.Name, 0x202e) {
			t.Errorf("rule name still carries a bidi override: %q", r.Name)
		}
		if strings.ContainsRune(r.Description, 0x200b) {
			t.Errorf("description still carries a zero-width space: %q", r.Description)
		}
		if want := "allow-webx"; r.Name != want {
			t.Errorf("Name = %q, want %q", r.Name, want)
		}
		if want := "desctext"; r.Description != want {
			t.Errorf("Description = %q, want %q", r.Description, want)
		}
		// Members travel through the same decode and must survive intact.
		if len(r.DestZones) != 1 || r.DestZones[0] != "untrust" {
			t.Errorf("DestZones = %v, want [untrust]", r.DestZones)
		}
	}
}
