package config

import (
	"bytes"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureWarnings redirects package warnings for the duration of a test.
func captureWarnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := warnOut
	warnOut = &buf
	t.Cleanup(func() { warnOut = prev })
	return &buf
}

// silenceStdLogger reproduces what main does before loading anything: the
// standard logger is pointed at io.Discard so stray log.Printf output cannot
// corrupt the TUI. Any warning that goes through that logger is invisible.
func silenceStdLogger(t *testing.T) {
	t.Helper()
	prev := log.Writer()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(prev) })
}

// TestLoadState_WarningSurvivesSilencedLogger is the regression that matters:
// the permissive-mode warning was written with log.Printf, but main silences
// the standard logger before LoadState runs, so the warning documented in
// SECURITY.md and configuration.md had never once reached a user.
func TestLoadState_WarningSurvivesSilencedLogger(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	stateDir := filepath.Join(tmpDir, ".pyre")
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "state.json"), []byte(`{"connections":{}}`), 0644); err != nil {
		t.Fatalf("seeding state: %v", err)
	}

	silenceStdLogger(t)
	buf := captureWarnings(t)

	if _, err := LoadState(); err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if !strings.Contains(buf.String(), "permissive mode") {
		t.Errorf("no permissive-mode warning reached the user; got %q", buf.String())
	}
}

// TestLoad_WarningSurvivesSilencedLogger covers the same path for
// ~/.pyre.yaml, which had no test at all.
func TestLoad_WarningSurvivesSilencedLogger(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, ".pyre.yaml"), []byte("settings:\n  theme: dark\n"), 0644); err != nil {
		t.Fatalf("seeding config: %v", err)
	}

	silenceStdLogger(t)
	buf := captureWarnings(t)

	if _, err := Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !strings.Contains(buf.String(), "permissive mode") {
		t.Errorf("no permissive-mode warning reached the user; got %q", buf.String())
	}
}

// TestLoad_NoWarningForSecureMode keeps the check from crying wolf.
func TestLoad_NoWarningForSecureMode(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, ".pyre.yaml"), []byte("settings:\n  theme: dark\n"), 0600); err != nil {
		t.Fatalf("seeding config: %v", err)
	}

	buf := captureWarnings(t)

	if _, err := Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if strings.Contains(buf.String(), "permissive mode") {
		t.Errorf("unexpected warning for a 0600 config: %q", buf.String())
	}
}

// TestSave_WritesToTheLoadedPath covers --config being read-only: the flag
// selected which file to read, but every save resolved ~/.pyre.yaml, so edits
// made while pointed at an alternate config landed in the default file.
func TestSave_WritesToTheLoadedPath(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	customDir := t.TempDir()
	customPath := filepath.Join(customDir, "custom.yaml")
	if err := os.WriteFile(customPath, []byte("connections:\n  a.example.com: {}\n"), 0600); err != nil {
		t.Fatalf("seeding custom config: %v", err)
	}

	cfg, err := LoadWithFlags(CLIFlags{Config: customPath})
	if err != nil {
		t.Fatalf("LoadWithFlags: %v", err)
	}
	cfg.SetConnection("b.example.com", ConnectionConfig{Username: "operator"})
	if saveErr := cfg.Save(); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	saved, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatalf("reading custom config: %v", err)
	}
	if !strings.Contains(string(saved), "b.example.com") {
		t.Errorf("custom config does not contain the new connection:\n%s", saved)
	}
	if _, err := os.Stat(filepath.Join(homeDir, ".pyre.yaml")); !os.IsNotExist(err) {
		t.Error("~/.pyre.yaml was written even though --config named another file")
	}
}

// TestSave_DefaultsToHomeConfig keeps the ordinary path working.
func TestSave_DefaultsToHomeConfig(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	cfg, err := LoadWithFlags(CLIFlags{})
	if err != nil {
		t.Fatalf("LoadWithFlags: %v", err)
	}
	cfg.SetConnection("a.example.com", ConnectionConfig{})
	if saveErr := cfg.Save(); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	if _, err := os.Stat(filepath.Join(homeDir, ".pyre.yaml")); err != nil {
		t.Errorf("expected ~/.pyre.yaml to be written: %v", err)
	}
}
