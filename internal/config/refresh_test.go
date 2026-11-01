package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRefreshInterval(t *testing.T) {
	for _, tt := range []struct {
		raw      string
		want     time.Duration
		wantNote bool
		wantErr  bool
	}{
		{raw: "", want: 0},
		{raw: "0s", want: 0},
		{raw: "30s", want: 30 * time.Second},
		{raw: "2m", want: 2 * time.Minute},
		{raw: "10s", want: MinRefreshInterval},
		{raw: "1s", want: MinRefreshInterval, wantNote: true},
		{raw: "-5s", wantErr: true},
		{raw: "30", wantErr: true}, // a bare number has no unit
		{raw: "soon", wantErr: true},
	} {
		got, note, err := Settings{RefreshIntervalRaw: tt.raw}.RefreshInterval()
		if (err != nil) != tt.wantErr {
			t.Errorf("%q: err = %v, want error %v", tt.raw, err, tt.wantErr)
			continue
		}
		if got != tt.want || (note != "") != tt.wantNote {
			t.Errorf("%q: got %v note %q, want %v (note %v)", tt.raw, got, note, tt.want, tt.wantNote)
		}
	}
}

// TestLoadWithFlags_WarnsAboutRefreshInterval keeps a typo from silently
// leaving auto-refresh off, or a too-short interval from being raised
// without saying so.
func TestLoadWithFlags_WarnsAboutRefreshInterval(t *testing.T) {
	for _, tt := range []struct{ raw, want string }{
		{"soon", "auto-refresh starts off"},
		{"2s", "below the 10s minimum"},
	} {
		buf := captureWarnings(t)
		path := filepath.Join(t.TempDir(), "pyre.yaml")
		if err := os.WriteFile(path, []byte("settings:\n  refresh_interval: "+tt.raw+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, err := LoadWithFlags(CLIFlags{Config: path})
		if err != nil {
			t.Fatalf("%q: LoadWithFlags: %v", tt.raw, err)
		}
		if cfg.Settings.RefreshIntervalRaw != tt.raw {
			t.Errorf("%q: loaded %q", tt.raw, cfg.Settings.RefreshIntervalRaw)
		}
		if !strings.Contains(buf.String(), tt.want) {
			t.Errorf("%q: warnings = %q, want %q", tt.raw, buf.String(), tt.want)
		}
	}
}
