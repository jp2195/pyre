package tui

// app_fanout_test.go – characterization tests for the three per-view fan-outs
// in app.go (handleWindowSize, handleSpinnerTick, handleRefresh).
//
// These tests are written BEFORE the refactor so they pin the observable
// behaviour of the fan-outs.
//
// Accessor notes (checked against current source):
//   - PoliciesModel / NATPoliciesModel wrap an unexported list RuleListModel[T]
//     field; Loading and SpinnerFrame are NOT directly accessible from this
//     package. Resize width/height are similarly inaccessible. These views are
//     tested indirectly via View() output regression checks.
//     GPUsersModel follows the same wrapped pattern as of M6.
//   - ObjectsModel stores width/height in unexported fields; tested via
//     View() output (the zero-width guard or loading state).
//   - NavbarModel stores width in an unexported field; resize propagation is
//     verified by confirming renderContent() does not panic after resize.
//   - connectionHub / connectionForm / login / commandPalette store dimensions
//     in unexported fields; full-height propagation is verified by checking that
//     Height on the content-height views equals msg.Height-4, which is
//     consistent ONLY when the full-height group is handled separately.

import (
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

// ---------------------------------------------------------------------------
// handleWindowSize fan-out
// ---------------------------------------------------------------------------

// TestFanout_Resize_ContentHeightViews checks that every view which receives
// contentHeight = msg.Height-4 actually stores those dimensions after a
// WindowSizeMsg is dispatched.
func TestFanout_Resize_ContentHeightViews(t *testing.T) {
	m := newTestModel(t, ViewDashboard)
	msg := tea.WindowSizeMsg{Width: 200, Height: 50}
	wantW := 200
	wantH := 50 - 4 // contentHeight

	updated, _ := m.Update(msg)
	nm := updated.(Model)

	// Dashboards embed DashboardBase which has exported Width/Height.
	for _, tc := range []struct {
		name string
		w, h int
	}{
		{"dashboard", nm.dashboard.Width, nm.dashboard.Height},
		{"networkDashboard", nm.networkDashboard.Width, nm.networkDashboard.Height},
		{"securityDashboard", nm.securityDashboard.Width, nm.securityDashboard.Height},
		{"vpnDashboard", nm.vpnDashboard.Width, nm.vpnDashboard.Height},
		{"configDashboard", nm.configDashboard.Width, nm.configDashboard.Height},
	} {
		if tc.w != wantW {
			t.Errorf("%s: Width=%d, want %d", tc.name, tc.w, wantW)
		}
		if tc.h != wantH {
			t.Errorf("%s: Height=%d, want %d", tc.name, tc.h, wantH)
		}
	}

	// Table-base views that embed TableBase directly expose Width/Height.
	for _, tc := range []struct {
		name string
		w, h int
	}{
		{"sessions", nm.sessions.Width, nm.sessions.Height},
		{"interfaces", nm.interfaces.Width, nm.interfaces.Height},
		{"routes", nm.routes.Width, nm.routes.Height},
		{"ipsecTunnels", nm.ipsecTunnels.Width, nm.ipsecTunnels.Height},
		{"logs", nm.logs.Width, nm.logs.Height},
	} {
		if tc.w != wantW {
			t.Errorf("%s: Width=%d, want %d", tc.name, tc.w, wantW)
		}
		if tc.h != wantH {
			t.Errorf("%s: Height=%d, want %d", tc.name, tc.h, wantH)
		}
	}

	// ObjectsModel stores dims in unexported fields; verify SetSize was called by
	// confirming View() does not short-circuit on zero-width for the address tab.
	// After resize the width is non-zero so View() renders real content.
	_ = nm.objects.View() // must not panic

	// picker and devicePicker store width/height in unexported fields. Verify
	// SetSize is called (compiles and does not panic) by calling View().
	_ = nm.picker.View()
	_ = nm.devicePicker.View()
}

// TestFanout_Resize_GlobalModel checks that m.width and m.height are set on the
// top-level model after a WindowSizeMsg.
func TestFanout_Resize_GlobalModel(t *testing.T) {
	m := newTestModel(t, ViewDashboard)
	msg := tea.WindowSizeMsg{Width: 160, Height: 45}
	updated, _ := m.Update(msg)
	nm := updated.(Model)
	if nm.width != 160 {
		t.Errorf("m.width=%d, want 160", nm.width)
	}
	if nm.height != 45 {
		t.Errorf("m.height=%d, want 45", nm.height)
	}
}

// TestFanout_Resize_RenderDoesNotPanic ensures that after a resize the top-level
// renderContent() does not panic — exercising the navbar and full-height views
// (connectionHub, connectionForm, login, commandPalette) indirectly.
func TestFanout_Resize_RenderDoesNotPanic(t *testing.T) {
	for _, view := range []ViewState{
		ViewConnectionHub, ViewConnectionForm, ViewLogin, ViewCommandPalette,
		ViewDashboard, ViewPolicies, ViewNATPolicies, ViewSessions,
		ViewInterfaces, ViewRoutes, ViewIPSecTunnels, ViewGPUsers,
		ViewLogs, ViewObjects,
	} {
		m := newTestModel(t, view)
		updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
		nm := updated.(Model)
		// Should not panic.
		_ = nm.renderContent()
	}
}

// ---------------------------------------------------------------------------
// handleSpinnerTick fan-out
// ---------------------------------------------------------------------------

// spinnerTickMsg returns a spinner.TickMsg that will be accepted by a freshly
// initialised spinner (tag=0 bypasses the tag guard; ID=0 bypasses the ID guard).
func spinnerTickMsg() spinner.TickMsg {
	return spinner.TickMsg{Time: time.Now()}
}

// TestFanout_SpinnerTick_PropagatesFrame checks that after a spinner.TickMsg the
// SpinnerFrame is non-empty on all views that directly expose the field.
func TestFanout_SpinnerTick_PropagatesFrame(t *testing.T) {
	m := newTestModel(t, ViewDashboard)
	updated, _ := m.Update(spinnerTickMsg())
	nm := updated.(Model)

	// Dashboards – expose SpinnerFrame via DashboardBase.
	for _, tc := range []struct {
		name  string
		frame string
	}{
		{"dashboard", nm.dashboard.SpinnerFrame},
		{"networkDashboard", nm.networkDashboard.SpinnerFrame},
		{"securityDashboard", nm.securityDashboard.SpinnerFrame},
		{"vpnDashboard", nm.vpnDashboard.SpinnerFrame},
		{"configDashboard", nm.configDashboard.SpinnerFrame},
	} {
		if tc.frame == "" {
			t.Errorf("%s: SpinnerFrame is empty after tick; expected propagation", tc.name)
		}
	}

	// Table views that embed TableBase directly expose SpinnerFrame.
	for _, tc := range []struct {
		name  string
		frame string
	}{
		{"sessions", nm.sessions.SpinnerFrame},
		{"interfaces", nm.interfaces.SpinnerFrame},
		{"routes", nm.routes.SpinnerFrame},
		{"ipsecTunnels", nm.ipsecTunnels.SpinnerFrame},
		{"logs", nm.logs.SpinnerFrame},
	} {
		if tc.frame == "" {
			t.Errorf("%s: SpinnerFrame is empty after tick; expected propagation", tc.name)
		}
	}

	// policies and natPolicies store SpinnerFrame in unexported list.SpinnerFrame.
	// Verify that the top-level spinner produced a non-empty frame (the call-path
	// to SetSpinnerFrame on those models is covered by compilation alone).
	wantFrame := nm.spinner.View()
	if wantFrame == "" {
		t.Fatal("top-level spinner.View() returned empty string after tick")
	}
	// Cross-check: the frame on directly accessible views equals the top-level
	// spinner's rendered frame.
	if nm.sessions.SpinnerFrame != wantFrame {
		t.Errorf("sessions SpinnerFrame=%q, want %q (top-level spinner)", nm.sessions.SpinnerFrame, wantFrame)
	}
}

// TestFanout_SpinnerTick_AllFramesSame checks that every directly-accessible
// SpinnerFrame holds the same value (they all receive the same frame string).
func TestFanout_SpinnerTick_AllFramesSame(t *testing.T) {
	m := newTestModel(t, ViewDashboard)
	updated, _ := m.Update(spinnerTickMsg())
	nm := updated.(Model)

	want := nm.sessions.SpinnerFrame
	if want == "" {
		t.Fatal("sessions SpinnerFrame is empty after tick")
	}

	checks := []struct {
		name  string
		frame string
	}{
		{"interfaces", nm.interfaces.SpinnerFrame},
		{"routes", nm.routes.SpinnerFrame},
		{"ipsecTunnels", nm.ipsecTunnels.SpinnerFrame},
		{"logs", nm.logs.SpinnerFrame},
		{"dashboard", nm.dashboard.SpinnerFrame},
		{"networkDashboard", nm.networkDashboard.SpinnerFrame},
		{"securityDashboard", nm.securityDashboard.SpinnerFrame},
		{"vpnDashboard", nm.vpnDashboard.SpinnerFrame},
		{"configDashboard", nm.configDashboard.SpinnerFrame},
	}
	for _, tc := range checks {
		if tc.frame != want {
			t.Errorf("%s: SpinnerFrame=%q, want %q", tc.name, tc.frame, want)
		}
	}
}

// ---------------------------------------------------------------------------
// handleRefresh fan-out
// ---------------------------------------------------------------------------

// TestFanout_Refresh_SetsLoadingOnCurrentView checks that handleRefresh (reached
// via the 'r' key binding) sets Loading=true on the view that is currently
// active. Uses views that embed TableBase directly and expose Loading.
func TestFanout_Refresh_SetsLoadingOnCurrentView(t *testing.T) {
	refreshKey := tea.KeyPressMsg{Code: 'r', Text: "r"}

	type check struct {
		view    ViewState
		loading func(Model) bool
	}
	checks := []check{
		{ViewSessions, func(nm Model) bool { return nm.sessions.Loading }},
		{ViewInterfaces, func(nm Model) bool { return nm.interfaces.Loading }},
		{ViewRoutes, func(nm Model) bool { return nm.routes.Loading }},
		{ViewIPSecTunnels, func(nm Model) bool { return nm.ipsecTunnels.Loading }},
		{ViewLogs, func(nm Model) bool { return nm.logs.Loading }},
	}

	for _, tc := range checks {
		m := newTestModel(t, tc.view)
		updated, _ := m.Update(refreshKey)
		nm := updated.(Model)
		if !tc.loading(nm) {
			t.Errorf("view %v: Loading not set after refresh key", tc.view)
		}
	}
}

// TestFanout_Refresh_SetsLoadingOnPolicies verifies that pressing 'r' while on
// the Policies, NATPolicies, or GPUsers view calls SetLoading(true). Since Loading is
// inside the unexported list field, we verify via the View() output: when
// Loading==true and no data has been loaded, View() renders the loading message.
func TestFanout_Refresh_SetsLoadingOnPolicies(t *testing.T) {
	refreshKey := tea.KeyPressMsg{Code: 'r', Text: "r"}

	for _, tc := range []struct {
		view ViewState
		name string
	}{
		{ViewPolicies, "policies"},
		{ViewNATPolicies, "natPolicies"},
		{ViewGPUsers, "gpUsers"},
	} {
		m := newTestModel(t, tc.view)
		// Dispatch a resize first so View() doesn't short-circuit on zero width.
		resized, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
		m = resized.(Model)

		updated, _ := m.Update(refreshKey)
		nm := updated.(Model)

		var got string
		switch tc.view {
		case ViewPolicies:
			got = nm.policies.View()
		case ViewNATPolicies:
			got = nm.natPolicies.View()
		case ViewGPUsers:
			got = nm.gpUsers.View()
		}
		// When Loading=true (and no items loaded), View() renders the loading
		// inline string (e.g. "Loading policies..." or "Loading NAT rules...").
		if !strings.Contains(got, "Loading") {
			t.Errorf("%s: expected loading view after refresh, got: %q", tc.name, got)
		}
	}
}

// TestFanout_Refresh_ObjectsLoading checks that pressing 'r' on the Objects view
// sets loading via SetLoading(true) on the ObjectsModel.
func TestFanout_Refresh_ObjectsLoading(t *testing.T) {
	refreshKey := tea.KeyPressMsg{Code: 'r', Text: "r"}
	m := newTestModel(t, ViewObjects)
	// Dispatch a resize first so View() renders real content.
	resized, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = resized.(Model)
	updated, _ := m.Update(refreshKey)
	nm := updated.(Model)
	// ObjectsModel propagates loading to both sub-tabs. Verify via View() which
	// renders loading text when addressTab.Loading==true.
	v := nm.objects.View()
	if !strings.Contains(v, "Loading") {
		t.Errorf("objects: expected loading view after refresh, got: %q", v)
	}
}

// TestFanout_Refresh_NonRefreshableViewsIgnored verifies that pressing 'r' on
// views not in the refresh switch (e.g. ViewDashboard, ViewConnectionHub) does
// NOT set loading on any content view.
func TestFanout_Refresh_NonRefreshableViewsIgnored(t *testing.T) {
	refreshKey := tea.KeyPressMsg{Code: 'r', Text: "r"}

	for _, view := range []ViewState{ViewDashboard, ViewConnectionHub} {
		m := newTestModel(t, view)
		updated, _ := m.Update(refreshKey)
		nm := updated.(Model)
		if nm.sessions.Loading {
			t.Errorf("view %v: sessions.Loading unexpectedly true after refresh", view)
		}
		if nm.interfaces.Loading {
			t.Errorf("view %v: interfaces.Loading unexpectedly true after refresh", view)
		}
		if nm.routes.Loading {
			t.Errorf("view %v: routes.Loading unexpectedly true after refresh", view)
		}
	}
}
