package views

import "testing"

func TestIPSecTunnelsModel_SetSpinnerFrame_ReachesList(t *testing.T) {
	m := NewIPSecTunnelsModel()
	m = m.SetSpinnerFrame("◢")
	if m.list.SpinnerFrame != "◢" {
		t.Errorf("list.SpinnerFrame = %q, want ◢", m.list.SpinnerFrame)
	}
}
