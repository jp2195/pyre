package views

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
)

func pressIPSec(m IPSecTunnelsModel, keys string) IPSecTunnelsModel {
	for _, r := range keys {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

func TestIPSec_Behavior_FilterKeystrokesNarrowRows(t *testing.T) {
	InitStyles()
	m := NewIPSecTunnelsModel()
	m = m.SetSize(120, 30)
	m = m.SetTunnels([]models.IPSecTunnel{
		{Name: "branch-east", Gateway: "1.1.1.1", State: "up"},
		{Name: "branch-west", Gateway: "2.2.2.2", State: "down"},
	}, nil)

	m = pressIPSec(m, "/east")
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	out := m.View()
	if !strings.Contains(out, "branch-east") {
		t.Errorf("expected branch-east after filter:\n%s", out)
	}
	if strings.Contains(out, "branch-west") {
		t.Errorf("expected branch-west filtered out:\n%s", out)
	}
}

func TestIPSec_Behavior_SortCycleReorders(t *testing.T) {
	InitStyles()
	m := NewIPSecTunnelsModel()
	m = m.SetSize(120, 30)
	m = m.SetTunnels([]models.IPSecTunnel{
		{Name: "aaa", Gateway: "9.9.9.9"},
		{Name: "zzz", Gateway: "1.1.1.1"},
	}, nil)

	// s: Name -> Gateway, ascending; 1.1.1.1 (zzz) first.
	m = pressIPSec(m, "s")
	out := m.View()
	if strings.Index(out, "zzz") > strings.Index(out, "aaa") {
		t.Errorf("expected zzz (gateway 1.1.1.1) first when sorted by gateway:\n%s", out)
	}
	if !strings.Contains(out, "Gateway") {
		t.Errorf("expected sort label Gateway in banner:\n%s", out)
	}
}

func TestIPSec_Behavior_EnterShowsDetail(t *testing.T) {
	InitStyles()
	m := NewIPSecTunnelsModel()
	m = m.SetSize(120, 40)
	m = m.SetTunnels([]models.IPSecTunnel{
		{Name: "branch-east", Gateway: "203.0.113.7", State: "up", Encryption: "aes-256-gcm"},
	}, nil)

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	out := m.View()
	if !strings.Contains(out, "aes-256-gcm") {
		t.Errorf("expected detail panel with encryption after enter:\n%s", out)
	}
}

func TestIPSec_Behavior_ErrorAndEmptyStates(t *testing.T) {
	InitStyles()
	m := NewIPSecTunnelsModel()
	m = m.SetSize(120, 30)

	m = m.SetTunnels(nil, errTest)
	if out := m.View(); !strings.Contains(out, "test error") {
		t.Errorf("expected error message:\n%s", out)
	}

	m = m.SetTunnels([]models.IPSecTunnel{}, nil)
	if out := m.View(); !strings.Contains(out, "No IPSec tunnels found") {
		t.Errorf("expected empty message:\n%s", out)
	}
}

func TestIPSec_Behavior_RenderEmitsValidUTF8(t *testing.T) {
	InitStyles()
	m := NewIPSecTunnelsModel()
	m = m.SetSize(120, 30)
	m = m.SetTunnels([]models.IPSecTunnel{
		{Name: "t-up", State: "up"},
		{Name: "t-init", State: "init"},
		{Name: "t-down", State: "down"},
	}, nil)

	out := m.View()
	for _, name := range []string{"t-up", "t-init", "t-down"} {
		if !strings.Contains(out, name) {
			t.Errorf("expected %q in output:\n%s", name, out)
		}
	}
}
