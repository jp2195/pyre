package views

import (
	"testing"
	"time"

	"github.com/jp2195/pyre/internal/models"
)

func TestMatchGPUser(t *testing.T) {
	u := models.GlobalProtectUser{Username: "Alice", Domain: "corp", Gateway: "gw-east", ClientIP: "203.0.113.9"}
	for _, q := range []string{"alice", "corp", "gw-east", "203.0.113"} {
		if !matchGPUser(u, q) {
			t.Errorf("matchGPUser should match %q", q)
		}
	}
	if matchGPUser(u, "nomatch") {
		t.Error("matchGPUser matched an unrelated query")
	}
}

func TestCompareGPUser(t *testing.T) {
	early := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	late := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	a := models.GlobalProtectUser{Username: "alice", Gateway: "gw-a", LoginTime: early, Duration: "1h"}
	b := models.GlobalProtectUser{Username: "bob", Gateway: "gw-b", LoginTime: late, Duration: "2h"}
	cases := []struct {
		idx  int
		want bool
	}{
		{0, true}, // Username: alice < bob
		{1, true}, // Gateway
		{2, true}, // LoginTime
		{3, true}, // Duration
	}
	for _, c := range cases {
		if got := compareGPUser(a, b, c.idx); got != c.want {
			t.Errorf("compareGPUser(idx=%d) = %v, want %v", c.idx, got, c.want)
		}
	}
}
