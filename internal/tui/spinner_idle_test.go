package tui

import "testing"

// TestAnyLoadingGate verifies the condition that keeps the spinner tick chain
// alive: idle detail views report not-loading, a fetch-in-flight view reports
// loading, and a dashboard with no data yet counts as loading.
func TestAnyLoadingGate(t *testing.T) {
	// Detail view with no fetch in flight: idle.
	m := newTestModel(t, ViewPolicies)
	if m.anyLoading() {
		t.Error("fresh idle Policies view should not report loading")
	}

	// Same view once a fetch is dispatched (loading flag set).
	m.policies = m.policies.SetLoading(true)
	if !m.anyLoading() {
		t.Error("Policies view with a fetch in flight should report loading")
	}

	// A dashboard with no data yet counts as loading (dashboards signal via
	// nil data, not a flag).
	md := newTestModel(t, ViewDashboard)
	if !md.anyLoading() {
		t.Error("dashboard with no data should report loading")
	}
}
