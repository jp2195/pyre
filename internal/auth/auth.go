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

	// mu guards the mutable Panorama fields below. Panorama detection and
	// device list refreshes run as Bubble Tea Cmd goroutines, while the
	// Update loop reads/writes these fields from the main goroutine — so
	// access must be serialized to satisfy the race detector.
	mu             sync.RWMutex
	IsPanorama     bool
	ManagedDevices []models.ManagedDevice
	TargetSerial   string // Current target device serial (empty = Panorama itself)
}

// SetPanoramaInfo records whether this connection is a Panorama.
// Safe for concurrent use; pairs with PanoramaInfo() readers.
func (c *Connection) SetPanoramaInfo(isPanorama bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.IsPanorama = isPanorama
}

// PanoramaInfo returns whether this connection is a Panorama.
func (c *Connection) PanoramaInfo() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.IsPanorama
}

// SetManagedDevices replaces the device list. Safe for concurrent use.
func (c *Connection) SetManagedDevices(devs []models.ManagedDevice) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ManagedDevices = devs
}

// ManagedDevicesSnapshot returns a copy of the current device list so
// callers can iterate without holding the lock.
func (c *Connection) ManagedDevicesSnapshot() []models.ManagedDevice {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.ManagedDevices == nil {
		return nil
	}
	out := make([]models.ManagedDevice, len(c.ManagedDevices))
	copy(out, c.ManagedDevices)
	return out
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
// It returns an error if the underlying API client cannot be constructed
// (for example, a user-supplied CA bundle that cannot be loaded).
func (s *Session) AddConnection(host string, connConfig *config.ConnectionConfig, apiKey string) (*Connection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	client, err := api.NewClient(host, apiKey, api.ClientOptions{
		Insecure:   connConfig.Insecure,
		CACertPath: connConfig.CACertPath,
	})
	if err != nil {
		return nil, err
	}
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

	return conn, nil
}

func (s *Session) RemoveConnection(host string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Zero credential fields before dropping the reference so the secret
	// stops being reachable from any surviving *Connection pointer a
	// caller might still hold. Credentials are never persisted anywhere,
	// so this in-memory copy is the only one.
	if conn, ok := s.Connections[host]; ok {
		conn.APIKey = ""
		if conn.Config != nil {
			conn.Config.APIKey = ""
			conn.Config.Password = ""
		}
		if conn.Client != nil {
			_ = conn.Client.Close() //nolint:errcheck // best-effort cleanup; primary error path already underway
		}
	}
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

func ResolveCredentials(cfg *config.Config, flags config.CLIFlags) (*Credentials, error) {
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
		}
	}

	// Host-based API key resolution order (documented in CLAUDE.md):
	//   1. PYRE_<HOST>_API_KEY environment variable.
	//   2. Fall through to PromptForPassword=true so the TUI prompts.
	// pyre does not persist credentials. Users manage them via env vars,
	// CLI flags, or the interactive login flow (session-only).
	if creds.Host != "" && creds.APIKey == "" {
		envName := normalizeHostForEnv(creds.Host)
		if envKey := os.Getenv("PYRE_" + envName + "_API_KEY"); envKey != "" {
			creds.APIKey = envKey
		}
	}

	// If we have host but no API key, signal that we need to prompt for password
	if creds.Host != "" && creds.APIKey == "" {
		creds.PromptForPassword = true
	}

	// Hosts from the TUI forms are validated at input time; hosts arriving
	// via --host / PYRE_HOST / config bypass those forms, so validate here
	// before any URL is built from them.
	if creds.Host != "" {
		if msg := ValidateHost(creds.Host); msg != "" {
			return nil, fmt.Errorf("invalid host %q: %s", creds.Host, msg)
		}
	}

	return creds, nil
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

// normalizeHostForEnv converts a connection host into an env-var-safe
// suffix. Strips any :port (including bracketed IPv6 forms) and
// replaces ".", "-", and ":" with "_" before uppercasing.
func normalizeHostForEnv(host string) string {
	if host == "" {
		return ""
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	r := strings.NewReplacer(".", "_", "-", "_", ":", "_")
	return strings.ToUpper(r.Replace(host))
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
//
// The target serial is stored on the Connection and is passed explicitly to
// each API call that needs it (see Target()). The underlying *api.Client
// holds no target state, which eliminates races when concurrent fetches
// use different targets.
func (c *Connection) SetTarget(device *models.ManagedDevice) error {
	if device == nil {
		c.mu.Lock()
		c.TargetSerial = ""
		c.mu.Unlock()
		return nil
	}

	// Validate serial number format
	if err := validateSerial(device.Serial); err != nil {
		return err
	}

	c.mu.Lock()
	c.TargetSerial = device.Serial
	c.mu.Unlock()
	return nil
}

// Target returns the current Panorama target serial, or "" for Panorama
// itself / standalone firewalls. Safe for concurrent callers.
func (c *Connection) Target() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.TargetSerial
}

// GetTargetDevice returns a pointer to a copy of the currently targeted
// managed device, or nil if targeting Panorama itself / no match is found.
// The returned pointer is safe to retain and mutate: it does not alias the
// live ManagedDevices slice, so a concurrent SetManagedDevices cannot race
// with the caller's use of the result.
func (c *Connection) GetTargetDevice() *models.ManagedDevice {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.TargetSerial == "" {
		return nil
	}
	for i := range c.ManagedDevices {
		if c.ManagedDevices[i].Serial == c.TargetSerial {
			// Return a copy so the caller can use it without
			// racing with a subsequent SetManagedDevices that
			// would replace the backing slice.
			d := c.ManagedDevices[i]
			return &d
		}
	}
	return nil
}

// RefreshManagedDevices fetches the latest list of managed devices from Panorama.
func (c *Connection) RefreshManagedDevices(ctx context.Context) error {
	c.mu.RLock()
	isPano := c.IsPanorama
	c.mu.RUnlock()
	if !isPano {
		return nil
	}

	// GetManagedDevices intentionally targets Panorama itself regardless of
	// the connection's current TargetSerial, so no save/restore dance is
	// required with the stateless client.
	devices, err := c.Client.GetManagedDevices(ctx)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.ManagedDevices = devices
	c.mu.Unlock()
	return nil
}

// ConnectedDeviceCount returns the count of connected managed devices.
func (c *Connection) ConnectedDeviceCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	count := 0
	for _, d := range c.ManagedDevices {
		if d.Connected {
			count++
		}
	}
	return count
}
