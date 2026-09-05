package api

import (
	"net/http"
	"net/url"
	"testing"
)

// TestNewClient_BracketsBareIPv6Host covers a host shape that passed
// validation and then broke every request. ValidateHost accepts any
// net.ParseIP result, including a bare IPv6 literal, but "https://::1/api/"
// is not a parseable URL: Go reads the trailing group as a port. Every call
// failed with `invalid port` before a packet was sent.
func TestNewClient_BracketsBareIPv6Host(t *testing.T) {
	for _, tc := range []struct {
		name     string
		host     string
		wantHost string
		wantPort string
	}{
		{"bare IPv6", "2001:db8::1", "2001:db8::1", ""},
		{"IPv6 loopback", "::1", "::1", ""},
		{"already bracketed", "[2001:db8::1]", "2001:db8::1", ""},
		{"bracketed with port", "[2001:db8::1]:8443", "2001:db8::1", "8443"},
		{"IPv4", "10.0.0.1", "10.0.0.1", ""},
		{"hostname with port", "fw.example.com:8443", "fw.example.com", "8443"},
		{"hostname", "fw.example.com", "fw.example.com", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := NewClient(tc.host, "key", ClientOptions{})
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			u, err := url.Parse(c.baseURL)
			if err != nil {
				t.Fatalf("base URL %q does not parse: %v", c.baseURL, err)
			}
			if u.Hostname() != tc.wantHost {
				t.Errorf("hostname = %q, want %q (base URL %q)", u.Hostname(), tc.wantHost, c.baseURL)
			}
			if u.Port() != tc.wantPort {
				t.Errorf("port = %q, want %q (base URL %q)", u.Port(), tc.wantPort, c.baseURL)
			}
			if _, err := http.NewRequest(http.MethodGet, c.baseURL, nil); err != nil {
				t.Errorf("base URL %q cannot build a request: %v", c.baseURL, err)
			}
		})
	}
}
