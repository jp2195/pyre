package tui

// viewslots.go – single registration table that drives handleWindowSize,
// handleSpinnerTick, and handleRefresh.
//
// Each viewSlot encodes all three fan-out roles for one sub-view model:
//   resize    – always non-nil; called for every slot during handleWindowSize.
//   spinner   – non-nil for the 14 views that display a spinner frame
//               (9 table views + 5 dashboards).
//   loading   – non-nil for the 9 refreshable views; called with true on refresh.
//   refreshFor – the ViewState that triggers a refresh for this slot; 0 when the
//                slot is not refreshable.
//
// Adding a new view means adding exactly one entry here.

type viewSlot struct {
	resize     func(m *Model, w, h, contentH int)
	spinner    func(m *Model, frame string)
	loading    func(m *Model, v bool)
	refreshFor ViewState
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
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.networkDashboard = m.networkDashboard.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.networkDashboard = m.networkDashboard.SetSpinnerFrame(frame)
			},
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.securityDashboard = m.securityDashboard.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.securityDashboard = m.securityDashboard.SetSpinnerFrame(frame)
			},
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.vpnDashboard = m.vpnDashboard.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.vpnDashboard = m.vpnDashboard.SetSpinnerFrame(frame)
			},
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.configDashboard = m.configDashboard.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.configDashboard = m.configDashboard.SetSpinnerFrame(frame)
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
			loading: func(m *Model, v bool) {
				m.policies = m.policies.SetLoading(v)
			},
			refreshFor: ViewPolicies,
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.natPolicies = m.natPolicies.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.natPolicies = m.natPolicies.SetSpinnerFrame(frame)
			},
			loading: func(m *Model, v bool) {
				m.natPolicies = m.natPolicies.SetLoading(v)
			},
			refreshFor: ViewNATPolicies,
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.sessions = m.sessions.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.sessions = m.sessions.SetSpinnerFrame(frame)
			},
			loading: func(m *Model, v bool) {
				m.sessions = m.sessions.SetLoading(v)
			},
			refreshFor: ViewSessions,
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.interfaces = m.interfaces.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.interfaces = m.interfaces.SetSpinnerFrame(frame)
			},
			loading: func(m *Model, v bool) {
				m.interfaces = m.interfaces.SetLoading(v)
			},
			refreshFor: ViewInterfaces,
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.routes = m.routes.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.routes = m.routes.SetSpinnerFrame(frame)
			},
			loading: func(m *Model, v bool) {
				m.routes = m.routes.SetLoading(v)
			},
			refreshFor: ViewRoutes,
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.ipsecTunnels = m.ipsecTunnels.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.ipsecTunnels = m.ipsecTunnels.SetSpinnerFrame(frame)
			},
			loading: func(m *Model, v bool) {
				m.ipsecTunnels = m.ipsecTunnels.SetLoading(v)
			},
			refreshFor: ViewIPSecTunnels,
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.gpUsers = m.gpUsers.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.gpUsers = m.gpUsers.SetSpinnerFrame(frame)
			},
			loading: func(m *Model, v bool) {
				m.gpUsers = m.gpUsers.SetLoading(v)
			},
			refreshFor: ViewGPUsers,
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.logs = m.logs.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.logs = m.logs.SetSpinnerFrame(frame)
			},
			loading: func(m *Model, v bool) {
				m.logs = m.logs.SetLoading(v)
			},
			refreshFor: ViewLogs,
		},
		{
			resize: func(m *Model, w, h, contentH int) {
				m.objects = m.objects.SetSize(w, contentH)
			},
			spinner: func(m *Model, frame string) {
				m.objects = m.objects.SetSpinnerFrame(frame)
			},
			loading: func(m *Model, v bool) {
				m.objects = m.objects.SetLoading(v)
			},
			refreshFor: ViewObjects,
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
