package auth_test

import (
	"strings"
	"testing"

	"github.com/jp2195/pyre/internal/auth"
)

// TestValidateHost_RejectsURLs covers the most likely thing to be pasted into
// the host field: the address bar of the firewall's web interface.
//
// The check accepted them. "https://10.0.104.50" contains one colon, so
// net.SplitHostPort reads the host as "https" and the rest as a port, and
// "https" is a valid hostname. The client then builds a URL by prefixing
// "https://" and appending "/api/", so the request went somewhere
// meaningless and failed with a message about a name that does not resolve.
func TestValidateHost_RejectsURLs(t *testing.T) {
	urls := []string{
		"https://10.0.104.50",
		"http://10.0.104.50",
		"https://firewall.example.com",
		"https://10.0.104.50/api",
		"10.0.104.50/api",
		"firewall.example.com/",
	}
	for _, host := range urls {
		msg := auth.ValidateHost(host)
		if msg == "" {
			t.Errorf("ValidateHost(%q) accepted it, want a message", host)
			continue
		}
		if !strings.Contains(msg, "host") && !strings.Contains(msg, "scheme") && !strings.Contains(msg, "path") {
			t.Errorf("ValidateHost(%q) = %q, which does not say what to fix", host, msg)
		}
	}
}

// TestValidateHost_StillAcceptsWhatItShould guards the fix against
// over-reaching: addresses, names, and host:port forms remain valid.
func TestValidateHost_StillAcceptsWhatItShould(t *testing.T) {
	for _, host := range []string{
		"10.0.104.50",
		"10.0.104.50:8443",
		"firewall.example.com",
		"firewall.example.com:443",
		"fe80::1",
		"[fe80::1]:443",
		"",
	} {
		if msg := auth.ValidateHost(host); msg != "" {
			t.Errorf("ValidateHost(%q) = %q, want it accepted", host, msg)
		}
	}
}
