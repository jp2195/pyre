package auth

import (
	"testing"

	"github.com/jp2195/pyre/internal/config"
	"github.com/jp2195/pyre/internal/models"
)

func TestNewSession(t *testing.T) {
	cfg := config.DefaultConfig()
	session := NewSession(cfg)

	if session == nil {
		t.Fatal("expected non-nil session")
	}
	if session.Connections == nil {
		t.Error("expected Connections map to be initialized")
	}
	if session.Config != cfg {
		t.Error("expected Config to match")
	}
	if session.ActiveFirewall != "" {
		t.Error("expected ActiveFirewall to be empty")
	}
}

func TestSession_AddConnection(t *testing.T) {
	cfg := config.DefaultConfig()
	session := NewSession(cfg)

	fwConfig := &config.ConnectionConfig{
		Insecure: true,
	}

	conn, err := session.AddConnection("10.0.0.1", fwConfig, "test-api-key")
	if err != nil {
		t.Fatalf("AddConnection returned error: %v", err)
	}

	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Host != "10.0.0.1" {
		t.Errorf("expected host '10.0.0.1', got %q", conn.Host)
	}
	if conn.APIKey != "test-api-key" {
		t.Errorf("expected API key 'test-api-key', got %q", conn.APIKey)
	}
	if !conn.Connected {
		t.Error("expected Connected to be true")
	}
	if session.ActiveFirewall != "10.0.0.1" {
		t.Errorf("expected ActiveFirewall '10.0.0.1', got %q", session.ActiveFirewall)
	}
}

func TestSession_GetActiveConnection(t *testing.T) {
	cfg := config.DefaultConfig()
	session := NewSession(cfg)

	// No active connection
	conn := session.GetActiveConnection()
	if conn != nil {
		t.Error("expected nil when no active connection")
	}

	// Add a connection
	fwConfig := &config.ConnectionConfig{}
	_, _ = session.AddConnection("10.0.0.1", fwConfig, "api-key")

	conn = session.GetActiveConnection()
	if conn == nil {
		t.Fatal("expected non-nil active connection")
	}
	if conn.Host != "10.0.0.1" {
		t.Errorf("expected host '10.0.0.1', got %q", conn.Host)
	}
}

func TestSession_SetActiveFirewall(t *testing.T) {
	cfg := config.DefaultConfig()
	session := NewSession(cfg)

	fwConfig := &config.ConnectionConfig{}
	_, _ = session.AddConnection("10.0.0.1", fwConfig, "key1")
	_, _ = session.AddConnection("10.0.0.2", fwConfig, "key2")

	// Set active to second host
	ok := session.SetActiveFirewall("10.0.0.2")
	if !ok {
		t.Error("expected SetActiveFirewall to succeed")
	}
	if session.ActiveFirewall != "10.0.0.2" {
		t.Errorf("expected ActiveFirewall '10.0.0.2', got %q", session.ActiveFirewall)
	}

	// Try to set non-existent firewall
	ok = session.SetActiveFirewall("nonexistent")
	if ok {
		t.Error("expected SetActiveFirewall to fail for nonexistent firewall")
	}
	// Active should remain 10.0.0.2
	if session.ActiveFirewall != "10.0.0.2" {
		t.Errorf("expected ActiveFirewall to remain '10.0.0.2', got %q", session.ActiveFirewall)
	}
}

func TestSession_RemoveConnection(t *testing.T) {
	cfg := config.DefaultConfig()
	session := NewSession(cfg)

	fwConfig := &config.ConnectionConfig{}
	_, _ = session.AddConnection("10.0.0.1", fwConfig, "key1")
	_, _ = session.AddConnection("10.0.0.2", fwConfig, "key2")

	// Remove active connection
	session.RemoveConnection("10.0.0.1")

	if _, ok := session.Connections["10.0.0.1"]; ok {
		t.Error("expected 10.0.0.1 to be removed")
	}
	// Active should switch to remaining connection
	if session.ActiveFirewall != "10.0.0.2" {
		t.Errorf("expected ActiveFirewall to switch to '10.0.0.2', got %q", session.ActiveFirewall)
	}

	// Remove last connection
	session.RemoveConnection("10.0.0.2")
	if session.ActiveFirewall != "" {
		t.Errorf("expected ActiveFirewall to be empty, got %q", session.ActiveFirewall)
	}
}

func TestSession_ListConnections(t *testing.T) {
	cfg := config.DefaultConfig()
	session := NewSession(cfg)

	// Empty list
	conns := session.ListConnections()
	if len(conns) != 0 {
		t.Errorf("expected 0 connections, got %d", len(conns))
	}

	// Add connections
	fwConfig := &config.ConnectionConfig{}
	_, _ = session.AddConnection("10.0.0.1", fwConfig, "key1")
	_, _ = session.AddConnection("10.0.0.2", fwConfig, "key2")

	conns = session.ListConnections()
	if len(conns) != 2 {
		t.Errorf("expected 2 connections, got %d", len(conns))
	}
}

func TestSession_IsConnected(t *testing.T) {
	cfg := config.DefaultConfig()
	session := NewSession(cfg)

	// Not connected
	if session.IsConnected("10.0.0.1") {
		t.Error("expected IsConnected to be false for nonexistent firewall")
	}

	// Add connection
	fwConfig := &config.ConnectionConfig{}
	_, _ = session.AddConnection("10.0.0.1", fwConfig, "key1")

	if !session.IsConnected("10.0.0.1") {
		t.Error("expected IsConnected to be true")
	}
}

func TestResolveCredentials(t *testing.T) {
	cfg := config.DefaultConfig()

	// Test with flags
	flags := config.CLIFlags{
		Host:     "10.0.0.1",
		APIKey:   "flag-api-key",
		Insecure: true,
	}

	creds, err := ResolveCredentials(cfg, flags)
	if err != nil {
		t.Fatalf("ResolveCredentials: %v", err)
	}

	if creds.Host != "10.0.0.1" {
		t.Errorf("expected Host '10.0.0.1', got %q", creds.Host)
	}
	if creds.APIKey != "flag-api-key" {
		t.Errorf("expected APIKey 'flag-api-key', got %q", creds.APIKey)
	}
	if !creds.Insecure {
		t.Error("expected Insecure to be true")
	}
}

func TestResolveCredentials_EnvVars(t *testing.T) {
	cfg := config.DefaultConfig()
	flags := config.CLIFlags{}

	t.Setenv("PYRE_HOST", "env-host")
	t.Setenv("PYRE_API_KEY", "env-api-key")
	t.Setenv("PYRE_INSECURE", "true")

	creds, err := ResolveCredentials(cfg, flags)
	if err != nil {
		t.Fatalf("ResolveCredentials: %v", err)
	}

	if creds.Host != "env-host" {
		t.Errorf("expected Host 'env-host', got %q", creds.Host)
	}
	if creds.APIKey != "env-api-key" {
		t.Errorf("expected APIKey 'env-api-key', got %q", creds.APIKey)
	}
	if !creds.Insecure {
		t.Error("expected Insecure to be true from env")
	}
}

func TestResolveCredentials_ConfigDefault(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Default = "config-host"
	cfg.Connections["config-host"] = config.ConnectionConfig{
		Insecure: true,
	}

	flags := config.CLIFlags{}

	creds, err := ResolveCredentials(cfg, flags)
	if err != nil {
		t.Fatalf("ResolveCredentials: %v", err)
	}

	if creds.Host != "config-host" {
		t.Errorf("expected Host 'config-host', got %q", creds.Host)
	}
	if !creds.Insecure {
		t.Error("expected Insecure to be true from config")
	}
}

func TestCredentials_Methods(t *testing.T) {
	tests := []struct {
		name          string
		creds         Credentials
		wantHasHost   bool
		wantHasAPIKey bool
	}{
		{
			name:          "empty credentials",
			creds:         Credentials{},
			wantHasHost:   false,
			wantHasAPIKey: false,
		},
		{
			name:          "host only",
			creds:         Credentials{Host: "10.0.0.1"},
			wantHasHost:   true,
			wantHasAPIKey: false,
		},
		{
			name:          "api key only",
			creds:         Credentials{APIKey: "key"},
			wantHasHost:   false,
			wantHasAPIKey: true,
		},
		{
			name:          "complete credentials",
			creds:         Credentials{Host: "10.0.0.1", APIKey: "key"},
			wantHasHost:   true,
			wantHasAPIKey: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.creds.HasHost(); got != tt.wantHasHost {
				t.Errorf("HasHost() = %v, want %v", got, tt.wantHasHost)
			}
			if got := tt.creds.HasAPIKey(); got != tt.wantHasAPIKey {
				t.Errorf("HasAPIKey() = %v, want %v", got, tt.wantHasAPIKey)
			}
		})
	}
}

func TestConnection_GetTargetDevice(t *testing.T) {
	conn := &Connection{
		Host:         "10.0.0.1",
		TargetSerial: "",
		ManagedDevices: []models.ManagedDevice{
			{Serial: "serial1", Hostname: "fw1", IPAddress: "10.0.0.1"},
			{Serial: "serial2", Hostname: "fw2", IPAddress: "10.0.0.2"},
		},
	}

	// No target set
	device := conn.GetTargetDevice()
	if device != nil {
		t.Error("expected nil when no target set")
	}

	// Set target
	conn.TargetSerial = "serial1"
	device = conn.GetTargetDevice()
	if device == nil {
		t.Fatal("expected non-nil device")
	}
	if device.Hostname != "fw1" {
		t.Errorf("expected hostname 'fw1', got %q", device.Hostname)
	}

	// Invalid target
	conn.TargetSerial = "nonexistent"
	device = conn.GetTargetDevice()
	if device != nil {
		t.Error("expected nil for nonexistent target")
	}
}

func TestConnection_ConnectedDeviceCount(t *testing.T) {
	conn := &Connection{
		ManagedDevices: []models.ManagedDevice{
			{Serial: "s1", Connected: true},
			{Serial: "s2", Connected: false},
			{Serial: "s3", Connected: true},
			{Serial: "s4", Connected: false},
		},
	}

	count := conn.ConnectedDeviceCount()
	if count != 2 {
		t.Errorf("expected 2 connected devices, got %d", count)
	}
}

// TestSession_AddConnection_MakesNewConnectionActive pins the invariant that
// connecting to a host focuses it. Claiming the active slot only when it was
// empty left a second login pointed at the first firewall.
func TestSession_AddConnection_MakesNewConnectionActive(t *testing.T) {
	session := NewSession(config.DefaultConfig())
	fwConfig := &config.ConnectionConfig{}

	if _, err := session.AddConnection("10.0.0.1", fwConfig, "key1"); err != nil {
		t.Fatalf("AddConnection: %v", err)
	}
	if _, err := session.AddConnection("10.0.0.2", fwConfig, "key2"); err != nil {
		t.Fatalf("AddConnection: %v", err)
	}

	if session.ActiveFirewall != "10.0.0.2" {
		t.Errorf("ActiveFirewall = %q, want 10.0.0.2 (the most recently connected host)", session.ActiveFirewall)
	}
	conn := session.GetActiveConnection()
	if conn == nil || conn.Host != "10.0.0.2" {
		t.Errorf("GetActiveConnection did not return the newly added connection")
	}
}
