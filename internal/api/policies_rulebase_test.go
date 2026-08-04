package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/jp2195/pyre/internal/api"
	"github.com/jp2195/pyre/internal/models"
	"github.com/jp2195/pyre/internal/testutil"
)

func policiesTestClient(t *testing.T) *api.Client {
	t.Helper()
	mock := testutil.NewMockPANOS()
	t.Cleanup(mock.Close)
	c, err := api.NewClient(mock.Host(), "test-key", api.ClientOptions{Insecure: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestGetSecurityPolicies_ParsesRulesInOrder(t *testing.T) {
	c := policiesTestClient(t)

	rules, err := c.GetSecurityPolicies(context.Background(), "")
	if err != nil {
		t.Fatalf("GetSecurityPolicies: %v", err)
	}
	if len(rules) != 4 {
		t.Fatalf("got %d rules, want 4", len(rules))
	}

	wantOrder := []string{"allow-outbound", "allow-dns", "deny-all", "deprecated-rule"}
	for i, name := range wantOrder {
		if rules[i].Name != name {
			t.Errorf("rules[%d].Name = %q, want %q", i, rules[i].Name, name)
		}
		if rules[i].Position != i+1 {
			t.Errorf("rules[%d].Position = %d, want %d", i, rules[i].Position, i+1)
		}
		if rules[i].RuleBase != models.RuleBaseLocal {
			t.Errorf("rules[%d].RuleBase = %v, want RuleBaseLocal", i, rules[i].RuleBase)
		}
	}

	if rules[0].Action != "allow" || rules[2].Action != "deny" {
		t.Errorf("actions = %q/%q, want allow/deny", rules[0].Action, rules[2].Action)
	}
	if !rules[3].Disabled {
		t.Error("deprecated-rule should be Disabled")
	}
	if len(rules[0].Applications) != 2 || rules[0].Applications[0] != "web-browsing" {
		t.Errorf("rules[0].Applications = %v, want [web-browsing ssl]", rules[0].Applications)
	}
	if !rules[0].LogEnd || rules[3].LogEnd {
		t.Errorf("LogEnd flags = %v/%v, want true/false", rules[0].LogEnd, rules[3].LogEnd)
	}
}

func TestGetSecurityPolicies_AppliesHitCounts(t *testing.T) {
	c := policiesTestClient(t)

	rules, err := c.GetSecurityPolicies(context.Background(), "")
	if err != nil {
		t.Fatalf("GetSecurityPolicies: %v", err)
	}

	wantHits := map[string]struct {
		count   int64
		lastHit time.Time
	}{
		"allow-outbound":  {1543289, time.Unix(1737456000, 0)},
		"allow-dns":       {892341, time.Unix(1737455900, 0)},
		"deny-all":        {12453, time.Unix(1737455800, 0)},
		"deprecated-rule": {0, time.Time{}},
	}
	for _, r := range rules {
		want, ok := wantHits[r.Name]
		if !ok {
			t.Errorf("unexpected rule %q", r.Name)
			continue
		}
		if r.HitCount != want.count {
			t.Errorf("%s HitCount = %d, want %d", r.Name, r.HitCount, want.count)
		}
		if !r.LastHit.Equal(want.lastHit) {
			t.Errorf("%s LastHit = %v, want %v", r.Name, r.LastHit, want.lastHit)
		}
	}
}

func TestGetNATRules_ParsesSourceTranslation(t *testing.T) {
	c := policiesTestClient(t)

	rules, err := c.GetNATRules(context.Background(), "")
	if err != nil {
		t.Fatalf("GetNATRules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(rules))
	}

	r := rules[0]
	if r.Name != "outbound-nat" || r.Position != 1 || r.RuleBase != models.RuleBaseLocal {
		t.Errorf("rule = %q pos=%d base=%v, want outbound-nat pos=1 local", r.Name, r.Position, r.RuleBase)
	}
	if r.SourceTransType != models.SourceTransDynamicIPPort {
		t.Errorf("SourceTransType = %v, want SourceTransDynamicIPPort", r.SourceTransType)
	}
	if r.TranslatedSource != "ethernet1/1" || !r.SourceInterfaceIP {
		t.Errorf("TranslatedSource = %q (interfaceIP=%v), want ethernet1/1 (true)", r.TranslatedSource, r.SourceInterfaceIP)
	}
	if len(r.Services) != 1 || r.Services[0] != "any" {
		t.Errorf("Services = %v, want [any]", r.Services)
	}
	// The mock's hit-count op response only contains security rules, so NAT
	// rules keep zero hit stats — this also pins the lenient behavior when
	// hit data has no matching rule names.
	if r.HitCount != 0 {
		t.Errorf("HitCount = %d, want 0", r.HitCount)
	}
}
