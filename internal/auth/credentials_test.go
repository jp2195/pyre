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

// TestResolveCredentials_ConnectionFlagUsesItsOwnHostKey covers a credential
// mix-up: -c overwrote the host *after* resolution, so the host-scoped lookup
// ran against the default connection. A key belonging to one firewall was
// paired with a different firewall's host, and the -c host's own key was
// never consulted.
func TestResolveCredentials_ConnectionFlagUsesItsOwnHostKey(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Connections["default.example.com"] = config.ConnectionConfig{}
	cfg.Connections["target.example.com"] = config.ConnectionConfig{
		Username: "operator",
		Insecure: true,
	}
	cfg.Default = "default.example.com"

	t.Setenv("PYRE_DEFAULT_EXAMPLE_COM_API_KEY", "wrong-key")
	t.Setenv("PYRE_TARGET_EXAMPLE_COM_API_KEY", "right-key")

	creds, err := auth.ResolveCredentials(cfg, config.CLIFlags{Connection: "target.example.com"})
	if err != nil {
		t.Fatalf("ResolveCredentials: %v", err)
	}

	if creds.Host != "target.example.com" {
		t.Errorf("Host = %q, want target.example.com", creds.Host)
	}
	if creds.APIKey != "right-key" {
		t.Errorf("APIKey = %q, want right-key (the -c host's own key)", creds.APIKey)
	}
	if creds.Username != "operator" {
		t.Errorf("Username = %q, want operator (from the selected connection)", creds.Username)
	}
	if !creds.Insecure {
		t.Error("Insecure = false, want true (from the selected connection)")
	}
	if creds.PromptForPassword {
		t.Error("PromptForPassword = true, but a key was resolved for this host")
	}
}

// TestResolveCredentials_ConnectionFlagUnknownHost checks the -c lookup fails
// loudly rather than resolving credentials for some other host.
func TestResolveCredentials_ConnectionFlagUnknownHost(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Connections["known.example.com"] = config.ConnectionConfig{}

	if _, err := auth.ResolveCredentials(cfg, config.CLIFlags{Connection: "missing.example.com"}); err == nil {
		t.Fatal("expected an error for a -c host that is not in the config")
	}
}

// TestResolveCredentials_ConnectionFlagPromptsWhenNoKey confirms the flag
// still falls through to the interactive login when no key is available.
func TestResolveCredentials_ConnectionFlagPromptsWhenNoKey(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Connections["target.example.com"] = config.ConnectionConfig{Username: "operator"}

	creds, err := auth.ResolveCredentials(cfg, config.CLIFlags{Connection: "target.example.com"})
	if err != nil {
		t.Fatalf("ResolveCredentials: %v", err)
	}
	if !creds.PromptForPassword {
		t.Error("PromptForPassword = false, want true when no API key is available")
	}
}
