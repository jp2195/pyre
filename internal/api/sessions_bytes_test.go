package api_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jp2195/pyre/internal/api"
)

// `show session all` reports one figure per session, total-byte-count. That
// value was assigned to a field named BytesIn and rendered as "Bytes In",
// with "Bytes Out" showing 0 beside it. An operator reading that pane saw a
// precise-looking directional split that the device never reported: every
// session appeared to have received everything and sent nothing.
const sessionsResponse = `<response status="success"><result>
<entry>
<idx>62870</idx><vsys>vsys1</vsys><application>web-browsing</application>
<state>ACTIVE</state><type>FLOW</type>
<source>10.0.0.5</source><sport>59940</sport>
<dst>198.51.100.20</dst><dport>443</dport>
<from>trust</from><to>untrust</to>
<xsource>203.0.113.7</xsource><xsport>22047</xsport>
<proto>6</proto><security-rule>allow-outbound</security-rule>
<start-time>Mon Jan 06 10:00:00 2025</start-time>
<total-byte-count>8877</total-byte-count>
</entry>
</result></response>`

func TestGetSessions_ReportsATotalNotADirectionalSplit(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, sessionsResponse)
	}))
	defer srv.Close()

	client, err := api.NewClient(strings.TrimPrefix(srv.URL, "https://"), "k", api.ClientOptions{Insecure: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	sessions, err := client.GetSessions(context.Background(), "", "")
	if err != nil {
		t.Fatalf("GetSessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	s := sessions[0]

	if s.TotalBytes != 8877 {
		t.Errorf("TotalBytes = %d, want 8877 (the one figure the device reports)", s.TotalBytes)
	}
	if s.ID != 62870 {
		t.Errorf("ID = %d, want 62870", s.ID)
	}
	if s.Protocol != "tcp" {
		t.Errorf("Protocol = %q, want tcp", s.Protocol)
	}
}
