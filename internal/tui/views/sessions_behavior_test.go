package views

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
)

func sessionsFixture() []models.Session {
	return []models.Session{
		{ID: 101, Application: "web-browsing", SourceIP: "10.0.0.1", DestIP: "8.8.8.8", State: "ACTIVE"},
		{ID: 202, Application: "ssl", SourceIP: "10.0.0.2", DestIP: "1.1.1.1", State: "ACTIVE"},
	}
}

func TestSessions_Behavior_DKeyEmitsFetchDetailCmd(t *testing.T) {
	InitStyles()
	m := NewSessionsModel()
	m = m.SetSize(120, 40)
	m = m.SetSessions(sessionsFixture(), nil)

	// Default sort: ID descending, so cursor 0 = session 202.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // expand detail
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if cmd == nil {
		t.Fatal("expected a command from d keypress with detail expanded")
	}
	msg := cmd()
	fd, ok := msg.(FetchDetailCmd)
	if !ok {
		t.Fatalf("expected FetchDetailCmd, got %T", msg)
	}
	if fd.SessionID != 202 {
		t.Errorf("SessionID = %d, want 202 (cursor row)", fd.SessionID)
	}
}

func TestSessions_Behavior_DKeyIgnoredWhileLoadingOrCollapsed(t *testing.T) {
	InitStyles()
	m := NewSessionsModel()
	m = m.SetSize(120, 40)
	m = m.SetSessions(sessionsFixture(), nil)

	// Collapsed: d does nothing.
	m2, cmd := m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if cmd != nil {
		t.Error("expected no command from d while collapsed")
	}
	m = m2

	// Expanded, first d starts the fetch; second d while loading is ignored.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	_, cmd = m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if cmd != nil {
		t.Error("expected no command from second d while detail is loading")
	}
}

func TestSessions_Behavior_SetDetailShowsExtendedSections(t *testing.T) {
	InitStyles()
	m := NewSessionsModel()
	m = m.SetSize(120, 50)
	m = m.SetSessions(sessionsFixture(), nil)

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	out := m.View()
	if !strings.Contains(out, "d: fetch extended details") {
		t.Fatalf("expected fetch hint before detail load:\n%s", out)
	}

	// Cursor row is session 202 (ID-descending default sort).
	m = m.SetDetail(&models.SessionDetail{ID: 202, NATRule: "nat-out", NATDestIP: "192.0.2.1"}, nil)
	out = m.View()
	if !strings.Contains(out, "NAT Details") {
		t.Errorf("expected NAT Details section after SetDetail:\n%s", out)
	}
	if !strings.Contains(out, "nat-out") {
		t.Errorf("expected NAT rule value after SetDetail:\n%s", out)
	}
}

func TestSessions_Behavior_CursorMoveClearsCachedDetail(t *testing.T) {
	InitStyles()
	m := NewSessionsModel()
	m = m.SetSize(120, 50)
	m = m.SetSessions(sessionsFixture(), nil)

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = m.SetDetail(&models.SessionDetail{ID: 202, NATRule: "nat-out", NATDestIP: "192.0.2.1"}, nil)

	m, _ = m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	out := m.View()
	if strings.Contains(out, "NAT Details") {
		t.Errorf("expected cached detail cleared after cursor move:\n%s", out)
	}
	if !strings.Contains(out, "d: fetch extended details") {
		t.Errorf("expected fetch hint again after cursor move:\n%s", out)
	}
}

func TestSessions_Behavior_FilterKeystrokesNarrowRows(t *testing.T) {
	InitStyles()
	m := NewSessionsModel()
	m = m.SetSize(120, 40)
	m = m.SetSessions(sessionsFixture(), nil)

	for _, r := range "/ssl" {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	out := m.View()
	if !strings.Contains(out, "ssl") {
		t.Errorf("expected ssl session after filter:\n%s", out)
	}
	if strings.Contains(out, "web-browsing") {
		t.Errorf("expected web-browsing filtered out:\n%s", out)
	}
}
