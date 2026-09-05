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

// parsePANTimeIn tries each layout in panosTimeLayouts and returns the first
// successful parse, interpreting the value in loc.
//
// PAN-OS timestamps are bare device-local wall clock with no offset, so the
// location has to come from outside the string. Interpreting them as UTC,
// which is what time.Parse does with a zoneless layout, puts every timestamp
// out by the device's offset: on a PA-440 keeping EDT, a log line written one
// second earlier displayed as "4h ago". See Client.deviceLocation.
func parsePANTimeIn(s string, loc *time.Location) (time.Time, error) {
	for _, layout := range panosTimeLayouts {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized PAN-OS timestamp %q", s)
}

// parsePANTime interprets s in the device's zone.
func (c *Client) parsePANTime(s string) (time.Time, error) {
	return parsePANTimeIn(s, c.deviceLocation())
}
