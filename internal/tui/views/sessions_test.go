package views

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
)

func TestNewSessionsModel(t *testing.T) {
	m := NewSessionsModel()

	if m.list.Cursor != 0 {
		t.Errorf("expected Cursor=0, got %d", m.list.Cursor)
	}
	if m.list.FilterMode {
		t.Error("expected FilterMode=false")
	}
	if m.list.sortBy != 0 {
		t.Errorf("expected default sort by ID, got %d", m.list.sortBy)
	}
}

func TestSessionsModel_SetSize(t *testing.T) {
	m := NewSessionsModel()
	m = m.SetSize(100, 50)

	if m.list.Width != 100 {
		t.Errorf("expected Width=100, got %d", m.list.Width)
	}
	if m.list.Height != 50 {
		t.Errorf("expected Height=50, got %d", m.list.Height)
	}
}

func TestSessionsModel_SetLoading(t *testing.T) {
	m := NewSessionsModel()

	m = m.SetLoading(true)
	if !m.list.Loading {
		t.Error("expected Loading=true")
	}

	m = m.SetLoading(false)
	if m.list.Loading {
		t.Error("expected Loading=false")
	}
}

func TestSessionsModel_SetSessions(t *testing.T) {
	m := NewSessionsModel()

	sessions := []models.Session{
		{ID: 1, Application: "web-browsing", SourceIP: "10.0.0.1", DestIP: "8.8.8.8"},
		{ID: 2, Application: "ssl", SourceIP: "10.0.0.2", DestIP: "1.1.1.1"},
	}

	m = m.SetSessions(sessions, nil)

	if len(m.list.Items()) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(m.list.Items()))
	}
	if m.list.Loading {
		t.Error("expected Loading=false after SetSessions")
	}
	if m.list.Cursor != 0 {
		t.Error("expected Cursor to reset to 0")
	}
}

func TestSessionsModel_SetSessions_WithError(t *testing.T) {
	m := NewSessionsModel()

	m = m.SetSessions(nil, errTest)

	if m.list.Err != errTest {
		t.Error("expected error to be set")
	}
}

func TestSessionsModel_Filtering(t *testing.T) {
	m := NewSessionsModel()
	m = m.SetSize(100, 50)

	sessions := []models.Session{
		{ID: 1, Application: "web-browsing", SourceIP: "10.0.0.1"},
		{ID: 2, Application: "ssl", SourceIP: "10.0.0.2"},
		{ID: 3, Application: "dns", SourceIP: "192.168.1.1"},
	}

	m = m.SetSessions(sessions, nil)

	// Initially all sessions should be visible
	if len(m.list.Filtered()) != 3 {
		t.Errorf("expected 3 filtered sessions, got %d", len(m.list.Filtered()))
	}

	// Apply filter
	m.list.Filter.SetValue("ssl")
	m.list.applyFilter()

	if len(m.list.Filtered()) != 1 {
		t.Errorf("expected 1 filtered session for 'ssl', got %d", len(m.list.Filtered()))
	}

	// Clear filter
	m.list.Filter.SetValue("")
	m.list.applyFilter()

	if len(m.list.Filtered()) != 3 {
		t.Errorf("expected 3 filtered sessions after clear, got %d", len(m.list.Filtered()))
	}
}

func TestSessionsModel_Sorting(t *testing.T) {
	m := NewSessionsModel()

	sessions := []models.Session{
		{ID: 3, Application: "dns", BytesIn: 100, BytesOut: 50, StartTime: time.Now().Add(-1 * time.Hour)},
		{ID: 1, Application: "web", BytesIn: 500, BytesOut: 200, StartTime: time.Now().Add(-30 * time.Minute)},
		{ID: 2, Application: "ssl", BytesIn: 200, BytesOut: 100, StartTime: time.Now().Add(-2 * time.Hour)},
	}

	m = m.SetSessions(sessions, nil)

	// Default sort by ID descending (SortAsc defaults to false)
	if m.list.Filtered()[0].ID != 3 {
		t.Errorf("expected first session to have ID 3 with default descending sort, got %d", m.list.Filtered()[0].ID)
	}

	// Cycle to bytes sort
	m.list.cycleSort()
	if m.list.sortBy != 1 {
		t.Errorf("expected sort by Bytes, got %d", m.list.sortBy)
	}

	// Cycle to age sort
	m.list.cycleSort()
	if m.list.sortBy != 2 {
		t.Errorf("expected sort by Age, got %d", m.list.sortBy)
	}

	// Cycle to application sort
	m.list.cycleSort()
	if m.list.sortBy != 3 {
		t.Errorf("expected sort by Application, got %d", m.list.sortBy)
	}

	// Cycle back to ID
	m.list.cycleSort()
	if m.list.sortBy != 0 {
		t.Errorf("expected sort by ID, got %d", m.list.sortBy)
	}
}

func TestSessionsModel_SortLabel(t *testing.T) {
	m := NewSessionsModel()

	// ID sort
	label := m.list.sortLabel()
	if label == "" {
		t.Error("expected non-empty sort label")
	}

	// Bytes sort
	m.list.sortBy = 1
	label = m.list.sortLabel()
	if label == "" {
		t.Error("expected non-empty sort label for bytes")
	}

	// Age sort
	m.list.sortBy = 2
	label = m.list.sortLabel()
	if label == "" {
		t.Error("expected non-empty sort label for age")
	}

	// Application sort
	m.list.sortBy = 3
	label = m.list.sortLabel()
	if label == "" {
		t.Error("expected non-empty sort label for application")
	}
}

func TestSessionsModel_Update_Navigation(t *testing.T) {
	m := NewSessionsModel()
	m = m.SetSize(100, 50)

	sessions := []models.Session{
		{ID: 1},
		{ID: 2},
		{ID: 3},
	}
	m = m.SetSessions(sessions, nil)

	// Move down
	m, _ = m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if m.list.Cursor != 1 {
		t.Errorf("expected Cursor=1 after j, got %d", m.list.Cursor)
	}

	// Move up
	m, _ = m.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	if m.list.Cursor != 0 {
		t.Errorf("expected Cursor=0 after k, got %d", m.list.Cursor)
	}

	// Sort key
	m, _ = m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	if m.list.sortBy != 1 {
		t.Errorf("expected sort to change after s, got %d", m.list.sortBy)
	}
}

func TestSessionsModel_View(t *testing.T) {
	m := NewSessionsModel()
	m = m.SetSize(100, 50)

	// View without sessions
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}

	// View with sessions
	sessions := []models.Session{
		{ID: 1, Application: "web", SourceIP: "10.0.0.1", DestIP: "8.8.8.8", State: "ACTIVE"},
	}
	m = m.SetSessions(sessions, nil)

	view = m.View()
	if view == "" {
		t.Error("expected non-empty view with sessions")
	}
}

func TestSessionsModel_View_ZeroWidth(t *testing.T) {
	m := NewSessionsModel()
	// Don't set size

	view := m.View()
	if !strings.Contains(view, "Loading...") {
		t.Errorf("expected view to contain 'Loading...' with zero width, got %q", view)
	}
}

// Error for testing
var errTest = &testError{}

type testError struct{}

func (e *testError) Error() string { return "test error" }
