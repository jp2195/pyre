package tui

import "testing"

// TestDetailViewRegistryComplete guards the invariant that every detail
// view has a fully-populated registry entry and that order and map agree.
func TestDetailViewRegistryComplete(t *testing.T) {
	if len(detailViewOrder) != len(detailViews) {
		t.Fatalf("detailViewOrder has %d entries, detailViews has %d", len(detailViewOrder), len(detailViews))
	}
	for _, vs := range detailViewOrder {
		e, ok := detailViews[vs]
		if !ok {
			t.Fatalf("detailViews missing entry for ViewState %d", vs)
		}
		if e.name == "" {
			t.Errorf("ViewState %d: empty name", vs)
		}
		if e.hasData == nil || e.setLoading == nil || e.setSize == nil ||
			e.setSpinner == nil || e.update == nil || e.render == nil || e.refresh == nil {
			t.Errorf("%s: incomplete viewEntry (nil closure)", e.name)
		}
	}
	// Every detail view must also be reachable from the navbar tables.
	for _, vs := range detailViewOrder {
		if _, ok := viewToNavbar[vs]; !ok {
			t.Errorf("ViewState %d (%s) has no viewToNavbar entry", vs, detailViews[vs].name)
		}
	}
}
