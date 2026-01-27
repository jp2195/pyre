package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	if cfg.Connections == nil {
		t.Error("expected Connections map to be initialized")
	}

	if cfg.Settings.SessionPageSize != 50 {
		t.Errorf("expected SessionPageSize 50, got %d", cfg.Settings.SessionPageSize)
	}

	if cfg.Settings.Theme != "default" {
		t.Errorf("expected Theme 'default', got %q", cfg.Settings.Theme)
	}

	if cfg.Settings.DefaultView != "dashboard" {
		t.Errorf("expected DefaultView 'dashboard', got %q", cfg.Settings.DefaultView)
	}
}

func TestConfig_GetConnection(t *testing.T) {
	cfg := DefaultConfig()
	// Host is now the key
	cfg.Connections["192.168.1.1"] = ConnectionConfig{
		Insecure: true,
	}

	// Test existing connection
	conn, ok := cfg.GetConnection("192.168.1.1")
	if !ok {
		t.Error("expected to find 192.168.1.1")
	}
	if !conn.Insecure {
		t.Error("expected Insecure to be true")
	}

	// Test non-existing connection
	_, ok = cfg.GetConnection("nonexistent")
	if ok {
		t.Error("expected not to find nonexistent connection")
	}
}

func TestConfig_GetDefaultConnection(t *testing.T) {
	cfg := DefaultConfig()

	// Test with no default set
	host, _, ok := cfg.GetDefaultConnection()
	if ok {
		t.Error("expected no default connection")
	}
	if host != "" {
		t.Errorf("expected empty host, got %q", host)
	}

	// Test with default set (host is now the key)
	cfg.Connections["10.0.0.1"] = ConnectionConfig{}
	cfg.Default = "10.0.0.1"

	host, _, ok = cfg.GetDefaultConnection()
	if !ok {
		t.Error("expected to find default connection")
	}
	if host != "10.0.0.1" {
		t.Errorf("expected host '10.0.0.1', got %q", host)
	}

	// Test with invalid default
	cfg.Default = "nonexistent"
	_, _, ok = cfg.GetDefaultConnection()
	if ok {
		t.Error("expected not to find invalid default connection")
	}
}

func TestConfig_ApplyFlags(t *testing.T) {
	cfg := DefaultConfig()

	// Test with no flags
	flags := CLIFlags{}
	cfg.ApplyFlags(flags)
	if cfg.Default != "" {
		t.Error("expected no default connection with empty flags")
	}

	// Test with host flag
	flags = CLIFlags{
		Host:     "192.168.1.100",
		Insecure: true,
	}
	cfg.ApplyFlags(flags)

	// Default should now be the host itself
	if cfg.Default != "192.168.1.100" {
		t.Errorf("expected default connection '192.168.1.100', got %q", cfg.Default)
	}

	conn, ok := cfg.GetConnection("192.168.1.100")
	if !ok {
		t.Error("expected to find '192.168.1.100' connection")
	}
	if !conn.Insecure {
		t.Error("expected Insecure to be true")
	}
}

func TestLoad_NoConfigFile(t *testing.T) {
	// This test relies on the config file not existing at the default location
	// or the test running in an environment without ~/.pyre.yaml
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	// Should return default config
	if cfg.Settings.SessionPageSize != 50 {
		t.Errorf("expected default SessionPageSize 50, got %d", cfg.Settings.SessionPageSize)
	}
}

func TestLoadWithFlags_CustomConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-pyre.yaml")

	// Host is now the key in the YAML
	configContent := `
default: 10.0.0.1
connections:
  10.0.0.1:
    insecure: true
settings:
  session_page_size: 100
  theme: dark
  default_view: policies
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	flags := CLIFlags{
		Config: configPath,
	}

	cfg, err := LoadWithFlags(flags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Default != "10.0.0.1" {
		t.Errorf("expected Default '10.0.0.1', got %q", cfg.Default)
	}

	conn, ok := cfg.GetConnection("10.0.0.1")
	if !ok {
		t.Error("expected to find 10.0.0.1")
	}
	if !conn.Insecure {
		t.Error("expected Insecure to be true")
	}

	if cfg.Settings.SessionPageSize != 100 {
		t.Errorf("expected SessionPageSize 100, got %d", cfg.Settings.SessionPageSize)
	}
	if cfg.Settings.Theme != "dark" {
		t.Errorf("expected Theme 'dark', got %q", cfg.Settings.Theme)
	}
	if cfg.Settings.DefaultView != "policies" {
		t.Errorf("expected DefaultView 'policies', got %q", cfg.Settings.DefaultView)
	}
}

func TestLoadWithFlags_InvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	// Write invalid YAML
	if err := os.WriteFile(configPath, []byte("invalid: yaml: content:"), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	flags := CLIFlags{
		Config: configPath,
	}

	_, err := LoadWithFlags(flags)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadWithFlags_NonexistentConfig(t *testing.T) {
	flags := CLIFlags{
		Config: "/nonexistent/path/config.yaml",
	}

	_, err := LoadWithFlags(flags)
	if err == nil {
		t.Error("expected error for nonexistent config file")
	}
}

func TestLoadWithFlags_FlagsOverrideConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	configContent := `
default: 10.0.0.1
connections:
  10.0.0.1:
    insecure: false
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Flags should override config
	flags := CLIFlags{
		Config:   configPath,
		Host:     "192.168.1.1",
		Insecure: true,
	}

	cfg, err := LoadWithFlags(flags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Default should be the host from flags
	if cfg.Default != "192.168.1.1" {
		t.Errorf("expected flags to override default, got %q", cfg.Default)
	}

	// Both connections should exist
	_, ok := cfg.GetConnection("10.0.0.1")
	if !ok {
		t.Error("expected 10.0.0.1 to still exist")
	}

	conn, ok := cfg.GetConnection("192.168.1.1")
	if !ok {
		t.Error("expected 192.168.1.1 connection from flags")
	}
	if !conn.Insecure {
		t.Error("expected Insecure from flags to be true")
	}
}

func TestSSHConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ssh-test.yaml")

	configContent := `
connections:
  10.0.0.1:
    ssh:
      port: 2222
      username: admin
      password: secret
      private_key_path: /path/to/key
      timeout: 60
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	flags := CLIFlags{Config: configPath}
	cfg, err := LoadWithFlags(flags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	conn, ok := cfg.GetConnection("10.0.0.1")
	if !ok {
		t.Fatal("expected to find 10.0.0.1")
	}

	if conn.SSH.Port != 2222 {
		t.Errorf("expected SSH port 2222, got %d", conn.SSH.Port)
	}
	if conn.SSH.Username != "admin" {
		t.Errorf("expected SSH username 'admin', got %q", conn.SSH.Username)
	}
	if conn.SSH.Password != "secret" {
		t.Errorf("expected SSH password 'secret', got %q", conn.SSH.Password)
	}
	if conn.SSH.PrivateKeyPath != "/path/to/key" {
		t.Errorf("expected SSH key path '/path/to/key', got %q", conn.SSH.PrivateKeyPath)
	}
	if conn.SSH.Timeout != 60 {
		t.Errorf("expected SSH timeout 60, got %d", conn.SSH.Timeout)
	}
}

func TestConnectionType(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "type-test.yaml")

	configContent := `
connections:
  10.0.0.1:
    type: panorama
  10.0.0.2:
    type: firewall
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	flags := CLIFlags{Config: configPath}
	cfg, err := LoadWithFlags(flags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	panorama, _ := cfg.GetConnection("10.0.0.1")
	if panorama.Type != "panorama" {
		t.Errorf("expected type 'panorama', got %q", panorama.Type)
	}

	firewall, _ := cfg.GetConnection("10.0.0.2")
	if firewall.Type != "firewall" {
		t.Errorf("expected type 'firewall', got %q", firewall.Type)
	}
}

func TestLoadWithFlags_DefaultLoad(t *testing.T) {
	// Test with empty flags (uses Load() internally)
	flags := CLIFlags{}
	cfg, err := LoadWithFlags(flags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestConfig_NilConnectionsAfterLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "minimal.yaml")

	// Config with no connections section
	configContent := `
settings:
  session_page_size: 100
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	flags := CLIFlags{Config: configPath}
	cfg, err := LoadWithFlags(flags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Connections should be initialized even if not in config
	if cfg.Connections == nil {
		t.Error("expected Connections map to be initialized")
	}
}

func TestLoad_ReadError(t *testing.T) {
	// Test Load() when config file exists but has read issues
	// This is harder to test directly, but we can test the path where
	// the file doesn't exist
	cfg, err := Load()
	if err != nil {
		// Error is acceptable if there's no home directory, etc.
		t.Logf("Load returned error (may be expected): %v", err)
	}
	if cfg == nil && err == nil {
		t.Error("expected either config or error")
	}
}

func TestCLIFlags_AllFields(t *testing.T) {
	flags := CLIFlags{
		Host:       "10.0.0.1",
		APIKey:     "test-key",
		Insecure:   true,
		Config:     "/path/to/config",
		Connection: "my-fw",
	}

	if flags.Host != "10.0.0.1" {
		t.Errorf("expected Host '10.0.0.1', got %q", flags.Host)
	}
	if flags.APIKey != "test-key" {
		t.Errorf("expected APIKey 'test-key', got %q", flags.APIKey)
	}
	if !flags.Insecure {
		t.Error("expected Insecure to be true")
	}
	if flags.Config != "/path/to/config" {
		t.Errorf("expected Config '/path/to/config', got %q", flags.Config)
	}
	if flags.Connection != "my-fw" {
		t.Errorf("expected Connection 'my-fw', got %q", flags.Connection)
	}
}

func TestSettings_AllFields(t *testing.T) {
	settings := Settings{
		SessionPageSize: 100,
		Theme:           "dark",
		DefaultView:     "policies",
	}

	if settings.SessionPageSize != 100 {
		t.Errorf("expected SessionPageSize 100, got %d", settings.SessionPageSize)
	}
	if settings.Theme != "dark" {
		t.Errorf("expected Theme 'dark', got %q", settings.Theme)
	}
	if settings.DefaultView != "policies" {
		t.Errorf("expected DefaultView 'policies', got %q", settings.DefaultView)
	}
}

func TestSSHConfig_AllFields(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ssh-all-fields.yaml")

	configContent := `
connections:
  10.0.0.1:
    insecure: true
    type: firewall
    ssh:
      port: 2222
      username: admin
      password: secret123
      private_key_path: /path/to/key
      timeout: 120
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	flags := CLIFlags{Config: configPath}
	cfg, err := LoadWithFlags(flags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	conn, ok := cfg.GetConnection("10.0.0.1")
	if !ok {
		t.Fatal("expected to find 10.0.0.1")
	}

	if !conn.Insecure {
		t.Error("expected Insecure to be true")
	}
	if conn.Type != "firewall" {
		t.Errorf("expected Type 'firewall', got %q", conn.Type)
	}
	if conn.SSH.Port != 2222 {
		t.Errorf("expected SSH Port 2222, got %d", conn.SSH.Port)
	}
	if conn.SSH.Username != "admin" {
		t.Errorf("expected SSH Username 'admin', got %q", conn.SSH.Username)
	}
	if conn.SSH.Password != "secret123" {
		t.Errorf("expected SSH Password 'secret123', got %q", conn.SSH.Password)
	}
	if conn.SSH.PrivateKeyPath != "/path/to/key" {
		t.Errorf("expected SSH PrivateKeyPath '/path/to/key', got %q", conn.SSH.PrivateKeyPath)
	}
	if conn.SSH.Timeout != 120 {
		t.Errorf("expected SSH Timeout 120, got %d", conn.SSH.Timeout)
	}
}

func TestConfig_ApplyFlags_NoHost(t *testing.T) {
	cfg := DefaultConfig()

	// Add existing connection
	cfg.Connections["10.0.0.1"] = ConnectionConfig{}
	cfg.Default = "10.0.0.1"

	// Apply empty flags - should not change anything
	flags := CLIFlags{}
	cfg.ApplyFlags(flags)

	if cfg.Default != "10.0.0.1" {
		t.Errorf("expected Default to remain '10.0.0.1', got %q", cfg.Default)
	}
}

func TestConfig_AddConnection(t *testing.T) {
	cfg := DefaultConfig()

	err := cfg.AddConnection("10.0.0.1", ConnectionConfig{Insecure: true})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	conn, ok := cfg.GetConnection("10.0.0.1")
	if !ok {
		t.Error("expected to find 10.0.0.1")
	}
	if !conn.Insecure {
		t.Error("expected Insecure to be true")
	}

	// Try to add duplicate
	err = cfg.AddConnection("10.0.0.1", ConnectionConfig{})
	if err == nil {
		t.Error("expected error for duplicate connection")
	}
}

func TestConfig_UpdateConnection(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Connections["10.0.0.1"] = ConnectionConfig{}

	err := cfg.UpdateConnection("10.0.0.1", ConnectionConfig{Insecure: true})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	conn, _ := cfg.GetConnection("10.0.0.1")
	if !conn.Insecure {
		t.Error("expected Insecure to be true")
	}

	// Try to update nonexistent
	err = cfg.UpdateConnection("nonexistent", ConnectionConfig{})
	if err == nil {
		t.Error("expected error for nonexistent connection")
	}
}

func TestConfig_DeleteConnection(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Connections["10.0.0.1"] = ConnectionConfig{}
	cfg.Default = "10.0.0.1"

	err := cfg.DeleteConnection("10.0.0.1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	_, ok := cfg.GetConnection("10.0.0.1")
	if ok {
		t.Error("expected 10.0.0.1 to be deleted")
	}

	// Default should be cleared
	if cfg.Default != "" {
		t.Errorf("expected Default to be cleared, got %q", cfg.Default)
	}

	// Try to delete nonexistent
	err = cfg.DeleteConnection("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent connection")
	}
}

func TestConfig_HasConnections(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.HasConnections() {
		t.Error("expected HasConnections to be false for empty config")
	}

	cfg.Connections["10.0.0.1"] = ConnectionConfig{}

	if !cfg.HasConnections() {
		t.Error("expected HasConnections to be true")
	}
}

func TestConfig_ConnectionHosts(t *testing.T) {
	cfg := DefaultConfig()

	hosts := cfg.ConnectionHosts()
	if len(hosts) != 0 {
		t.Errorf("expected 0 hosts, got %d", len(hosts))
	}

	cfg.Connections["10.0.0.1"] = ConnectionConfig{}
	cfg.Connections["10.0.0.2"] = ConnectionConfig{}

	hosts = cfg.ConnectionHosts()
	if len(hosts) != 2 {
		t.Errorf("expected 2 hosts, got %d", len(hosts))
	}
}
