package api_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jp2195/pyre/internal/api"
)

// PAN-OS reports timestamps as bare device-local wall clock with no offset,
// and `show system info` carries no time-zone field to interpret them with.
// Both facts are taken from a PA-440 running 11.2.10-h8:
//
//	<time>Fri Sep  4 21:46:19 2026</time>          (system info, no zone)
//	<time_generated>2026/09/04 21:46:19</time_generated>   (system log)
//
// Parsing those as UTC, which is what time.Parse does with a zoneless layout,
// puts every timestamp out by the device's offset. On that device the error
// was four hours: a log line written one second earlier displayed as "4h ago".
//
// The device's clock is the only offset source available, so it is derived by
// comparing the reported wall clock against ours.

// deviceServer replays the two responses these tests need, using a device
// whose clock reads offsetFromUTC away from real UTC.
func deviceServer(t *testing.T, offsetFromUTC time.Duration) *httptest.Server {
	t.Helper()
	wall := func() time.Time { return time.Now().UTC().Add(offsetFromUTC) }

	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		q := r.URL.Query()

		switch q.Get("type") {
		case "op":
			// Only system info matters here; the real device format has a
			// space-padded day, which is why the layout is written this way.
			_, _ = io.WriteString(w, fmt.Sprintf(
				`<response status="success"><result><system>`+
					`<hostname>fw-1</hostname><model>PA-440</model>`+
					`<serial>000000000000</serial><sw-version>11.2.10-h8</sw-version>`+
					`<time>%s</time>`+
					`</system></result></response>`,
				wall().Format("Mon Jan _2 15:04:05 2006")))
		case "log":
			if q.Get("action") == "get" {
				_, _ = io.WriteString(w, fmt.Sprintf(
					`<response status="success"><result><job><status>FIN</status></job>`+
						`<log><logs><entry>`+
						`<time_generated>%s</time_generated>`+
						`<type>SYSTEM</type><subtype>general</subtype>`+
						`<severity>informational</severity><opaque>fresh entry</opaque>`+
						`</entry></logs></log></result></response>`,
					wall().Format("2006/01/02 15:04:05")))
				return
			}
			_, _ = io.WriteString(w, `<response status="success"><result><job>1</job></result></response>`)
		default:
			_, _ = io.WriteString(w, `<response status="success"><result></result></response>`)
		}
	}))
}

func newDeviceClient(t *testing.T, srv *httptest.Server) *api.Client {
	t.Helper()
	c, err := api.NewClient(strings.TrimPrefix(srv.URL, "https://"), "k", api.ClientOptions{Insecure: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// TestLogTimestamps_UseTheDeviceClock is the regression that matters: a log
// line the device wrote a moment ago must read as a moment ago, whatever zone
// the device keeps its clock in.
func TestLogTimestamps_UseTheDeviceClock(t *testing.T) {
	for _, tc := range []struct {
		name   string
		offset time.Duration
	}{
		{"device four hours behind UTC", -4 * time.Hour},
		{"device nine hours ahead of UTC", 9 * time.Hour},
		{"device on UTC", 0},
		{"device on a half-hour offset", 5*time.Hour + 30*time.Minute},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := deviceServer(t, tc.offset)
			defer srv.Close()
			client := newDeviceClient(t, srv)
			ctx := context.Background()

			// The offset is learned from the device's own clock.
			if _, err := client.GetSystemInfo(ctx, ""); err != nil {
				t.Fatalf("GetSystemInfo: %v", err)
			}

			page, err := client.GetSystemLogs(ctx, api.LogQuery{Max: 10}, "")
			if err != nil {
				t.Fatalf("GetSystemLogs: %v", err)
			}
			logs := page.Entries
			if len(logs) == 0 {
				t.Fatal("expected one log entry")
			}

			age := time.Since(logs[0].Time)
			if age < -2*time.Minute || age > 2*time.Minute {
				t.Errorf("log written just now reads as %v old; timestamps are being interpreted in the wrong zone", age.Round(time.Minute))
			}
		})
	}
}

// TestAbsoluteTimestamps_RenderInDeviceWallClock checks the other half. An
// absolute time shown next to a log line should match what the device itself
// would print, so it can be correlated with the device's CLI and its own logs
// rather than silently re-based into the reader's zone.
func TestAbsoluteTimestamps_RenderInDeviceWallClock(t *testing.T) {
	const offset = -7 * time.Hour
	srv := deviceServer(t, offset)
	defer srv.Close()
	client := newDeviceClient(t, srv)
	ctx := context.Background()

	if _, err := client.GetSystemInfo(ctx, ""); err != nil {
		t.Fatalf("GetSystemInfo: %v", err)
	}
	page, err := client.GetSystemLogs(ctx, api.LogQuery{Max: 10}, "")
	if err != nil {
		t.Fatalf("GetSystemLogs: %v", err)
	}
	logs := page.Entries
	if len(logs) == 0 {
		t.Fatal("expected one log entry")
	}

	wantWall := time.Now().UTC().Add(offset).Format("2006-01-02 15:04")
	gotWall := logs[0].Time.Format("2006-01-02 15:04")
	if gotWall != wantWall {
		t.Errorf("absolute time renders as %q, want the device's own wall clock %q", gotWall, wantWall)
	}
}

// TestSystemInfoTime_IsTheDeviceWallClock covers the field the offset is
// derived from, including the space-padded day the real device emits.
func TestSystemInfoTime_IsTheDeviceWallClock(t *testing.T) {
	const offset = 3 * time.Hour
	srv := deviceServer(t, offset)
	defer srv.Close()
	client := newDeviceClient(t, srv)

	info, err := client.GetSystemInfo(context.Background(), "")
	if err != nil {
		t.Fatalf("GetSystemInfo: %v", err)
	}
	if info.CurrentTime.IsZero() {
		t.Fatal("CurrentTime not parsed; the device emits a space-padded day")
	}
	if drift := time.Since(info.CurrentTime); drift < -2*time.Minute || drift > 2*time.Minute {
		t.Errorf("device clock reads %v away from now, want ~0", drift.Round(time.Minute))
	}
}
