package config

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DefaultFirewall string                    `yaml:"default_firewall"`
	Firewalls       map[string]FirewallConfig `yaml:"firewalls"`
	Settings        Settings                  `yaml:"settings"`
}

type SSHConfig struct {
	Port           int    `yaml:"port"`             // Default: 22
	Username       string `yaml:"username"`
	Password       string `yaml:"password"`         // Or use key
	PrivateKeyPath string `yaml:"private_key_path"`
	Timeout        int    `yaml:"timeout"`          // Seconds, default: 30
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
	data, err := os.ReadFile(configPath)
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
	return cfg, nil
}
