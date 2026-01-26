package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DefaultFirewall string                    `yaml:"default_firewall"`
	Firewalls       map[string]FirewallConfig `yaml:"firewalls"`
	Settings        Settings                  `yaml:"settings"`
	Warnings        []string                  `yaml:"-"` // Security warnings from config validation
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

type FirewallConfig struct {
	Host     string    `yaml:"host"`
	Insecure bool      `yaml:"insecure"`
	SSH      SSHConfig `yaml:"ssh"`
	Type     string    `yaml:"type,omitempty"` // "panorama" or empty (auto-detect)
}

type Settings struct {
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	SessionPageSize int           `yaml:"session_page_size"`
	Theme           string        `yaml:"theme"`
	DefaultView     string        `yaml:"default_view"`
}

func DefaultConfig() *Config {
	return &Config{
		Firewalls: make(map[string]FirewallConfig),
		Settings: Settings{
			RefreshInterval: 5 * time.Second,
			SessionPageSize: 50,
			Theme:           "default",
			DefaultView:     "dashboard",
		},
	}
}

func Load() (*Config, error) {
	cfg := DefaultConfig()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return cfg, nil
	}

	configPath := filepath.Join(homeDir, ".pyre.yaml")
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

	if cfg.Firewalls == nil {
		cfg.Firewalls = make(map[string]FirewallConfig)
	}

	return cfg, nil
}

func (c *Config) GetFirewall(name string) (FirewallConfig, bool) {
	fw, ok := c.Firewalls[name]
	return fw, ok
}

func (c *Config) GetDefaultFirewall() (string, FirewallConfig, bool) {
	if c.DefaultFirewall == "" {
		return "", FirewallConfig{}, false
	}
	fw, ok := c.Firewalls[c.DefaultFirewall]
	return c.DefaultFirewall, fw, ok
}

type CLIFlags struct {
	Host     string
	Username string
	APIKey   string
	Insecure bool
	Config   string
}

func (c *Config) ApplyFlags(flags CLIFlags) {
	if flags.Host != "" {
		c.Firewalls["cli"] = FirewallConfig{
			Host:     flags.Host,
			Insecure: flags.Insecure,
		}
		c.DefaultFirewall = "cli"
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
	for name, fw := range c.Firewalls {
		// Warn about SSH password in config file (deprecated)
		if fw.SSH.Password != "" {
			c.Warnings = append(c.Warnings, fmt.Sprintf(
				"SECURITY WARNING: Firewall %q has SSH password in config file. "+
					"Use PYRE_SSH_PASSWORD or PYRE_%s_SSH_PASSWORD environment variable instead.",
				name, name))
		}
	}
}
