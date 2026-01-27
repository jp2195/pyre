package views

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jp2195/pyre/internal/models"
)

func TestNewLogsModel(t *testing.T) {
	m := NewLogsModel()

	if m.Cursor != 0 {
		t.Errorf("expected Cursor=0, got %d", m.Cursor)
	}
	if m.activeLogType != models.LogTypeSystem {
		t.Errorf("expected default log type to be System, got %v", m.activeLogType)
	}
	if m.SortAsc {
		t.Error("expected SortAsc=false by default (newest first)")
	}
}

func TestLogsModel_SetSize(t *testing.T) {
	m := NewLogsModel()
	m = m.SetSize(100, 50)

	if m.Width != 100 {
		t.Errorf("expected Width=100, got %d", m.Width)
	}
	if m.Height != 50 {
		t.Errorf("expected Height=50, got %d", m.Height)
	}
}

func TestLogsModel_SetLoading(t *testing.T) {
	m := NewLogsModel()

	m = m.SetLoading(true)
	if !m.Loading {
		t.Error("expected Loading=true")
	}

	m = m.SetLoading(false)
	if m.Loading {
		t.Error("expected Loading=false")
	}
}

func TestLogsModel_SetSystemLogs(t *testing.T) {
	m := NewLogsModel()

	logs := []models.SystemLogEntry{
		{Time: time.Now(), Severity: "warning", Description: "Test warning"},
		{Time: time.Now(), Severity: "info", Description: "Test info"},
	}

	m = m.SetSystemLogs(logs, nil)

	if len(m.systemLogs) != 2 {
		t.Errorf("expected 2 system logs, got %d", len(m.systemLogs))
	}
	if m.Loading {
		t.Error("expected Loading=false after SetSystemLogs")
	}
}

func TestLogsModel_SetSystemLogs_WithError(t *testing.T) {
	m := NewLogsModel()

	err := errors.New("API error")
	m = m.SetSystemLogs(nil, err)

	if m.Err != err {
		t.Error("expected error to be set")
	}
}

func TestLogsModel_SetTrafficLogs(t *testing.T) {
	m := NewLogsModel()

	logs := []models.TrafficLogEntry{
		{Time: time.Now(), Action: "allow", SourceIP: "10.0.0.1"},
		{Time: time.Now(), Action: "deny", SourceIP: "192.168.1.1"},
	}

	m = m.SetTrafficLogs(logs, nil)

	if len(m.trafficLogs) != 2 {
		t.Errorf("expected 2 traffic logs, got %d", len(m.trafficLogs))
	}
	if m.Loading {
		t.Error("expected Loading=false after SetTrafficLogs")
	}
}

func TestLogsModel_SetTrafficLogs_WithError(t *testing.T) {
	m := NewLogsModel()

	err := errors.New("API error")
	m = m.SetTrafficLogs(nil, err)

	if m.Err != err {
		t.Error("expected error to be set")
	}
}

func TestLogsModel_SetThreatLogs(t *testing.T) {
	m := NewLogsModel()

	logs := []models.ThreatLogEntry{
		{Time: time.Now(), Severity: "critical", ThreatName: "Test Threat"},
		{Time: time.Now(), Severity: "high", ThreatName: "Another Threat"},
	}

	m = m.SetThreatLogs(logs, nil)

	if len(m.threatLogs) != 2 {
		t.Errorf("expected 2 threat logs, got %d", len(m.threatLogs))
	}
	if m.Loading {
		t.Error("expected Loading=false after SetThreatLogs")
	}
}

func TestLogsModel_SetThreatLogs_WithError(t *testing.T) {
	m := NewLogsModel()

	err := errors.New("API error")
	m = m.SetThreatLogs(nil, err)

	if m.Err != err {
		t.Error("expected error to be set")
	}
}

func TestLogsModel_SetError(t *testing.T) {
	m := NewLogsModel()

	err := errors.New("test error")
	m = m.SetError(err)

	if m.Err != err {
		t.Error("expected error to be set")
	}
	if m.Loading {
		t.Error("expected Loading=false after SetError")
	}
}

func TestLogsModel_Update_Navigation(t *testing.T) {
	m := NewLogsModel()
	m = m.SetSize(100, 50)

	logs := []models.SystemLogEntry{
		{Time: time.Now(), Description: "Log 1"},
		{Time: time.Now(), Description: "Log 2"},
		{Time: time.Now(), Description: "Log 3"},
	}
	m = m.SetSystemLogs(logs, nil)

	// Move down
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.Cursor != 1 {
		t.Errorf("expected Cursor=1 after j, got %d", m.Cursor)
	}

	// Move up
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.Cursor != 0 {
		t.Errorf("expected Cursor=0 after k, got %d", m.Cursor)
	}
}

func TestLogsModel_View(t *testing.T) {
	m := NewLogsModel()
	m = m.SetSize(100, 50)

	// View without logs
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}

	// View with system logs
	logs := []models.SystemLogEntry{
		{Time: time.Now(), Severity: "warning", Description: "Test"},
	}
	m = m.SetSystemLogs(logs, nil)

	view = m.View()
	if view == "" {
		t.Error("expected non-empty view with logs")
	}
}

func TestLogsModel_View_ZeroWidth(t *testing.T) {
	m := NewLogsModel()
	// Don't set size

	view := m.View()
	if !strings.Contains(view, "Loading...") {
		t.Errorf("expected view to contain 'Loading...' with zero width, got %q", view)
	}
}

func TestLogSortField_Constants(t *testing.T) {
	if LogSortTime != 0 {
		t.Errorf("expected LogSortTime=0, got %d", LogSortTime)
	}
	if LogSortSeverity != 1 {
		t.Errorf("expected LogSortSeverity=1, got %d", LogSortSeverity)
	}
	if LogSortSource != 2 {
		t.Errorf("expected LogSortSource=2, got %d", LogSortSource)
	}
	if LogSortAction != 3 {
		t.Errorf("expected LogSortAction=3, got %d", LogSortAction)
	}
}

func TestLogsModel_Update_LogTypeCycleForward(t *testing.T) {
	m := NewLogsModel()
	m = m.SetSize(100, 50)

	// Default should be System
	if m.activeLogType != models.LogTypeSystem {
		t.Errorf("expected default log type System, got %v", m.activeLogType)
	}

	// Press ] to cycle forward: System -> Traffic
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	if m.activeLogType != models.LogTypeTraffic {
		t.Errorf("expected Traffic after ], got %v", m.activeLogType)
	}

	// Press ] again: Traffic -> Threat
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	if m.activeLogType != models.LogTypeThreat {
		t.Errorf("expected Threat after ], got %v", m.activeLogType)
	}

	// Press ] again: Threat -> System (wraps around)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	if m.activeLogType != models.LogTypeSystem {
		t.Errorf("expected System after ] (wrap), got %v", m.activeLogType)
	}
}

func TestLogsModel_Update_LogTypeCycleBackward(t *testing.T) {
	m := NewLogsModel()
	m = m.SetSize(100, 50)

	// Default should be System
	if m.activeLogType != models.LogTypeSystem {
		t.Errorf("expected default log type System, got %v", m.activeLogType)
	}

	// Press [ to cycle backward: System -> Threat
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})
	if m.activeLogType != models.LogTypeThreat {
		t.Errorf("expected Threat after [, got %v", m.activeLogType)
	}

	// Press [ again: Threat -> Traffic
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})
	if m.activeLogType != models.LogTypeTraffic {
		t.Errorf("expected Traffic after [, got %v", m.activeLogType)
	}

	// Press [ again: Traffic -> System (wraps around)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})
	if m.activeLogType != models.LogTypeSystem {
		t.Errorf("expected System after [ (wrap), got %v", m.activeLogType)
	}
}

func TestLogsModel_SetSize_ClampsCursor(t *testing.T) {
	m := NewLogsModel()
	m = m.SetSize(100, 50)

	// Add some logs
	logs := []models.SystemLogEntry{
		{Time: time.Now(), Description: "Log 1"},
		{Time: time.Now(), Description: "Log 2"},
		{Time: time.Now(), Description: "Log 3"},
		{Time: time.Now(), Description: "Log 4"},
		{Time: time.Now(), Description: "Log 5"},
	}
	m = m.SetSystemLogs(logs, nil)

	// Move cursor to end
	m.Cursor = 4
	if m.Cursor != 4 {
		t.Errorf("expected cursor at 4, got %d", m.Cursor)
	}

	// Filter to reduce items
	m.Filter.SetValue("Log 1")
	m.applyFilter()

	// Resize - should clamp cursor
	m = m.SetSize(100, 50)

	// Cursor should be clamped to valid range
	if m.Cursor >= m.filteredCount() {
		t.Errorf("cursor %d should be less than filtered count %d after resize", m.Cursor, m.filteredCount())
	}
}
