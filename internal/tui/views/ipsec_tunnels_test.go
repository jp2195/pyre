package views

import (
	"testing"

	"github.com/jp2195/pyre/internal/models"
)

func TestMatchIPSecTunnel(t *testing.T) {
	tun := models.IPSecTunnel{Name: "site-b", Gateway: "gw-b", State: "up", Protocol: "ESP", Encryption: "aes-256-gcm"}
	for _, q := range []string{"site-b", "gw-b", "up", "esp", "aes-256"} {
		if !matchIPSecTunnel(tun, q) {
			t.Errorf("matchIPSecTunnel should match %q", q)
		}
	}
	if matchIPSecTunnel(tun, "nomatch") {
		t.Error("matchIPSecTunnel matched an unrelated query")
	}
}

func TestCompareIPSecTunnel(t *testing.T) {
	a := models.IPSecTunnel{Name: "a", Gateway: "gw-a", State: "down", BytesIn: 1, BytesOut: 1}
	b := models.IPSecTunnel{Name: "b", Gateway: "gw-b", State: "up", BytesIn: 100, BytesOut: 100}
	cases := []struct {
		idx  int
		want bool
	}{
		{0, true}, // Name
		{1, true}, // Gateway
		{2, true}, // State: "down" < "up"
		{3, true}, // Traffic: 2 < 200
	}
	for _, c := range cases {
		if got := compareIPSecTunnel(a, b, c.idx); got != c.want {
			t.Errorf("compareIPSecTunnel(idx=%d) = %v, want %v", c.idx, got, c.want)
		}
	}
}
