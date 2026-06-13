package views

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
)

func TestNewPoliciesModel(t *testing.T) {
	m := NewPoliciesModel()

	if m.list.Cursor != 0 {
		t.Errorf("expected Cursor=0, got %d", m.list.Cursor)
	}
	if m.HasData() {
		t.Error("expected HasData=false for new model")
	}
}

func TestPoliciesModel_SetSize(t *testing.T) {
	m := NewPoliciesModel()
	m = m.SetSize(100, 50)

	if m.list.Width != 100 {
		t.Errorf("expected Width=100, got %d", m.list.Width)
	}
	if m.list.Height != 50 {
		t.Errorf("expected Height=50, got %d", m.list.Height)
	}
}

func TestPoliciesModel_SetLoading(t *testing.T) {
	m := NewPoliciesModel()

	m = m.SetLoading(true)
	if !m.list.Loading {
		t.Error("expected Loading=true")
	}

	m = m.SetLoading(false)
	if m.list.Loading {
		t.Error("expected Loading=false")
	}
}

func TestPoliciesModel_SetPolicies(t *testing.T) {
	m := NewPoliciesModel()

	policies := []models.SecurityRule{
		{Name: "Allow-Web", Action: "allow", SourceZones: []string{"trust"}},
		{Name: "Deny-All", Action: "deny", SourceZones: []string{"any"}},
	}

	m = m.SetPolicies(policies, nil)

	if len(m.list.Items()) != 2 {
		t.Errorf("expected 2 policies, got %d", len(m.list.Items()))
	}
	if m.list.Loading {
		t.Error("expected Loading=false after SetPolicies")
	}
	if m.list.Cursor != 0 {
		t.Error("expected Cursor to reset to 0")
	}
}

func TestPoliciesModel_SetPolicies_WithError(t *testing.T) {
	m := NewPoliciesModel()

	err := errors.New("API error")
	m = m.SetPolicies(nil, err)

	if m.list.Err != err {
		t.Error("expected error to be set")
	}
}

func TestPoliciesModel_Filtering(t *testing.T) {
	m := NewPoliciesModel()
	m = m.SetSize(100, 50)

	policies := []models.SecurityRule{
		{Name: "Allow-Web", Action: "allow", Applications: []string{"web-browsing"}},
		{Name: "Allow-SSH", Action: "allow", Applications: []string{"ssh"}},
		{Name: "Deny-All", Action: "deny", Applications: []string{"any"}},
	}

	m = m.SetPolicies(policies, nil)

	// Initially all policies should be visible
	if len(m.list.Filtered()) != 3 {
		t.Errorf("expected 3 filtered policies, got %d", len(m.list.Filtered()))
	}

	// Filter by name
	m.list.Filter.SetValue("Allow")
	m.list.applyFilter()

	if len(m.list.Filtered()) != 2 {
		t.Errorf("expected 2 filtered policies for 'Allow', got %d", len(m.list.Filtered()))
	}

	// Filter by application
	m.list.Filter.SetValue("ssh")
	m.list.applyFilter()

	if len(m.list.Filtered()) != 1 {
		t.Errorf("expected 1 filtered policy for 'ssh', got %d", len(m.list.Filtered()))
	}

	// Clear filter
	m.list.Filter.SetValue("")
	m.list.applyFilter()

	if len(m.list.Filtered()) != 3 {
		t.Errorf("expected 3 filtered policies after clear, got %d", len(m.list.Filtered()))
	}
}

func TestPoliciesModel_Filtering_ByZone(t *testing.T) {
	m := NewPoliciesModel()
	m = m.SetSize(100, 50)

	policies := []models.SecurityRule{
		{Name: "Trust-to-Untrust", SourceZones: []string{"trust"}, DestZones: []string{"untrust"}},
		{Name: "Internal-Only", SourceZones: []string{"internal"}, DestZones: []string{"internal"}},
	}

	m = m.SetPolicies(policies, nil)

	// Filter by source zone
	m.list.Filter.SetValue("trust")
	m.list.applyFilter()

	if len(m.list.Filtered()) != 1 {
		t.Errorf("expected 1 filtered policy for 'trust', got %d", len(m.list.Filtered()))
	}

	// Filter by destination zone
	m.list.Filter.SetValue("internal")
	m.list.applyFilter()

	if len(m.list.Filtered()) != 1 {
		t.Errorf("expected 1 filtered policy for 'internal', got %d", len(m.list.Filtered()))
	}
}

func TestPoliciesModel_Update_Navigation(t *testing.T) {
	m := NewPoliciesModel()
	m = m.SetSize(100, 50)

	policies := []models.SecurityRule{
		{Name: "Rule1"},
		{Name: "Rule2"},
		{Name: "Rule3"},
	}
	m = m.SetPolicies(policies, nil)

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

func TestPoliciesModel_View(t *testing.T) {
	m := NewPoliciesModel()
	m = m.SetSize(100, 50)

	// View without policies
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}

	// View with policies
	policies := []models.SecurityRule{
		{Name: "Test-Rule", Action: "allow", SourceZones: []string{"trust"}},
	}
	m = m.SetPolicies(policies, nil)

	view = m.View()
	if view == "" {
		t.Error("expected non-empty view with policies")
	}
}

func TestPoliciesModel_View_ZeroWidth(t *testing.T) {
	m := NewPoliciesModel()
	// Don't set size

	view := m.View()
	if !strings.Contains(view, "Loading...") {
		t.Errorf("expected view to contain 'Loading...' with zero width, got %q", view)
	}
}

func TestPoliciesModel_SetSpinnerFrame_ReachesList(t *testing.T) {
	m := NewPoliciesModel()
	m = m.SetSpinnerFrame("◢")
	if m.list.SpinnerFrame != "◢" {
		t.Errorf("list.SpinnerFrame = %q, want ◢", m.list.SpinnerFrame)
	}
}
