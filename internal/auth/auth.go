package auth

import (
	"context"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/jp2195/pyre/internal/api"
	"github.com/jp2195/pyre/internal/config"
	"github.com/jp2195/pyre/internal/models"
	"github.com/jp2195/pyre/internal/ssh"
)

// serialPattern validates Palo Alto device serial numbers (alphanumeric, typically 12-15 chars)
var serialPattern = regexp.MustCompile(`^[A-Za-z0-9]{8,20}$`)

type Session struct {
	mu             sync.RWMutex
	ActiveFirewall string
	Connections    map[string]*Connection
	Config         *config.Config
}

type Connection struct {
	Name       string
	Config     *config.FirewallConfig
	APIKey     string
	Client     *api.Client
	SSHClient  *ssh.Client
	Connected  bool
	SSHEnabled bool

	// SSH username from login (password must come from env var for security)
	SSHUsername string

	// Panorama fields
	IsPanorama     bool
	ManagedDevices []models.ManagedDevice
	TargetSerial   string // Current target device serial (empty = Panorama itself)
	TargetIP       string // Current target device mgmt IP (for SSH)
}

func NewSession(cfg *config.Config) *Session {
	return &Session{
		Connections: make(map[string]*Connection),
		Config:      cfg,
	}
}

func (s *Session) GetActiveConnection() *Connection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.ActiveFirewall == "" {
		return nil
	}
	return s.Connections[s.ActiveFirewall]
}

func (s *Session) SetActiveFirewall(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.Connections[name]; ok {
		s.ActiveFirewall = name
		return true
	}
	return false
}

func (s *Session) AddConnection(name string, fwConfig *config.FirewallConfig, apiKey string) *Connection {
	return s.AddConnectionWithSSH(name, fwConfig, apiKey, "", nil)
}

// AddConnectionWithSSH creates a new connection with SSH username and optional pre-established SSH client.
// If sshClient is provided, it will be used directly. Otherwise, SSH can be established later
// using credentials from environment variables.
func (s *Session) AddConnectionWithSSH(name string, fwConfig *config.FirewallConfig, apiKey, sshUsername string, sshClient *ssh.Client) *Connection {
	s.mu.Lock()
	defer s.mu.Unlock()

	client := api.NewClient(fwConfig.Host, apiKey, api.WithInsecure(fwConfig.Insecure))
	conn := &Connection{
		Name:        name,
		Config:      fwConfig,
		APIKey:      apiKey,
		Client:      client,
		Connected:   true,
		SSHUsername: sshUsername,
		SSHClient:   sshClient,
		SSHEnabled:  sshClient != nil,
	}
	s.Connections[name] = conn

	if s.ActiveFirewall == "" {
		s.ActiveFirewall = name
	}

	return conn
}

func (s *Session) RemoveConnection(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Connections, name)
	if s.ActiveFirewall == name {
		s.ActiveFirewall = ""
		for n := range s.Connections {
			s.ActiveFirewall = n
			break
		}
	}
}

func (s *Session) ListConnections() []*Connection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	conns := make([]*Connection, 0, len(s.Connections))
	for _, c := range s.Connections {
		conns = append(conns, c)
	}
	return conns
}

func (s *Session) IsConnected(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	conn, ok := s.Connections[name]
	return ok && conn.Connected
}

type Credentials struct {
	Host     string
	APIKey   string
	Username string
	Password string
	Insecure bool
}

func ResolveCredentials(cfg *config.Config, flags config.CLIFlags) *Credentials {
	creds := &Credentials{}

	// CLI flags take highest priority
	if flags.Host != "" {
		creds.Host = flags.Host
	}
	if flags.APIKey != "" {
		creds.APIKey = flags.APIKey
	}
	// If --insecure flag is explicitly true, use it
	if flags.Insecure {
		creds.Insecure = true
	}

	// Environment variables (if not set by flags)
	if envHost := os.Getenv("PYRE_HOST"); envHost != "" && creds.Host == "" {
		creds.Host = envHost
	}
	if envKey := os.Getenv("PYRE_API_KEY"); envKey != "" && creds.APIKey == "" {
		creds.APIKey = envKey
	}
	if os.Getenv("PYRE_INSECURE") == "true" && !creds.Insecure {
		creds.Insecure = true
	}

	// Config file defaults (if not set by flags or env)
	if creds.Host == "" {
		if name, fw, ok := cfg.GetDefaultFirewall(); ok {
			creds.Host = fw.Host
			// Use config insecure if not already set by flags or env
			if !creds.Insecure && fw.Insecure {
				creds.Insecure = true
			}

			envKey := os.Getenv("PYRE_" + name + "_API_KEY")
			if envKey != "" && creds.APIKey == "" {
				creds.APIKey = envKey
			}
		}
	}

	return creds
}

func (c *Credentials) HasHost() bool {
	return c.Host != ""
}

func (c *Credentials) HasAPIKey() bool {
	return c.APIKey != ""
}

func (c *Credentials) NeedsInteractiveAuth() bool {
	return c.Host == "" || c.APIKey == ""
}

// validateSerial checks if the serial number has a valid format.
func validateSerial(serial string) error {
	if serial == "" {
		return nil
	}
	if !serialPattern.MatchString(serial) {
		return fmt.Errorf("invalid serial number format: %s", serial)
	}
	return nil
}

// validateIP checks if the IP address is valid.
func validateIP(ip string) error {
	if ip == "" {
		return nil
	}
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}
	return nil
}

// SetTarget sets the current target device for Panorama.
// Pass nil to target Panorama itself.
// Returns an error if the device serial or IP is invalid.
func (c *Connection) SetTarget(device *models.ManagedDevice) error {
	if device == nil {
		c.TargetSerial = ""
		c.TargetIP = ""
		c.Client.ClearTarget()
		return nil
	}

	// Validate serial number format
	if err := validateSerial(device.Serial); err != nil {
		return err
	}

	// Validate IP address format
	if err := validateIP(device.IPAddress); err != nil {
		return err
	}

	c.TargetSerial = device.Serial
	c.TargetIP = device.IPAddress
	c.Client.SetTarget(device.Serial)
	return nil
}

// GetTargetDevice returns the currently targeted managed device, or nil if targeting Panorama.
func (c *Connection) GetTargetDevice() *models.ManagedDevice {
	if c.TargetSerial == "" {
		return nil
	}
	for i := range c.ManagedDevices {
		if c.ManagedDevices[i].Serial == c.TargetSerial {
			return &c.ManagedDevices[i]
		}
	}
	return nil
}

// RefreshManagedDevices fetches the latest list of managed devices from Panorama.
func (c *Connection) RefreshManagedDevices(ctx context.Context) error {
	if !c.IsPanorama {
		return nil
	}

	// Temporarily clear target to query Panorama directly
	savedTarget := c.Client.GetTarget()
	c.Client.ClearTarget()
	defer c.Client.SetTarget(savedTarget)

	devices, err := c.Client.GetManagedDevices(ctx)
	if err != nil {
		return err
	}
	c.ManagedDevices = devices
	return nil
}

// ConnectedDeviceCount returns the count of connected managed devices.
func (c *Connection) ConnectedDeviceCount() int {
	count := 0
	for _, d := range c.ManagedDevices {
		if d.Connected {
			count++
		}
	}
	return count
}

// ConnectSSH establishes an SSH connection for the given firewall connection.
func (c *Connection) ConnectSSH(ctx context.Context) error {
	sshCfg := c.getSSHConfig()

	// If no SSH username is configured, skip SSH connection
	if sshCfg.Username == "" {
		return nil
	}

	// For Panorama with a target device, connect to the target's IP
	host := c.Config.Host
	if c.IsPanorama && c.TargetIP != "" {
		// Validate target IP before using it
		if err := validateIP(c.TargetIP); err != nil {
			return fmt.Errorf("invalid target IP for SSH: %w", err)
		}
		host = c.TargetIP
	}

	client, err := ssh.NewClient(host, sshCfg)
	if err != nil {
		return err
	}

	if err := client.Connect(ctx); err != nil {
		return err
	}

	c.SSHClient = client
	c.SSHEnabled = true
	return nil
}

// DisconnectSSH closes the SSH connection.
func (c *Connection) DisconnectSSH() error {
	if c.SSHClient != nil {
		err := c.SSHClient.Close()
		c.SSHClient = nil
		c.SSHEnabled = false
		return err
	}
	return nil
}

// HasSSH returns true if SSH is configured for this connection.
func (c *Connection) HasSSH() bool {
	sshCfg := c.getSSHConfig()
	return sshCfg.Username != ""
}

// getSSHConfig returns the SSH configuration, combining config file, env vars, and login credentials.
// Note: SSH passwords must come from environment variables (PYRE_SSH_PASSWORD) for security.
func (c *Connection) getSSHConfig() config.SSHConfig {
	var sshCfg config.SSHConfig
	if c.Config != nil {
		sshCfg = c.Config.SSH
	}

	// Apply environment variable overrides
	sshCfg = resolveSSHCredentials(c.Name, sshCfg)

	// Use login username if no username configured yet
	if sshCfg.Username == "" && c.SSHUsername != "" {
		sshCfg.Username = c.SSHUsername
	}

	return sshCfg
}

// resolveSSHCredentials applies environment variable overrides to SSH config.
func resolveSSHCredentials(fwName string, cfg config.SSHConfig) config.SSHConfig {
	// Global SSH env vars
	if envUser := os.Getenv("PYRE_SSH_USERNAME"); envUser != "" && cfg.Username == "" {
		cfg.Username = envUser
	}
	if envPass := os.Getenv("PYRE_SSH_PASSWORD"); envPass != "" && cfg.Password == "" {
		cfg.Password = envPass
	}
	if envKey := os.Getenv("PYRE_SSH_KEY_PATH"); envKey != "" && cfg.PrivateKeyPath == "" {
		cfg.PrivateKeyPath = envKey
	}
	if os.Getenv("PYRE_SSH_INSECURE") == "true" {
		cfg.Insecure = true
	}

	// Per-firewall SSH password: PYRE_<FIREWALL>_SSH_PASSWORD
	envName := strings.ToUpper(strings.ReplaceAll(fwName, "-", "_"))
	if envPass := os.Getenv("PYRE_" + envName + "_SSH_PASSWORD"); envPass != "" {
		cfg.Password = envPass
	}

	return cfg
}
