package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

// boundTestServer answers a system-info op, a log submit, and a log get. It
// records the order of the request types it saw and the submit's parameters.
// sysInfo decides what the op returns: when it is empty the op fails, which
// is the case where the device's clock cannot be learned.
type boundTestServer struct {
	mu       sync.Mutex
	seen     []string
	submit   url.Values
	sysInfo  string
	opFailed bool
}

func newBoundClient(t *testing.T, srv *boundTestServer) *Client {
	t.Helper()
	return newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		srv.mu.Lock()
		defer srv.mu.Unlock()
		switch {
		case q.Get("type") == "op":
			srv.seen = append(srv.seen, "op")
			if srv.sysInfo == "" {
				srv.opFailed = true
				fmt.Fprint(w, `<response status="error" code="403"><msg><line>Permission denied</line></msg></response>`)
				return
			}
			fmt.Fprintf(w, `<response status="success"><result><system><hostname>fw-1</hostname><time>%s</time></system></result></response>`, srv.sysInfo)
		case q.Get("action") == "get":
			srv.seen = append(srv.seen, "log-get")
			fmt.Fprint(w, `<response status="success"><result><job><status>FIN</status></job><log><logs count="0"></logs></log></result></response>`)
		default:
			srv.seen = append(srv.seen, "log-submit")
			srv.submit = q
			fmt.Fprint(w, `<response status="success"><result><job>42</job></result></response>`)
		}
	})
}

// boundOf pulls the receive_time bound out of an assembled expression.
func boundOf(t *testing.T, expr string) string {
	t.Helper()
	const prefix = "(receive_time geq '"
	_, after, ok := strings.Cut(expr, prefix)
	if !ok {
		t.Fatalf("no time bound in %q", expr)
	}
	rest := after
	j := strings.Index(rest, "'")
	if j < 0 {
		t.Fatalf("unterminated time bound in %q", expr)
	}
	return rest[:j]
}

// A bound is a timestamp we write for the device to interpret in its own wall
// clock, and that offset is learned rather than reported: PAN-OS carries no
// time-zone field. Until the clock has been read the client falls back to the
// operator's zone, so a preset written then is silently shifted by the
// device's offset -- an hour's window pointing at the wrong hour, with
// nothing on screen to say so. The bound must therefore learn the clock
// first, rather than formatting against a guess.
func TestGetSystemLogs_LearnsTheDeviceClockBeforeWritingABound(t *testing.T) {
	shrinkPollTimings(t, 5, 10*time.Millisecond)

	// A device offset that cannot coincide with the test runner's own.
	_, localOffset := time.Now().Zone()
	deviceOffset := localOffset + 3*3600
	srv := &boundTestServer{
		sysInfo: time.Now().UTC().Add(time.Duration(deviceOffset) * time.Second).Format("2006/01/02 15:04:05"),
	}
	c := newBoundClient(t, srv)

	since := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	page, err := c.GetSystemLogs(context.Background(), LogQuery{Max: 10, Since: since}, "")
	if err != nil {
		t.Fatalf("GetSystemLogs: %v", err)
	}

	srv.mu.Lock()
	seen, submit := srv.seen, srv.submit
	srv.mu.Unlock()

	if len(seen) == 0 || seen[0] != "op" {
		t.Fatalf("request order = %v, want the system-info op first so the bound has a zone", seen)
	}
	// The offset is derived from a live clock read over the wire, so it can
	// land a second either side of the nominal one. What must not happen is
	// the bound following our zone: that is three hours out, not one second.
	const layout = "2006/01/02 15:04:05"
	got, err := time.ParseInLocation(layout, boundOf(t, submit.Get("query")), time.UTC)
	if err != nil {
		t.Fatalf("parsing the bound that was sent: %v", err)
	}
	want, err := time.ParseInLocation(layout, since.In(time.FixedZone("", deviceOffset)).Format(layout), time.UTC)
	if err != nil {
		t.Fatalf("parsing the expected bound: %v", err)
	}
	if diff := got.Sub(want); diff > time.Second || diff < -time.Second {
		t.Errorf("bound = %s, want %s (the device's wall clock, not ours)",
			got.Format(layout), want.Format(layout))
	}
	if page.Warning != "" {
		t.Errorf("Warning = %q, want none: the clock was learned", page.Warning)
	}
}

// The clock is learned once. A second page must not spend another op command
// re-asking a question already answered.
func TestGetSystemLogs_LearnsTheDeviceClockOnlyOnce(t *testing.T) {
	shrinkPollTimings(t, 5, 10*time.Millisecond)

	_, localOffset := time.Now().Zone()
	srv := &boundTestServer{
		sysInfo: time.Now().UTC().Add(time.Duration(localOffset+3*3600) * time.Second).Format("2006/01/02 15:04:05"),
	}
	c := newBoundClient(t, srv)

	q := LogQuery{Max: 10, Since: time.Now().Add(-time.Hour)}
	for range 2 {
		if _, err := c.GetSystemLogs(context.Background(), q, ""); err != nil {
			t.Fatalf("GetSystemLogs: %v", err)
		}
	}

	srv.mu.Lock()
	defer srv.mu.Unlock()
	var ops int
	for _, s := range srv.seen {
		if s == "op" {
			ops++
		}
	}
	if ops != 1 {
		t.Errorf("system-info ops = %d, want 1: the learned offset is cached on the client", ops)
	}
}

// When the device cannot be asked, the bound is still sent -- dropping the
// time clause would answer "the last hour" with everything, which is its own
// silent wrong answer -- but the guess is reported rather than hidden.
func TestGetSystemLogs_ReportsAGuessedZoneRatherThanHidingIt(t *testing.T) {
	shrinkPollTimings(t, 5, 10*time.Millisecond)

	srv := &boundTestServer{} // no sysInfo: the op fails
	c := newBoundClient(t, srv)

	since := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	page, err := c.GetSystemLogs(context.Background(), LogQuery{Max: 10, Since: since}, "")
	if err != nil {
		t.Fatalf("GetSystemLogs: %v", err)
	}

	srv.mu.Lock()
	failed, submit := srv.opFailed, srv.submit
	srv.mu.Unlock()

	if !failed {
		t.Fatal("the system-info op was never attempted")
	}
	if page.Warning == "" {
		t.Error("a bound written in a guessed zone was reported as if it were exact")
	}
	// The clause is still there: showing everything when the operator asked
	// for the last hour would be the other way to be silently wrong.
	if got := boundOf(t, submit.Get("query")); got == "" {
		t.Error("the time clause was dropped instead of being sent with a warning")
	}
}

// No bound, no reason to ask the device anything extra.
func TestGetSystemLogs_UnboundedQueryDoesNotLearnTheClock(t *testing.T) {
	shrinkPollTimings(t, 5, 10*time.Millisecond)

	srv := &boundTestServer{sysInfo: time.Now().UTC().Format("2006/01/02 15:04:05")}
	c := newBoundClient(t, srv)

	if _, err := c.GetSystemLogs(context.Background(), LogQuery{Max: 10}, ""); err != nil {
		t.Fatalf("GetSystemLogs: %v", err)
	}

	srv.mu.Lock()
	defer srv.mu.Unlock()
	for _, s := range srv.seen {
		if s == "op" {
			t.Error("a query with no time bound asked the device for its clock")
		}
	}
}
