package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// State holds persistent state about connections (last used, connect count, etc.)
type State struct {
	Connections map[string]ConnectionState `json:"connections"`
}

// ConnectionState holds state for a single connection
type ConnectionState struct {
	LastConnected time.Time `json:"last_connected,omitempty"`
	LastUser      string    `json:"last_user,omitempty"`
	ConnectCount  int       `json:"connect_count"`
}

// StatePath returns the path to the state file (~/.pyre/state.json)
func StatePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".pyre", "state.json"), nil
}

// LoadState loads the state from disk, or returns an empty state if not found
func LoadState() (*State, error) {
	state := &State{
		Connections: make(map[string]ConnectionState),
	}

	statePath, err := StatePath()
	if err != nil {
		return state, nil
	}

	data, err := os.ReadFile(statePath) // #nosec G304 -- Path is constructed from user's home directory
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	if err := json.Unmarshal(data, state); err != nil {
		// If state file is corrupted, start fresh
		return &State{Connections: make(map[string]ConnectionState)}, nil
	}

	if state.Connections == nil {
		state.Connections = make(map[string]ConnectionState)
	}

	return state, nil
}

// Save writes the state to disk
func (s *State) Save() error {
	statePath, err := StatePath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(statePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(statePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// UpdateConnection updates the state for a connection after a successful connection
func (s *State) UpdateConnection(host, user string) {
	if s.Connections == nil {
		s.Connections = make(map[string]ConnectionState)
	}

	state := s.Connections[host]
	state.LastConnected = time.Now()
	state.LastUser = user
	state.ConnectCount++
	s.Connections[host] = state
}

// GetConnection returns the state for a connection, or nil if not found
func (s *State) GetConnection(host string) *ConnectionState {
	if s.Connections == nil {
		return nil
	}
	state, ok := s.Connections[host]
	if !ok {
		return nil
	}
	return &state
}

// DeleteConnection removes state for a connection
func (s *State) DeleteConnection(host string) {
	if s.Connections != nil {
		delete(s.Connections, host)
	}
}
