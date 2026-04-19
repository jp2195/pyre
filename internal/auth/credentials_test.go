package auth_test

import (
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/jp2195/pyre/internal/auth"
	"github.com/jp2195/pyre/internal/config"
)

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

// TestResolveCredentials_HostEnvVarWins exercises priority rule 3 in the
// documented resolution order: the PYRE_<HOST>_API_KEY environment variable
// is consulted after the CLI flag and PYRE_API_KEY but before the keychain.
// With no CLI flag and no global env var, the per-host env var must win.
func TestResolveCredentials_HostEnvVarWins(t *testing.T) {
	keyring.MockInit()
	host := "fw1.example.com"

	t.Setenv("PYRE_API_KEY", "")
	t.Setenv("PYRE_FW1_EXAMPLE_COM_API_KEY", "env-key")

	creds := auth.ResolveCredentials(newConfigWithHost(host), config.CLIFlags{})
	if creds.APIKey != "env-key" {
		t.Fatalf("APIKey = %q, want %q", creds.APIKey, "env-key")
	}
	if creds.PromptForPassword {
		t.Error("PromptForPassword must be false when an API key was resolved")
	}
}

// TestResolveCredentials_KeychainFallback asserts that when no env var is
// set, the keychain entry (priority 4) is used.
func TestResolveCredentials_KeychainFallback(t *testing.T) {
	keyring.MockInit()
	host := "fw2.example.com"

	// No per-host env var, no global env var.
	t.Setenv("PYRE_API_KEY", "")
	t.Setenv("PYRE_FW2_EXAMPLE_COM_API_KEY", "")

	if err := config.SetAPIKey(host, "keychain-key"); err != nil {
		t.Fatalf("seeding keychain: %v", err)
	}

	creds := auth.ResolveCredentials(newConfigWithHost(host), config.CLIFlags{})
	if creds.APIKey != "keychain-key" {
		t.Fatalf("APIKey = %q, want %q", creds.APIKey, "keychain-key")
	}
	if creds.PromptForPassword {
		t.Error("PromptForPassword must be false when keychain supplies a key")
	}
}

// TestResolveCredentials_NoKeyPromptsForPassword asserts that when neither
// env var nor keychain supplies a key, ResolveCredentials signals the TUI
// prompt flow via PromptForPassword=true (priority 5).
func TestResolveCredentials_NoKeyPromptsForPassword(t *testing.T) {
	keyring.MockInit()
	host := "fw3.example.com"

	t.Setenv("PYRE_API_KEY", "")
	t.Setenv("PYRE_FW3_EXAMPLE_COM_API_KEY", "")
	// MockInit starts with an empty store, so GetAPIKey will return
	// ErrCredentialNotFound for this host.

	creds := auth.ResolveCredentials(newConfigWithHost(host), config.CLIFlags{})
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

// TestResolveCredentials_EnvVarBeatsKeychain enforces the ordering: when
// both a per-host env var and a keychain entry exist, the env var wins.
func TestResolveCredentials_EnvVarBeatsKeychain(t *testing.T) {
	keyring.MockInit()
	host := "fw4.example.com"

	t.Setenv("PYRE_API_KEY", "")
	t.Setenv("PYRE_FW4_EXAMPLE_COM_API_KEY", "env-wins")
	if err := config.SetAPIKey(host, "keychain-loses"); err != nil {
		t.Fatalf("seeding keychain: %v", err)
	}

	creds := auth.ResolveCredentials(newConfigWithHost(host), config.CLIFlags{})
	if creds.APIKey != "env-wins" {
		t.Fatalf("APIKey = %q, want %q (env var must beat keychain)", creds.APIKey, "env-wins")
	}
}

// TestConnection_ClearsCredentialsOnRemove verifies that RemoveConnection
// zeros the APIKey / Password fields on the previously-added *Connection so
// any surviving reference in caller code stops pointing at the secret. The
// keychain (which holds the persistent copy) is not touched here.
func TestConnection_ClearsCredentialsOnRemove(t *testing.T) {
	keyring.MockInit()

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

	// Session state bookkeeping: no active connection should remain when
	// the only connection is removed.
	if got := session.GetActiveConnection(); got != nil {
		t.Errorf("GetActiveConnection after RemoveConnection = %+v, want nil", got)
	}
}
