package config

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadState_WarnsOnPermissiveMode(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	// Build the state path the way StatePath() does (~/.pyre/state.json).
	stateDir := filepath.Join(tmpDir, ".pyre")
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	statePath := filepath.Join(stateDir, "state.json")
	if err := os.WriteFile(statePath, []byte(`{"connections":{}}`), 0644); err != nil {
		t.Fatalf("seeding state: %v", err)
	}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	if _, err := LoadState(); err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if !strings.Contains(buf.String(), "permissive mode") {
		t.Errorf("expected permissive-mode warning for 0644 state file, log output:\n%s", buf.String())
	}

	// 0600 must NOT warn.
	if err := os.Chmod(statePath, 0600); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	buf.Reset()
	if _, err := LoadState(); err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if strings.Contains(buf.String(), "permissive mode") {
		t.Errorf("unexpected warning for 0600 state file:\n%s", buf.String())
	}
}
