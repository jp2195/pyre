package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// makeResp builds an *XMLResponse whose Result.Inner matches the given body.
// The body is the inner XML of <result>...</result> (so callers pass e.g.
// "<job><status>FIN</status></job>").
func makeResp(inner string) *XMLResponse {
	r := &XMLResponse{Status: "success"}
	r.Result.Inner = []byte(inner)
	return r
}

func TestClassifyJobStatus(t *testing.T) {
	cases := []struct {
		name     string
		resp     *XMLResponse
		wantKind logJobStatus
		wantRaw  string
	}{
		{"nil response", nil, logJobRunning, ""},
		{"empty inner", makeResp(""), logJobRunning, ""},
		{"missing job element", makeResp("<other>x</other>"), logJobRunning, ""},
		{"FIN -> done", makeResp("<job><status>FIN</status></job>"), logJobDone, "FIN"},
		{"DONE -> done", makeResp("<job><status>DONE</status></job>"), logJobDone, "DONE"},
		{"FAIL -> failed", makeResp("<job><status>FAIL</status></job>"), logJobFailed, "FAIL"},
		{"CANC -> failed", makeResp("<job><status>CANC</status></job>"), logJobFailed, "CANC"},
		{"ACT -> running", makeResp("<job><status>ACT</status></job>"), logJobRunning, "ACT"},
		{"PEND -> running", makeResp("<job><status>PEND</status></job>"), logJobRunning, "PEND"},
		{"lowercase fin -> done (ToUpper)", makeResp("<job><status>fin</status></job>"), logJobDone, "fin"},
		{"malformed XML -> running", makeResp("<job><status>FIN"), logJobRunning, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotKind, gotRaw := classifyJobStatus(tc.resp)
			if gotKind != tc.wantKind {
				t.Errorf("kind = %d, want %d", gotKind, tc.wantKind)
			}
			if gotRaw != tc.wantRaw {
				t.Errorf("raw = %q, want %q", gotRaw, tc.wantRaw)
			}
		})
	}
}

// shrinkPollTimings shrinks the package-level poll knobs for the duration of
// a test and restores them on cleanup. Caller chooses attempts/interval.
func shrinkPollTimings(t *testing.T, attempts int, interval time.Duration) {
	t.Helper()
	origAttempts := logPollMaxAttempts
	origInterval := logPollInterval
	logPollMaxAttempts = attempts
	logPollInterval = interval
	t.Cleanup(func() {
		logPollMaxAttempts = origAttempts
		logPollInterval = origInterval
	})
}

// newTestClient spins up an https test server running handler and returns a
// Client pointed at it. The Close is registered with t.Cleanup.
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewTLSServer(handler)
	t.Cleanup(srv.Close)
	host := strings.TrimPrefix(srv.URL, "https://")
	c, err := NewClient(host, "K", ClientOptions{Insecure: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestPollLogJob_ReturnsSuccessAfterOnePoll(t *testing.T) {
	shrinkPollTimings(t, 5, 10*time.Millisecond)
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<response status="success"><result><job><status>FIN</status></job></result></response>`)
	})

	resp, err := c.pollLogJob(context.Background(), "42", "")
	if err != nil {
		t.Fatalf("pollLogJob: unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if !strings.Contains(string(resp.Result.Inner), "<status>FIN</status>") {
		t.Errorf("unexpected inner: %s", resp.Result.Inner)
	}
}

func TestPollLogJob_ReturnsFailureOnFAIL(t *testing.T) {
	shrinkPollTimings(t, 5, 10*time.Millisecond)
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<response status="success"><result><job><status>FAIL</status></job></result></response>`)
	})

	resp, err := c.pollLogJob(context.Background(), "99", "")
	if err == nil {
		t.Fatalf("expected error, got resp=%v", resp)
	}
	if !strings.Contains(err.Error(), "reported failure") {
		t.Errorf("error = %v, want contains \"reported failure\"", err)
	}
}

func TestPollLogJob_TimesOutAfterMaxAttempts(t *testing.T) {
	shrinkPollTimings(t, 3, 5*time.Millisecond)
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<response status="success"><result><job><status>ACT</status></job></result></response>`)
	})

	resp, err := c.pollLogJob(context.Background(), "1", "")
	if err == nil {
		t.Fatalf("expected timeout error, got resp=%v", resp)
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error = %v, want contains \"timed out\"", err)
	}
}

func TestPollLogJob_RetriesTransportErrors(t *testing.T) {
	shrinkPollTimings(t, 10, 5*time.Millisecond)
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			// Non-XML body ensures request() returns a parse error.
			fmt.Fprint(w, "upstream unavailable")
			return
		}
		fmt.Fprint(w, `<response status="success"><result><job><status>FIN</status></job></result></response>`)
	})

	resp, err := c.pollLogJob(context.Background(), "7", "")
	if err != nil {
		t.Fatalf("pollLogJob: unexpected error after retries: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if got := calls.Load(); got < 3 {
		t.Errorf("expected >=3 server calls, got %d", got)
	}
}

func TestPollLogJob_FailsAfterThreeConsecutiveErrors(t *testing.T) {
	shrinkPollTimings(t, 10, 5*time.Millisecond)
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprint(w, "gateway error")
	})

	resp, err := c.pollLogJob(context.Background(), "11", "")
	if err == nil {
		t.Fatalf("expected error, got resp=%v", resp)
	}
	if !strings.Contains(err.Error(), "consecutive errors") {
		t.Errorf("error = %v, want contains \"consecutive errors\"", err)
	}
}

func TestPollLogJob_FirstAttemptIsImmediate(t *testing.T) {
	// Use a deliberately long interval. If the loop sleeps before the first
	// poll, this test will exceed its budget.
	shrinkPollTimings(t, 30, 2*time.Second)
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		fmt.Fprint(w, `<response status="success"><result><job><status>FIN</status></job></result></response>`)
	})

	start := time.Now()
	resp, err := c.pollLogJob(context.Background(), "1234", "")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("pollLogJob returned err: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil resp")
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("first poll should be immediate; elapsed=%v (interval=2s)", elapsed)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("expected 1 LogGet call on first-attempt success, got %d", got)
	}
}

func TestPollLogJob_RespectsContextCancellation(t *testing.T) {
	// Long per-poll interval would ordinarily dominate the loop; cancellation
	// must interrupt the select immediately.
	shrinkPollTimings(t, 30, 500*time.Millisecond)
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<response status="success"><result><job><status>ACT</status></job></result></response>`)
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	resp, err := c.pollLogJob(ctx, "5", "")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("expected cancellation error, got resp=%v", resp)
	}
	if err != context.Canceled {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	// Should return well before attempt exhaustion (30 * 500ms = 15s).
	if elapsed > 3*time.Second {
		t.Errorf("cancellation took %v, expected <3s", elapsed)
	}
}
