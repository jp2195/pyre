package api_test

import (
	"context"
	"testing"

	"github.com/joshuamontgomery/pyre/internal/api"
	"github.com/joshuamontgomery/pyre/internal/testutil"
)

func TestGetSystemInfo(t *testing.T) {
	mock := testutil.NewMockPANOS()
	defer mock.Close()

	client := api.NewClient(mock.Host(), "test-api-key", api.WithInsecure(true))

	info, err := client.GetSystemInfo(context.Background())
	if err != nil {
		t.Fatalf("GetSystemInfo failed: %v", err)
	}

	if info.Hostname != mock.Hostname {
		t.Errorf("expected hostname %q, got %q", mock.Hostname, info.Hostname)
	}
	if info.Model != mock.Model {
		t.Errorf("expected model %q, got %q", mock.Model, info.Model)
	}
	if info.Serial != mock.Serial {
		t.Errorf("expected serial %q, got %q", mock.Serial, info.Serial)
	}
	if info.Version != mock.Version {
		t.Errorf("expected version %q, got %q", mock.Version, info.Version)
	}
	if info.Uptime == "" {
		t.Error("expected uptime to be set")
	}
}

func TestGetSystemResources(t *testing.T) {
	mock := testutil.NewMockPANOS()
	defer mock.Close()

	client := api.NewClient(mock.Host(), "test-api-key", api.WithInsecure(true))

	res, err := client.GetSystemResources(context.Background())
	if err != nil {
		t.Fatalf("GetSystemResources failed: %v", err)
	}

	if res.CPUPercent < 0 || res.CPUPercent > 100 {
		t.Errorf("expected CPU percent between 0-100, got %f", res.CPUPercent)
	}
	if res.Load1 <= 0 {
		t.Errorf("expected Load1 > 0, got %f", res.Load1)
	}
}

func TestGetSessionInfo(t *testing.T) {
	mock := testutil.NewMockPANOS()
	defer mock.Close()

	client := api.NewClient(mock.Host(), "test-api-key", api.WithInsecure(true))

	info, err := client.GetSessionInfo(context.Background())
	if err != nil {
		t.Fatalf("GetSessionInfo failed: %v", err)
	}

	if info.ActiveCount != 15432 {
		t.Errorf("expected ActiveCount 15432, got %d", info.ActiveCount)
	}
	if info.MaxCount != 262144 {
		t.Errorf("expected MaxCount 262144, got %d", info.MaxCount)
	}
	if info.CPS != 1250 {
		t.Errorf("expected CPS 1250, got %d", info.CPS)
	}
}

func TestGetSessions(t *testing.T) {
	mock := testutil.NewMockPANOS()
	defer mock.Close()

	client := api.NewClient(mock.Host(), "test-api-key", api.WithInsecure(true))

	sessions, err := client.GetSessions(context.Background(), "")
	if err != nil {
		t.Fatalf("GetSessions failed: %v", err)
	}

	if len(sessions) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(sessions))
	}

	s := sessions[0]
	if s.ID != 12345 {
		t.Errorf("expected session ID 12345, got %d", s.ID)
	}
	if s.Application != "web-browsing" {
		t.Errorf("expected application web-browsing, got %s", s.Application)
	}
	if s.SourceIP != "192.168.1.100" {
		t.Errorf("expected source IP 192.168.1.100, got %s", s.SourceIP)
	}
	if s.DestPort != 443 {
		t.Errorf("expected dest port 443, got %d", s.DestPort)
	}
}

func TestGetSecurityPolicies(t *testing.T) {
	mock := testutil.NewMockPANOS()
	defer mock.Close()

	client := api.NewClient(mock.Host(), "test-api-key", api.WithInsecure(true))

	policies, err := client.GetSecurityPolicies(context.Background())
	if err != nil {
		t.Fatalf("GetSecurityPolicies failed: %v", err)
	}

	if len(policies) != 4 {
		t.Fatalf("expected 4 policies, got %d", len(policies))
	}

	p := policies[0]
	if p.Name != "allow-outbound" {
		t.Errorf("expected policy name allow-outbound, got %s", p.Name)
	}
	if p.Action != "allow" {
		t.Errorf("expected action allow, got %s", p.Action)
	}
	if p.Disabled {
		t.Error("expected policy to be enabled")
	}

	disabledPolicy := policies[3]
	if disabledPolicy.Name != "deprecated-rule" {
		t.Errorf("expected policy name deprecated-rule, got %s", disabledPolicy.Name)
	}
	if !disabledPolicy.Disabled {
		t.Error("expected deprecated-rule to be disabled")
	}
}

func TestGetHAStatus(t *testing.T) {
	mock := testutil.NewMockPANOS()
	defer mock.Close()

	client := api.NewClient(mock.Host(), "test-api-key", api.WithInsecure(true))

	status, err := client.GetHAStatus(context.Background())
	if err != nil {
		t.Fatalf("GetHAStatus failed: %v", err)
	}

	if !status.Enabled {
		t.Error("expected HA to be enabled")
	}
	if status.State != "active" {
		t.Errorf("expected state active, got %s", status.State)
	}
	if status.PeerState != "passive" {
		t.Errorf("expected peer state passive, got %s", status.PeerState)
	}
	if status.SyncState != "synchronized" {
		t.Errorf("expected sync state synchronized, got %s", status.SyncState)
	}
}

func TestGetInterfaces(t *testing.T) {
	mock := testutil.NewMockPANOS()
	defer mock.Close()

	client := api.NewClient(mock.Host(), "test-api-key", api.WithInsecure(true))

	interfaces, err := client.GetInterfaces(context.Background())
	if err != nil {
		t.Fatalf("GetInterfaces failed: %v", err)
	}

	if len(interfaces) != 4 {
		t.Fatalf("expected 4 interfaces, got %d", len(interfaces))
	}

	iface := interfaces[0]
	if iface.Name != "ethernet1/1" {
		t.Errorf("expected interface name ethernet1/1, got %s", iface.Name)
	}
	if iface.Zone != "untrust" {
		t.Errorf("expected zone untrust, got %s", iface.Zone)
	}
	if iface.State != "up" {
		t.Errorf("expected state up, got %s", iface.State)
	}

	downIface := interfaces[3]
	if downIface.State != "down" {
		t.Errorf("expected interface 4 state down, got %s", downIface.State)
	}
}
