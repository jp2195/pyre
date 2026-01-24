package auth

import (
	"fmt"
	"os"
	"testing"

	"github.com/jp2195/pyre/internal/config"
	"github.com/jp2195/pyre/internal/models"
)

func TestNewSession(t *testing.T) {
	cfg := config.DefaultConfig()
	session := NewSession(cfg)

	if session == nil {
		t.Fatal("expected non-nil session")
	}
	if session.Connections == nil {
		t.Error("expected Connections map to be initialized")
	}
	if session.Config != cfg {
		t.Error("expected Config to match")
	}
	if session.ActiveFirewall != "" {
		t.Error("expected ActiveFirewall to be empty")
	}
}

func TestSession_AddConnection(t *testing.T) {
	cfg := config.DefaultConfig()
	session := NewSession(cfg)

	fwConfig := &config.FirewallConfig{
		Host:     "10.0.0.1",
		Insecure: true,
	}

	conn := session.AddConnection("test-fw", fwConfig, "test-api-key")

	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Name != "test-fw" {
		t.Errorf("expected name 'test-fw', got %q", conn.Name)
	}
	if conn.APIKey != "test-api-key" {
		t.Errorf("expected API key 'test-api-key', got %q", conn.APIKey)
	}
	if !conn.Connected {
		t.Error("expected Connected to be true")
	}
	if session.ActiveFirewall != "test-fw" {
		t.Errorf("expected ActiveFirewall 'test-fw', got %q", session.ActiveFirewall)
	}
}

func TestSession_AddConnectionWithSSH(t *testing.T) {
	cfg := config.DefaultConfig()
	session := NewSession(cfg)

	fwConfig := &config.FirewallConfig{
		Host:     "10.0.0.1",
		Insecure: true,
	}

	conn := session.AddConnectionWithSSH("test-fw", fwConfig, "test-api-key", "admin", "password123")

	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.SSHUsername != "admin" {
		t.Errorf("expected SSH username 'admin', got %q", conn.SSHUsername)
	}
	if conn.SSHPassword != "password123" {
		t.Errorf("expected SSH password 'password123', got %q", conn.SSHPassword)
	}
}

func TestSession_GetActiveConnection(t *testing.T) {
	cfg := config.DefaultConfig()
	session := NewSession(cfg)

	// No active connection
	conn := session.GetActiveConnection()
	if conn != nil {
		t.Error("expected nil when no active connection")
	}

	// Add a connection
	fwConfig := &config.FirewallConfig{Host: "10.0.0.1"}
	session.AddConnection("test-fw", fwConfig, "api-key")

	conn = session.GetActiveConnection()
	if conn == nil {
		t.Fatal("expected non-nil active connection")
	}
	if conn.Name != "test-fw" {
		t.Errorf("expected name 'test-fw', got %q", conn.Name)
	}
}

func TestSession_SetActiveFirewall(t *testing.T) {
	cfg := config.DefaultConfig()
	session := NewSession(cfg)

	fwConfig := &config.FirewallConfig{Host: "10.0.0.1"}
	session.AddConnection("fw1", fwConfig, "key1")
	session.AddConnection("fw2", fwConfig, "key2")

	// Set active to fw2
	ok := session.SetActiveFirewall("fw2")
	if !ok {
		t.Error("expected SetActiveFirewall to succeed")
	}
	if session.ActiveFirewall != "fw2" {
		t.Errorf("expected ActiveFirewall 'fw2', got %q", session.ActiveFirewall)
	}

	// Try to set non-existent firewall
	ok = session.SetActiveFirewall("nonexistent")
	if ok {
		t.Error("expected SetActiveFirewall to fail for nonexistent firewall")
	}
	// Active should remain fw2
	if session.ActiveFirewall != "fw2" {
		t.Errorf("expected ActiveFirewall to remain 'fw2', got %q", session.ActiveFirewall)
	}
}

func TestSession_RemoveConnection(t *testing.T) {
	cfg := config.DefaultConfig()
	session := NewSession(cfg)

	fwConfig := &config.FirewallConfig{Host: "10.0.0.1"}
	session.AddConnection("fw1", fwConfig, "key1")
	session.AddConnection("fw2", fwConfig, "key2")

	// Remove active connection
	session.RemoveConnection("fw1")

	if _, ok := session.Connections["fw1"]; ok {
		t.Error("expected fw1 to be removed")
	}
	// Active should switch to remaining connection
	if session.ActiveFirewall != "fw2" {
		t.Errorf("expected ActiveFirewall to switch to 'fw2', got %q", session.ActiveFirewall)
	}

	// Remove last connection
	session.RemoveConnection("fw2")
	if session.ActiveFirewall != "" {
		t.Errorf("expected ActiveFirewall to be empty, got %q", session.ActiveFirewall)
	}
}

func TestSession_ListConnections(t *testing.T) {
	cfg := config.DefaultConfig()
	session := NewSession(cfg)

	// Empty list
	conns := session.ListConnections()
	if len(conns) != 0 {
		t.Errorf("expected 0 connections, got %d", len(conns))
	}

	// Add connections
	fwConfig := &config.FirewallConfig{Host: "10.0.0.1"}
	session.AddConnection("fw1", fwConfig, "key1")
	session.AddConnection("fw2", fwConfig, "key2")

	conns = session.ListConnections()
	if len(conns) != 2 {
		t.Errorf("expected 2 connections, got %d", len(conns))
	}
}

func TestSession_IsConnected(t *testing.T) {
	cfg := config.DefaultConfig()
	session := NewSession(cfg)

	// Not connected
	if session.IsConnected("fw1") {
		t.Error("expected IsConnected to be false for nonexistent firewall")
	}

	// Add connection
	fwConfig := &config.FirewallConfig{Host: "10.0.0.1"}
	session.AddConnection("fw1", fwConfig, "key1")

	if !session.IsConnected("fw1") {
		t.Error("expected IsConnected to be true")
	}
}

func TestResolveCredentials(t *testing.T) {
	cfg := config.DefaultConfig()

	// Test with flags
	flags := config.CLIFlags{
		Host:     "10.0.0.1",
		APIKey:   "flag-api-key",
		Insecure: true,
	}

	creds := ResolveCredentials(cfg, flags)

	if creds.Host != "10.0.0.1" {
		t.Errorf("expected Host '10.0.0.1', got %q", creds.Host)
	}
	if creds.APIKey != "flag-api-key" {
		t.Errorf("expected APIKey 'flag-api-key', got %q", creds.APIKey)
	}
	if !creds.Insecure {
		t.Error("expected Insecure to be true")
	}
}

func TestResolveCredentials_EnvVars(t *testing.T) {
	cfg := config.DefaultConfig()
	flags := config.CLIFlags{}

	// Set environment variables
	os.Setenv("PYRE_HOST", "env-host")
	os.Setenv("PYRE_API_KEY", "env-api-key")
	os.Setenv("PYRE_INSECURE", "true")
	defer func() {
		os.Unsetenv("PYRE_HOST")
		os.Unsetenv("PYRE_API_KEY")
		os.Unsetenv("PYRE_INSECURE")
	}()

	creds := ResolveCredentials(cfg, flags)

	if creds.Host != "env-host" {
		t.Errorf("expected Host 'env-host', got %q", creds.Host)
	}
	if creds.APIKey != "env-api-key" {
		t.Errorf("expected APIKey 'env-api-key', got %q", creds.APIKey)
	}
	if !creds.Insecure {
		t.Error("expected Insecure to be true from env")
	}
}

func TestResolveCredentials_ConfigDefault(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DefaultFirewall = "default-fw"
	cfg.Firewalls["default-fw"] = config.FirewallConfig{
		Host:     "config-host",
		Insecure: true,
	}

	flags := config.CLIFlags{}

	creds := ResolveCredentials(cfg, flags)

	if creds.Host != "config-host" {
		t.Errorf("expected Host 'config-host', got %q", creds.Host)
	}
	if !creds.Insecure {
		t.Error("expected Insecure to be true from config")
	}
}

func TestCredentials_Methods(t *testing.T) {
	tests := []struct {
		name                     string
		creds                    Credentials
		wantHasHost              bool
		wantHasAPIKey            bool
		wantNeedsInteractiveAuth bool
	}{
		{
			name:                     "empty credentials",
			creds:                    Credentials{},
			wantHasHost:              false,
			wantHasAPIKey:            false,
			wantNeedsInteractiveAuth: true,
		},
		{
			name:                     "host only",
			creds:                    Credentials{Host: "10.0.0.1"},
			wantHasHost:              true,
			wantHasAPIKey:            false,
			wantNeedsInteractiveAuth: true,
		},
		{
			name:                     "api key only",
			creds:                    Credentials{APIKey: "key"},
			wantHasHost:              false,
			wantHasAPIKey:            true,
			wantNeedsInteractiveAuth: true,
		},
		{
			name:                     "complete credentials",
			creds:                    Credentials{Host: "10.0.0.1", APIKey: "key"},
			wantHasHost:              true,
			wantHasAPIKey:            true,
			wantNeedsInteractiveAuth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.creds.HasHost(); got != tt.wantHasHost {
				t.Errorf("HasHost() = %v, want %v", got, tt.wantHasHost)
			}
			if got := tt.creds.HasAPIKey(); got != tt.wantHasAPIKey {
				t.Errorf("HasAPIKey() = %v, want %v", got, tt.wantHasAPIKey)
			}
			if got := tt.creds.NeedsInteractiveAuth(); got != tt.wantNeedsInteractiveAuth {
				t.Errorf("NeedsInteractiveAuth() = %v, want %v", got, tt.wantNeedsInteractiveAuth)
			}
		})
	}
}

func TestConnection_GetTargetDevice(t *testing.T) {
	conn := &Connection{
		Name:         "test-conn",
		TargetSerial: "",
		ManagedDevices: []models.ManagedDevice{
			{Serial: "serial1", Hostname: "fw1", IPAddress: "10.0.0.1"},
			{Serial: "serial2", Hostname: "fw2", IPAddress: "10.0.0.2"},
		},
	}

	// No target set
	device := conn.GetTargetDevice()
	if device != nil {
		t.Error("expected nil when no target set")
	}

	// Set target
	conn.TargetSerial = "serial1"
	device = conn.GetTargetDevice()
	if device == nil {
		t.Fatal("expected non-nil device")
	}
	if device.Hostname != "fw1" {
		t.Errorf("expected hostname 'fw1', got %q", device.Hostname)
	}

	// Invalid target
	conn.TargetSerial = "nonexistent"
	device = conn.GetTargetDevice()
	if device != nil {
		t.Error("expected nil for nonexistent target")
	}
}

func TestConnection_ConnectedDeviceCount(t *testing.T) {
	conn := &Connection{
		ManagedDevices: []models.ManagedDevice{
			{Serial: "s1", Connected: true},
			{Serial: "s2", Connected: false},
			{Serial: "s3", Connected: true},
			{Serial: "s4", Connected: false},
		},
	}

	count := conn.ConnectedDeviceCount()
	if count != 2 {
		t.Errorf("expected 2 connected devices, got %d", count)
	}
}

func TestConnection_HasSSH(t *testing.T) {
	// No SSH config
	conn := &Connection{
		Name:   "test-conn",
		Config: &config.FirewallConfig{},
	}

	if conn.HasSSH() {
		t.Error("expected HasSSH to be false with no username")
	}

	// With SSH username in config
	conn.Config = &config.FirewallConfig{
		SSH: config.SSHConfig{
			Username: "admin",
		},
	}

	if !conn.HasSSH() {
		t.Error("expected HasSSH to be true with username")
	}

	// With SSH username from login credentials
	conn.Config = &config.FirewallConfig{}
	conn.SSHUsername = "admin"

	if !conn.HasSSH() {
		t.Error("expected HasSSH to be true with login username")
	}
}

func TestResolveSSHCredentials(t *testing.T) {
	// Set environment variables
	os.Setenv("PYRE_SSH_USERNAME", "env-user")
	os.Setenv("PYRE_SSH_PASSWORD", "env-pass")
	os.Setenv("PYRE_SSH_KEY_PATH", "/env/key/path")
	os.Setenv("PYRE_TEST_FW_SSH_PASSWORD", "fw-specific-pass")
	defer func() {
		os.Unsetenv("PYRE_SSH_USERNAME")
		os.Unsetenv("PYRE_SSH_PASSWORD")
		os.Unsetenv("PYRE_SSH_KEY_PATH")
		os.Unsetenv("PYRE_TEST_FW_SSH_PASSWORD")
	}()

	cfg := config.SSHConfig{}
	result := resolveSSHCredentials("test-fw", cfg)

	if result.Username != "env-user" {
		t.Errorf("expected Username 'env-user', got %q", result.Username)
	}
	// Per-firewall password should override global
	if result.Password != "fw-specific-pass" {
		t.Errorf("expected Password 'fw-specific-pass', got %q", result.Password)
	}
	if result.PrivateKeyPath != "/env/key/path" {
		t.Errorf("expected PrivateKeyPath '/env/key/path', got %q", result.PrivateKeyPath)
	}
}

func TestResolveSSHCredentials_ConfigTakesPrecedence(t *testing.T) {
	// Set environment variables
	os.Setenv("PYRE_SSH_USERNAME", "env-user")
	defer os.Unsetenv("PYRE_SSH_USERNAME")

	// Config with username already set
	cfg := config.SSHConfig{
		Username: "config-user",
	}
	result := resolveSSHCredentials("test-fw", cfg)

	// Config should be preserved, env should not override
	if result.Username != "config-user" {
		t.Errorf("expected Username 'config-user', got %q", result.Username)
	}
}

func TestConnection_getSSHConfig(t *testing.T) {
	conn := &Connection{
		Name: "test-fw",
		Config: &config.FirewallConfig{
			SSH: config.SSHConfig{
				Port:     2222,
				Username: "config-admin",
				Timeout:  60,
			},
		},
		SSHUsername: "login-admin",
		SSHPassword: "login-pass",
	}

	cfg := conn.getSSHConfig()

	// Config values should be preserved
	if cfg.Port != 2222 {
		t.Errorf("expected Port 2222, got %d", cfg.Port)
	}
	if cfg.Timeout != 60 {
		t.Errorf("expected Timeout 60, got %d", cfg.Timeout)
	}
	// Config username takes precedence
	if cfg.Username != "config-admin" {
		t.Errorf("expected Username 'config-admin', got %q", cfg.Username)
	}
}

func TestConnection_getSSHConfig_LoginFallback(t *testing.T) {
	conn := &Connection{
		Name: "test-fw",
		Config: &config.FirewallConfig{
			SSH: config.SSHConfig{
				Port: 22,
				// No username
			},
		},
		SSHUsername: "login-admin",
		SSHPassword: "login-pass",
	}

	cfg := conn.getSSHConfig()

	// Should fall back to login credentials
	if cfg.Username != "login-admin" {
		t.Errorf("expected Username 'login-admin', got %q", cfg.Username)
	}
	if cfg.Password != "login-pass" {
		t.Errorf("expected Password 'login-pass', got %q", cfg.Password)
	}
}

func TestConnection_getSSHConfig_NilConfig(t *testing.T) {
	conn := &Connection{
		Name:        "test-fw",
		Config:      nil,
		SSHUsername: "login-admin",
		SSHPassword: "login-pass",
	}

	cfg := conn.getSSHConfig()

	// Should use login credentials
	if cfg.Username != "login-admin" {
		t.Errorf("expected Username 'login-admin', got %q", cfg.Username)
	}
}

// KeygenError tests

func TestKeygenError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *KeygenError
		expected string
	}{
		{
			name:     "message only",
			err:      &KeygenError{Message: "authentication failed"},
			expected: "authentication failed",
		},
		{
			name:     "message with cause",
			err:      &KeygenError{Message: "connection failed", Cause: os.ErrPermission},
			expected: "connection failed: permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestKeygenError_Unwrap(t *testing.T) {
	cause := os.ErrNotExist
	err := &KeygenError{Message: "test", Cause: cause}

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}

	// Test with nil cause
	err2 := &KeygenError{Message: "test"}
	if err2.Unwrap() != nil {
		t.Error("expected Unwrap() to return nil when Cause is nil")
	}
}

func TestIsAuthenticationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "KeygenError with invalid credential",
			err:      &KeygenError{Message: "Invalid credential for user"},
			expected: true,
		},
		{
			name:     "KeygenError with authentication failed",
			err:      &KeygenError{Message: "authentication failed"},
			expected: true,
		},
		{
			name:     "KeygenError with invalid username",
			err:      &KeygenError{Message: "Invalid username provided"},
			expected: true,
		},
		{
			name:     "KeygenError with invalid password",
			err:      &KeygenError{Message: "Invalid password"},
			expected: true,
		},
		{
			name:     "KeygenError with connection error",
			err:      &KeygenError{Message: "connection refused"},
			expected: false,
		},
		{
			name:     "regular error with auth message",
			err:      fmt.Errorf("authentication failed"),
			expected: true,
		},
		{
			name:     "regular error with invalid credentials",
			err:      fmt.Errorf("invalid credentials"),
			expected: true,
		},
		{
			name:     "regular error with invalid username or password",
			err:      fmt.Errorf("invalid username or password"),
			expected: true,
		},
		{
			name:     "regular error without auth message",
			err:      fmt.Errorf("network error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAuthenticationError(tt.err)
			if got != tt.expected {
				t.Errorf("IsAuthenticationError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "KeygenError",
			err:      &KeygenError{Message: "connection refused"},
			expected: true,
		},
		{
			name:     "regular error",
			err:      fmt.Errorf("some error"),
			expected: false,
		},
		{
			name:     "wrapped KeygenError",
			err:      fmt.Errorf("wrapped: %w", &KeygenError{Message: "connection failed"}),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsConnectionError(tt.err)
			if got != tt.expected {
				t.Errorf("IsConnectionError() = %v, want %v", got, tt.expected)
			}
		})
	}
}
