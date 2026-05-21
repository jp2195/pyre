package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jp2195/pyre/internal/testutil"
)

func TestGetAddresses_ParsesAllTypes(t *testing.T) {
	mock := testutil.NewMockPANOS()
	defer mock.Close()

	client, err := NewClient(mock.Host(), "test-key", ClientOptions{Insecure: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := client.GetAddresses(context.Background(), "")
	if err != nil {
		t.Fatalf("GetAddresses: %v", err)
	}

	want := map[string]struct{ typ, value string }{
		"web-servers":      {"ip-netmask", "10.0.0.0/24"},
		"azure-east-range": {"ip-range", "52.224.0.1-52.255.255.255"},
		"partner-vpn":      {"fqdn", "vpn.partner.example.com"},
		"legacy-wildcard":  {"ip-wildcard", "10.0.0.0/0.0.255.255"},
	}

	if len(got) < len(want) {
		t.Fatalf("got %d address objects, want at least %d", len(got), len(want))
	}

	byName := make(map[string]int)
	for i, a := range got {
		byName[a.Name] = i
	}
	for name, expect := range want {
		idx, ok := byName[name]
		if !ok {
			t.Errorf("missing address object %q in result", name)
			continue
		}
		a := got[idx]
		if a.Type != expect.typ {
			t.Errorf("%s: Type=%q want %q", name, a.Type, expect.typ)
		}
		if a.Value != expect.value {
			t.Errorf("%s: Value=%q want %q", name, a.Value, expect.value)
		}
	}
}

func TestGetAddresses_EmptySharedScope(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		xpath := r.URL.Query().Get("xpath")
		if strings.HasSuffix(xpath, "/vsys/entry[@name='vsys1']/address") {
			_, _ = w.Write([]byte(`<response status="success"><result><address><entry name="only-vsys"><ip-netmask>1.2.3.0/24</ip-netmask></entry></address></result></response>`))
			return
		}
		// Everything else (including shared) returns empty success.
		_, _ = w.Write([]byte(`<response status="success"><result></result></response>`))
	}))
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "https://")
	client, err := NewClient(host, "test-key", ClientOptions{Insecure: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := client.GetAddresses(context.Background(), "")
	if err != nil {
		t.Fatalf("GetAddresses: %v", err)
	}
	if len(got) != 1 || got[0].Name != "only-vsys" {
		t.Fatalf("expected 1 object 'only-vsys', got %+v", got)
	}
}

func TestGetAddresses_MalformedEntry_Skipped(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		xpath := r.URL.Query().Get("xpath")
		if strings.HasSuffix(xpath, "/vsys/entry[@name='vsys1']/address") {
			_, _ = w.Write([]byte(`<response status="success"><result><address>
<entry name="good"><ip-netmask>10.0.0.0/24</ip-netmask></entry>
<entry name="orphan"></entry>
<entry name="also-good"><fqdn>example.com</fqdn></entry>
</address></result></response>`))
			return
		}
		_, _ = w.Write([]byte(`<response status="success"><result></result></response>`))
	}))
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "https://")
	client, err := NewClient(host, "test-key", ClientOptions{Insecure: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := client.GetAddresses(context.Background(), "")
	if err != nil {
		t.Fatalf("GetAddresses: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 valid objects (orphan skipped), got %d: %+v", len(got), got)
	}
	for _, a := range got {
		if a.Name == "orphan" {
			t.Errorf("orphan entry should have been skipped, got %+v", a)
		}
	}
}

func TestGetAddresses_PropagatesAPIError(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "https://")
	client, err := NewClient(host, "test-key", ClientOptions{Insecure: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.GetAddresses(context.Background(), "")
	if err == nil {
		t.Fatal("expected error from 500 response, got nil")
	}
}

func TestGetServices_TCPAndUDP_PortRange(t *testing.T) {
	mock := testutil.NewMockPANOS()
	defer mock.Close()

	client, err := NewClient(mock.Host(), "test-key", ClientOptions{Insecure: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := client.GetServices(context.Background(), "")
	if err != nil {
		t.Fatalf("GetServices: %v", err)
	}

	want := map[string]struct {
		proto, dest, src string
	}{
		"tcp-443":   {"tcp", "443", "1024-65535"},
		"tcp-mssql": {"tcp", "1433,1434", ""},
		"udp-dns":   {"udp", "53", ""},
	}

	if len(got) < len(want) {
		t.Fatalf("got %d service objects, want at least %d", len(got), len(want))
	}

	byName := make(map[string]int)
	for i, s := range got {
		byName[s.Name] = i
	}
	for name, expect := range want {
		idx, ok := byName[name]
		if !ok {
			t.Errorf("missing service object %q", name)
			continue
		}
		s := got[idx]
		if s.Protocol != expect.proto {
			t.Errorf("%s: Protocol=%q want %q", name, s.Protocol, expect.proto)
		}
		if s.DestPort != expect.dest {
			t.Errorf("%s: DestPort=%q want %q", name, s.DestPort, expect.dest)
		}
		if s.SrcPort != expect.src {
			t.Errorf("%s: SrcPort=%q want %q", name, s.SrcPort, expect.src)
		}
	}
}
