package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Default     string                      `yaml:"default,omitempty"`
	Connections map[string]ConnectionConfig `yaml:"connections,omitempty"`
	Settings    Settings                    `yaml:"settings"`
	Warnings    []string                    `yaml:"-"` // Security warnings from config validation
}

type SSHConfig struct {
	Port           int    `yaml:"port"` // Default: 22
	Username       string `yaml:"username"`
	Password       string `yaml:"password,omitempty"` // Deprecated: use env vars instead
	PrivateKeyPath string `yaml:"private_key_path"`
	Timeout        int    `yaml:"timeout"`          // Seconds, default: 30
	KnownHostsPath string `yaml:"known_hosts_path"` // Default: ~/.ssh/known_hosts
	Insecure       bool   `yaml:"insecure"`         // Skip host key verification (not recommended)
}

// ConnectionConfig represents a firewall or Panorama connection configuration
// Note: The host/IP is used as the map key in Config.Connections, not stored here
type ConnectionConfig struct {
	Username string    `yaml:"username,omitempty"` // Username for API authentication
	Type     string    `yaml:"type,omitempty"`     // "firewall" (default) or "panorama"
	Insecure bool      `yaml:"insecure,omitempty"`
	SSH      SSHConfig `yaml:"ssh,omitempty"`
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
	if _, err := os.Stat(configPath); err == nil {
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

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

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
	cfg.validateSecuritySettings()
	return cfg, nil
}

// validateSecuritySettings checks for deprecated or insecure configuration settings
// and adds warnings to the config.
func (c *Config) validateSecuritySettings() {
	for host, conn := range c.Connections {
		// Warn about SSH password in config file (deprecated)
		if conn.SSH.Password != "" {
			c.Warnings = append(c.Warnings, fmt.Sprintf(
				"SECURITY WARNING: Connection %q has SSH password in config file. "+
					"Use PYRE_SSH_PASSWORD environment variable instead.",
				host))
		}
	}
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
