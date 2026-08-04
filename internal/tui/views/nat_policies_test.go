package views

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
)

func TestNewNATPoliciesModel(t *testing.T) {
	m := NewNATPoliciesModel()

	if m.list.Cursor != 0 {
		t.Errorf("expected Cursor=0, got %d", m.list.Cursor)
	}
	if m.HasData() {
		t.Error("expected HasData=false for new model")
	}
}

func TestNATPoliciesModel_SetSize(t *testing.T) {
	m := NewNATPoliciesModel()
	m = m.SetSize(100, 50)

	if m.list.Width != 100 {
		t.Errorf("expected Width=100, got %d", m.list.Width)
	}
	if m.list.Height != 50 {
		t.Errorf("expected Height=50, got %d", m.list.Height)
	}
}

func TestNATPoliciesModel_SetLoading(t *testing.T) {
	m := NewNATPoliciesModel()

	m = m.SetLoading(true)
	if !m.list.Loading {
		t.Error("expected Loading=true")
	}

	m = m.SetLoading(false)
	if m.list.Loading {
		t.Error("expected Loading=false")
	}
}

func TestNATPoliciesModel_SetRules(t *testing.T) {
	m := NewNATPoliciesModel()

	rules := []models.NATRule{
		{Name: "NAT-Outbound", SourceZones: []string{"trust"}},
		{Name: "NAT-Inbound", SourceZones: []string{"untrust"}},
	}

	m = m.SetRules(rules, nil)

	if len(m.list.Items()) != 2 {
		t.Errorf("expected 2 rules, got %d", len(m.list.Items()))
	}
	if m.list.Loading {
		t.Error("expected Loading=false after SetRules")
	}
	if m.list.Cursor != 0 {
		t.Error("expected Cursor to reset to 0")
	}
}

func TestNATPoliciesModel_SetRules_WithError(t *testing.T) {
	m := NewNATPoliciesModel()

	err := errors.New("API error")
	m = m.SetRules(nil, err)

	if m.list.Err != err {
		t.Error("expected error to be set")
	}
}

func TestNATPoliciesModel_Update_Navigation(t *testing.T) {
	m := NewNATPoliciesModel()
	m = m.SetSize(100, 50)

	rules := []models.NATRule{
		{Name: "Rule1"},
		{Name: "Rule2"},
		{Name: "Rule3"},
	}
	m = m.SetRules(rules, nil)

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
}

func TestNATPoliciesModel_View(t *testing.T) {
	m := NewNATPoliciesModel()
	m = m.SetSize(100, 50)

	// View without rules
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}

	// View with rules
	rules := []models.NATRule{
		{Name: "Test-NAT", SourceZones: []string{"trust"}},
	}
	m = m.SetRules(rules, nil)

	view = m.View()
	if view == "" {
		t.Error("expected non-empty view with rules")
	}
}

func TestNATPoliciesModel_View_ZeroWidth(t *testing.T) {
	m := NewNATPoliciesModel()
	// Don't set size

	view := m.View()
	if !strings.Contains(view, "Loading...") {
		t.Errorf("expected view to contain 'Loading...' with zero width, got %q", view)
	}
}

func TestNATPoliciesModel_SetSpinnerFrame_ReachesList(t *testing.T) {
	m := NewNATPoliciesModel()
	m = m.SetSpinnerFrame("◢")
	if m.list.SpinnerFrame != "◢" {
		t.Errorf("list.SpinnerFrame = %q, want ◢", m.list.SpinnerFrame)
	}
}
