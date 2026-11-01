package tui

import "github.com/jp2195/pyre/internal/tui/views"

// viewslots.go – single registration table that drives handleWindowSize,
// handleSpinnerTick, handleRefresh, and resetViewData.
//
// Each viewSlot encodes all fan-out roles for one sub-view model:
//   resize    – always non-nil; called for every slot during handleWindowSize.
//   spinner   – non-nil for the 14 views that display a spinner frame
//               (9 table views + 5 dashboards).
//   loading   – non-nil for the 9 refreshable views; called with true on refresh.
//   reset     – non-nil for the 14 views that cache per-device data; called on
//               a connection or Panorama-target switch (see resetViewData).
//   isFiltering – non-nil for the 9 views with a filter input; drives
//                 currentViewFiltering.
//   isExpanded – non-nil for the auto-refreshed list views; holds auto-refresh
//                while a detail panel is open.
//   refreshFor – the ViewState that triggers a refresh for this slot; 0 when the
//                slot is not refreshable.
//
// Adding a new view means adding exactly one entry here.

type viewSlot struct {
	resize  func(m *Model, w, h, contentH int)
	spinner func(m *Model, frame string)
	loading func(m *Model, v bool)
	// reset replaces the sub-view model with a freshly constructed one,
	// dropping data belonging to the previously selected device. Slots that
	// hold no per-device data (navbar, forms, pickers) leave it nil.
	reset func(m *Model)
	// isLoading reports whether this slot's view currently has a fetch in
	// flight; non-nil for the refreshable views. Used by anyLoading to gate
	// the spinner tick chain so it stops when nothing is loading.
	isLoading func(m *Model) bool
	// isFiltering reports whether the view's filter (or, in logs, query)
	// input has focus, so keys go to the input instead of global bindings.
	// Non-nil for every view with a filter.
	isFiltering func(m *Model) bool
	// isExpanded reports whether a detail panel is open. Auto-refresh holds
	// while it is, so the row being read does not change underneath it.
	// Non-nil for the auto-refreshed list views; logs holds on its own rule.
	isExpanded func(m *Model) bool
	// refreshFor is the ViewState that triggers a refresh for this slot; 0 when the
	// slot is not refreshable.
	// NOTE: the zero value collides with ViewConnectionHub (= 0); this is only safe
	// because no slot sets loading without an explicit refreshFor. Set both or neither.
	refreshFor ViewState
}

// currentSlot is the slot for the view on screen, if that view has one.
// Matching on isLoading as well as refreshFor keeps the zero refreshFor of
// non-refreshable slots from matching ViewConnectionHub (also 0).
func (m Model) currentSlot() (viewSlot, bool) {
	for _, s := range viewSlots() {
		if s.isLoading != nil && s.refreshFor == m.currentView {
			return s, true
		}
	}
	return viewSlot{}, false
}

// viewSlots returns the canonical ordered registration table.
// All 21 sub-view fields appear here exactly once.
func viewSlots() []viewSlot {
	return []viewSlot{
		// --- Navbar (width-only resize; no spinner; not refreshable) ---
		{
			resize: func(m *Model, w, h, contentH int) {
				m.navbar = m.navbar.SetSize(w)
			},
		},

		// --- Full-height overlay views (no spinner; not refreshable) ---
		{
			resize: func(m *Model, w, h, contentH int) {
				m.connectionHub = m.connectionHub.SetSize(w, h)
			},
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.connectionForm = m.connectionForm.SetSize(w, h)
			},
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.login = m.login.SetSize(w, h)
			},
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.commandPalette = m.commandPalette.SetSize(w, h)
			},
		},

		// --- Dashboard views (contentHeight; spinner; not individually refreshable) ---
		{
			resize: func(m *Model, w, h, contentH int) {
				m.dashboard = m.dashboard.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.dashboard = m.dashboard.SetSpinnerFrame(frame)
			},
			reset: func(m *Model) {
				m.dashboard = views.NewDashboardModel()
			},
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.networkDashboard = m.networkDashboard.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.networkDashboard = m.networkDashboard.SetSpinnerFrame(frame)
			},
			reset: func(m *Model) {
				m.networkDashboard = views.NewNetworkDashboardModel()
			},
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.securityDashboard = m.securityDashboard.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.securityDashboard = m.securityDashboard.SetSpinnerFrame(frame)
			},
			reset: func(m *Model) {
				m.securityDashboard = views.NewSecurityDashboardModel()
			},
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.vpnDashboard = m.vpnDashboard.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.vpnDashboard = m.vpnDashboard.SetSpinnerFrame(frame)
			},
			reset: func(m *Model) {
				m.vpnDashboard = views.NewVPNDashboardModel()
			},
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.configDashboard = m.configDashboard.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.configDashboard = m.configDashboard.SetSpinnerFrame(frame)
			},
			reset: func(m *Model) {
				m.configDashboard = views.NewConfigDashboardModel()
			},
		},

		// --- Table views (contentHeight; spinner; refreshable) ---
		{
			resize: func(m *Model, w, h, contentH int) {
				m.policies = m.policies.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.policies = m.policies.SetSpinnerFrame(frame)
			},
			reset: func(m *Model) {
				m.policies = views.NewPoliciesModel()
			},
			loading: func(m *Model, v bool) {
				m.policies = m.policies.SetLoading(v)
			},
			isLoading:   func(m *Model) bool { return m.policies.IsLoading() },
			isFiltering: func(m *Model) bool { return m.policies.IsFilterMode() },
			refreshFor:  ViewPolicies,
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.natPolicies = m.natPolicies.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.natPolicies = m.natPolicies.SetSpinnerFrame(frame)
			},
			reset: func(m *Model) {
				m.natPolicies = views.NewNATPoliciesModel()
			},
			loading: func(m *Model, v bool) {
				m.natPolicies = m.natPolicies.SetLoading(v)
			},
			isLoading:   func(m *Model) bool { return m.natPolicies.IsLoading() },
			isFiltering: func(m *Model) bool { return m.natPolicies.IsFilterMode() },
			refreshFor:  ViewNATPolicies,
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.sessions = m.sessions.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.sessions = m.sessions.SetSpinnerFrame(frame)
			},
			reset: func(m *Model) {
				m.sessions = views.NewSessionsModel()
			},
			loading: func(m *Model, v bool) {
				m.sessions = m.sessions.SetLoading(v)
			},
			isLoading:   func(m *Model) bool { return m.sessions.IsLoading() },
			isFiltering: func(m *Model) bool { return m.sessions.IsFilterMode() },
			isExpanded:  func(m *Model) bool { return m.sessions.IsExpanded() },
			refreshFor:  ViewSessions,
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.interfaces = m.interfaces.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.interfaces = m.interfaces.SetSpinnerFrame(frame)
			},
			reset: func(m *Model) {
				m.interfaces = views.NewInterfacesModel()
			},
			loading: func(m *Model, v bool) {
				m.interfaces = m.interfaces.SetLoading(v)
			},
			isLoading:   func(m *Model) bool { return m.interfaces.IsLoading() },
			isFiltering: func(m *Model) bool { return m.interfaces.IsFilterMode() },
			isExpanded:  func(m *Model) bool { return m.interfaces.IsExpanded() },
			refreshFor:  ViewInterfaces,
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.routes = m.routes.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.routes = m.routes.SetSpinnerFrame(frame)
			},
			reset: func(m *Model) {
				m.routes = views.NewRoutesModel()
			},
			loading: func(m *Model, v bool) {
				m.routes = m.routes.SetLoading(v)
			},
			isLoading:   func(m *Model) bool { return m.routes.Loading },
			isFiltering: func(m *Model) bool { return m.routes.IsFilterMode() },
			isExpanded:  func(m *Model) bool { return m.routes.Expanded },
			refreshFor:  ViewRoutes,
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.ipsecTunnels = m.ipsecTunnels.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.ipsecTunnels = m.ipsecTunnels.SetSpinnerFrame(frame)
			},
			reset: func(m *Model) {
				m.ipsecTunnels = views.NewIPSecTunnelsModel()
			},
			loading: func(m *Model, v bool) {
				m.ipsecTunnels = m.ipsecTunnels.SetLoading(v)
			},
			isLoading:   func(m *Model) bool { return m.ipsecTunnels.IsLoading() },
			isFiltering: func(m *Model) bool { return m.ipsecTunnels.IsFilterMode() },
			isExpanded:  func(m *Model) bool { return m.ipsecTunnels.IsExpanded() },
			refreshFor:  ViewIPSecTunnels,
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.gpUsers = m.gpUsers.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.gpUsers = m.gpUsers.SetSpinnerFrame(frame)
			},
			reset: func(m *Model) {
				m.gpUsers = views.NewGPUsersModel()
			},
			loading: func(m *Model, v bool) {
				m.gpUsers = m.gpUsers.SetLoading(v)
			},
			isLoading:   func(m *Model) bool { return m.gpUsers.IsLoading() },
			isFiltering: func(m *Model) bool { return m.gpUsers.IsFilterMode() },
			isExpanded:  func(m *Model) bool { return m.gpUsers.IsExpanded() },
			refreshFor:  ViewGPUsers,
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.logs = m.logs.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.logs = m.logs.SetSpinnerFrame(frame)
			},
			reset: func(m *Model) {
				m.logs = views.NewLogsModel()
			},
			loading: func(m *Model, v bool) {
				m.logs = m.logs.SetLoading(v)
			},
			isLoading:   func(m *Model) bool { return m.logs.Loading },
			isFiltering: func(m *Model) bool { return m.logs.IsFilterMode() || m.logs.IsQueryMode() },
			refreshFor:  ViewLogs,
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.objects = m.objects.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.objects = m.objects.SetSpinnerFrame(frame)
			},
			reset: func(m *Model) {
				m.objects = views.NewObjectsModel()
			},
			loading: func(m *Model, v bool) {
				m.objects = m.objects.SetLoading(v)
			},
			isLoading:   func(m *Model) bool { return m.objects.IsLoading() },
			isFiltering: func(m *Model) bool { return m.objects.IsFilterMode() },
			refreshFor:  ViewObjects,
		},

		// --- Picker views (contentHeight; no spinner; not refreshable) ---
		{
			resize: func(m *Model, w, h, contentH int) {
				m.picker = m.picker.SetSize(w, contentH)
			},
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.devicePicker = m.devicePicker.SetSize(w, contentH)
			},
		},
	}
}
