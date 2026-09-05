package views

import (
	"strings"
	"testing"

	"github.com/jp2195/pyre/internal/models"
)

// TestRoutesView_IsFramedLikeOtherListViews pins the Routes view to the same
// shape as every other list view. It alone rendered bare: no panel border and
// no title line, so moving between Analyze views made the screen jump between
// framed and unframed content.
func TestRoutesView_IsFramedLikeOtherListViews(t *testing.T) {
	InitStyles()

	m := NewRoutesModel().SetSize(140, 40)
	m = m.SetRoutes([]models.RouteEntry{
		{Destination: "10.0.0.0/24", Nexthop: "direct", Interface: "ethernet1/1", Protocol: "connected", VirtualRouter: "default"},
		{Destination: "0.0.0.0/0", Nexthop: "10.0.0.1", Interface: "ethernet1/1", Protocol: "static", VirtualRouter: "default"},
	}, nil)

	out := plain(m.View())

	if !strings.Contains(out, "╭") || !strings.Contains(out, "╰") {
		t.Error("routes view renders without the panel border every other list view has")
	}
	if !strings.Contains(out, "Routes") {
		t.Error("routes view has no title")
	}
	// The banner other list views carry: how many items, and how to act on them.
	if !strings.Contains(out, "2 routes") {
		t.Errorf("routes view does not report its item count:\n%s", out)
	}
	if !strings.Contains(out, "filter") {
		t.Error("routes view does not advertise its filter key")
	}
}

// TestRoutesView_NeighborsTabIsFramedToo covers the other tab.
func TestRoutesView_NeighborsTabIsFramedToo(t *testing.T) {
	InitStyles()

	m := NewRoutesModel().SetSize(140, 40)
	m = m.SetBGPNeighbors([]models.BGPNeighbor{
		{PeerAddress: "10.0.0.2", PeerAS: 65001, State: "Established", VirtualRouter: "default"},
	}, nil)
	m.activeTab = RoutesTabNeighbors

	out := plain(m.View())
	if !strings.Contains(out, "╭") || !strings.Contains(out, "╰") {
		t.Error("neighbors tab renders without a panel border")
	}
	if !strings.Contains(out, "10.0.0.2") {
		t.Error("neighbors tab does not render its data")
	}
}

// TestInterfaceRow_StripsRouterPrefix covers a column that spent three of its
// twelve characters repeating itself. PAN-OS reports the forwarding domain as
// "lr:default" under advanced routing and "vr:default" under legacy, but the
// routing mode is global to a device, so the prefix is identical on every row
// and carries no per-row information. The column heading already says VR.
func TestInterfaceRow_StripsRouterPrefix(t *testing.T) {
	InitStyles()

	for _, tc := range []struct{ in, want string }{
		{"lr:default", "default"},
		{"vr:default", "default"},
		{"vr:corp-vr", "corp-vr"},
		{"default", "default"},
		{"", ""},
	} {
		if got := routerName(tc.in); got != tc.want {
			t.Errorf("routerName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}

	row := formatInterfaceRow(models.Interface{
		Name: "ethernet1/1", Type: "ethernet", Zone: "untrust",
		IP: "10.0.0.1/24", MAC: "aa:bb:cc:dd:ee:01", VirtualRouter: "lr:default",
	}, 200)
	if strings.Contains(row, "lr:") {
		t.Errorf("interface row still carries the router prefix: %q", row)
	}
	if !strings.Contains(row, "default") {
		t.Errorf("interface row lost the router name: %q", row)
	}
}
