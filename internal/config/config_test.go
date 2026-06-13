package config

import (
	"bytes"
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

	if cfg.Settings.Theme != "default" {
		t.Errorf("expected Theme 'default', got %q", cfg.Settings.Theme)
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
	if cfg.Settings.Theme != "default" {
		t.Errorf("expected default Theme 'default', got %q", cfg.Settings.Theme)
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
  theme: dark
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

	if cfg.Settings.Theme != "dark" {
		t.Errorf("expected Theme 'dark', got %q", cfg.Settings.Theme)
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
  theme: dark
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
		Theme: "dark",
	}

	if settings.Theme != "dark" {
		t.Errorf("expected Theme 'dark', got %q", settings.Theme)
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

// TestConfig_DoesNotPersistCredentials is a regression guard for the
// contract documented on ConnectionConfig: APIKey and Password carry
// `yaml:"-"` tags and must never round-trip to ~/.pyre.yaml. If someone
// later removes those tags (or adds a new credential field without one),
// this test fires.
func TestConfig_DoesNotPersistCredentials(t *testing.T) {
	const (
		secretKey = "SECRET-KEY-ABCD1234"
		secretPwd = "hunter2"
	)

	// Redirect HOME so Save() writes into a test-scoped directory. Save()
	// resolves the destination via os.UserHomeDir() → ~/.pyre.yaml.
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	// macOS honours $HOME for UserHomeDir(); set USERPROFILE too so the
	// same test works on Windows if the suite is ever cross-run.
	t.Setenv("USERPROFILE", tmpDir)

	cfg := DefaultConfig()
	cfg.Default = "10.0.0.1"
	cfg.Connections["10.0.0.1"] = ConnectionConfig{
		Username: "admin",
		APIKey:   secretKey,
		Password: secretPwd,
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	configPath := filepath.Join(tmpDir, ".pyre.yaml")
	data, err := os.ReadFile(configPath) // #nosec G304 -- test-controlled path
	if err != nil {
		t.Fatalf("reading saved config: %v", err)
	}

	if bytes.Contains(data, []byte(secretKey)) {
		t.Errorf("API key leaked to disk; file contents:\n%s", data)
	}
	if bytes.Contains(data, []byte(secretPwd)) {
		t.Errorf("password leaked to disk; file contents:\n%s", data)
	}

	// Load the file back and confirm credentials are empty on the
	// round-tripped struct.
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	conn, ok := loaded.GetConnection("10.0.0.1")
	if !ok {
		t.Fatalf("expected 10.0.0.1 connection after reload")
	}
	if conn.APIKey != "" {
		t.Errorf("APIKey round-tripped from disk: %q", conn.APIKey)
	}
	if conn.Password != "" {
		t.Errorf("Password round-tripped from disk: %q", conn.Password)
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

func TestAtomicWriteFile_WritesContentAndPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")

	if err := atomicWriteFile(path, []byte("hello: world\n"), 0600); err != nil {
		t.Fatalf("atomicWriteFile: %v", err)
	}

	data, err := os.ReadFile(path) // #nosec G304 -- test-controlled path
	if err != nil {
		t.Fatalf("reading result: %v", err)
	}
	if string(data) != "hello: world\n" {
		t.Errorf("content = %q, want %q", data, "hello: world\n")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("mode = %#o, want 0600", info.Mode().Perm())
	}
}

func TestAtomicWriteFile_OverwritesExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")
	if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
		t.Fatalf("seeding file: %v", err)
	}

	if err := atomicWriteFile(path, []byte("new"), 0600); err != nil {
		t.Fatalf("atomicWriteFile: %v", err)
	}

	data, err := os.ReadFile(path) // #nosec G304 -- test-controlled path
	if err != nil {
		t.Fatalf("reading result: %v", err)
	}
	if string(data) != "new" {
		t.Errorf("content = %q, want %q", data, "new")
	}
}

// TestAtomicWriteFile_LeavesNoTempFiles guards the rename-into-place contract:
// after a successful write the directory holds exactly the target file.
func TestAtomicWriteFile_LeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")

	if err := atomicWriteFile(path, []byte("x"), 0600); err != nil {
		t.Fatalf("atomicWriteFile: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "out.yaml" {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("directory contents = %v, want exactly [out.yaml]", names)
	}
}

func TestAtomicWriteFile_ErrorOnMissingDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-such-subdir", "out.yaml")

	err := atomicWriteFile(path, []byte("x"), 0600)
	if err == nil {
		t.Fatal("expected error writing into nonexistent directory")
	}
}

// TestConfig_Save_CreatesBackup guards Save's backup contract: when
// ~/.pyre.yaml already exists, its prior contents are preserved in
// ~/.pyre.yaml.bak before the new contents land.
func TestConfig_Save_CreatesBackup(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	configPath := filepath.Join(tmpDir, ".pyre.yaml")
	if err := os.WriteFile(configPath, []byte("default: old-host\n"), 0600); err != nil {
		t.Fatalf("seeding config: %v", err)
	}

	cfg := DefaultConfig()
	cfg.Default = "new-host"
	cfg.Connections["new-host"] = ConnectionConfig{}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	backup, err := os.ReadFile(configPath + ".bak") // #nosec G304 -- test-controlled path
	if err != nil {
		t.Fatalf("reading backup: %v", err)
	}
	if string(backup) != "default: old-host\n" {
		t.Errorf("backup = %q, want prior contents", backup)
	}

	current, err := os.ReadFile(configPath) // #nosec G304 -- test-controlled path
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	if !bytes.Contains(current, []byte("new-host")) {
		t.Errorf("saved config missing new content:\n%s", current)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("config mode = %#o, want 0600", info.Mode().Perm())
	}
}

func TestConfig_Save_NoBackupOnFirstWrite(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	cfg := DefaultConfig()
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	_, err := os.Stat(filepath.Join(tmpDir, ".pyre.yaml.bak"))
	if err == nil {
		t.Error("expected no backup file on first save, but it exists")
	} else if !os.IsNotExist(err) {
		t.Errorf("unexpected stat error (neither nil nor not-exist): %v", err)
	}
}

func TestConfig_CRUD_NilConnectionsMap(t *testing.T) {
	// A zero-value Config (nil Connections) must behave: Add initializes
	// the map; Update and Delete report not-found instead of panicking.
	c := &Config{}
	if err := c.AddConnection("10.0.0.1", ConnectionConfig{}); err != nil {
		t.Errorf("AddConnection on nil map: %v", err)
	}
	if _, ok := c.GetConnection("10.0.0.1"); !ok {
		t.Error("expected connection after Add on nil map")
	}

	c2 := &Config{}
	if err := c2.UpdateConnection("10.0.0.1", ConnectionConfig{}); err == nil {
		t.Error("expected error from UpdateConnection on nil map")
	}
	c3 := &Config{}
	if err := c3.DeleteConnection("10.0.0.1"); err == nil {
		t.Error("expected error from DeleteConnection on nil map")
	}
}
