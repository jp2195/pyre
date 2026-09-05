package views

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
)

// TestLogsModel_ErrorIsScopedToItsLogType covers a shared-state bug: system,
// traffic, and threat logs are three independent fetches rendered in three
// tabs, but all three setters wrote to one Err field and the view rendered it
// regardless of which tab was open. One failing fetch blanked the other two
// tabs, so an operator investigating an incident saw "Error" where perfectly
// good traffic logs had already loaded.
func TestLogsModel_ErrorIsScopedToItsLogType(t *testing.T) {
	m := NewLogsModel()
	m = m.SetSize(120, 40)
	m = m.SetSystemLogs([]models.SystemLogEntry{
		{Time: time.Now(), Severity: "high", Type: "general", Description: "system-entry-marker"},
	}, nil)
	m = m.SetTrafficLogs(nil, errors.New("traffic-fetch-failed"))

	// The System tab is active by default and its fetch succeeded.
	out := m.View()
	if strings.Contains(out, "traffic-fetch-failed") {
		t.Error("System tab renders the traffic tab's error")
	}
	if !strings.Contains(out, "system-entry-marker") {
		t.Error("System tab does not render its own logs")
	}

	// Switching to Traffic must surface that tab's error.
	next, _ := m.Update(tea.KeyPressMsg{Code: ']', Text: "]"})
	if got := next.View(); !strings.Contains(got, "traffic-fetch-failed") {
		t.Error("Traffic tab does not render its own error")
	}
}

// TestLogsModel_PerTypeErrorsDoNotOverwrite checks each setter records its
// own failure rather than the last one winning.
func TestLogsModel_PerTypeErrorsDoNotOverwrite(t *testing.T) {
	sysErr := errors.New("system-failed")
	threatErr := errors.New("threat-failed")

	m := NewLogsModel()
	m = m.SetSystemLogs(nil, sysErr)
	m = m.SetThreatLogs(nil, threatErr)
	m = m.SetTrafficLogs([]models.TrafficLogEntry{{Time: time.Now()}}, nil)

	if !errors.Is(m.systemErr, sysErr) {
		t.Errorf("systemErr = %v, want %v", m.systemErr, sysErr)
	}
	if !errors.Is(m.threatErr, threatErr) {
		t.Errorf("threatErr = %v, want %v", m.threatErr, threatErr)
	}
	if m.trafficErr != nil {
		t.Errorf("trafficErr = %v, want nil after a successful fetch", m.trafficErr)
	}
}
