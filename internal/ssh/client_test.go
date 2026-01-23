package ssh

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/joshuamontgomery/pyre/internal/config"
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
