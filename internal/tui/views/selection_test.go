package views

import (
	"testing"
	"time"

	"github.com/jp2195/pyre/internal/models"
)

func TestSelectedLine_Traffic(t *testing.T) {
	m := NewLogsModel()
	m.activeLogType = models.LogTypeTraffic
	at := time.Date(2026, 11, 1, 9, 30, 0, 0, time.FixedZone("EST", -5*3600))
	m.filteredTraffic = []models.TrafficLogEntry{{
		Time: at, SourceIP: "10.0.0.5", SourcePort: 51000, DestIP: "203.0.113.9", DestPort: 443,
		Application: "ssl", Action: "allow", Rule: "allow-web",
	}}
	got, ok := m.SelectedLine()
	want := "2026-11-01T09:30:00-05:00 10.0.0.5:51000 -> 203.0.113.9:443 ssl allow rule=allow-web"
	if !ok || got != want {
		t.Errorf("SelectedLine = %q, %v\nwant          %q", got, ok, want)
	}
}

func TestSelectedLine_SystemSkipsEmptyFields(t *testing.T) {
	m := NewLogsModel()
	m.activeLogType = models.LogTypeSystem
	m.filteredSystem = []models.SystemLogEntry{{Severity: "high", Description: "HA link down"}}
	if got, _ := m.SelectedLine(); got != "high HA link down" {
		t.Errorf("SelectedLine = %q; empty time and type should leave no gaps", got)
	}
}

func TestSelectedPeer_FollowsTheTab(t *testing.T) {
	m := NewRoutesModel()
	m.filtered = []models.RouteEntry{{Destination: "0.0.0.0/0"}}
	if l, v, ok := m.SelectedPeer(); !ok || l != "route" || v != "0.0.0.0/0" {
		t.Errorf("routes tab: %q %q %v", l, v, ok)
	}

	m.activeTab = RoutesTabNeighbors
	m.bgpNeighbors = []models.BGPNeighbor{{PeerAddress: "192.0.2.1"}}
	m.ospfNeighbors = []models.OSPFNeighbor{{NeighborID: "10.255.0.2"}}
	if l, v, _ := m.SelectedPeer(); l != "BGP peer" || v != "192.0.2.1" {
		t.Errorf("first neighbor: %q %q", l, v)
	}
	m.neighborCursor = 1
	if l, v, _ := m.SelectedPeer(); l != "OSPF neighbor" || v != "10.255.0.2" {
		t.Errorf("second neighbor (OSPF, after the BGP rows): %q %q", l, v)
	}
	m.neighborCursor = 2
	if _, _, ok := m.SelectedPeer(); ok {
		t.Error("cursor past the neighbor list still selected something")
	}
}

func TestSelectedName_FollowsTheTab(t *testing.T) {
	m := NewObjectsModel()
	m.addressTab.filtered = []models.AddressObject{{Name: "web-servers"}}
	m.serviceTab.filtered = []models.ServiceObject{{Name: "tcp-8443"}}
	if l, v, _ := m.SelectedName(); l != "address" || v != "web-servers" {
		t.Errorf("address tab: %q %q", l, v)
	}
	m.tab = ObjectsTabService
	if l, v, _ := m.SelectedName(); l != "service" || v != "tcp-8443" {
		t.Errorf("service tab: %q %q", l, v)
	}
}
