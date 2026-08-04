package views

import (
	"strings"
	"testing"
	"time"

	"github.com/jp2195/pyre/internal/config"
	"github.com/jp2195/pyre/internal/tui/theme"
)

// TestConnectionHubModel_CursorOnInterleavedPanorama regression-guards a bug
// where the View() function split connections into separate firewall/panorama
// sections while m.cursor indexed the original sorted slice. When a Panorama
// was interleaved (by recency) between firewalls, the cursor highlighted the
// wrong row.
func TestConnectionHubModel_CursorOnInterleavedPanorama(t *testing.T) {
	theme.Init("dark")
	InitStyles()

	now := time.Now()

	cfg := config.DefaultConfig()
	cfg.Connections["fw-1"] = config.ConnectionConfig{Type: "firewall"}
	cfg.Connections["panorama-mid"] = config.ConnectionConfig{Type: "panorama"}
	cfg.Connections["fw-2"] = config.ConnectionConfig{Type: "firewall"}

	state := &config.State{
		Connections: map[string]config.ConnectionState{
			"fw-1":         {LastConnected: now.Add(-1 * time.Hour)},
			"panorama-mid": {LastConnected: now.Add(-2 * time.Hour)},
			"fw-2":         {LastConnected: now.Add(-3 * time.Hour)},
		},
	}

	m := NewConnectionHubModel().SetConnections(cfg, state)
	m = m.SetSize(120, 30)

	// Verify ordering: recency sort should produce fw-1, panorama-mid, fw-2.
	if len(m.connections) != 3 {
		t.Fatalf("expected 3 connections, got %d", len(m.connections))
	}
	want := []string{"fw-1", "panorama-mid", "fw-2"}
	for i, w := range want {
		if m.connections[i].Host != w {
			t.Fatalf("expected sorted order %v, got %s at index %d",
				want, m.connections[i].Host, i)
		}
	}

	// Render with cursor on each row in turn; the rendered line containing
	// the host name must also carry the selected style for that cursor index.
	for idx, host := range want {
		mc := m
		mc.cursor = idx
		out := mc.View()

		hostLine := findLineContaining(out, host)
		if hostLine == "" {
			t.Fatalf("cursor=%d: host %q not present in output:\n%s", idx, host, out)
		}
		if !hasSelectionStyling(hostLine) {
			t.Errorf("cursor=%d: expected host %q to be rendered with selection styling, line was:\n%q\nfull output:\n%s",
				idx, host, hostLine, out)
		}

		// Other rows should not be selected.
		for otherIdx, otherHost := range want {
			if otherIdx == idx {
				continue
			}
			line := findLineContaining(out, otherHost)
			if line == "" {
				continue
			}
			if hasSelectionStyling(line) {
				t.Errorf("cursor=%d: host %q should NOT be selected but its line had selection styling: %q",
					idx, otherHost, line)
			}
		}
	}
}

// findLineContaining returns the first line of out that contains substr, or "".
func findLineContaining(out, substr string) string {
	for line := range strings.SplitSeq(out, "\n") {
		if strings.Contains(line, substr) {
			return line
		}
	}
	return ""
}

// hasSelectionStyling reports whether the line was rendered with
// TableRowSelectedStyle. The selected style sets a non-default background via
// an ANSI 48; escape sequence; the normal row style sets only padding.
func hasSelectionStyling(line string) bool {
	// "\x1b[48" is the CSI introducer for a background color sequence.
	return strings.Contains(line, "\x1b[48")
}
