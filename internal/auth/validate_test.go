package auth_test

import (
	"strings"
	"testing"

	"github.com/jp2195/pyre/internal/auth"
)

func TestValidateHost(t *testing.T) {
	valid := []string{
		"",
		"192.168.1.1",
		"10.0.0.1:8443",
		"fw.example.com",
		"fw.example.com:443",
		"fw-01.internal",
		"2001:db8::1",
	}
	for _, host := range valid {
		if msg := auth.ValidateHost(host); msg != "" {
			t.Errorf("ValidateHost(%q) = %q, want valid", host, msg)
		}
	}

	invalid := []string{
		"evil.example/api",
		"user@fw.example.com",
		"fw.example.com?x=1",
		"fw example com",
		"host\twith\ttabs",
		strings.Repeat("a", 254),
	}
	for _, host := range invalid {
		if msg := auth.ValidateHost(host); msg == "" {
			t.Errorf("ValidateHost(%q) = valid, want rejection", host)
		}
	}
}
