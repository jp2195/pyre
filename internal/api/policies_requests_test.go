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

// A standalone firewall has no Panorama-pushed rulebases. Asked for one, it
// answers exactly this, which is what a real PA-440 returned for all three
// pre-rulebase and post-rulebase xpath spellings:
//
//	<response status="error"><msg><line>No such node</line></msg></response>
//
// The fetcher tries three candidate xpaths per rulebase location and both
// `show` and `get` for each, so a policy load spent twelve of its fourteen
// requests confirming, every single time, that a standalone box is still
// standalone. With requests now capped at four in flight, those twelve also
// queue ahead of the ones that matter.
const (
	noSuchNode = `<response status="error"><msg><line>No such node</line></msg></response>`
	// PAN-OS answers `get` on a missing node very differently from `show`:
	// success, with code 7 ("object not present") and an empty result. Both
	// responses are copied from a real PA-440. An earlier version of this
	// test assumed `get` also errored, so it passed while the device did not.
	getObjectNotPresent = `<response status="success" code="7"><result/></response>`
	localRules          = `<response status="success"><result><rules><entry name="allow-web"><action>allow</action><to><member>untrust</member></to><from><member>trust</member></from></entry></rules></result></response>`
	emptySucces         = `<response status="success"><result></result></response>`
)

// standaloneFirewall replays a standalone device and counts config requests.
func standaloneFirewall(t *testing.T) (*api.Client, func() int) {
	t.Helper()
	var mu sync.Mutex
	configRequests := 0

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		w.Header().Set("Content-Type", "application/xml")

		if q.Get("type") != "config" {
			// Hit counts and anything else.
			_, _ = io.WriteString(w, emptySucces)
			return
		}

		mu.Lock()
		configRequests++
		mu.Unlock()

		xpath := q.Get("xpath")
		switch {
		case strings.Contains(xpath, "pre-rulebase"), strings.Contains(xpath, "post-rulebase"):
			if q.Get("action") == "get" {
				_, _ = io.WriteString(w, getObjectNotPresent)
				return
			}
			_, _ = io.WriteString(w, noSuchNode)
		case strings.Contains(xpath, "/rulebase/security/rules"):
			_, _ = io.WriteString(w, localRules)
		default:
			_, _ = io.WriteString(w, noSuchNode)
		}
	}))
	t.Cleanup(srv.Close)

	c, err := api.NewClient(strings.TrimPrefix(srv.URL, "https://"), "k", api.ClientOptions{Insecure: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c, func() int {
		mu.Lock()
		defer mu.Unlock()
		return configRequests
	}
}

func TestGetSecurityPolicies_DoesNotReprobeAbsentRulebases(t *testing.T) {
	client, count := standaloneFirewall(t)
	ctx := context.Background()

	rules, err := client.GetSecurityPolicies(ctx, "")
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(rules))
	}
	first := count()
	t.Logf("first policy load issued %d config requests", first)

	// A refresh is the common case: the user presses r, or switches back to
	// the view. Having already learned this device has no Panorama
	// rulebases, it should not ask again.
	if _, err := client.GetSecurityPolicies(ctx, ""); err != nil {
		t.Fatalf("second load: %v", err)
	}
	second := count() - first
	t.Logf("second policy load issued %d config requests", second)

	if second >= first {
		t.Errorf("refresh cost %d requests after a first load of %d; absent rulebases are being re-probed every time", second, first)
	}
	if second > 2 {
		t.Errorf("refresh issued %d config requests, want at most 2 (the rulebase that actually exists)", second)
	}
}

// TestGetSecurityPolicies_StillFindsRulesAfterCaching guards the obvious way
// to break this: caching the resolved path must not stop returning rules.
func TestGetSecurityPolicies_StillFindsRulesAfterCaching(t *testing.T) {
	client, _ := standaloneFirewall(t)
	ctx := context.Background()

	for i := range 3 {
		rules, err := client.GetSecurityPolicies(ctx, "")
		if err != nil {
			t.Fatalf("load %d: %v", i, err)
		}
		if len(rules) != 1 || rules[0].Name != "allow-web" {
			t.Fatalf("load %d returned %d rules, want the one local rule", i, len(rules))
		}
	}
}

// TestGetSecurityPolicies_CacheIsPerTarget checks the cache cannot leak
// across Panorama targets: one managed device having no pre-rulebase says
// nothing about the next one.
func TestGetSecurityPolicies_CacheIsPerTarget(t *testing.T) {
	var mu sync.Mutex
	seen := map[string]int{}

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		w.Header().Set("Content-Type", "application/xml")
		if q.Get("type") != "config" {
			_, _ = io.WriteString(w, emptySucces)
			return
		}
		mu.Lock()
		seen[q.Get("target")]++
		mu.Unlock()
		if strings.Contains(q.Get("xpath"), "/rulebase/security/rules") {
			_, _ = io.WriteString(w, localRules)
			return
		}
		_, _ = io.WriteString(w, noSuchNode)
	}))
	defer srv.Close()

	client, err := api.NewClient(strings.TrimPrefix(srv.URL, "https://"), "k", api.ClientOptions{Insecure: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	ctx := context.Background()
	if _, err := client.GetSecurityPolicies(ctx, "serial-a"); err != nil {
		t.Fatalf("target a: %v", err)
	}
	if _, err := client.GetSecurityPolicies(ctx, "serial-b"); err != nil {
		t.Fatalf("target b: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if seen["serial-b"] == 0 {
		t.Error("second target reused the first target's resolved paths")
	}
}
