package api_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jp2195/pyre/internal/api"
)

// newCountingServer returns a TLS test server that records the high-water
// mark of concurrently in-flight requests.
func newCountingServer(t *testing.T, hold time.Duration) (*httptest.Server, func() int) {
	t.Helper()
	var mu sync.Mutex
	var inFlight, peak int

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		inFlight++
		if inFlight > peak {
			peak = inFlight
		}
		mu.Unlock()

		// Hold the request open so overlap is observable.
		time.Sleep(hold)

		mu.Lock()
		inFlight--
		mu.Unlock()

		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, `<response status="success"><result></result></response>`)
	}))
	return srv, func() int {
		mu.Lock()
		defer mu.Unlock()
		return peak
	}
}

// TestClient_LimitsConcurrentRequests pins the request footprint against a
// single firewall. A dashboard load fans out about sixteen fetches at once
// and a policy load issues roughly fifteen config calls, all against a
// management plane that is slow and shared with the web UI. Every one is
// recorded in the firewall's own log, so a burst is both a performance
// problem and a noisy audit trail.
func TestClient_LimitsConcurrentRequests(t *testing.T) {
	srv, peak := newCountingServer(t, 20*time.Millisecond)
	defer srv.Close()

	client, err := api.NewClient(strings.TrimPrefix(srv.URL, "https://"), "k", api.ClientOptions{Insecure: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	var wg sync.WaitGroup
	for range 24 {
		wg.Go(func() {
			_, _ = client.Op(context.Background(), "<show><system><info></info></system></show>", "")
		})
	}
	wg.Wait()

	if got := peak(); got > api.MaxConcurrentRequests {
		t.Errorf("peak concurrent requests = %d, want at most %d", got, api.MaxConcurrentRequests)
	}
	// Guard against the cap being satisfied by accident (a serialized client
	// would also pass the check above but would make the TUI crawl).
	if got := peak(); got < 2 {
		t.Errorf("peak concurrent requests = %d; requests are not overlapping at all", got)
	}
}

// TestClient_CancellationIsHonoredWhileQueued checks that a request waiting
// on the concurrency cap still respects its context, so a view switch does
// not have to wait out a queue of the previous device's requests.
func TestClient_CancellationIsHonoredWhileQueued(t *testing.T) {
	srv, _ := newCountingServer(t, 200*time.Millisecond)
	defer srv.Close()

	client, err := api.NewClient(strings.TrimPrefix(srv.URL, "https://"), "k", api.ClientOptions{Insecure: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// Saturate the cap.
	var wg sync.WaitGroup
	for range api.MaxConcurrentRequests {
		wg.Go(func() {
			_, _ = client.Op(context.Background(), "<show/>", "")
		})
	}

	// This one has to queue; cancel it while it waits.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.Op(ctx, "<show/>", ""); err == nil {
		t.Error("expected a canceled queued request to return an error")
	}
	wg.Wait()
}

// TestClient_SendsUserAgent checks pyre identifies itself. Every call lands
// in the firewall's log, and an unidentified client is a worse answer to
// "what made these API calls?" than it needs to be.
func TestClient_SendsUserAgent(t *testing.T) {
	var got string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
		_, _ = io.WriteString(w, `<response status="success"><result></result></response>`)
	}))
	defer srv.Close()

	client, err := api.NewClient(strings.TrimPrefix(srv.URL, "https://"), "k", api.ClientOptions{Insecure: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := client.Op(context.Background(), "<show/>", ""); err != nil {
		t.Fatalf("Op: %v", err)
	}

	if !strings.HasPrefix(got, "pyre") {
		t.Errorf("User-Agent = %q, want it to identify pyre", got)
	}
}

// TestNewTransport_UsesEnvironmentProxy checks outbound calls can be routed
// through a corporate or inspection proxy, which every other Go HTTP client
// on the machine already honors.
func TestNewTransport_UsesEnvironmentProxy(t *testing.T) {
	tr, err := api.NewTransport(api.ClientOptions{})
	if err != nil {
		t.Fatalf("NewTransport: %v", err)
	}
	if tr.Proxy == nil {
		t.Error("transport ignores HTTPS_PROXY / HTTP_PROXY")
	}
}
