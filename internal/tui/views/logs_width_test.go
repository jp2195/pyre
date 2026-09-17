package views

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/models"
)

// The log tables used one fixed set of column widths regardless of terminal
// size, so on a wide terminal they stopped around column 110 and truncated
// data there was plenty of room for. Several columns were also too narrow for
// their own vocabulary: Action at 7 cells cannot hold "sinkhole", and
// Severity at 9 cannot hold "informational", both of which this device emits.

func threatLogsModel(t *testing.T, width int) LogsModel {
	t.Helper()
	InitStyles()
	m := NewLogsModel().SetSize(width, 40)
	m, _ = m.SetThreatLogs([]models.ThreatLogEntry{{
		Time:           time.Date(2026, 9, 4, 21, 0, 0, 0, time.UTC),
		Severity:       "informational",
		ThreatName:     "Proxy:mask.test-dns.net-a-rather-long-indicator",
		SourceIP:       "10.0.40.15",
		DestIP:         "203.0.113.99",
		Action:         "sinkhole",
		ThreatCategory: "adns-proxy",
	}}, LogPageMeta{}, nil)
	m.activeLogType = models.LogTypeThreat
	return m
}

func TestThreatLogTable_ShowsFullVocabulary(t *testing.T) {
	out := plain(threatLogsModel(t, 200).View())

	for _, want := range []string{"informational", "sinkhole"} {
		if !strings.Contains(out, want) {
			t.Errorf("threat table truncates %q, which is a value the device emits:\n%s", want, out)
		}
	}
}

func TestThreatLogTable_UsesTheAvailableWidth(t *testing.T) {
	const width = 200
	out := plain(threatLogsModel(t, width).View())

	if !strings.Contains(out, "Proxy:mask.test-dns.net-a-rather-long-indicator") {
		t.Error("threat name truncated despite room for it at 200 columns")
	}

	widest := 0
	for l := range strings.SplitSeq(out, "\n") {
		if w := lipgloss.Width(strings.TrimRight(l, " ")); w > widest {
			widest = w
		}
	}
	if widest < width/2 {
		t.Errorf("widest rendered line is %d cells of %d available; the table is not using the terminal", widest, width)
	}
}

func TestThreatLogTable_FitsNarrowTerminals(t *testing.T) {
	const width = 90
	out := plain(threatLogsModel(t, width).View())
	for i, l := range strings.Split(out, "\n") {
		if w := lipgloss.Width(l); w > width {
			t.Fatalf("line %d is %d cells wide at a %d-column terminal", i+1, w, width)
		}
	}
}

func TestTrafficLogTable_ShowsFullActionAndUsesWidth(t *testing.T) {
	InitStyles()
	m := NewLogsModel().SetSize(200, 40)
	m, _ = m.SetTrafficLogs([]models.TrafficLogEntry{{
		Time:        time.Date(2026, 9, 4, 21, 0, 0, 0, time.UTC),
		Action:      "reset-both",
		SourceIP:    "10.0.0.5",
		DestIP:      "198.51.100.20",
		Application: "web-browsing",
		Rule:        "permit-with-a-long-rule-name",
		Bytes:       8877,
	}}, LogPageMeta{}, nil)
	m.activeLogType = models.LogTypeTraffic

	out := plain(m.View())
	if !strings.Contains(out, "reset-both") {
		t.Errorf("traffic table truncates the action:\n%s", out)
	}
	if !strings.Contains(out, "permit-with-a-long-rule-name") {
		t.Error("rule name truncated despite room for it at 200 columns")
	}
}
