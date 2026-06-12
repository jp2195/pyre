package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/jp2195/pyre/internal/tui/views"
)

const headerLines = 2 // main row + sub-tab row

func TestRenderHeader_DisconnectedAtStandardWidth(t *testing.T) {
	m := newTestModel(t, ViewDashboard)

	out := m.renderHeader()
	if !strings.Contains(out, "pyre") {
		t.Errorf("expected title 'pyre' in header:\n%s", out)
	}
	if !strings.Contains(out, "disconnected") {
		t.Errorf("expected disconnected status in header:\n%s", out)
	}
	// NavHeaderBorder.Width clamps every line to m.width, so an over-wide
	// row produced by broken padding math would WRAP into extra lines
	// rather than exceed the width. Pin the line count to catch that.
	lines := strings.Split(out, "\n")
	if len(lines) != headerLines {
		t.Errorf("header has %d lines, want %d (padding overflow wraps the row):\n%s", len(lines), headerLines, out)
	}
}

// TestRenderHeader_NarrowWidthsDoNotPanic guards the padding math: every
// strings.Repeat count derives from a max(...,0) clamp, so even absurdly
// narrow terminals must render without panicking.
func TestRenderHeader_NarrowWidthsDoNotPanic(t *testing.T) {
	for _, width := range []int{1, 5, 10, 20, 40} {
		m := newTestModel(t, ViewPolicies)
		m.width = width

		out := m.renderHeader() // panics on negative Repeat before clamping
		if out == "" {
			t.Errorf("width %d: expected non-empty header", width)
		}
	}
}

func TestCurrentViewName_AllViews(t *testing.T) {
	cases := []struct {
		view ViewState
		dash views.DashboardType
		want string
	}{
		{ViewDashboard, views.DashboardMain, "Monitor/Overview"},
		{ViewDashboard, views.DashboardNetwork, "Monitor/Network"},
		{ViewDashboard, views.DashboardSecurity, "Monitor/Security"},
		{ViewDashboard, views.DashboardVPN, "Monitor/VPN"},
		{ViewDashboard, views.DashboardConfig, "Tools/Config"},
		{ViewPolicies, views.DashboardMain, "Analyze/Policies"},
		{ViewNATPolicies, views.DashboardMain, "Analyze/NAT"},
		{ViewSessions, views.DashboardMain, "Analyze/Sessions"},
		{ViewInterfaces, views.DashboardMain, "Analyze/Interfaces"},
		{ViewRoutes, views.DashboardMain, "Analyze/Routes"},
		{ViewIPSecTunnels, views.DashboardMain, "Analyze/IPSec"},
		{ViewGPUsers, views.DashboardMain, "Analyze/GP Users"},
		{ViewLogs, views.DashboardMain, "Analyze/Logs"},
		{ViewObjects, views.DashboardMain, "Analyze/Objects"},
		{ViewPicker, views.DashboardMain, "Connections"},
		{ViewDevicePicker, views.DashboardMain, "Connections/Devices"},
		{ViewCommandPalette, views.DashboardMain, "Commands"},
		// These states have no explicit case and fall through to "".
		{ViewConnectionHub, views.DashboardMain, ""},
		{ViewConnectionForm, views.DashboardMain, ""},
		{ViewLogin, views.DashboardMain, ""},
	}
	for _, tc := range cases {
		m := newTestModel(t, tc.view)
		m.currentView = tc.view
		m.currentDashboard = tc.dash
		if got := m.currentViewName(); got != tc.want {
			t.Errorf("currentViewName(%v, %v) = %q, want %q", tc.view, tc.dash, got, tc.want)
		}
	}
}

func TestRenderFooter_LoadingShowsSpinner(t *testing.T) {
	m := newTestModel(t, ViewDashboard)
	m.loading = true

	out := m.renderFooter()
	if !strings.Contains(out, "Loading...") {
		t.Errorf("expected Loading... in footer:\n%s", out)
	}
}

func TestRenderFooter_ShowsError(t *testing.T) {
	m := newTestModel(t, ViewDashboard)
	m.err = errors.New("kaboom")

	out := m.renderFooter()
	if !strings.Contains(out, "Error: kaboom") {
		t.Errorf("expected error line in footer:\n%s", out)
	}
	// Help hints still render below the error.
	if !strings.Contains(out, "quit") {
		t.Errorf("expected help hints alongside error:\n%s", out)
	}
}

func TestRenderFooter_DefaultHints(t *testing.T) {
	m := newTestModel(t, ViewDashboard)

	out := m.renderFooter()
	for _, hint := range []string{"1-3", "section", "cycle", "refresh", "conn", "commands", "help", "quit"} {
		if !strings.Contains(out, hint) {
			t.Errorf("expected hint %q in footer:\n%s", hint, out)
		}
	}
	// No Panorama connection in the test model: the devices hint must be absent.
	if strings.Contains(out, "devices") {
		t.Errorf("did not expect devices hint without a Panorama connection:\n%s", out)
	}
}
