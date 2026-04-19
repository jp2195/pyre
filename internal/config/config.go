package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v4"
)

// Config represents the application configuration
type Config struct {
	Default     string                      `yaml:"default,omitempty"`
	Connections map[string]ConnectionConfig `yaml:"connections,omitempty"`
	Settings    Settings                    `yaml:"settings"`
}

// ConnectionConfig describes a single PAN-OS endpoint.
// SSH is intentionally unsupported; access is XML API only.
// Note: The host/IP is used as the map key in Config.Connections, not stored here.
//
// Credential fields (APIKey, Password) are tagged `yaml:"-"` so they are
// NEVER round-tripped to ~/.pyre.yaml. pyre does not persist credentials.
// At runtime they come from CLI flags, environment variables, or the
// interactive login flow (session-only); at disconnect they are zeroed
// (see internal/auth).
type ConnectionConfig struct {
	Username   string `yaml:"username,omitempty"`     // Username for API authentication
	Type       string `yaml:"type,omitempty"`         // "firewall" (default) or "panorama"
	Insecure   bool   `yaml:"insecure,omitempty"`     // Skip TLS verification (self-signed certs)
	CACertPath string `yaml:"ca_cert_path,omitempty"` // Optional PEM-encoded CA bundle for TLS verification

	// APIKey is the per-host PAN-OS API key. Never persisted to disk.
	APIKey string `yaml:"-"`
	// Password is the cleartext password used for initial keygen. Never
	// persisted to disk; only present in memory during the login flow.
	Password string `yaml:"-"`
}

type Settings struct {
	SessionPageSize int    `yaml:"session_page_size"`
	Theme           string `yaml:"theme"`
	DefaultView     string `yaml:"default_view"`
}

// ConfigPath returns the path to the config file (~/.pyre.yaml)
func ConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".pyre.yaml"), nil
}

func DefaultConfig() *Config {
	return &Config{
		Connections: make(map[string]ConnectionConfig),
		Settings: Settings{
			SessionPageSize: 50,
			Theme:           "default",
			DefaultView:     "dashboard",
		},
	}
}

func Load() (*Config, error) {
	cfg := DefaultConfig()

	configPath, err := ConfigPath()
	if err != nil {
		return cfg, nil
	}

	// Warn if the config file is readable by group/other. Plan A removed
	// the Warnings slice in favour of direct log.Printf at startup, which
	// prints before the TUI initialises and is therefore user-visible.
	if info, statErr := os.Stat(configPath); statErr == nil {
		if info.Mode().Perm()&0o077 != 0 {
			log.Printf("warning: %s has permissive mode %#o; run `chmod 600 %s`",
				configPath, info.Mode().Perm(), configPath)
		}
	}

	data, err := os.ReadFile(configPath) // #nosec G304 -- Path is constructed from user's home directory
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if cfg.Connections == nil {
		cfg.Connections = make(map[string]ConnectionConfig)
	}

	return cfg, nil
}

// configForSave represents the format we write to disk (new format only)
type configForSave struct {
	Default     string                      `yaml:"default,omitempty"`
	Connections map[string]ConnectionConfig `yaml:"connections,omitempty"`
	Settings    Settings                    `yaml:"settings"`
}

// Save writes the config to ~/.pyre.yaml, creating a backup first
func (c *Config) Save() error {
	configPath, err := ConfigPath()
	if err != nil {
		return err
	}

	// Create backup if file exists
	if _, statErr := os.Stat(configPath); statErr == nil {
		backupPath := configPath + ".bak"
		data, readErr := os.ReadFile(configPath) // #nosec G304 -- Path is constructed from user's home directory
		if readErr == nil {
			if writeErr := os.WriteFile(backupPath, data, 0600); writeErr != nil {
				return fmt.Errorf("failed to create backup: %w", writeErr)
			}
		}
	}

	// Write new format only
	saveConfig := configForSave{
		Default:     c.Default,
		Connections: c.Connections,
		Settings:    c.Settings,
	}

	data, err := yaml.Marshal(saveConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := atomicWriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// atomicWriteFile writes data to a file atomically by writing to a temp file
// in the same directory and renaming it. This prevents corruption from crashes
// or concurrent writes since rename is atomic on Unix.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, filepath.Base(path)+".tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := f.Name()

	// Clean up temp file on any error. Best-effort: if Remove fails here
	// there's nothing useful we can do (the primary error is already on
	// its way to the caller).
	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := f.Chmod(perm); err != nil {
		_ = f.Close()
		return fmt.Errorf("setting permissions: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return fmt.Errorf("writing data: %w", err)
	}

	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("syncing file: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("renaming temp file: %w", err)
	}

	success = true
	return nil
}

// AddConnection adds a new connection to the config (keyed by host)
func (c *Config) AddConnection(host string, conn ConnectionConfig) error {
	if c.Connections == nil {
		c.Connections = make(map[string]ConnectionConfig)
	}
	if _, exists := c.Connections[host]; exists {
		return fmt.Errorf("connection %q already exists", host)
	}
	c.Connections[host] = conn
	return nil
}

// UpdateConnection updates an existing connection in the config
func (c *Config) UpdateConnection(host string, conn ConnectionConfig) error {
	if c.Connections == nil {
		return fmt.Errorf("connection %q not found", host)
	}
	if _, exists := c.Connections[host]; !exists {
		return fmt.Errorf("connection %q not found", host)
	}
	c.Connections[host] = conn
	return nil
}

// DeleteConnection removes a connection from the config
func (c *Config) DeleteConnection(host string) error {
	if c.Connections == nil {
		return fmt.Errorf("connection %q not found", host)
	}
	if _, exists := c.Connections[host]; !exists {
		return fmt.Errorf("connection %q not found", host)
	}
	delete(c.Connections, host)

	// Clear default if it was the deleted connection
	if c.Default == host {
		c.Default = ""
	}
	return nil
}

// GetConnection returns a connection by host
func (c *Config) GetConnection(host string) (ConnectionConfig, bool) {
	conn, ok := c.Connections[host]
	return conn, ok
}

// GetDefaultConnection returns the default connection (host and config)
func (c *Config) GetDefaultConnection() (string, ConnectionConfig, bool) {
	if c.Default == "" {
		return "", ConnectionConfig{}, false
	}
	conn, ok := c.Connections[c.Default]
	return c.Default, conn, ok // Returns: host, config, ok
}

type CLIFlags struct {
	Host       string
	Username   string
	APIKey     string
	Insecure   bool
	Config     string
	Connection string // -c flag for selecting a specific connection
}

func (c *Config) ApplyFlags(flags CLIFlags) {
	if flags.Host != "" {
		if c.Connections == nil {
			c.Connections = make(map[string]ConnectionConfig)
		}
		c.Connections[flags.Host] = ConnectionConfig{
			Insecure: flags.Insecure,
		}
		c.Default = flags.Host
	}
}

func LoadWithFlags(flags CLIFlags) (*Config, error) {
	var cfg *Config
	var err error

	if flags.Config != "" {
		data, readErr := os.ReadFile(flags.Config)
		if readErr != nil {
			return nil, readErr
		}
		cfg = DefaultConfig()
		if err = yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
		if cfg.Connections == nil {
			cfg.Connections = make(map[string]ConnectionConfig)
		}
	} else {
		cfg, err = Load()
		if err != nil {
			return nil, err
		}
	}

	cfg.ApplyFlags(flags)
	return cfg, nil
}

// HasConnections returns true if there are any configured connections
func (c *Config) HasConnections() bool {
	return len(c.Connections) > 0
}

// ConnectionHosts returns a list of all connection hosts
func (c *Config) ConnectionHosts() []string {
	hosts := make([]string, 0, len(c.Connections))
	for host := range c.Connections {
		hosts = append(hosts, host)
	}
	return hosts
}
