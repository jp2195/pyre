package views

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
)

func pressGPUsers(m GPUsersModel, keys string) GPUsersModel {
	for _, r := range keys {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

func TestGPUsers_Behavior_FilterKeystrokesNarrowRows(t *testing.T) {
	InitStyles()
	m := NewGPUsersModel()
	m = m.SetSize(120, 30)
	m = m.SetUsers([]models.GlobalProtectUser{
		{Username: "alice", Gateway: "gw-east"},
		{Username: "bob", Gateway: "gw-west"},
	}, nil)

	out := m.View()
	if !strings.Contains(out, "alice") || !strings.Contains(out, "bob") {
		t.Fatalf("expected both users initially:\n%s", out)
	}

	m = pressGPUsers(m, "/alice")
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	out = m.View()
	if !strings.Contains(out, "alice") {
		t.Errorf("expected alice after filter:\n%s", out)
	}
	if strings.Contains(out, "bob") {
		t.Errorf("expected bob filtered out:\n%s", out)
	}
}

func TestGPUsers_Behavior_EscClearsFilter(t *testing.T) {
	InitStyles()
	m := NewGPUsersModel()
	m = m.SetSize(120, 30)
	m = m.SetUsers([]models.GlobalProtectUser{
		{Username: "alice"},
		{Username: "bob"},
	}, nil)

	m = pressGPUsers(m, "/alice")
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})

	out := m.View()
	if !strings.Contains(out, "bob") {
		t.Errorf("expected filter cleared so bob is visible again:\n%s", out)
	}
}

func TestGPUsers_Behavior_SortCycleReorders(t *testing.T) {
	InitStyles()
	m := NewGPUsersModel()
	m = m.SetSize(120, 30)
	m = m.SetUsers([]models.GlobalProtectUser{
		{Username: "alice", Gateway: "zz-gw"},
		{Username: "zed", Gateway: "aa-gw"},
	}, nil)

	// Press s once: Username -> Gateway, ascending; aa-gw (zed) sorts first.
	m = pressGPUsers(m, "s")
	out := m.View()
	if strings.Index(out, "zed") > strings.Index(out, "alice") {
		t.Errorf("expected zed (aa-gw) first when sorted by gateway:\n%s", out)
	}
	if !strings.Contains(out, "Gateway") {
		t.Errorf("expected sort label Gateway in banner:\n%s", out)
	}
}

func TestGPUsers_Behavior_EnterShowsDetail(t *testing.T) {
	InitStyles()
	m := NewGPUsersModel()
	m = m.SetSize(120, 40)
	m = m.SetUsers([]models.GlobalProtectUser{
		{Username: "alice", Domain: "corp.example"},
	}, nil)

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	out := m.View()
	if !strings.Contains(out, "corp.example") {
		t.Errorf("expected detail panel with domain after enter:\n%s", out)
	}
}

func TestGPUsers_Behavior_ScrollIndicator(t *testing.T) {
	InitStyles()
	m := NewGPUsersModel()
	m = m.SetSize(120, 12) // visibleRows = 12-8 = 4
	m = m.SetUsers(gpUsersFixture(20), nil)

	out := m.View()
	if !strings.Contains(out, "of 20") {
		t.Errorf("expected scroll indicator 'of 20':\n%s", out)
	}
}

func TestGPUsers_Behavior_ErrorAndEmptyStates(t *testing.T) {
	InitStyles()
	m := NewGPUsersModel()
	m = m.SetSize(120, 30)

	m = m.SetUsers(nil, errTest)
	if out := m.View(); !strings.Contains(out, "test error") {
		t.Errorf("expected error message:\n%s", out)
	}

	m = m.SetUsers([]models.GlobalProtectUser{}, nil)
	if out := m.View(); !strings.Contains(out, "No GlobalProtect users found") {
		t.Errorf("expected empty message:\n%s", out)
	}
}
