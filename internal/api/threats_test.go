package api_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/jp2195/pyre/internal/api"
)

// The Threats panel used to be built from dataplane global counters matching
// flow_threat_*, bucketed by each counter's severity field. On a real PA-440
// running 11.2.10-h8 that is wrong twice over:
//
//   - The filtered counter command fails outright:
//     <response status="error"><msg><line>An error occurred...</line></msg></response>
//   - Asking for all global counters returns 23 entries, none of whose names
//     contain "threat", and whose severity field holds drop / info / warn.
//     Those never match critical / high / medium / low, so every severity
//     bucket was zero even where the command worked, while TotalThreats
//     accumulated raw packet counts since boot.
//
// The threat log is the actual record of threats, and it carries real
// severities and actions. These fixtures use the vocabulary that device
// returned, including the sinkhole action.
const threatLogsResponse = `<response status="success"><result>
<job><status>FIN</status></job>
<log><logs count="6">
<entry><time_generated>2026/09/04 21:00:00</time_generated><severity>critical</severity><action>reset-both</action><threatid>Bad Thing</threatid></entry>
<entry><time_generated>2026/09/04 21:00:01</time_generated><severity>high</severity><action>drop</action><threatid>Worse Thing</threatid></entry>
<entry><time_generated>2026/09/04 21:00:02</time_generated><severity>medium</severity><action>sinkhole</action><threatid>Proxy:mask.test-dns.net</threatid><threat_name>Proxy:mask.test-dns.net</threat_name></entry>
<entry><time_generated>2026/09/04 21:00:03</time_generated><severity>low</severity><action>alert</action><threatid>Noise</threatid></entry>
<entry><time_generated>2026/09/04 21:00:04</time_generated><severity>informational</severity><action>alert</action><threatid>Chatter</threatid></entry>
<entry><time_generated>2026/09/04 21:00:05</time_generated><severity>medium</severity><action>block-url</action><threatid>Blocked Site</threatid></entry>
</logs></log></result></response>`

// threatServer replays a threat-log query and records which op commands were
// issued, so a test can assert the summary no longer comes from counters.
func threatServer(t *testing.T) (*api.Client, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var ops []string

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		w.Header().Set("Content-Type", "application/xml")
		switch q.Get("type") {
		case "op":
			mu.Lock()
			ops = append(ops, q.Get("cmd"))
			mu.Unlock()
			// Mirror the real device: the counter command fails.
			_, _ = io.WriteString(w, `<response status="error"><msg><line>An error occurred. See dagger.log for information.</line></msg></response>`)
		case "log":
			if q.Get("action") == "get" {
				_, _ = io.WriteString(w, threatLogsResponse)
				return
			}
			_, _ = io.WriteString(w, `<response status="success"><result><job>1</job></result></response>`)
		default:
			_, _ = io.WriteString(w, `<response status="success"><result></result></response>`)
		}
	}))
	t.Cleanup(srv.Close)

	c, err := api.NewClient(strings.TrimPrefix(srv.URL, "https://"), "k", api.ClientOptions{Insecure: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), ops...)
	}
}

func TestGetThreatSummary_CountsBySeverity(t *testing.T) {
	client, _ := threatServer(t)

	s, err := client.GetThreatSummary(context.Background(), "")
	if err != nil {
		t.Fatalf("GetThreatSummary: %v", err)
	}

	for _, tc := range []struct {
		name string
		got  int64
		want int64
	}{
		{"total", s.TotalThreats, 6},
		{"critical", s.CriticalCount, 1},
		{"high", s.HighCount, 1},
		{"medium", s.MediumCount, 2},
		{"low+informational", s.LowCount, 2},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %d, want %d", tc.name, tc.got, tc.want)
		}
	}
}

// TestGetThreatSummary_ClassifiesActions checks the blocked/alerted split
// against the real action vocabulary. sinkhole redirects the traffic, so it
// counts as stopped rather than merely observed.
func TestGetThreatSummary_ClassifiesActions(t *testing.T) {
	client, _ := threatServer(t)

	s, err := client.GetThreatSummary(context.Background(), "")
	if err != nil {
		t.Fatalf("GetThreatSummary: %v", err)
	}
	// reset-both, drop, sinkhole, block-url stopped something; two alerts did not.
	if s.BlockedCount != 4 {
		t.Errorf("blocked = %d, want 4", s.BlockedCount)
	}
	if s.AlertedCount != 2 {
		t.Errorf("alerted = %d, want 2", s.AlertedCount)
	}
	if s.BlockedCount+s.AlertedCount != s.TotalThreats {
		t.Errorf("blocked+alerted = %d, want it to account for all %d threats",
			s.BlockedCount+s.AlertedCount, s.TotalThreats)
	}
}

// TestGetThreatSummary_DoesNotUseDataplaneCounters is the regression guard.
// The counter command fails on real hardware, so any implementation that
// still depends on it reports nothing at all.
func TestGetThreatSummary_DoesNotUseDataplaneCounters(t *testing.T) {
	client, ops := threatServer(t)

	if _, err := client.GetThreatSummary(context.Background(), ""); err != nil {
		t.Fatalf("GetThreatSummary: %v", err)
	}
	for _, cmd := range ops() {
		if strings.Contains(cmd, "counter") {
			t.Errorf("threat summary still issues a counter command: %s", cmd)
		}
	}
}

// TestGetThreatSummary_ReportsItsSampleSize keeps the panel honest. The query
// is capped, so the figures describe the most recent N threats rather than
// every threat the device has ever logged.
func TestGetThreatSummary_ReportsItsSampleSize(t *testing.T) {
	client, _ := threatServer(t)

	s, err := client.GetThreatSummary(context.Background(), "")
	if err != nil {
		t.Fatalf("GetThreatSummary: %v", err)
	}
	if s.SampleLimit <= 0 {
		t.Error("SampleLimit not reported; the view cannot say what the numbers cover")
	}
	if s.TotalThreats > int64(s.SampleLimit) {
		t.Errorf("total %d exceeds the sample limit %d", s.TotalThreats, s.SampleLimit)
	}
}

// TestGetThreatLogs_ThreatIDIsNotNumeric covers a type mismatch that broke
// the Threat tab outright on a real PA-440. PAN-OS reports threatid as a
// name, not a number: "Proxy:mask.test-dns.net", "generic:example-threat.com",
// "new:totracking.com". Modelling it as an int64 made the XML decode fail, so
// GetThreatLogs returned an error for every fetch and the view showed no
// threats at all rather than the ones the device had recorded.
func TestGetThreatLogs_ThreatIDIsNotNumeric(t *testing.T) {
	client, _ := threatServer(t)

	page, err := client.GetThreatLogs(context.Background(), api.LogQuery{Max: 10}, "")
	if err != nil {
		t.Fatalf("GetThreatLogs: %v", err)
	}
	logs := page.Entries
	if len(logs) != 6 {
		t.Fatalf("got %d entries, want 6", len(logs))
	}
	if got := logs[2].ThreatID; got != "Proxy:mask.test-dns.net" {
		t.Errorf("ThreatID = %q, want the name the device reported", got)
	}
}

// TestGetThreatLogs_ThreatNameElement covers a mapping that silently produced
// a blank column. The device sends the human-readable name in <threat_name>;
// pyre read <threat>, which does not exist in a PAN-OS threat log, so the
// Threat column and the detail pane were always empty.
func TestGetThreatLogs_ThreatNameElement(t *testing.T) {
	client, _ := threatServer(t)

	page, err := client.GetThreatLogs(context.Background(), api.LogQuery{Max: 10}, "")
	if err != nil {
		t.Fatalf("GetThreatLogs: %v", err)
	}
	logs := page.Entries
	if got := logs[2].ThreatName; got != "Proxy:mask.test-dns.net" {
		t.Errorf("ThreatName = %q, want it read from <threat_name>", got)
	}
}
