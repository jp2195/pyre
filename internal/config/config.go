package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v4"
)

// Config represents the application configuration
type Config struct {
	Default     string                      `yaml:"default,omitempty"`
	Connections map[string]ConnectionConfig `yaml:"connections,omitempty"`
	Settings    Settings                    `yaml:"settings"`

	// path is the file this config was loaded from, so saves go back to it.
	// --config previously selected which file to read while every save
	// resolved ~/.pyre.yaml, silently splitting reads from writes.
	// Unexported, so it never round-trips into the YAML.
	path string
}

// Path returns the file this config is read from and written to, falling
// back to ~/.pyre.yaml for a config that was not produced by Load.
// It returns "" only when the home directory cannot be resolved.
func (c *Config) Path() string {
	if c.path != "" {
		return c.path
	}
	p, err := ConfigPath()
	if err != nil {
		return ""
	}
	return p
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
	Theme string `yaml:"theme"`
}

// warnOut receives startup warnings such as insecure file permissions.
//
// It is deliberately NOT the standard logger: main points that at a debug
// file or io.Discard before any config is loaded, so warnings written with
// log.Printf were discarded every time and the permission warning promised
// by SECURITY.md never reached anyone. Tests swap this writer.
var warnOut io.Writer = os.Stderr

// warnf writes a startup warning that survives logger reconfiguration.
func warnf(format string, args ...any) {
	// Best-effort: a startup warning that cannot be written is not worth
	// failing the load over.
	_, _ = fmt.Fprintf(warnOut, "warning: "+format+"\n", args...) //nolint:errcheck // best-effort warning
}

// warnIfPermissive warns when path is readable or writable by group or other.
// Both the config and the state file are expected to be 0600.
func warnIfPermissive(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		warnf("%s has permissive mode %#o; run `chmod 600 %s`", path, perm, path)
	}
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
			Theme: "default",
		},
	}
}

func Load() (*Config, error) {
	cfg := DefaultConfig()

	configPath, err := ConfigPath()
	if err != nil {
		return cfg, nil
	}

	cfg.path = configPath
	warnIfPermissive(configPath)

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

// Marshal returns the YAML representation Save writes to disk. Credential
// fields never appear because ConnectionConfig tags them `yaml:"-"`
// (regression-guarded by TestConfig_DoesNotPersistCredentials).
//
// Marshal must be called from the TUI event loop; the returned bytes are
// safe to hand to a background writer (see WriteConfigBytes).
func (c *Config) Marshal() ([]byte, error) {
	data, err := yaml.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	return data, nil
}

// WriteConfigBytes writes pre-marshaled config data to configPath, creating
// a .bak backup of the previous file first. The path is passed in rather than
// resolved here so a config loaded via --config is written back to that same
// file. Safe to call from a background goroutine because it touches no shared
// in-memory state; obtain the path from Config.Path on the event loop.
func WriteConfigBytes(configPath string, data []byte) error {
	if configPath == "" {
		return fmt.Errorf("no config path resolved")
	}

	// Create backup if file exists
	if _, statErr := os.Stat(configPath); statErr == nil {
		backupPath := configPath + ".bak"
		prev, readErr := os.ReadFile(configPath) // #nosec G304 -- Path is constructed from user's home directory
		if readErr == nil {
			if writeErr := atomicWriteFile(backupPath, prev, 0600); writeErr != nil {
				return fmt.Errorf("failed to create backup: %w", writeErr)
			}
		}
	}

	if err := atomicWriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}

// Save writes the config back to the file it was loaded from, creating a
// backup first.
func (c *Config) Save() error {
	data, err := c.Marshal()
	if err != nil {
		return err
	}
	return WriteConfigBytes(c.Path(), data)
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
			_ = os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup of partially-written temp file
		}
	}()

	if err := f.Chmod(perm); err != nil {
		_ = f.Close() //nolint:errcheck // cleanup on error path
		return fmt.Errorf("setting permissions: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		_ = f.Close() //nolint:errcheck // cleanup on error path
		return fmt.Errorf("writing data: %w", err)
	}

	if err := f.Sync(); err != nil {
		_ = f.Close() //nolint:errcheck // cleanup on error path
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

// SetConnection stores conn under host, replacing any existing entry.
//
// Unlike AddConnection it does not treat an existing host as an error:
// saving a connection the user already tracks means updating it, and the
// caller has no useful recovery from "already exists" other than doing
// exactly this.
func (c *Config) SetConnection(host string, conn ConnectionConfig) {
	if c.Connections == nil {
		c.Connections = make(map[string]ConnectionConfig)
	}
	c.Connections[host] = conn
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
		data, readErr := os.ReadFile(flags.Config) // #nosec G304 -- path supplied by the user via --config
		if readErr != nil {
			return nil, readErr
		}
		cfg = DefaultConfig()
		if err = yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
		cfg.path = flags.Config
		warnIfPermissive(flags.Config)
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
