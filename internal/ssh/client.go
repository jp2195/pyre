package ssh

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/jp2195/pyre/internal/config"
)

// Client represents an SSH client for connecting to PAN-OS devices.
// Fields are ordered for optimal memory alignment on 64-bit systems.
type Client struct {
	host    string            // 16 bytes (string header)
	config  *ssh.ClientConfig // 8 bytes (pointer)
	client  *ssh.Client       // 8 bytes (pointer)
	timeout time.Duration     // 8 bytes (int64)
	port    int               // 8 bytes (int on 64-bit)
}

// CommandResult contains the result of an SSH command execution.
type CommandResult struct {
	Command  string
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
	Error    error
}

// NewClient creates a new SSH client for the given host and configuration.
func NewClient(host string, cfg config.SSHConfig) (*Client, error) {
	port := cfg.Port
	if port == 0 {
		port = 22
	}

	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	var authMethods []ssh.AuthMethod

	// Try private key auth first
	if cfg.PrivateKeyPath != "" {
		keyPath := expandPath(cfg.PrivateKeyPath)
		key, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key: %w", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}

		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// Add password auth if provided
	if cfg.Password != "" {
		authMethods = append(authMethods, ssh.Password(cfg.Password))
	}

	// Add keyboard-interactive for PAN-OS compatibility
	if cfg.Password != "" {
		authMethods = append(authMethods, ssh.KeyboardInteractive(
			func(user, instruction string, questions []string, echos []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range questions {
					answers[i] = cfg.Password
				}
				return answers, nil
			},
		))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no authentication method provided")
	}

	sshConfig := &ssh.ClientConfig{
		User:            cfg.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}

	return &Client{
		host:    host,
		port:    port,
		config:  sshConfig,
		timeout: timeout,
	}, nil
}

// Connect establishes an SSH connection to the device.
func (c *Client) Connect(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)

	// Use context for connection timeout
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to dial: %w", err)
	}

	// Create SSH connection on top of TCP connection
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, c.config)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to establish SSH connection: %w", err)
	}

	c.client = ssh.NewClient(sshConn, chans, reqs)
	return nil
}

// Close closes the SSH connection.
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// IsConnected returns true if the client has an active connection.
func (c *Client) IsConnected() bool {
	return c.client != nil
}

// Execute runs a command on the remote device and returns the result.
func (c *Client) Execute(ctx context.Context, cmd string) (*CommandResult, error) {
	if c.client == nil {
		return nil, fmt.Errorf("not connected")
	}

	start := time.Now()

	session, err := c.client.NewSession()
	if err != nil {
		return &CommandResult{
			Command:  cmd,
			Duration: time.Since(start),
			Error:    fmt.Errorf("failed to create session: %w", err),
		}, nil
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	// Run command with context awareness
	done := make(chan error, 1)
	go func() {
		done <- session.Run(cmd)
	}()

	select {
	case <-ctx.Done():
		session.Signal(ssh.SIGTERM)
		return &CommandResult{
			Command:  cmd,
			Duration: time.Since(start),
			Error:    ctx.Err(),
		}, nil
	case err := <-done:
		result := &CommandResult{
			Command:  cmd,
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
			Duration: time.Since(start),
		}

		if err != nil {
			if exitErr, ok := err.(*ssh.ExitError); ok {
				result.ExitCode = exitErr.ExitStatus()
			} else {
				result.Error = err
			}
		}

		return result, nil
	}
}

// PAN-OS specific helper methods

// ShowLog retrieves log entries from the device.
func (c *Client) ShowLog(ctx context.Context, logType string, lines int) (*CommandResult, error) {
	if lines <= 0 {
		lines = 50
	}
	cmd := fmt.Sprintf("less mp-log %s.log | tail %d", logType, lines)
	return c.Execute(ctx, cmd)
}

// ShowClockInfo retrieves clock and NTP information.
func (c *Client) ShowClockInfo(ctx context.Context) (*CommandResult, error) {
	return c.Execute(ctx, "show clock")
}

// TestConnectivity performs a ping test to the target host.
func (c *Client) TestConnectivity(ctx context.Context, target string) (*CommandResult, error) {
	cmd := fmt.Sprintf("ping host %s count 3", target)
	return c.Execute(ctx, cmd)
}

// ShowPanoramaStatus retrieves Panorama connection status.
func (c *Client) ShowPanoramaStatus(ctx context.Context) (*CommandResult, error) {
	return c.Execute(ctx, "show panorama-status")
}

// ShowHAState retrieves detailed HA state information.
func (c *Client) ShowHAState(ctx context.Context) (*CommandResult, error) {
	return c.Execute(ctx, "show high-availability state")
}

// ShowHALink retrieves HA link status.
func (c *Client) ShowHALink(ctx context.Context) (*CommandResult, error) {
	return c.Execute(ctx, "show high-availability link-monitoring")
}

// ShowCommitHistory retrieves recent commit history.
func (c *Client) ShowCommitHistory(ctx context.Context) (*CommandResult, error) {
	return c.Execute(ctx, "show jobs all")
}

// ShowConfigLock retrieves configuration lock status.
func (c *Client) ShowConfigLock(ctx context.Context) (*CommandResult, error) {
	return c.Execute(ctx, "show config-lock all")
}

// ShowLicenseInfo retrieves license information.
func (c *Client) ShowLicenseInfo(ctx context.Context) (*CommandResult, error) {
	return c.Execute(ctx, "request license info")
}

// ShowTopProcesses retrieves top resource-consuming processes.
func (c *Client) ShowTopProcesses(ctx context.Context) (*CommandResult, error) {
	return c.Execute(ctx, "show system resources")
}

// ShowSessionTable retrieves session table summary.
func (c *Client) ShowSessionTable(ctx context.Context) (*CommandResult, error) {
	return c.Execute(ctx, "show session info")
}

// ShowDataplaneStats retrieves dataplane statistics.
func (c *Client) ShowDataplaneStats(ctx context.Context) (*CommandResult, error) {
	return c.Execute(ctx, "show running resource-monitor")
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return home + path[1:]
		}
	}
	return path
}
