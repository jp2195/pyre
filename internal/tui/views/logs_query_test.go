package views

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
)

func typeString(m LogsModel, s string) LogsModel {
	for _, r := range s {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

func TestLogsModel_FOpensQueryBar(t *testing.T) {
	m := NewLogsModel().SetSize(100, 40)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	if !m.IsQueryMode() {
		t.Fatal("f did not open the query bar")
	}
}

func TestLogsModel_QueryCommitsOnEnter(t *testing.T) {
	m := NewLogsModel().SetSize(100, 40)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	m = typeString(m, "addr.src in 203.0.113.5")

	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.IsQueryMode() {
		t.Error("enter did not leave query mode")
	}
	if m.Query() != "addr.src in 203.0.113.5" {
		t.Errorf("Query() = %q, want the typed expression", m.Query())
	}
	if cmd == nil {
		t.Fatal("committing a query did not refetch")
	}
	req, ok := cmd().(FetchLogsCmd)
	if !ok {
		t.Fatalf("expected FetchLogsCmd, got %T", cmd())
	}
	if req.Query != "addr.src in 203.0.113.5" {
		t.Errorf("request carried %q", req.Query)
	}
}

func TestLogsModel_QueryCancelsOnEsc(t *testing.T) {
	m := NewLogsModel().SetSize(100, 40)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	m = typeString(m, "addr.src in 203.0.113.5")
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})

	if m.IsQueryMode() {
		t.Error("esc did not leave query mode")
	}
	if m.Query() != "" {
		t.Errorf("esc stored the query anyway: %q", m.Query())
	}
}

// The device rejects a bad query before any job exists. The previous rows
// must stay on screen and the device's own message must be shown.
func TestLogsModel_RejectedQueryKeepsRowsAndShowsMessage(t *testing.T) {
	m := NewLogsModel().SetSize(100, 40)
	m = m.SetSystemLogs([]models.SystemLogEntry{{Type: "SYSTEM", Description: "keep me"}}, LogPageMeta{}, nil)

	m = m.SetSystemLogs(nil, LogPageMeta{}, errors.New("syntax error at 14:50:57"))

	if got := m.rowCount(models.LogTypeSystem); got != 1 {
		t.Errorf("rows after a rejected query = %d, want 1 (previous rows kept)", got)
	}
	view := m.View()
	if !strings.Contains(view, "syntax error at 14:50:57") {
		t.Errorf("view does not show the device message:\n%s", view)
	}
}
