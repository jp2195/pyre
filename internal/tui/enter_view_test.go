package tui

import (
	"testing"

	"charm.land/bubbles/v2/spinner"
)

// TestSwitchView_StartsTheSpinner guards the spinner tick through the merged
// path. The tick used to be batched by handleNavigationMsg; enterView owns
// it now, and a lost tick freezes the loading frame.
func TestSwitchView_StartsTheSpinner(t *testing.T) {
	for _, v := range []ViewState{ViewPolicies, ViewSessions, ViewLogs} {
		m := newTestModel(t, ViewDashboard)
		next, cmd := m.Update(SwitchViewMsg{View: v})
		got := next.(Model)
		if got.currentView != v {
			t.Errorf("view %d: currentView = %d", v, got.currentView)
		}
		ticked := false
		for _, msg := range runCmdTree(cmd) {
			if _, ok := msg.(spinner.TickMsg); ok {
				ticked = true
			}
		}
		if !ticked {
			t.Errorf("view %d: switching to an empty view did not start the spinner tick", v)
		}
	}
}

// TestSwitchView_MatchesTheNavbar checks the palette and the navbar now land
// in the same state for the same view.
func TestSwitchView_MatchesTheNavbar(t *testing.T) {
	m := newTestModel(t, ViewDashboard)
	next, _ := m.Update(SwitchViewMsg{View: ViewSessions})
	viaPalette := next.(Model)
	if !viaPalette.sessions.IsLoading() {
		t.Error("palette switch to an empty Sessions view did not mark it loading")
	}
	if item := viaPalette.navbar.ActiveItem(); item == nil || item.ID != "sessions" {
		t.Errorf("navbar not synced to Sessions: %+v", item)
	}
}
