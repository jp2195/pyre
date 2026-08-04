package views

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jp2195/pyre/internal/models"
)

func TestNewDashboardModel(t *testing.T) {
	m := NewDashboardModel()

	if m.Width != 0 {
		t.Errorf("expected Width=0, got %d", m.Width)
	}
	if m.Height != 0 {
		t.Errorf("expected Height=0, got %d", m.Height)
	}
	if m.systemInfo != nil {
		t.Error("expected systemInfo=nil")
	}
}

func TestDashboardModel_SetSize(t *testing.T) {
	m := NewDashboardModel()
	m = m.SetSize(100, 50)

	if m.Width != 100 {
		t.Errorf("expected Width=100, got %d", m.Width)
	}
	if m.Height != 50 {
		t.Errorf("expected Height=50, got %d", m.Height)
	}
}

func TestDashboardModel_SetSystemInfo(t *testing.T) {
	m := NewDashboardModel()

	info := &models.SystemInfo{
		Hostname: "fw01",
		Model:    "PA-3220",
	}
	m = m.SetSystemInfo(info, nil)

	if m.systemInfo != info {
		t.Error("expected systemInfo to be set")
	}
	if m.sysInfoErr != nil {
		t.Error("expected no error")
	}

	// With error
	err := errors.New("API error")
	m = m.SetSystemInfo(nil, err)
	if m.sysInfoErr != err {
		t.Error("expected error to be set")
	}
}

func TestDashboardModel_SetResources(t *testing.T) {
	m := NewDashboardModel()

	res := &models.Resources{
		CPUPercent:    50.0,
		MemoryPercent: 75.0,
	}
	m = m.SetResources(res, nil)

	if m.resources != res {
		t.Error("expected resources to be set")
	}

	// With error
	err := errors.New("API error")
	m = m.SetResources(nil, err)
	if m.resourceErr != err {
		t.Error("expected error to be set")
	}
}

func TestDashboardModel_SetSessionInfo(t *testing.T) {
	m := NewDashboardModel()

	info := &models.SessionInfo{
		ActiveCount: 1000,
		MaxCount:    5000,
	}
	m = m.SetSessionInfo(info, nil)

	if m.sessionInfo != info {
		t.Error("expected sessionInfo to be set")
	}

	// With error
	err := errors.New("API error")
	m = m.SetSessionInfo(nil, err)
	if m.sessionErr != err {
		t.Error("expected error to be set")
	}
}

func TestDashboardModel_SetHAStatus(t *testing.T) {
	m := NewDashboardModel()

	status := &models.HAStatus{
		Enabled:   true,
		State:     "active",
		PeerState: "passive",
	}
	m = m.SetHAStatus(status, nil)

	if m.haStatus != status {
		t.Error("expected haStatus to be set")
	}

	// With error
	err := errors.New("API error")
	m = m.SetHAStatus(nil, err)
	if m.haErr != err {
		t.Error("expected error to be set")
	}
}

func TestDashboardModel_SetInterfaces(t *testing.T) {
	m := NewDashboardModel()

	ifaces := []models.Interface{
		{Name: "ethernet1/1", State: "up"},
		{Name: "ethernet1/2", State: "down"},
	}
	m = m.SetInterfaces(ifaces, nil)

	if len(m.interfaces) != 2 {
		t.Errorf("expected 2 interfaces, got %d", len(m.interfaces))
	}

	// With error
	err := errors.New("API error")
	m = m.SetInterfaces(nil, err)
	if m.ifaceErr != err {
		t.Error("expected error to be set")
	}
}

func TestDashboardModel_SetThreatSummary(t *testing.T) {
	m := NewDashboardModel()

	summary := &models.ThreatSummary{
		TotalThreats:  100,
		CriticalCount: 5,
		HighCount:     20,
	}
	m = m.SetThreatSummary(summary, nil)

	if m.threatSummary != summary {
		t.Error("expected threatSummary to be set")
	}

	// With error
	err := errors.New("API error")
	m = m.SetThreatSummary(nil, err)
	if m.threatErr != err {
		t.Error("expected error to be set")
	}
}

func TestDashboardName(t *testing.T) {
	tests := []struct {
		dt   DashboardType
		want string
	}{
		{DashboardMain, "Main"},
		{DashboardNetwork, "Network"},
		{DashboardSecurity, "Security"},
		{DashboardVPN, "VPN"},
		{DashboardConfig, "Config"},
		{DashboardType(99), "Main"}, // Unknown type defaults to Main
	}

	for _, tt := range tests {
		got := DashboardName(tt.dt)
		if got != tt.want {
			t.Errorf("DashboardName(%d) = %q, want %q", tt.dt, got, tt.want)
		}
	}
}

func TestDashboardType_Constants(t *testing.T) {
	if DashboardMain != 0 {
		t.Errorf("expected DashboardMain=0, got %d", DashboardMain)
	}
	if DashboardNetwork != 1 {
		t.Errorf("expected DashboardNetwork=1, got %d", DashboardNetwork)
	}
	if DashboardSecurity != 2 {
		t.Errorf("expected DashboardSecurity=2, got %d", DashboardSecurity)
	}
	if DashboardVPN != 3 {
		t.Errorf("expected DashboardVPN=3, got %d", DashboardVPN)
	}
	if DashboardConfig != 4 {
		t.Errorf("expected DashboardConfig=4, got %d", DashboardConfig)
	}
}

// TestSecurityDashboard_ThreatSeverityShowsErrorNotSpinner guards against a
// failed threat fetch rendering as a permanent "Loading..." spinner. The
// sibling Threat Summary panel already degrades to "Not available"; this
// panel spun forever, so the user could not tell a slow fetch from a dead one.
func TestSecurityDashboard_ThreatSeverityShowsErrorNotSpinner(t *testing.T) {
	m := NewSecurityDashboardModel()
	m = m.SetSpinnerFrame("|")
	m = m.SetThreatSummary(nil, errors.New("op command failed"))

	out := m.renderThreatSeverity(60)
	if strings.Contains(out, "Loading") {
		t.Errorf("threat severity panel spins forever on error; got:\n%s", out)
	}
	if !strings.Contains(out, "Not available") {
		t.Errorf("expected 'Not available' on error; got:\n%s", out)
	}
}

// TestHasData_WaitsForEveryPanel guards the frozen-spinner bug: HasData drives
// the app-wide spinner tick chain (Model.anyLoading). When it reported "done"
// while some panels were still fetching, the tick chain was dropped and those
// panels sat on "Loading..." with a spinner frozen mid-frame. A dashboard is
// only settled once every source has produced data OR an error.
func TestHasData_WaitsForEveryPanel(t *testing.T) {
	t.Run("security waits for policies", func(t *testing.T) {
		m := NewSecurityDashboardModel()
		m = m.SetThreatSummary(&models.ThreatSummary{}, nil)
		if m.HasData() {
			t.Error("security dashboard reported settled while Zero-Hit/Most-Hit " +
				"rules were still loading; spinner would freeze")
		}
		m = m.SetPolicies(nil, errors.New("boom"))
		if !m.HasData() {
			t.Error("an errored policy fetch still settles the dashboard")
		}
	})

	t.Run("config waits for pending changes", func(t *testing.T) {
		m := NewConfigDashboardModel()
		m = m.SetPolicies([]models.SecurityRule{}, nil)
		if m.HasData() {
			t.Error("config dashboard settled before pending changes arrived")
		}
		m = m.SetPendingChanges(nil, errors.New("boom"))
		if !m.HasData() {
			t.Error("an errored changes fetch still settles the dashboard")
		}
	})

	t.Run("vpn waits for gp users", func(t *testing.T) {
		m := NewVPNDashboardModel()
		m = m.SetIPSecTunnels([]models.IPSecTunnel{}, nil)
		if m.HasData() {
			t.Error("vpn dashboard settled before GlobalProtect users arrived")
		}
		m = m.SetGlobalProtectUsers(nil, errors.New("boom"))
		if !m.HasData() {
			t.Error("an errored gp-user fetch still settles the dashboard")
		}
	})

	t.Run("overview waits for resources", func(t *testing.T) {
		m := NewDashboardModel()
		m = m.SetSystemInfo(&models.SystemInfo{}, nil)
		if m.HasData() {
			t.Error("overview settled while the Resources panel was still loading")
		}
	})
}

// TestClampToHeight_NeverOverflowsTerminal pins that a dashboard's panel stack
// is trimmed to the visible height. It used to render at full height
// regardless of terminal size: at 80x24 that pushed Disk Usage, Hardware
// Status and the entire footer off-screen, and at 120x40 it corrupted the
// display. DashboardBase.Height was stored and never read.
func TestClampToHeight_NeverOverflowsTerminal(t *testing.T) {
	InitStyles()
	content := strings.Join(makeLines(60), "\n")

	d := DashboardBase{Width: 120, Height: 20}
	got := strings.Split(d.ClampToHeight(content), "\n")
	if len(got) > d.Height {
		t.Errorf("rendered %d lines into a %d-line viewport", len(got), d.Height)
	}
	if !strings.Contains(got[len(got)-1], "more") {
		t.Errorf("expected a scroll indicator on the last line, got %q", got[len(got)-1])
	}

	// Content that already fits is returned untouched.
	short := strings.Join(makeLines(5), "\n")
	if d.ClampToHeight(short) != short {
		t.Error("content that fits must not be modified")
	}
}

// TestScrollBy_ClampsToContent ensures scrolling cannot run past either end,
// so the view never goes blank and k always returns you to the top.
func TestScrollBy_ClampsToContent(t *testing.T) {
	d := DashboardBase{Width: 120, Height: 20}
	const contentHeight = 60

	if got := d.ScrollBy(-5, contentHeight).Offset; got != 0 {
		t.Errorf("scrolling up from the top gave Offset=%d, want 0", got)
	}

	maxOff := d.MaxScrollOffset(contentHeight)
	if got := d.ScrollBy(9999, contentHeight).Offset; got != maxOff {
		t.Errorf("scrolling past the end gave Offset=%d, want %d", got, maxOff)
	}

	// At max offset the last content line must still be visible.
	d.Offset = maxOff
	lines := strings.Split(d.ClampToHeight(strings.Join(makeLines(contentHeight), "\n")), "\n")
	if !strings.Contains(strings.Join(lines, "\n"), "line-60") {
		t.Error("the final content line is unreachable at max scroll offset")
	}

	// Content that fits cannot scroll at all.
	if got := d.MaxScrollOffset(5); got != 0 {
		t.Errorf("MaxScrollOffset for fitting content = %d, want 0", got)
	}
}

func makeLines(n int) []string {
	out := make([]string, 0, n)
	for i := range n {
		out = append(out, fmt.Sprintf("line-%d", i+1))
	}
	return out
}
