package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/config"
	"github.com/jp2195/pyre/internal/models"
	"github.com/jp2195/pyre/internal/testutil"
	"github.com/jp2195/pyre/internal/tui/views"
)

// connectedSessionsModel returns a model on the Sessions view with an active
// connection, n sessions loaded, and auto-refresh enabled and due. Port 1
// refuses at once, so a fetch it issues completes without a server.
func connectedSessionsModel(t *testing.T, n int) Model {
	t.Helper()
	m := newTestModel(t, ViewSessions)
	if _, err := m.session.AddConnection("127.0.0.1:1", &config.ConnectionConfig{}, "k"); err != nil {
		t.Fatalf("AddConnection: %v", err)
	}
	m.resetViewData()
	m.sessions = m.sessions.SetSessions(sessionsN(n), nil)
	m.autoRefresh = autoRefreshState{enabled: true, interval: 30 * time.Second}
	return m
}

func sessionsN(n int) []models.Session {
	out := make([]models.Session, n)
	for i := range out {
		out[i] = models.Session{ID: int64(i + 1)}
	}
	return out
}

func pressKey(t *testing.T, m Model, s string) Model {
	t.Helper()
	var msg tea.KeyPressMsg
	switch s {
	case "enter":
		msg = tea.KeyPressMsg{Code: tea.KeyEnter}
	default:
		r := []rune(s)[0]
		msg = tea.KeyPressMsg{Code: r, Text: s}
	}
	next, _ := m.Update(msg)
	return next.(Model)
}

func TestAutoRefresh_FiresWithoutBlankingTheView(t *testing.T) {
	m := connectedSessionsModel(t, 3)
	now := time.Now()

	next, cmd := m.handleClockTick(now)
	if cmd == nil {
		t.Fatal("due auto-refresh issued no command")
	}
	if next.autoRefresh.pending == 0 {
		t.Error("auto-refresh did not record the refresh it is waiting on")
	}
	// The whole point of tracking separately: a loading flag would blank
	// the table every interval.
	if next.sessions.IsLoading() {
		t.Error("auto-refresh set the loading flag; the table would blank")
	}
	if want := now.Add(30 * time.Second); !next.autoRefresh.nextDue.Equal(want) {
		t.Errorf("nextDue = %v, want %v", next.autoRefresh.nextDue, want)
	}
}

func TestAutoRefresh_Holds(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name  string
		setup func(t *testing.T, m Model) Model
	}{
		{"disabled", func(_ *testing.T, m Model) Model { m.autoRefresh.enabled = false; return m }},
		{"not yet due", func(_ *testing.T, m Model) Model { m.autoRefresh.nextDue = now.Add(time.Second); return m }},
		{"refresh in flight", func(_ *testing.T, m Model) Model { m.autoRefresh.pending = 7; return m }},
		{"manual fetch loading", func(_ *testing.T, m Model) Model { m.sessions = m.sessions.SetLoading(true); return m }},
		{"typing a filter", func(t *testing.T, m Model) Model { return pressKey(t, m, "/") }},
		{"detail open", func(t *testing.T, m Model) Model { return pressKey(t, m, "enter") }},
		{"configuration view", func(_ *testing.T, m Model) Model { m.currentView = ViewPolicies; return m }},
		{"config dashboard", func(_ *testing.T, m Model) Model {
			m.currentView, m.currentDashboard = ViewDashboard, views.DashboardConfig
			return m
		}},
		{"no connection", func(_ *testing.T, m Model) Model {
			m.session.RemoveConnection("127.0.0.1:1")
			return m
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setup(t, connectedSessionsModel(t, 3))
			if _, cmd := m.handleClockTick(now); cmd != nil {
				t.Error("auto-refresh fired; want it held")
			}
		})
	}
}

// TestAutoRefresh_HoldIsShown keeps a held refresh from looking like a broken
// one: the footer says why it is waiting.
func TestAutoRefresh_HoldIsShown(t *testing.T) {
	m := pressKey(t, connectedSessionsModel(t, 3), "enter")
	if got := m.renderRefreshStatus(time.Now()); !strings.Contains(got, "auto held (detail open)") {
		t.Errorf("status = %q, want the hold and its reason", got)
	}
}

func TestAutoRefresh_LogsHoldOffTheNewestRow(t *testing.T) {
	m := connectedSessionsModel(t, 0)
	m.currentView = ViewLogs
	if hold := m.autoRefreshHold(); hold != "" {
		t.Fatalf("fresh logs view held (%q); want it following", hold)
	}
	m.logs.Cursor = 3
	if hold := m.autoRefreshHold(); hold != "not at newest" {
		t.Errorf("hold = %q with the cursor off the first row, want %q", hold, "not at newest")
	}
}

func TestRefreshDone_ReplaysAndStamps(t *testing.T) {
	m := connectedSessionsModel(t, 0)
	m.sessions = m.sessions.SetSessions(nil, nil)
	m.autoRefresh.pending = 4
	done := refreshDoneMsg{
		token: 4,
		epoch: m.connEpoch,
		key:   viewKey{view: ViewSessions},
		msgs:  []tea.Msg{epochMsg{epoch: m.connEpoch, inner: SessionsMsg{Sessions: sessionsN(5)}}},
	}

	next, _ := m.Update(done)
	got := next.(Model)

	if !got.sessions.HasData() {
		t.Error("replayed sessions were not applied")
	}
	if got.autoRefresh.pending != 0 {
		t.Error("pending not cleared; auto-refresh would never fire again")
	}
	if at, ok := got.updatedAt[viewKey{view: ViewSessions}]; !ok || time.Since(at) > time.Second {
		t.Errorf("updatedAt = %v (set %v), want just now", at, ok)
	}
}

// TestRefreshDone_DropsSuperseded covers a manual r pressed while an
// automatic refresh was in flight: the older result must not land after the
// newer request and clear its loading state early.
func TestRefreshDone_DropsSuperseded(t *testing.T) {
	for _, tt := range []struct {
		name         string
		token, epoch int
	}{
		{"superseded token", 3, 0},
		{"old connection", 4, -1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := connectedSessionsModel(t, 0)
			m.sessions = m.sessions.SetSessions(nil, nil)
			m.autoRefresh.pending = 4
			done := refreshDoneMsg{
				token: tt.token,
				epoch: m.connEpoch + tt.epoch,
				key:   viewKey{view: ViewSessions},
				msgs:  []tea.Msg{epochMsg{epoch: m.connEpoch + tt.epoch, inner: SessionsMsg{Sessions: sessionsN(5)}}},
			}
			next, _ := m.Update(done)
			if next.(Model).sessions.HasData() {
				t.Error("stale result was applied; want it dropped")
			}
		})
	}
}

func TestAutoRefresh_ToggleKey(t *testing.T) {
	m := connectedSessionsModel(t, 3)
	m.autoRefresh.enabled = false

	m = pressKey(t, m, "R")
	if !m.autoRefresh.enabled {
		t.Fatal("R did not turn auto-refresh on")
	}
	if !m.autoRefresh.nextDue.After(time.Now()) {
		t.Error("turning it on should schedule the first refresh one interval out, not fire at once")
	}
	if got := m.renderRefreshStatus(time.Now()); !strings.Contains(got, "auto 30s") {
		t.Errorf("status = %q, want it to show the interval", got)
	}

	m = pressKey(t, m, "R")
	if m.autoRefresh.enabled {
		t.Error("second R did not turn auto-refresh off")
	}
}

func TestAutoRefresh_StartsFromConfig(t *testing.T) {
	for _, tt := range []struct {
		raw      string
		enabled  bool
		interval time.Duration
	}{
		{"", false, config.DefaultRefreshInterval},
		{"45s", true, 45 * time.Second},
		{"2s", true, config.MinRefreshInterval},
		{"soon", false, config.DefaultRefreshInterval},
	} {
		cfg := &config.Config{Settings: config.Settings{RefreshIntervalRaw: tt.raw}}
		got := newAutoRefreshState(cfg)
		if got.enabled != tt.enabled || got.interval != tt.interval {
			t.Errorf("%q: enabled=%v interval=%v, want %v %v", tt.raw, got.enabled, got.interval, tt.enabled, tt.interval)
		}
	}
}

func TestRenderRefreshStatus_ShowsAge(t *testing.T) {
	m := connectedSessionsModel(t, 3)
	m.autoRefresh.enabled = false
	now := time.Now()
	m.updatedAt[viewKey{view: ViewSessions}] = now.Add(-12 * time.Second)
	if got := m.renderRefreshStatus(now); got != "updated 12s ago" {
		t.Errorf("status = %q, want %q", got, "updated 12s ago")
	}
}

func TestStampLoaded_FollowsTheData(t *testing.T) {
	m := connectedSessionsModel(t, 3)
	k := viewKey{view: ViewSessions}
	m.stampLoaded(time.Now())
	if _, ok := m.updatedAt[k]; !ok {
		t.Fatal("loaded sessions not stamped")
	}
	m.sessions = m.sessions.SetSessions(nil, nil)
	m.stampLoaded(time.Now())
	if _, ok := m.updatedAt[k]; ok {
		t.Error("stamp kept after the data went away; the footer would report an age for nothing")
	}
}

func TestResetViewData_ForgetsRefreshState(t *testing.T) {
	m := connectedSessionsModel(t, 3)
	m.autoRefresh.pending = 9
	m.stampLoaded(time.Now())
	m.resetViewData()
	if m.autoRefresh.pending != 0 || len(m.updatedAt) != 0 {
		t.Errorf("pending=%d stamps=%d after a device switch, want both cleared", m.autoRefresh.pending, len(m.updatedAt))
	}
}

func TestClockTick_Reschedules(t *testing.T) {
	m := newTestModel(t, ViewSessions)
	if _, cmd := m.Update(clockTickMsg{now: time.Now()}); cmd == nil {
		t.Error("clock tick returned no command; the clock would stop")
	}
}

func TestRunCmdTree_FlattensNestedBatches(t *testing.T) {
	msg := func(s string) tea.Cmd { return func() tea.Msg { return s } }
	cmd := tea.Batch(msg("a"), tea.Batch(msg("b"), nil, msg("c")), nil)
	got := runCmdTree(cmd)
	if len(got) != 3 {
		t.Fatalf("got %v, want the three leaf messages", got)
	}
}

func TestFormatAge(t *testing.T) {
	for d, want := range map[time.Duration]string{
		-time.Second:     "0s",
		12 * time.Second: "12s",
		90 * time.Second: "1m",
		3 * time.Hour:    "3h",
	} {
		if got := formatAge(d); got != want {
			t.Errorf("formatAge(%v) = %q, want %q", d, got, want)
		}
	}
	if got := formatInterval(90 * time.Second); got != "90s" {
		t.Errorf("formatInterval(90s) = %q, want an exact %q", got, "90s")
	}
}

// TestAutoRefresh_EndToEnd runs a real tracked refresh against the mock
// PAN-OS server: the command tree is executed for real (Overview batches
// fourteen fetches), its result is replayed through Update, and the screen
// ends up holding data, stamped, with nothing left pending.
func TestAutoRefresh_EndToEnd(t *testing.T) {
	srv := testutil.NewMockPANOS()
	t.Cleanup(srv.Close)

	for _, tt := range []struct {
		name string
		view ViewState
		has  func(m Model) bool
	}{
		{"sessions", ViewSessions, func(m Model) bool { return m.sessions.HasData() }},
		{"overview", ViewDashboard, func(m Model) bool { return m.dashboard.HasData() }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel(t, tt.view)
			if _, err := m.session.AddConnection(srv.Host(), &config.ConnectionConfig{Insecure: true}, "k"); err != nil {
				t.Fatalf("AddConnection: %v", err)
			}
			m.resetViewData()
			// First load, as navigation does it. Until a dashboard has data
			// it counts as loading, and auto-refresh rightly waits for it.
			for _, msg := range runCmdTree(m.refreshCurrentView()) {
				next, _ := m.Update(msg)
				m = next.(Model)
			}
			if !tt.has(m) {
				t.Fatal("first load against the mock produced no data")
			}
			m.autoRefresh = autoRefreshState{enabled: true, interval: 30 * time.Second}

			next, cmd := m.handleClockTick(time.Now())
			if cmd == nil {
				t.Fatal("no refresh issued")
			}
			done, ok := cmd().(refreshDoneMsg)
			if !ok {
				t.Fatal("tracked refresh did not report back as one refreshDoneMsg")
			}
			after, _ := next.Update(done)
			got := after.(Model)

			if !tt.has(got) {
				t.Error("screen has no data after the refresh was replayed")
			}
			if got.autoRefresh.pending != 0 {
				t.Error("refresh still pending after its result was applied")
			}
			if _, ok := got.updatedAt[got.currentKey()]; !ok {
				t.Error("refresh not stamped")
			}
		})
	}
}

// TestFooter_RefreshStatusFits keeps the status from pushing the footer past
// the terminal edge, the defect class render.go's hint fitting exists for.
func TestFooter_RefreshStatusFits(t *testing.T) {
	for _, width := range []int{40, 60, 80, 120} {
		m := pressKey(t, connectedSessionsModel(t, 3), "enter") // longest hold text
		m.width = width
		m.updatedAt[viewKey{view: ViewSessions}] = time.Now().Add(-42 * time.Minute)
		for line := range strings.SplitSeq(m.renderFooter(), "\n") {
			if w := lipgloss.Width(line); w > width {
				t.Errorf("width %d: footer line is %d cells: %q", width, w, line)
			}
		}
	}
	m := connectedSessionsModel(t, 3)
	m.width = 120
	m.updatedAt[viewKey{view: ViewSessions}] = time.Now()
	if out := m.renderFooter(); !strings.Contains(out, "updated 0s ago · auto 30s") {
		t.Errorf("wide footer = %q, want the full status", out)
	}
}
