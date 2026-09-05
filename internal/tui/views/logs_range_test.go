package views

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/models"
)

func TestLogRange_SinceAndLabel(t *testing.T) {
	now := time.Date(2026, 9, 5, 14, 0, 0, 0, time.UTC)
	tests := []struct {
		rng       LogRange
		wantLabel string
		wantSince time.Time
	}{
		{LogRangeAll, "all", time.Time{}},
		{LogRange15m, "15m", now.Add(-15 * time.Minute)},
		{LogRange1h, "1h", now.Add(-time.Hour)},
		{LogRange24h, "24h", now.Add(-24 * time.Hour)},
		{LogRange7d, "7d", now.Add(-7 * 24 * time.Hour)},
	}
	for _, tt := range tests {
		t.Run(tt.wantLabel, func(t *testing.T) {
			if got := tt.rng.Label(); got != tt.wantLabel {
				t.Errorf("Label() = %q, want %q", got, tt.wantLabel)
			}
			if got := tt.rng.Since(now); !got.Equal(tt.wantSince) {
				t.Errorf("Since() = %v, want %v", got, tt.wantSince)
			}
		})
	}
}

// The default must be `all`: no bound, nothing hidden, which is exactly what
// the view did before server-side queries existed.
func TestLogsModel_DefaultRangeIsAll(t *testing.T) {
	m := NewLogsModel()
	if m.Range() != LogRangeAll {
		t.Errorf("default range = %v, want LogRangeAll", m.Range())
	}
	if since := m.Range().Since(time.Now()); !since.IsZero() {
		t.Errorf("default range bound = %v, want zero", since)
	}
	if m.Query() != "" {
		t.Errorf("default Query() = %q, want empty", m.Query())
	}
}

func TestLogsModel_TCyclesRangeAndRefetches(t *testing.T) {
	m := NewLogsModel()
	m, _ = m.SetSystemLogs([]models.SystemLogEntry{{Type: "SYSTEM"}}, LogPageMeta{}, nil)

	want := []LogRange{LogRange15m, LogRange1h, LogRange24h, LogRange7d, LogRangeAll}
	for i, wantRange := range want {
		var cmd tea.Cmd
		m, cmd = m.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
		if m.Range() != wantRange {
			t.Fatalf("after %d presses range = %v, want %v", i+1, m.Range(), wantRange)
		}
		if cmd == nil {
			t.Fatalf("press %d returned no refetch command", i+1)
		}
		if _, ok := cmd().(FetchLogsCmd); !ok {
			t.Fatalf("press %d did not emit FetchLogsCmd", i+1)
		}
	}
}

func TestLogsModel_ShiftTCyclesBackward(t *testing.T) {
	m := NewLogsModel()
	m, _ = m.Update(tea.KeyPressMsg{Code: 'T', Text: "T"})
	if m.Range() != LogRange7d {
		t.Errorf("range = %v, want LogRange7d (wrapped backward from all)", m.Range())
	}
}

// Changing the range leaves the tabs the operator cannot see holding rows
// from the old range, so they read as stale and are refetched when next
// shown rather than silently mixing two ranges in one table.
func TestLogsModel_RangeChangeMakesOtherTabsStale(t *testing.T) {
	m := NewLogsModel()
	m, _ = m.SetSystemLogs([]models.SystemLogEntry{{Type: "SYSTEM"}}, LogPageMeta{}, nil)
	m, _ = m.SetTrafficLogs([]models.TrafficLogEntry{{Action: "allow"}}, LogPageMeta{}, nil)

	if m.tabStale(models.LogTypeTraffic) {
		t.Fatal("traffic was stale before anything changed")
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: 't', Text: "t"})

	if !m.tabStale(models.LogTypeTraffic) {
		t.Error("traffic tab is not stale after a range change")
	}
}

// A page arriving under the current range clears that tab's staleness,
// because the rows now carry the range they were fetched under.
func TestLogsModel_CompletedPageClearsStaleness(t *testing.T) {
	m := NewLogsModel()
	m, _ = m.SetSystemLogs([]models.SystemLogEntry{{Type: "SYSTEM"}}, LogPageMeta{}, nil)

	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
	if !m.tabStale(models.LogTypeSystem) {
		t.Fatal("the active tab should read as stale until its new page lands")
	}

	// The page has to carry the request it answers: a tab is only marked
	// with the range and query the rows were actually fetched under.
	m, _ = m.SetSystemLogs([]models.SystemLogEntry{{Type: "SYSTEM"}},
		LogPageMeta{Req: mustFetchReq(t, cmd)}, nil)
	if m.tabStale(models.LogTypeSystem) {
		t.Error("staleness survived a page fetched under the current range")
	}
}

func TestLogsModel_StatusLineShowsRange(t *testing.T) {
	m := NewLogsModel().SetSize(100, 40)
	m, _ = m.SetSystemLogs([]models.SystemLogEntry{{Type: "SYSTEM"}}, LogPageMeta{}, nil)
	m, _ = m.Update(tea.KeyPressMsg{Code: 't', Text: "t"})

	if got := m.statusLine(); !strings.Contains(got, "15m") {
		t.Errorf("status line %q does not name the range", got)
	}
}

// The supported floor is 60 columns. The status line must fit at every width
// the view claims to support, and must still name the range at the floor.
func TestLogsModel_StatusLineFitsEveryWidth(t *testing.T) {
	for _, width := range []int{60, 80, 120} {
		m := NewLogsModel().SetSize(width, 40)
		m, _ = m.SetSystemLogs(make([]models.SystemLogEntry, 500), LogPageMeta{HasMore: true}, nil)
		m, _ = m.Update(tea.KeyPressMsg{Code: 't', Text: "t"})

		line := m.statusLine()
		for _, rendered := range strings.Split(line, "\n") {
			if got := lipgloss.Width(rendered); got > width {
				t.Errorf("width %d: status line is %d cells wide: %q", width, got, rendered)
			}
		}
		if !strings.Contains(line, "15m") {
			t.Errorf("width %d: status line %q dropped the range", width, line)
		}
	}
}

// The query is appended to the status line only once there is room for it at
// width 80. Pinning only one side of the threshold would let an off-by-one
// (>80 instead of >=80) slip through undetected, so this checks both: present
// at 80, absent at 79. There is no exported setter for the query, so the
// unexported field is set directly -- this test lives in the same package.
func TestLogsModel_StatusLineQueryThresholdAt80(t *testing.T) {
	tests := []struct {
		width     int
		wantQuery bool
	}{
		{80, true},
		{79, false},
	}
	for _, tt := range tests {
		m := NewLogsModel().SetSize(tt.width, 40)
		m.query = "action eq deny"

		got := m.statusLine()
		hasQuery := strings.Contains(got, "action eq deny")
		if hasQuery != tt.wantQuery {
			t.Errorf("width %d: query present = %v, want %v (line: %q)", tt.width, hasQuery, tt.wantQuery, got)
		}
	}
}

// Below the 60-column floor -- this project's supported minimum -- the status
// line must degrade to the compact "N+" form rather than the long "shown,
// more available" phrasing, which does not fit down there.
func TestLogsModel_StatusLineBelowFloorUsesCompactForm(t *testing.T) {
	m := NewLogsModel().SetSize(40, 40)
	m, _ = m.SetSystemLogs(make([]models.SystemLogEntry, 500), LogPageMeta{HasMore: true}, nil)

	got := m.statusLine()
	if !strings.Contains(got, "500+") {
		t.Errorf("status line at width 40 = %q, want the compact \"500+\" form", got)
	}
	if strings.Contains(got, "shown, more available") {
		t.Errorf("status line at width 40 = %q, used the long form below the floor", got)
	}
}
