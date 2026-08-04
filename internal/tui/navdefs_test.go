package tui

import (
	"reflect"
	"testing"

	"github.com/jp2195/pyre/internal/tui/views"
)

// TestNavbarGroups_MatchesLegacyLiteral pins navbarGroups() output to the
// exact content of the navbar literal it replaces (views/navbar.go as of
// M6): same group/item IDs, labels, positional keys, and order.
func TestNavbarGroups_MatchesLegacyLiteral(t *testing.T) {
	want := []views.NavGroup{
		{
			ID:    "monitor",
			Label: "Monitor",
			Key:   "1",
			Items: []views.NavItem{
				{ID: "overview", Label: "Overview", Key: "1"},
				{ID: "network", Label: "Network", Key: "2"},
				{ID: "security", Label: "Security", Key: "3"},
				{ID: "vpn", Label: "VPN", Key: "4"},
			},
		},
		{
			ID:    "analyze",
			Label: "Analyze",
			Key:   "2",
			Items: []views.NavItem{
				{ID: "policies", Label: "Policies", Key: "1"},
				{ID: "nat", Label: "NAT", Key: "2"},
				{ID: "objects", Label: "Objects", Key: "3"},
				{ID: "sessions", Label: "Sessions", Key: "4"},
				{ID: "interfaces", Label: "Interfaces", Key: "5"},
				{ID: "routes", Label: "Routes", Key: "6"},
				{ID: "ipsec", Label: "IPSec", Key: "7"},
				{ID: "gpusers", Label: "GP Users", Key: "8"},
				{ID: "logs", Label: "Logs", Key: "9"},
			},
		},
		{
			ID:    "tools",
			Label: "Tools",
			Key:   "3",
			Items: []views.NavItem{
				{ID: "config", Label: "Config", Key: "1"},
			},
		},
	}

	got := navbarGroups()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("navbarGroups() diverges from legacy navbar literal:\ngot:  %#v\nwant: %#v", got, want)
	}
}

// TestNavDefs_Invariants asserts properties the three legacy structures
// could not express: globally unique item IDs and non-nil hasData/fetch.
func TestNavDefs_Invariants(t *testing.T) {
	seen := map[string]bool{}
	for _, g := range navDefs {
		for _, it := range g.items {
			if seen[it.id] {
				t.Errorf("duplicate nav item id %q", it.id)
			}
			seen[it.id] = true
			if it.hasData == nil {
				t.Errorf("nav item %q has nil hasData", it.id)
			}
			if it.fetch == nil {
				t.Errorf("nav item %q has nil fetch", it.id)
			}
		}
	}
	if len(seen) != 14 {
		t.Errorf("navDefs defines %d items; want 14 (4 monitor + 9 analyze + 1 tools)", len(seen))
	}
}
