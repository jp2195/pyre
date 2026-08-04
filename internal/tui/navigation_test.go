package tui

import (
	"testing"

	"github.com/jp2195/pyre/internal/tui/views"
)

// TestNavTargetsAndViewToNavbarAreBijective asserts that navTargets and
// viewToNavbar form a bijection between navbar item IDs and (ViewState,
// DashboardType) pairs. The correspondence is:
//
//	navTargets[id]         -> (view, dashboard)
//	viewToNavbar[view]     -> []{isDashboard, dashboard, navbarID{group,item}}
//
// Every navTargets entry must have a matching entry in viewToNavbar, and
// every viewToNavbar entry must point to a valid navTargets key. For
// ViewDashboard entries the dashboard field distinguishes siblings; for
// non-dashboard views the dashboard field is ignored.
func TestNavTargetsAndViewToNavbarAreBijective(t *testing.T) {
	// Forward: every navTargets entry must round-trip through viewToNavbar.
	for id, target := range navTargets {
		entries, ok := viewToNavbar[target.view]
		if !ok {
			t.Errorf("navTargets[%q] -> view %v has no viewToNavbar entry", id, target.view)
			continue
		}
		var matched *navbarEntry
		for i := range entries {
			e := entries[i]
			if target.view == ViewDashboard {
				if e.isDashboard && e.dashboard == target.dashboard {
					matched = &e
					break
				}
			} else {
				if !e.isDashboard {
					matched = &e
					break
				}
			}
		}
		if matched == nil {
			t.Errorf("navTargets[%q] (view=%v dashboard=%v) has no matching viewToNavbar entry", id, target.view, target.dashboard)
			continue
		}
		if matched.id.item != id {
			t.Errorf("navTargets[%q] round-trips to viewToNavbar item %q; want %q", id, matched.id.item, id)
		}
	}

	// Reverse: every viewToNavbar entry must point at a valid navTargets key.
	for view, entries := range viewToNavbar {
		for _, e := range entries {
			target, ok := navTargets[e.id.item]
			if !ok {
				t.Errorf("viewToNavbar[%v] item %q has no navTargets entry", view, e.id.item)
				continue
			}
			if target.view != view {
				t.Errorf("viewToNavbar[%v] item %q points at navTargets entry with view %v", view, e.id.item, target.view)
			}
			if view == ViewDashboard {
				if !e.isDashboard {
					t.Errorf("viewToNavbar[ViewDashboard] item %q must set isDashboard=true", e.id.item)
				}
				if target.dashboard != e.dashboard {
					t.Errorf("viewToNavbar[ViewDashboard] item %q dashboard=%v; navTargets dashboard=%v", e.id.item, e.dashboard, target.dashboard)
				}
			} else {
				if e.isDashboard {
					t.Errorf("viewToNavbar[%v] non-dashboard entry %q incorrectly marked isDashboard=true", view, e.id.item)
				}
			}
		}
	}

	// Cardinality: the number of navTargets entries must equal the total
	// number of viewToNavbar entries across all views.
	total := 0
	for _, entries := range viewToNavbar {
		total += len(entries)
	}
	if total != len(navTargets) {
		t.Errorf("viewToNavbar has %d total entries; navTargets has %d", total, len(navTargets))
	}

	// Dashboard entries must cover every known DashboardType used in navTargets.
	dashSeen := map[views.DashboardType]bool{}
	for _, target := range navTargets {
		if target.view == ViewDashboard {
			dashSeen[target.dashboard] = true
		}
	}
	for dt := range dashSeen {
		found := false
		for _, e := range viewToNavbar[ViewDashboard] {
			if e.isDashboard && e.dashboard == dt {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("DashboardType %v referenced by navTargets but missing from viewToNavbar[ViewDashboard]", dt)
		}
	}
}
