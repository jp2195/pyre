package views

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
)

func routesFixture(n int) []models.RouteEntry {
	routes := make([]models.RouteEntry, n)
	for i := range routes {
		routes[i] = models.RouteEntry{
			Destination: fmt.Sprintf("10.0.%d.0/24", i),
			Nexthop:     "10.0.0.1",
			Protocol:    "static",
		}
	}
	return routes
}

func TestRoutesModel_SetSize_ClampsNegativeCursor(t *testing.T) {
	m := NewRoutesModel()
	m = m.SetRoutes(routesFixture(5), nil)
	m.Cursor = -1

	m = m.SetSize(80, 30)

	if m.Cursor != 0 {
		t.Errorf("Cursor = %d, want 0 after clamping negative cursor", m.Cursor)
	}
}

func TestRoutesModel_SetSize_ScrollsOffsetDownToCursor(t *testing.T) {
	m := NewRoutesModel()
	m = m.SetRoutes(routesFixture(20), nil)
	m.Cursor = 5
	m.Offset = 10

	m = m.SetSize(80, 30)

	if m.Offset != m.Cursor {
		t.Errorf("Offset = %d, want %d (cursor becomes new top of window)", m.Offset, m.Cursor)
	}
}

func TestRoutesModel_SetSize_ClampsNegativeNeighborCursor(t *testing.T) {
	m := NewRoutesModel()
	m = m.SetBGPNeighbors([]models.BGPNeighbor{{PeerAddress: "10.0.0.1", State: "Established"}}, nil)
	m.neighborCursor = -2

	m = m.SetSize(80, 30)

	if m.neighborCursor != 0 {
		t.Errorf("neighborCursor = %d, want 0 after clamping", m.neighborCursor)
	}
}

func TestRoutesModel_SetSize_ScrollsNeighborOffsetDownToCursor(t *testing.T) {
	m := NewRoutesModel()
	neighbors := make([]models.BGPNeighbor, 20)
	for i := range neighbors {
		neighbors[i] = models.BGPNeighbor{PeerAddress: fmt.Sprintf("10.0.0.%d", i+1), State: "Established"}
	}
	m = m.SetBGPNeighbors(neighbors, nil)
	m.neighborCursor = 5
	m.neighborOffset = 10

	m = m.SetSize(80, 30)

	if m.neighborOffset != m.neighborCursor {
		t.Errorf("neighborOffset = %d, want %d (cursor becomes new top of window)", m.neighborOffset, m.neighborCursor)
	}
}

func TestRoutesModel_SetSize_ScrollsNeighborOffsetUpToCursor(t *testing.T) {
	m := NewRoutesModel()
	neighbors := make([]models.BGPNeighbor, 40)
	for i := range neighbors {
		neighbors[i] = models.BGPNeighbor{PeerAddress: fmt.Sprintf("10.0.0.%d", i+1), State: "Established"}
	}
	m = m.SetBGPNeighbors(neighbors, nil)
	// Height 30 with overhead 10 gives 20 visible rows; cursor 35 with
	// offset 0 is far below the window, so the offset must scroll up to
	// cursor-visible+1 = 16.
	m.neighborCursor = 35
	m.neighborOffset = 0

	m = m.SetSize(80, 30)

	if m.neighborOffset != 16 {
		t.Errorf("neighborOffset = %d, want 16 (cursor - visibleRows + 1)", m.neighborOffset)
	}
	if m.neighborCursor != 35 {
		t.Errorf("neighborCursor = %d, want 35 (unchanged)", m.neighborCursor)
	}
}

// routesFixtureNew is a fixture for newer behavioral tests.
func routesFixtureNew() []models.RouteEntry {
	return []models.RouteEntry{
		{Destination: "10.1.0.0/16", Nexthop: "192.0.2.1", Interface: "ethernet1/1", Protocol: "bgp", VirtualRouter: "default", Metric: 20},
		{Destination: "10.2.0.0/16", Nexthop: "", Interface: "ethernet1/2", Protocol: "connected", VirtualRouter: "default"},
		{Destination: "0.0.0.0/0", Nexthop: "192.0.2.254", Interface: "ethernet1/1", Protocol: "static", VirtualRouter: "default", Metric: 10},
	}
}

func pressRoutes(m RoutesModel, keys string) RoutesModel {
	for _, r := range keys {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

func TestRoutes_TextFilterNarrowsRows(t *testing.T) {
	InitStyles()
	m := NewRoutesModel()
	m = m.SetSize(120, 30)
	m = m.SetRoutes(routesFixtureNew(), nil)

	m = pressRoutes(m, "/")
	if !m.IsFilterMode() {
		t.Fatal("expected filter mode after /")
	}
	m = pressRoutes(m, "10.1")
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	out := m.View()
	if !strings.Contains(out, "10.1.0.0/16") {
		t.Errorf("expected matching route visible:\n%s", out)
	}
	if strings.Contains(out, "10.2.0.0/16") {
		t.Errorf("expected non-matching route filtered out:\n%s", out)
	}
	if !strings.Contains(out, "1 of 3 routes") {
		t.Errorf("expected filtered summary '1 of 3 routes':\n%s", out)
	}
}

func TestRoutes_ProtocolFilterKeys(t *testing.T) {
	InitStyles()
	m := NewRoutesModel()
	m = m.SetSize(120, 30)
	m = m.SetRoutes(routesFixtureNew(), nil)

	m = pressRoutes(m, "b") // BGP only
	out := m.View()
	if !strings.Contains(out, "(filter: bgp)") {
		t.Errorf("expected protocol filter indicator:\n%s", out)
	}
	if !strings.Contains(out, "10.1.0.0/16") || strings.Contains(out, "10.2.0.0/16") {
		t.Errorf("expected only the bgp route:\n%s", out)
	}

	m = pressRoutes(m, "a") // back to all
	out = m.View()
	if !strings.Contains(out, "3 routes") {
		t.Errorf("expected all routes after 'a':\n%s", out)
	}
	if strings.Contains(out, "of 3 routes") || strings.Contains(out, "(filter:") {
		t.Errorf("expected filter fully cleared after 'a':\n%s", out)
	}
}

func TestRoutes_DefaultSortDestinationAscending(t *testing.T) {
	InitStyles()
	m := NewRoutesModel()
	m = m.SetSize(120, 30)
	m = m.SetRoutes(routesFixtureNew(), nil)

	out := m.View()
	// "0.0.0.0/0" sorts before "10.1.0.0/16" with SortAsc=true. Guard
	// presence first: a missing route would make Index return -1 and
	// pass the ordering check vacuously.
	if !strings.Contains(out, "0.0.0.0/0") || !strings.Contains(out, "10.1.0.0/16") {
		t.Fatalf("expected both routes in output:\n%s", out)
	}
	if strings.Index(out, "0.0.0.0/0") > strings.Index(out, "10.1.0.0/16") {
		t.Errorf("expected default route first (destination ascending):\n%s", out)
	}
}

func TestRoutes_TabSwitchToNeighbors(t *testing.T) {
	InitStyles()
	m := NewRoutesModel()
	m = m.SetSize(120, 30)
	m = m.SetRoutes(routesFixtureNew(), nil)
	m = m.SetBGPNeighbors([]models.BGPNeighbor{
		{PeerAddress: "203.0.113.5", PeerAS: 65001, State: "Established", PrefixesReceived: 42, Uptime: "3d", VirtualRouter: "default"},
	}, nil)
	m = m.SetOSPFNeighbors([]models.OSPFNeighbor{
		{NeighborID: "1.1.1.1", State: "Full", Area: "0.0.0.0", DeadTime: "00:38", VirtualRouter: "default"},
	}, nil)

	m = pressRoutes(m, "]")
	out := m.View()
	if !strings.Contains(out, "1 BGP peers") || !strings.Contains(out, "1 OSPF neighbors") {
		t.Errorf("expected neighbor summary after tab switch:\n%s", out)
	}
	if !strings.Contains(out, "203.0.113.5") {
		t.Errorf("expected BGP peer row:\n%s", out)
	}
	if !strings.Contains(out, "AS65001") {
		t.Errorf("expected AS number:\n%s", out)
	}
	if !strings.Contains(out, "1.1.1.1") {
		t.Errorf("expected OSPF neighbor row:\n%s", out)
	}

	// "[" switches back.
	m = pressRoutes(m, "[")
	out = m.View()
	if !strings.Contains(out, "3 routes") || strings.Contains(out, "of 3 routes") {
		t.Errorf("expected unfiltered routes tab after switching back:\n%s", out)
	}
}

func TestRoutes_NeighborsEmptyState(t *testing.T) {
	InitStyles()
	m := NewRoutesModel()
	m = m.SetSize(120, 30)
	m = m.SetBGPNeighbors([]models.BGPNeighbor{}, nil)
	m = m.SetOSPFNeighbors([]models.OSPFNeighbor{}, nil)

	m = pressRoutes(m, "]")
	if out := m.View(); !strings.Contains(out, "No BGP or OSPF neighbors configured") {
		t.Errorf("expected empty-state message:\n%s", out)
	}
}

func TestRoutes_NeighborNavigation(t *testing.T) {
	InitStyles()
	m := NewRoutesModel()
	m = m.SetSize(120, 30)
	m = m.SetBGPNeighbors([]models.BGPNeighbor{
		{PeerAddress: "203.0.113.1"}, {PeerAddress: "203.0.113.2"}, {PeerAddress: "203.0.113.3"},
	}, nil)
	m = pressRoutes(m, "]")

	m = pressRoutes(m, "j")
	if m.neighborCursor != 1 {
		t.Errorf("neighborCursor = %d after j, want 1", m.neighborCursor)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: 'G', Text: "G"})
	if m.neighborCursor != 2 {
		t.Errorf("neighborCursor = %d after G, want 2", m.neighborCursor)
	}
	m = pressRoutes(m, "g")
	if m.neighborCursor != 0 {
		t.Errorf("neighborCursor = %d after g, want 0", m.neighborCursor)
	}
	m = pressRoutes(m, "k") // at top already: stays 0
	if m.neighborCursor != 0 {
		t.Errorf("neighborCursor = %d after k at top, want 0", m.neighborCursor)
	}
}

func TestRoutes_ErrorAndLoadingStates(t *testing.T) {
	InitStyles()
	m := NewRoutesModel()
	m = m.SetSize(120, 30)

	if out := m.View(); !strings.Contains(out, "Loading routes...") {
		t.Errorf("expected loading state before data:\n%s", out)
	}

	m = m.SetRoutes(nil, errTest)
	if out := m.View(); !strings.Contains(out, "test error") {
		t.Errorf("expected error message:\n%s", out)
	}
	if !m.HasData() {
		t.Error("expected HasData true after error (error counts as data)")
	}
}

func TestRoutes_ViewZeroWidth(t *testing.T) {
	m := NewRoutesModel()
	if out := m.View(); out != "Loading..." {
		t.Errorf("expected zero-width fallback 'Loading...', got %q", out)
	}
}

// TestRoutes_NeighborRefreshClampsCursor guards the shrink-refresh case:
// with the cursor advanced, a refresh returning fewer neighbors must not
// strand cursor/offset past the new list (which rendered an empty tab).
func TestRoutes_NeighborRefreshClampsCursor(t *testing.T) {
	InitStyles()
	m := NewRoutesModel()
	m = m.SetSize(120, 30)
	m = m.SetBGPNeighbors([]models.BGPNeighbor{
		{PeerAddress: "203.0.113.1"}, {PeerAddress: "203.0.113.2"}, {PeerAddress: "203.0.113.3"},
	}, nil)
	m = pressRoutes(m, "]")
	m, _ = m.Update(tea.KeyPressMsg{Code: 'G', Text: "G"}) // cursor -> 2

	// Refresh returns a single neighbor.
	m = m.SetBGPNeighbors([]models.BGPNeighbor{{PeerAddress: "198.51.100.9"}}, nil)

	if m.neighborCursor > 0 {
		t.Errorf("neighborCursor = %d after shrink, want clamped to 0", m.neighborCursor)
	}
	if m.neighborOffset > m.neighborCursor {
		t.Errorf("neighborOffset = %d > cursor %d after shrink", m.neighborOffset, m.neighborCursor)
	}
	if out := m.View(); !strings.Contains(out, "198.51.100.9") {
		t.Errorf("expected remaining neighbor rendered after shrink:\n%s", out)
	}

	// Shrink to zero must also be safe.
	m = m.SetBGPNeighbors(nil, nil)
	m = m.SetOSPFNeighbors(nil, nil)
	_ = m.View() // must not panic
	if m.neighborCursor != 0 || m.neighborOffset != 0 {
		t.Errorf("cursor/offset = %d/%d with no neighbors, want 0/0", m.neighborCursor, m.neighborOffset)
	}
}
