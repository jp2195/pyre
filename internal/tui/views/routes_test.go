package views

import (
	"fmt"
	"testing"

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

	if m.Offset > m.Cursor {
		t.Errorf("Offset = %d, want <= Cursor (%d) so the cursor row is visible", m.Offset, m.Cursor)
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

	if m.neighborOffset > m.neighborCursor {
		t.Errorf("neighborOffset = %d, want <= neighborCursor (%d)", m.neighborOffset, m.neighborCursor)
	}
}
