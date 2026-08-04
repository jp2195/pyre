package tui

import (
	"testing"

	"github.com/jp2195/pyre/internal/auth"
	"github.com/jp2195/pyre/internal/config"
)

// newTestModel constructs a Model with defaults suitable for TUI unit tests.
// It uses an empty in-memory config and state and nil credentials, and sets
// the given initial view.
func newTestModel(t *testing.T, initialView ViewState) Model {
	t.Helper()
	cfg := &config.Config{
		Connections: map[string]config.ConnectionConfig{},
	}
	state := &config.State{
		Connections: map[string]config.ConnectionState{},
	}
	creds := &auth.Credentials{}
	m, err := NewModel(cfg, state, creds, initialView)
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	// Give the model a non-zero size so renderContent paths don't short-circuit.
	m.width = 120
	m.height = 40
	return m
}
