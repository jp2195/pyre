package views

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jp2195/pyre/internal/models"
)

func TestNewPoliciesModel(t *testing.T) {
	m := NewPoliciesModel()

	if m.Cursor != 0 {
		t.Errorf("expected Cursor=0, got %d", m.Cursor)
	}
	if m.sortBy != PolicySortPosition {
		t.Errorf("expected default sort by Position, got %d", m.sortBy)
	}
}

func TestPoliciesModel_SetSize(t *testing.T) {
	m := NewPoliciesModel()
	m = m.SetSize(100, 50)

	if m.Width != 100 {
		t.Errorf("expected Width=100, got %d", m.Width)
	}
	if m.Height != 50 {
		t.Errorf("expected Height=50, got %d", m.Height)
	}
}

func TestPoliciesModel_SetLoading(t *testing.T) {
	m := NewPoliciesModel()

	m = m.SetLoading(true)
	if !m.Loading {
		t.Error("expected Loading=true")
	}

	m = m.SetLoading(false)
	if m.Loading {
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

	if len(m.policies) != 2 {
		t.Errorf("expected 2 policies, got %d", len(m.policies))
	}
	if m.Loading {
		t.Error("expected Loading=false after SetPolicies")
	}
	if m.Cursor != 0 {
		t.Error("expected Cursor to reset to 0")
	}
}

func TestPoliciesModel_SetPolicies_WithError(t *testing.T) {
	m := NewPoliciesModel()

	err := errors.New("API error")
	m = m.SetPolicies(nil, err)

	if m.Err != err {
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
	if len(m.filtered) != 3 {
		t.Errorf("expected 3 filtered policies, got %d", len(m.filtered))
	}

	// Filter by name
	m.Filter.SetValue("Allow")
	m.applyFilter()

	if len(m.filtered) != 2 {
		t.Errorf("expected 2 filtered policies for 'Allow', got %d", len(m.filtered))
	}

	// Filter by application
	m.Filter.SetValue("ssh")
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 filtered policy for 'ssh', got %d", len(m.filtered))
	}

	// Clear filter
	m.Filter.SetValue("")
	m.applyFilter()

	if len(m.filtered) != 3 {
		t.Errorf("expected 3 filtered policies after clear, got %d", len(m.filtered))
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
	m.Filter.SetValue("trust")
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 filtered policy for 'trust', got %d", len(m.filtered))
	}

	// Filter by destination zone
	m.Filter.SetValue("internal")
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 filtered policy for 'internal', got %d", len(m.filtered))
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
	if view != "Loading..." {
		t.Errorf("expected 'Loading...' with zero width, got %q", view)
	}
}

func TestPolicySortField_Constants(t *testing.T) {
	if PolicySortPosition != 0 {
		t.Errorf("expected PolicySortPosition=0, got %d", PolicySortPosition)
	}
	if PolicySortName != 1 {
		t.Errorf("expected PolicySortName=1, got %d", PolicySortName)
	}
	if PolicySortHits != 2 {
		t.Errorf("expected PolicySortHits=2, got %d", PolicySortHits)
	}
	if PolicySortLastHit != 3 {
		t.Errorf("expected PolicySortLastHit=3, got %d", PolicySortLastHit)
	}
}
