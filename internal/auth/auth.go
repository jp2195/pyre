package auth

import (
	"context"
	"os"
	"strings"
	"sync"

	"github.com/joshuamontgomery/pyre/internal/api"
	"github.com/joshuamontgomery/pyre/internal/config"
	"github.com/joshuamontgomery/pyre/internal/models"
	"github.com/joshuamontgomery/pyre/internal/ssh"
)

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

	// SSH credentials from login (reused for SSH connection)
	SSHUsername string
	SSHPassword string

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
	return s.AddConnectionWithSSH(name, fwConfig, apiKey, "", "")
}

func (s *Session) AddConnectionWithSSH(name string, fwConfig *config.FirewallConfig, apiKey, sshUsername, sshPassword string) *Connection {
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
		SSHPassword: sshPassword,
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

	if flags.Host != "" {
		creds.Host = flags.Host
		creds.Insecure = flags.Insecure
	}
	if flags.APIKey != "" {
		creds.APIKey = flags.APIKey
	}

	if envHost := os.Getenv("PYRE_HOST"); envHost != "" && creds.Host == "" {
		creds.Host = envHost
	}
	if envKey := os.Getenv("PYRE_API_KEY"); envKey != "" && creds.APIKey == "" {
		creds.APIKey = envKey
	}
	if os.Getenv("PYRE_INSECURE") == "true" && !creds.Insecure {
		creds.Insecure = true
	}

	if creds.Host == "" {
		if name, fw, ok := cfg.GetDefaultFirewall(); ok {
			creds.Host = fw.Host
			creds.Insecure = fw.Insecure

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

// SetTarget sets the current target device for Panorama.
// Pass nil to target Panorama itself.
func (c *Connection) SetTarget(device *models.ManagedDevice) {
	if device == nil {
		c.TargetSerial = ""
		c.TargetIP = ""
		c.Client.ClearTarget()
	} else {
		c.TargetSerial = device.Serial
		c.TargetIP = device.IPAddress
		c.Client.SetTarget(device.Serial)
	}
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
func (c *Connection) getSSHConfig() config.SSHConfig {
	var sshCfg config.SSHConfig
	if c.Config != nil {
		sshCfg = c.Config.SSH
	}

	// Apply environment variable overrides
	sshCfg = resolveSSHCredentials(c.Name, sshCfg)

	// Use login credentials if no username configured yet
	if sshCfg.Username == "" && c.SSHUsername != "" {
		sshCfg.Username = c.SSHUsername
		sshCfg.Password = c.SSHPassword
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

	// Per-firewall SSH password: PYRE_<FIREWALL>_SSH_PASSWORD
	envName := strings.ToUpper(strings.ReplaceAll(fwName, "-", "_"))
	if envPass := os.Getenv("PYRE_" + envName + "_SSH_PASSWORD"); envPass != "" {
		cfg.Password = envPass
	}

	return cfg
}
