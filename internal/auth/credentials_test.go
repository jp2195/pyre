package auth_test

import (
	"strings"
	"testing"

	"github.com/jp2195/pyre/internal/auth"
	"github.com/jp2195/pyre/internal/config"
)

// mustResolve resolves credentials and fails the test on validation error.
func mustResolve(t *testing.T, cfg *config.Config, flags config.CLIFlags) *auth.Credentials {
	t.Helper()
	creds, err := auth.ResolveCredentials(cfg, flags)
	if err != nil {
		t.Fatalf("ResolveCredentials: %v", err)
	}
	return creds
}

// newConfigWithHost returns a *config.Config with a single default
// connection for host. ResolveCredentials consults the default connection
// when no CLI flag or env var supplies a host.
func newConfigWithHost(host string) *config.Config {
	return &config.Config{
		Default: host,
		Connections: map[string]config.ConnectionConfig{
			host: {},
		},
	}
}

// TestResolveCredentials_CLIFlagWins asserts that an explicit --api-key flag
// beats every other source.
func TestResolveCredentials_CLIFlagWins(t *testing.T) {
	host := "fw.example.com"
	t.Setenv("PYRE_API_KEY", "env-key")
	t.Setenv("PYRE_FW_EXAMPLE_COM_API_KEY", "host-env-key")

	creds := mustResolve(t, newConfigWithHost(host), config.CLIFlags{APIKey: "flag-key"})
	if creds.APIKey != "flag-key" {
		t.Fatalf("APIKey = %q, want flag-key", creds.APIKey)
	}
	if creds.PromptForPassword {
		t.Error("PromptForPassword must be false when a key was resolved")
	}
}

// TestResolveCredentials_GlobalEnvVar asserts that PYRE_API_KEY wins over
// the per-host env var fallback when no CLI flag is present.
func TestResolveCredentials_GlobalEnvVar(t *testing.T) {
	host := "fw.example.com"
	t.Setenv("PYRE_API_KEY", "global-env-key")
	t.Setenv("PYRE_FW_EXAMPLE_COM_API_KEY", "host-env-key")

	creds := mustResolve(t, newConfigWithHost(host), config.CLIFlags{})
	if creds.APIKey != "global-env-key" {
		t.Fatalf("APIKey = %q, want global-env-key", creds.APIKey)
	}
}

// TestResolveCredentials_HostEnvVarFallback asserts the per-host env var is
// consulted when neither a CLI flag nor PYRE_API_KEY is set.
func TestResolveCredentials_HostEnvVarFallback(t *testing.T) {
	host := "fw1.example.com"
	t.Setenv("PYRE_API_KEY", "")
	t.Setenv("PYRE_FW1_EXAMPLE_COM_API_KEY", "host-env-key")

	creds := mustResolve(t, newConfigWithHost(host), config.CLIFlags{})
	if creds.APIKey != "host-env-key" {
		t.Fatalf("APIKey = %q, want host-env-key", creds.APIKey)
	}
	if creds.PromptForPassword {
		t.Error("PromptForPassword must be false when an API key was resolved")
	}
}

// TestResolveCredentials_NoKeyPromptsForPassword asserts that when no source
// supplies a key, ResolveCredentials signals the TUI prompt flow.
func TestResolveCredentials_NoKeyPromptsForPassword(t *testing.T) {
	host := "fw3.example.com"
	t.Setenv("PYRE_API_KEY", "")
	t.Setenv("PYRE_FW3_EXAMPLE_COM_API_KEY", "")

	creds := mustResolve(t, newConfigWithHost(host), config.CLIFlags{})
	if creds.APIKey != "" {
		t.Errorf("APIKey = %q, want empty", creds.APIKey)
	}
	if !creds.PromptForPassword {
		t.Error("expected PromptForPassword=true when no key is available")
	}
	if creds.Host != host {
		t.Errorf("Host = %q, want %q", creds.Host, host)
	}
}

// TestConnection_ClearsCredentialsOnRemove verifies that RemoveConnection
// zeroes the APIKey / Password fields on the previously-added *Connection so
// any surviving reference in caller code stops pointing at the secret.
func TestConnection_ClearsCredentialsOnRemove(t *testing.T) {
	cfg := &config.Config{}
	session := auth.NewSession(cfg)

	const host = "10.0.0.42"
	const apiKey = "secret-api-key"

	connConfig := &config.ConnectionConfig{}
	connConfig.APIKey = apiKey
	connConfig.Password = "also-secret"

	conn, err := session.AddConnection(host, connConfig, apiKey)
	if err != nil {
		t.Fatalf("AddConnection: %v", err)
	}
	if conn.APIKey != apiKey {
		t.Fatalf("conn.APIKey after AddConnection = %q, want %q", conn.APIKey, apiKey)
	}

	session.RemoveConnection(host)

	if conn.APIKey != "" {
		t.Errorf("conn.APIKey after RemoveConnection = %q, want empty", conn.APIKey)
	}
	if conn.Config != nil {
		if conn.Config.APIKey != "" {
			t.Errorf("conn.Config.APIKey after RemoveConnection = %q, want empty", conn.Config.APIKey)
		}
		if conn.Config.Password != "" {
			t.Errorf("conn.Config.Password after RemoveConnection = %q, want empty", conn.Config.Password)
		}
	}

	if got := session.GetActiveConnection(); got != nil {
		t.Errorf("GetActiveConnection after RemoveConnection = %+v, want nil", got)
	}
}

// TestResolveCredentials_RejectsMalformedHost asserts that a host arriving
// via CLI flag that would change the request URL shape is rejected before
// any URL is built from it.
func TestResolveCredentials_RejectsMalformedHost(t *testing.T) {
	_, err := auth.ResolveCredentials(&config.Config{}, config.CLIFlags{Host: "evil.example/api"})
	if err == nil {
		t.Fatal("expected error for malformed --host, got nil")
	}
	if !strings.Contains(err.Error(), "invalid host") {
		t.Errorf("err = %v, want to mention 'invalid host'", err)
	}
}
