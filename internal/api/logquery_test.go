package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// The device interprets a bound in its own wall clock (fact 5), so the bound
// must be formatted in the device's zone and never ours. deviceLoc is pinned
// to a zone that cannot match the test runner's.
func TestBuildLogQuery_FormatsBoundInDeviceZone(t *testing.T) {
	c := &Client{}
	c.deviceLoc.Store(time.FixedZone("", 5*3600+30*60)) // +05:30

	// 2026-09-05T00:00:00Z is 05:30 on the 5th in the device's zone.
	since := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	got := c.buildLogQuery(LogQuery{Since: since})
	want := "(receive_time geq '2026/09/05 05:30:00')"
	if got != want {
		t.Errorf("buildLogQuery() = %q, want %q", got, want)
	}
}

func TestBuildLogQuery_Assembly(t *testing.T) {
	c := &Client{}
	c.deviceLoc.Store(time.UTC)
	since := time.Date(2026, 9, 5, 14, 30, 0, 0, time.UTC)

	tests := []struct {
		name string
		q    LogQuery
		want string
	}{
		{
			name: "empty query sends nothing",
			q:    LogQuery{},
			want: "",
		},
		{
			name: "bound only",
			q:    LogQuery{Since: since},
			want: "(receive_time geq '2026/09/05 14:30:00')",
		},
		{
			name: "user clause only, wrapped",
			q:    LogQuery{Query: "addr.src in 203.0.113.5"},
			want: "(addr.src in 203.0.113.5)",
		},
		{
			name: "bound and clause joined with and",
			q:    LogQuery{Since: since, Query: "addr.src in 203.0.113.5"},
			want: "(receive_time geq '2026/09/05 14:30:00') and (addr.src in 203.0.113.5)",
		},
		{
			name: "a top-level or cannot escape the bound",
			q:    LogQuery{Since: since, Query: "(action eq allow) or (action eq deny)"},
			want: "(receive_time geq '2026/09/05 14:30:00') and ((action eq allow) or (action eq deny))",
		},
		{
			name: "surrounding whitespace is trimmed",
			q:    LogQuery{Query: "   addr.src in 203.0.113.5   "},
			want: "(addr.src in 203.0.113.5)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := c.buildLogQuery(tt.q); got != tt.want {
				t.Errorf("buildLogQuery() = %q, want %q", got, tt.want)
			}
		})
	}
}

// The device ceiling is exactly 5000 and 5001 is an HTTP 400 (fact 3), so an
// over-large Max must never leave the process.
func TestLogQuery_Normalized(t *testing.T) {
	tests := []struct {
		name              string
		in                LogQuery
		wantMax, wantSkip int
	}{
		{"zero max becomes the default page", LogQuery{}, defaultLogRows, 0},
		{"negative max becomes the default page", LogQuery{Max: -5}, defaultLogRows, 0},
		{"max above the ceiling clamps", LogQuery{Max: 9999}, maxLogRows, 0},
		{"max at the ceiling is kept", LogQuery{Max: maxLogRows}, maxLogRows, 0},
		{"one row is legal", LogQuery{Max: 1}, 1, 0},
		{"negative skip becomes zero", LogQuery{Max: 10, Skip: -3}, 10, 0},
		{"positive skip is kept", LogQuery{Max: 10, Skip: 40}, 10, 40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in.normalized()
			if got.Max != tt.wantMax || got.Skip != tt.wantSkip {
				t.Errorf("normalized() = {Max:%d Skip:%d}, want {Max:%d Skip:%d}",
					got.Max, got.Skip, tt.wantMax, tt.wantSkip)
			}
		})
	}
}

// logPageServer answers a submit with a job id and the poll with rowCount
// synthetic system rows. It records the submit's query parameters.
func logPageServer(t *testing.T, rowCount int, gotParams *url.Values) *Client {
	t.Helper()
	return newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("action") == "get" {
			var rows strings.Builder
			for i := range rowCount {
				fmt.Fprintf(&rows,
					`<entry><time_generated>2026/09/05 14:%02d:00</time_generated><type>SYSTEM</type><severity>informational</severity><opaque>row %d</opaque></entry>`,
					i%60, i)
			}
			fmt.Fprintf(w, `<response status="success"><result><job><status>FIN</status></job><log><logs count="%d">%s</logs></log></result></response>`,
				rowCount, rows.String())
			return
		}
		*gotParams = r.URL.Query()
		fmt.Fprint(w, `<response status="success"><result><job>42</job></result></response>`)
	})
}

func TestGetSystemLogs_SendsClampedParamsAndReportsHasMore(t *testing.T) {
	shrinkPollTimings(t, 5, 10*time.Millisecond)

	tests := []struct {
		name        string
		q           LogQuery
		rowCount    int
		wantNlogs   string
		wantSkip    string
		wantHasMore bool
	}{
		{
			name:        "full page reports more available",
			q:           LogQuery{Max: 10},
			rowCount:    10,
			wantNlogs:   "10",
			wantSkip:    "",
			wantHasMore: true,
		},
		{
			name:        "short page is the end",
			q:           LogQuery{Max: 10},
			rowCount:    4,
			wantNlogs:   "10",
			wantSkip:    "",
			wantHasMore: false,
		},
		{
			name:        "empty page is the end",
			q:           LogQuery{Max: 10},
			rowCount:    0,
			wantNlogs:   "10",
			wantSkip:    "",
			wantHasMore: false,
		},
		{
			name:        "over-large max is clamped before it is sent",
			q:           LogQuery{Max: 9999},
			rowCount:    3,
			wantNlogs:   "5000",
			wantSkip:    "",
			wantHasMore: false,
		},
		{
			name:        "skip is passed through",
			q:           LogQuery{Max: 10, Skip: 20},
			rowCount:    2,
			wantNlogs:   "10",
			wantSkip:    "20",
			wantHasMore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var params url.Values
			c := logPageServer(t, tt.rowCount, &params)

			page, err := c.GetSystemLogs(context.Background(), tt.q, "")
			if err != nil {
				t.Fatalf("GetSystemLogs: %v", err)
			}
			if len(page.Entries) != tt.rowCount {
				t.Errorf("got %d entries, want %d", len(page.Entries), tt.rowCount)
			}
			if page.HasMore != tt.wantHasMore {
				t.Errorf("HasMore = %v, want %v", page.HasMore, tt.wantHasMore)
			}
			if got := params.Get("nlogs"); got != tt.wantNlogs {
				t.Errorf("nlogs = %q, want %q", got, tt.wantNlogs)
			}
			if got := params.Get("skip"); got != tt.wantSkip {
				t.Errorf("skip = %q, want %q", got, tt.wantSkip)
			}
		})
	}
}

// An empty LogQuery must send no query parameter at all, so the newest rows
// come back exactly as they did before this feature existed.
func TestGetSystemLogs_EmptyQuerySendsNoQueryParam(t *testing.T) {
	shrinkPollTimings(t, 5, 10*time.Millisecond)
	var params url.Values
	c := logPageServer(t, 2, &params)

	page, err := c.GetSystemLogs(context.Background(), LogQuery{Max: 10}, "")
	if err != nil {
		t.Fatalf("GetSystemLogs: %v", err)
	}
	if _, ok := params["query"]; ok {
		t.Errorf("query param was sent for an empty LogQuery: %q", params.Get("query"))
	}
	if page.Query != "" {
		t.Errorf("page.Query = %q, want empty", page.Query)
	}
}

// The page reports the expression that was actually sent, which is what the
// view shows when a query matches nothing.
func TestGetSystemLogs_PageReportsAssembledQuery(t *testing.T) {
	shrinkPollTimings(t, 5, 10*time.Millisecond)
	var params url.Values
	c := logPageServer(t, 1, &params)
	c.deviceLoc.Store(time.UTC)

	q := LogQuery{
		Max:   10,
		Since: time.Date(2026, 9, 5, 14, 30, 0, 0, time.UTC),
		Query: "addr.src in 203.0.113.5",
	}
	want := "(receive_time geq '2026/09/05 14:30:00') and (addr.src in 203.0.113.5)"

	page, err := c.GetSystemLogs(context.Background(), q, "")
	if err != nil {
		t.Fatalf("GetSystemLogs: %v", err)
	}
	if page.Query != want {
		t.Errorf("page.Query = %q, want %q", page.Query, want)
	}
	if got := params.Get("query"); got != want {
		t.Errorf("sent query = %q, want %q", got, want)
	}
}
