package auth

import (
	"testing"

	"github.com/jp2195/pyre/internal/config"
)

func TestNormalizeHostForEnv(t *testing.T) {
	tests := []struct {
		host string
		want string
	}{
		{"firewall.example.com", "FIREWALL_EXAMPLE_COM"},
		{"firewall.example.com:8443", "FIREWALL_EXAMPLE_COM"},
		{"10.0.0.1", "10_0_0_1"},
		{"10.0.0.1:8443", "10_0_0_1"},
		{"my-fw-01.example.com", "MY_FW_01_EXAMPLE_COM"},
		{"[2001:db8::1]:8443", "2001_DB8__1"},
		{"2001:db8::1", "2001_DB8__1"},
		{"", ""},
	}
	for _, tc := range tests {
		t.Run(tc.host, func(t *testing.T) {
			got := normalizeHostForEnv(tc.host)
			if got != tc.want {
				t.Errorf("normalizeHostForEnv(%q) = %q, want %q", tc.host, got, tc.want)
			}
		})
	}
}

func TestResolveCredentials_HostPortEnvVar(t *testing.T) {
	t.Setenv("PYRE_FIREWALL_EXAMPLE_COM_API_KEY", "secret-from-env")

	cfg := &config.Config{}
	creds := ResolveCredentials(cfg, config.CLIFlags{Host: "firewall.example.com:8443"})

	if creds.APIKey != "secret-from-env" {
		t.Errorf("expected env-var resolution despite :port, got APIKey=%q", creds.APIKey)
	}
	if creds.PromptForPassword {
		t.Error("PromptForPassword should be false when env-var resolves the key")
	}
}
