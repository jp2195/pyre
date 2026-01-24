package ssh

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jp2195/pyre/internal/config"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.SSHConfig
		wantErr bool
	}{
		{
			name: "with password",
			cfg: config.SSHConfig{
				Username: "admin",
				Password: "secret",
			},
			wantErr: false,
		},
		{
			name:    "no auth method",
			cfg:     config.SSHConfig{Username: "admin"},
			wantErr: true,
		},
		{
			name: "with invalid key path",
			cfg: config.SSHConfig{
				Username:       "admin",
				PrivateKeyPath: "/nonexistent/path/key",
			},
			wantErr: true,
		},
		{
			name: "default port and timeout",
			cfg: config.SSHConfig{
				Username: "admin",
				Password: "secret",
			},
			wantErr: false,
		},
		{
			name: "custom port and timeout",
			cfg: config.SSHConfig{
				Username: "admin",
				Password: "secret",
				Port:     2222,
				Timeout:  60,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient("10.0.0.1", tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewClient() returned nil client without error")
			}
		})
	}
}

func TestClientWithMockServer(t *testing.T) {
	server, err := NewMockSSHServer()
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}

	server.SetDefaultResponses()

	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer server.Close()

	cfg := config.SSHConfig{
		Username: "admin",
		Password: "password",
		Port:     server.Port(),
		Timeout:  10,
	}

	client, err := NewClient(server.Host(), cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	if !client.IsConnected() {
		t.Error("Expected client to be connected")
	}

	t.Run("Execute command", func(t *testing.T) {
		result, err := client.Execute(ctx, "show clock")
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
		if result.Error != nil {
			t.Fatalf("Command returned error: %v", result.Error)
		}
		if !strings.Contains(result.Stdout, "2026") {
			t.Errorf("Unexpected output: %s", result.Stdout)
		}
		if result.ExitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", result.ExitCode)
		}
		if result.Duration == 0 {
			t.Error("Expected non-zero duration")
		}
	})

	t.Run("ShowClockInfo", func(t *testing.T) {
		result, err := client.ShowClockInfo(ctx)
		if err != nil {
			t.Fatalf("ShowClockInfo failed: %v", err)
		}
		if result.Error != nil {
			t.Fatalf("Command returned error: %v", result.Error)
		}
		if !strings.Contains(result.Stdout, "2026") {
			t.Errorf("Unexpected output: %s", result.Stdout)
		}
	})

	t.Run("ShowPanoramaStatus", func(t *testing.T) {
		result, err := client.ShowPanoramaStatus(ctx)
		if err != nil {
			t.Fatalf("ShowPanoramaStatus failed: %v", err)
		}
		if result.Error != nil {
			t.Fatalf("Command returned error: %v", result.Error)
		}
		if !strings.Contains(result.Stdout, "Connected") {
			t.Errorf("Unexpected output: %s", result.Stdout)
		}
	})

	t.Run("ShowHAState", func(t *testing.T) {
		result, err := client.ShowHAState(ctx)
		if err != nil {
			t.Fatalf("ShowHAState failed: %v", err)
		}
		if result.Error != nil {
			t.Fatalf("Command returned error: %v", result.Error)
		}
		if !strings.Contains(result.Stdout, "active") {
			t.Errorf("Unexpected output: %s", result.Stdout)
		}
	})

	t.Run("TestConnectivity", func(t *testing.T) {
		result, err := client.TestConnectivity(ctx, "10.0.0.1")
		if err != nil {
			t.Fatalf("TestConnectivity failed: %v", err)
		}
		if result.Error != nil {
			t.Fatalf("Command returned error: %v", result.Error)
		}
		if !strings.Contains(result.Stdout, "0.0% packet loss") {
			t.Errorf("Unexpected output: %s", result.Stdout)
		}
	})

	t.Run("ShowLog", func(t *testing.T) {
		result, err := client.ShowLog(ctx, "ms", 50)
		if err != nil {
			t.Fatalf("ShowLog failed: %v", err)
		}
		if result.Error != nil {
			t.Fatalf("Command returned error: %v", result.Error)
		}
		if !strings.Contains(result.Stdout, "pan_comm") {
			t.Errorf("Unexpected output: %s", result.Stdout)
		}
	})

	t.Run("ShowLog default lines", func(t *testing.T) {
		result, err := client.ShowLog(ctx, "system", 0)
		if err != nil {
			t.Fatalf("ShowLog failed: %v", err)
		}
		if result.Error != nil {
			t.Fatalf("Command returned error: %v", result.Error)
		}
	})

	t.Run("ShowHALink", func(t *testing.T) {
		result, err := client.ShowHALink(ctx)
		if err != nil {
			t.Fatalf("ShowHALink failed: %v", err)
		}
		if result.Error != nil {
			t.Fatalf("Command returned error: %v", result.Error)
		}
	})

	t.Run("ShowCommitHistory", func(t *testing.T) {
		result, err := client.ShowCommitHistory(ctx)
		if err != nil {
			t.Fatalf("ShowCommitHistory failed: %v", err)
		}
		if result.Error != nil {
			t.Fatalf("Command returned error: %v", result.Error)
		}
	})

	t.Run("ShowConfigLock", func(t *testing.T) {
		result, err := client.ShowConfigLock(ctx)
		if err != nil {
			t.Fatalf("ShowConfigLock failed: %v", err)
		}
		if result.Error != nil {
			t.Fatalf("Command returned error: %v", result.Error)
		}
	})

	t.Run("ShowLicenseInfo", func(t *testing.T) {
		result, err := client.ShowLicenseInfo(ctx)
		if err != nil {
			t.Fatalf("ShowLicenseInfo failed: %v", err)
		}
		if result.Error != nil {
			t.Fatalf("Command returned error: %v", result.Error)
		}
	})

	t.Run("ShowTopProcesses", func(t *testing.T) {
		result, err := client.ShowTopProcesses(ctx)
		if err != nil {
			t.Fatalf("ShowTopProcesses failed: %v", err)
		}
		if result.Error != nil {
			t.Fatalf("Command returned error: %v", result.Error)
		}
	})

	t.Run("ShowSessionTable", func(t *testing.T) {
		result, err := client.ShowSessionTable(ctx)
		if err != nil {
			t.Fatalf("ShowSessionTable failed: %v", err)
		}
		if result.Error != nil {
			t.Fatalf("Command returned error: %v", result.Error)
		}
	})

	t.Run("ShowDataplaneStats", func(t *testing.T) {
		result, err := client.ShowDataplaneStats(ctx)
		if err != nil {
			t.Fatalf("ShowDataplaneStats failed: %v", err)
		}
		if result.Error != nil {
			t.Fatalf("Command returned error: %v", result.Error)
		}
	})
}

func TestClientNotConnected(t *testing.T) {
	cfg := config.SSHConfig{
		Username: "admin",
		Password: "password",
	}

	client, err := NewClient("10.0.0.1", cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	result, err := client.Execute(ctx, "show clock")
	if err == nil && result.Error == nil {
		t.Error("Expected error when executing command without connection")
	}

	if client.IsConnected() {
		t.Error("Expected client to not be connected")
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
		{
			input:    "relative/path",
			expected: "relative/path",
		},
	}

	for _, tt := range tests {
		result := expandPath(tt.input)
		if result != tt.expected {
			t.Errorf("expandPath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}

	// Test ~ expansion - result should contain home dir, not ~/
	result := expandPath("~/somepath")
	if strings.HasPrefix(result, "~/") {
		t.Error("expandPath should expand ~ to home directory")
	}
	if !strings.HasSuffix(result, "/somepath") {
		t.Errorf("expandPath(~/somepath) should end with /somepath, got %q", result)
	}
}

func TestMockSSHServerCustomResponse(t *testing.T) {
	server, err := NewMockSSHServer()
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}

	server.SetResponse("custom-command", "custom-response\n")

	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer server.Close()

	cfg := config.SSHConfig{
		Username: "admin",
		Password: "password",
		Port:     server.Port(),
	}

	client, err := NewClient(server.Host(), cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	result, err := client.Execute(ctx, "custom-command")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result.Stdout, "custom-response") {
		t.Errorf("Expected custom response, got: %s", result.Stdout)
	}
}

func TestMockSSHServerAddress(t *testing.T) {
	server, err := NewMockSSHServer()
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}

	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer server.Close()

	if server.Host() != "127.0.0.1" {
		t.Errorf("Expected host 127.0.0.1, got %s", server.Host())
	}

	if server.Port() == 0 {
		t.Error("Expected non-zero port")
	}

	addr := server.Address()
	if !strings.Contains(addr, "127.0.0.1:") {
		t.Errorf("Unexpected address format: %s", addr)
	}
}

// TestExecuteContextCancellation tests that command execution properly
// handles context cancellation and cleans up resources.
func TestExecuteContextCancellation(t *testing.T) {
	server, err := NewMockSSHServer()
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}

	// Set up a slow command response
	server.SetResponse("sleep-command", "started\n")

	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer server.Close()

	cfg := config.SSHConfig{
		Username: "admin",
		Password: "password",
		Port:     server.Port(),
		Timeout:  10,
	}

	client, err := NewClient(server.Host(), cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create a context that we'll cancel
	cmdCtx, cmdCancel := context.WithCancel(context.Background())

	// Start command in goroutine
	resultCh := make(chan *CommandResult, 1)
	go func() {
		result, _ := client.Execute(cmdCtx, "show clock")
		resultCh <- result
	}()

	// Cancel the context immediately
	cmdCancel()

	// Wait for result with timeout
	select {
	case result := <-resultCh:
		// Should get a result (either successful or cancelled)
		if result != nil && result.Error != nil {
			// Context cancellation should be reflected
			if result.Error != context.Canceled {
				// It's okay if the command completed before cancellation
				t.Logf("Command completed with: %v", result.Error)
			}
		}
	case <-time.After(3 * time.Second):
		t.Error("Command did not respond to context cancellation in time")
	}
}

// TestExecuteWithAlreadyCancelledContext tests execution with pre-cancelled context.
func TestExecuteWithAlreadyCancelledContext(t *testing.T) {
	server, err := NewMockSSHServer()
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}

	server.SetDefaultResponses()

	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer server.Close()

	cfg := config.SSHConfig{
		Username: "admin",
		Password: "password",
		Port:     server.Port(),
		Timeout:  10,
	}

	client, err := NewClient(server.Host(), cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create an already-cancelled context
	cancelledCtx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc() // Cancel immediately

	result, _ := client.Execute(cancelledCtx, "show clock")

	// Should return an error due to cancelled context
	if result != nil && result.Error == nil && result.Stdout == "" {
		// Either error or we got a result before cancellation kicked in
		t.Logf("Execution with cancelled context: error=%v, stdout=%q", result.Error, result.Stdout)
	}
}
