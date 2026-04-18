package auth

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/jp2195/pyre/internal/api"
	"github.com/jp2195/pyre/internal/config"
	"github.com/jp2195/pyre/internal/models"
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
	Host      string // Host/IP is the primary identifier
	Config    *config.ConnectionConfig
	APIKey    string
	Client    *api.Client
	Connected bool

	// Panorama fields
	IsPanorama     bool
	ManagedDevices []models.ManagedDevice
	TargetSerial   string // Current target device serial (empty = Panorama itself)
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

// AddConnection creates a new PAN-OS XML API connection for the given host.
func (s *Session) AddConnection(host string, connConfig *config.ConnectionConfig, apiKey string) *Connection {
	s.mu.Lock()
	defer s.mu.Unlock()

	client := api.NewClient(host, apiKey, api.WithInsecure(connConfig.Insecure))
	conn := &Connection{
		Host:      host,
		Config:    connConfig,
		APIKey:    apiKey,
		Client:    client,
		Connected: true,
	}
	s.Connections[host] = conn

	if s.ActiveFirewall == "" {
		s.ActiveFirewall = host
	}

	return conn
}

func (s *Session) RemoveConnection(host string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Connections, host)
	if s.ActiveFirewall == host {
		s.ActiveFirewall = ""
		for h := range s.Connections {
			s.ActiveFirewall = h
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

func (s *Session) IsConnected(host string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	conn, ok := s.Connections[host]
	return ok && conn.Connected
}

type Credentials struct {
	Host              string
	APIKey            string
	Username          string
	Password          string
	Insecure          bool
	PromptForPassword bool // True when host/user are set but no API key, so prompt for password
}

func ResolveCredentials(cfg *config.Config, flags config.CLIFlags) *Credentials {
	creds := &Credentials{}

	// CLI flags take highest priority
	if flags.Host != "" {
		creds.Host = flags.Host
	}
	if flags.Username != "" {
		creds.Username = flags.Username
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
		if host, conn, ok := cfg.GetDefaultConnection(); ok {
			creds.Host = host
			// Use config insecure if not already set by flags or env
			if !creds.Insecure && conn.Insecure {
				creds.Insecure = true
			}

			// Try host-based env var for API key (normalize host to valid env var name)
			envName := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(host, "-", "_"), ".", "_"))
			envKey := os.Getenv("PYRE_" + envName + "_API_KEY")
			if envKey != "" && creds.APIKey == "" {
				creds.APIKey = envKey
			}
		}
	}

	// If we have host but no API key, signal that we need to prompt for password
	if creds.Host != "" && creds.APIKey == "" {
		creds.PromptForPassword = true
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

// SetTarget sets the current target device for Panorama.
// Pass nil to target Panorama itself.
// Returns an error if the device serial is invalid.
func (c *Connection) SetTarget(device *models.ManagedDevice) error {
	if device == nil {
		c.TargetSerial = ""
		c.Client.ClearTarget()
		return nil
	}

	// Validate serial number format
	if err := validateSerial(device.Serial); err != nil {
		return err
	}

	c.TargetSerial = device.Serial
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
