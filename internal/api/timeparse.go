package api

import (
	"fmt"
	"time"
)

// panosTimeLayouts enumerates every time-layout string observed in PAN-OS
// XML API responses. The union is a superset of the per-call slices that
// previously lived in config.go, logs.go, monitoring.go, system.go, vpn.go,
// etc. parsePANTime walks the list in order; callers that need a specific
// subset can still call time.Parse directly, but most call sites just want
// "try every known layout."
var panosTimeLayouts = []string{
	"2006/01/02 15:04:05",
	"2006-01-02 15:04:05",
	"Mon Jan 2 15:04:05 2006",
	"Mon Jan 02 15:04:05 2006",
	"01/02/2006 15:04:05",
	"Jan 2 15:04:05 2006 MST",
	"January 02, 2006",
}

// parsePANTime tries each layout in panosTimeLayouts and returns the first
// successful parse. Returns an error listing the input if no layout matches.
// Callers that do not care about the error (and are content with the zero
// time) can discard it with `t, _ := parsePANTime(s)`.
func parsePANTime(s string) (time.Time, error) {
	for _, layout := range panosTimeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized PAN-OS timestamp %q", s)
}
