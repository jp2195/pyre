package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestEnsureBoundZone_LearnsTheClockOnce covers a duplicated device probe.
// A time-bounded log query has to be written in the device's wall clock, which
// the client learns once by asking for system info. The three log tabs fetch
// concurrently, so on the first bounded query they all found the zone unknown
// and each fired its own probe -- three identical op commands for one answer,
// against a device whose op queue is a shared resource.
func TestEnsureBoundZone_LearnsTheClockOnce(t *testing.T) {
	shrinkPollTimings(t, 5, 10*time.Millisecond)

	var probes atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		case q.Get("type") == "op":
			probes.Add(1)
			// Slow enough that the other callers are certain to arrive
			// while this one is still in flight.
			time.Sleep(50 * time.Millisecond)
			fmt.Fprint(w, `<response status="success"><result><system>`+
				`<hostname>fw-1</hostname><time>2026/09/05 12:00:00</time>`+
				`</system></result></response>`)
		case q.Get("action") == "get":
			fmt.Fprint(w, `<response status="success"><result><job><status>FIN</status></job>`+
				`<log><logs count="0"></logs></log></result></response>`)
		default:
			fmt.Fprint(w, `<response status="success"><result><job>1</job></result></response>`)
		}
	})

	q := LogQuery{Max: 10, Since: time.Now().Add(-time.Hour)}
	var wg sync.WaitGroup
	for range 3 {
		wg.Go(func() {
			_, _ = c.GetSystemLogs(context.Background(), q, "")
		})
	}
	wg.Wait()

	if got := probes.Load(); got != 1 {
		t.Errorf("learned the device clock with %d probes, want 1", got)
	}
}
