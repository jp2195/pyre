package views

import (
	"testing"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/models"
)

// TestLogTables_FitTheTerminalWidth checks that each log table renders inside
// the width it was given.
//
// The tables pick a column set from the terminal width, but the pick was made
// by comparing against a threshold rather than by checking the chosen set
// actually fits. The narrow traffic set is 88 cells at its smallest, so it
// overflowed every terminal below 88 columns, the standard 80 included. The
// overflow is not confined to the table: the screen is joined vertically, so
// lipgloss pads every other line to match and the whole interface shifts off
// the right edge.
func TestLogTables_FitTheTerminalWidth(t *testing.T) {
	now := time.Date(2026, 9, 4, 22, 6, 30, 0, time.UTC)

	system := []models.SystemLogEntry{{
		Time: now, Severity: "informational", Type: "general",
		Description: "A description long enough to fill whatever column it is given, and then some.",
	}}
	traffic := []models.TrafficLogEntry{{
		Time: now, Action: "reset-client", SourceIP: "203.0.113.42", DestIP: "198.51.100.7",
		Application: "web-browsing", Rule: "allow-outbound-web-traffic", Bytes: 123456789,
	}}
	threat := []models.ThreatLogEntry{{
		Time: now, Severity: "informational", ThreatName: "Suspicious DNS Query",
		SourceIP: "203.0.113.42", DestIP: "198.51.100.7", Action: "reset-client",
		ThreatCategory: "command-and-control",
	}}

	tabs := []struct {
		name string
		set  func(LogsModel) LogsModel
	}{
		{"system", func(m LogsModel) LogsModel {
			return m.SetSystemLogs(system, LogPageMeta{}, nil).SetActiveLogType(models.LogTypeSystem)
		}},
		{"traffic", func(m LogsModel) LogsModel {
			return m.SetTrafficLogs(traffic, LogPageMeta{}, nil).SetActiveLogType(models.LogTypeTraffic)
		}},
		{"threat", func(m LogsModel) LogsModel {
			return m.SetThreatLogs(threat, LogPageMeta{}, nil).SetActiveLogType(models.LogTypeThreat)
		}},
	}

	for _, tab := range tabs {
		// 60 columns is the narrowest terminal the application supports. The
		// tables themselves hold together lower than that: the compact sets
		// bottom out at 43 cells for system, 42 for threat, and 51 for
		// traffic, below which their remaining columns cannot be dropped
		// without gutting the table.
		for width := 60; width <= 220; width++ {
			m := tab.set(NewLogsModel()).SetSize(width, 30)
			for _, line := range splitLines(m.View()) {
				if got := lipgloss.Width(line); got > width {
					t.Errorf("%s at width %d: line is %d cells, %d over\n  %q",
						tab.name, width, got, got-width, line)
					break
				}
			}
		}
	}
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := range len(s) {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}
