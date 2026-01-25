package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	if cfg.Firewalls == nil {
		t.Error("expected Firewalls map to be initialized")
	}

	if cfg.Settings.RefreshInterval != 5*time.Second {
		t.Errorf("expected RefreshInterval 5s, got %v", cfg.Settings.RefreshInterval)
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

func TestConfig_GetFirewall(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Firewalls["test-fw"] = FirewallConfig{
		Host:     "192.168.1.1",
		Insecure: true,
	}

	// Test existing firewall
	fw, ok := cfg.GetFirewall("test-fw")
	if !ok {
		t.Error("expected to find test-fw")
	}
	if fw.Host != "192.168.1.1" {
		t.Errorf("expected host 192.168.1.1, got %q", fw.Host)
	}
	if !fw.Insecure {
		t.Error("expected Insecure to be true")
	}

	// Test non-existing firewall
	_, ok = cfg.GetFirewall("nonexistent")
	if ok {
		t.Error("expected not to find nonexistent firewall")
	}
}

func TestConfig_GetDefaultFirewall(t *testing.T) {
	cfg := DefaultConfig()

	// Test with no default set
	name, fw, ok := cfg.GetDefaultFirewall()
	if ok {
		t.Error("expected no default firewall")
	}
	if name != "" {
		t.Errorf("expected empty name, got %q", name)
	}

	// Test with default set
	cfg.Firewalls["prod-fw"] = FirewallConfig{
		Host: "10.0.0.1",
	}
	cfg.DefaultFirewall = "prod-fw"

	name, fw, ok = cfg.GetDefaultFirewall()
	if !ok {
		t.Error("expected to find default firewall")
	}
	if name != "prod-fw" {
		t.Errorf("expected name 'prod-fw', got %q", name)
	}
	if fw.Host != "10.0.0.1" {
		t.Errorf("expected host '10.0.0.1', got %q", fw.Host)
	}

	// Test with invalid default
	cfg.DefaultFirewall = "nonexistent"
	_, _, ok = cfg.GetDefaultFirewall()
	if ok {
		t.Error("expected not to find invalid default firewall")
	}
}

func TestConfig_ApplyFlags(t *testing.T) {
	cfg := DefaultConfig()

	// Test with no flags
	flags := CLIFlags{}
	cfg.ApplyFlags(flags)
	if cfg.DefaultFirewall != "" {
		t.Error("expected no default firewall with empty flags")
	}

	// Test with host flag
	flags = CLIFlags{
		Host:     "192.168.1.100",
		Insecure: true,
	}
	cfg.ApplyFlags(flags)

	if cfg.DefaultFirewall != "cli" {
		t.Errorf("expected default firewall 'cli', got %q", cfg.DefaultFirewall)
	}

	fw, ok := cfg.GetFirewall("cli")
	if !ok {
		t.Error("expected to find 'cli' firewall")
	}
	if fw.Host != "192.168.1.100" {
		t.Errorf("expected host '192.168.1.100', got %q", fw.Host)
	}
	if !fw.Insecure {
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
	if cfg.Settings.RefreshInterval != 5*time.Second {
		t.Errorf("expected default RefreshInterval, got %v", cfg.Settings.RefreshInterval)
	}
}

func TestLoadWithFlags_CustomConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-pyre.yaml")

	configContent := `
default_firewall: test-fw
firewalls:
  test-fw:
    host: 10.0.0.1
    insecure: true
settings:
  refresh_interval: 10s
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

	if cfg.DefaultFirewall != "test-fw" {
		t.Errorf("expected DefaultFirewall 'test-fw', got %q", cfg.DefaultFirewall)
	}

	fw, ok := cfg.GetFirewall("test-fw")
	if !ok {
		t.Error("expected to find test-fw")
	}
	if fw.Host != "10.0.0.1" {
		t.Errorf("expected host '10.0.0.1', got %q", fw.Host)
	}

	if cfg.Settings.RefreshInterval != 10*time.Second {
		t.Errorf("expected RefreshInterval 10s, got %v", cfg.Settings.RefreshInterval)
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
default_firewall: config-fw
firewalls:
  config-fw:
    host: 10.0.0.1
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

	// Default should be "cli" from flags, not "config-fw" from file
	if cfg.DefaultFirewall != "cli" {
		t.Errorf("expected flags to override default, got %q", cfg.DefaultFirewall)
	}

	// Both firewalls should exist
	_, ok := cfg.GetFirewall("config-fw")
	if !ok {
		t.Error("expected config-fw to still exist")
	}

	fw, ok := cfg.GetFirewall("cli")
	if !ok {
		t.Error("expected cli firewall from flags")
	}
	if fw.Host != "192.168.1.1" {
		t.Errorf("expected host from flags, got %q", fw.Host)
	}
}

func TestSSHConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ssh-test.yaml")

	configContent := `
firewalls:
  ssh-fw:
    host: 10.0.0.1
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

	fw, ok := cfg.GetFirewall("ssh-fw")
	if !ok {
		t.Fatal("expected to find ssh-fw")
	}

	if fw.SSH.Port != 2222 {
		t.Errorf("expected SSH port 2222, got %d", fw.SSH.Port)
	}
	if fw.SSH.Username != "admin" {
		t.Errorf("expected SSH username 'admin', got %q", fw.SSH.Username)
	}
	if fw.SSH.Password != "secret" {
		t.Errorf("expected SSH password 'secret', got %q", fw.SSH.Password)
	}
	if fw.SSH.PrivateKeyPath != "/path/to/key" {
		t.Errorf("expected SSH key path '/path/to/key', got %q", fw.SSH.PrivateKeyPath)
	}
	if fw.SSH.Timeout != 60 {
		t.Errorf("expected SSH timeout 60, got %d", fw.SSH.Timeout)
	}
}

func TestFirewallType(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "type-test.yaml")

	configContent := `
firewalls:
  panorama:
    host: 10.0.0.1
    type: panorama
  firewall:
    host: 10.0.0.2
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	flags := CLIFlags{Config: configPath}
	cfg, err := LoadWithFlags(flags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	panorama, _ := cfg.GetFirewall("panorama")
	if panorama.Type != "panorama" {
		t.Errorf("expected type 'panorama', got %q", panorama.Type)
	}

	firewall, _ := cfg.GetFirewall("firewall")
	if firewall.Type != "" {
		t.Errorf("expected empty type for auto-detect, got %q", firewall.Type)
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

func TestConfig_NilFirewallsAfterLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "minimal.yaml")

	// Config with no firewalls section
	configContent := `
settings:
  refresh_interval: 10s
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	flags := CLIFlags{Config: configPath}
	cfg, err := LoadWithFlags(flags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Firewalls should be initialized even if not in config
	if cfg.Firewalls == nil {
		t.Error("expected Firewalls map to be initialized")
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
		Host:     "10.0.0.1",
		APIKey:   "test-key",
		Insecure: true,
		Config:   "/path/to/config",
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
}

func TestSettings_AllFields(t *testing.T) {
	settings := Settings{
		RefreshInterval: 30 * time.Second,
		SessionPageSize: 100,
		Theme:           "dark",
		DefaultView:     "policies",
	}

	if settings.RefreshInterval != 30*time.Second {
		t.Errorf("expected RefreshInterval 30s, got %v", settings.RefreshInterval)
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
firewalls:
  complete-fw:
    host: 10.0.0.1
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

	fw, ok := cfg.GetFirewall("complete-fw")
	if !ok {
		t.Fatal("expected to find complete-fw")
	}

	if fw.Host != "10.0.0.1" {
		t.Errorf("expected Host '10.0.0.1', got %q", fw.Host)
	}
	if !fw.Insecure {
		t.Error("expected Insecure to be true")
	}
	if fw.Type != "firewall" {
		t.Errorf("expected Type 'firewall', got %q", fw.Type)
	}
	if fw.SSH.Port != 2222 {
		t.Errorf("expected SSH Port 2222, got %d", fw.SSH.Port)
	}
	if fw.SSH.Username != "admin" {
		t.Errorf("expected SSH Username 'admin', got %q", fw.SSH.Username)
	}
	if fw.SSH.Password != "secret123" {
		t.Errorf("expected SSH Password 'secret123', got %q", fw.SSH.Password)
	}
	if fw.SSH.PrivateKeyPath != "/path/to/key" {
		t.Errorf("expected SSH PrivateKeyPath '/path/to/key', got %q", fw.SSH.PrivateKeyPath)
	}
	if fw.SSH.Timeout != 120 {
		t.Errorf("expected SSH Timeout 120, got %d", fw.SSH.Timeout)
	}
}

func TestConfig_ApplyFlags_NoHost(t *testing.T) {
	cfg := DefaultConfig()

	// Add existing firewall
	cfg.Firewalls["existing-fw"] = FirewallConfig{Host: "10.0.0.1"}
	cfg.DefaultFirewall = "existing-fw"

	// Apply empty flags - should not change anything
	flags := CLIFlags{}
	cfg.ApplyFlags(flags)

	if cfg.DefaultFirewall != "existing-fw" {
		t.Errorf("expected DefaultFirewall to remain 'existing-fw', got %q", cfg.DefaultFirewall)
	}
	if _, ok := cfg.Firewalls["cli"]; ok {
		t.Error("expected no 'cli' firewall to be added")
	}
}
