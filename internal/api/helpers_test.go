package api

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"zero", 0, "0m"},
		{"sub-hour", 45 * time.Minute, "45m"},
		{"exactly one hour", time.Hour, "1h 0m"},
		{"hours and minutes", 3*time.Hour + 15*time.Minute, "3h 15m"},
		{"just under 24h", 23*time.Hour + 59*time.Minute, "23h 59m"},
		// The boundary case: exactly 24h should render as "1d 0h", not "24h 0m".
		{"exactly 24h", 24 * time.Hour, "1d 0h"},
		{"just over 24h", 25 * time.Hour, "1d 1h"},
		{"multi-day", 49*time.Hour + 30*time.Minute, "2d 1h"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatDuration(tc.d)
			if got != tc.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tc.d, got, tc.want)
			}
		})
	}
}

func TestProtoToName(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"6", "tcp"},
		{"17", "udp"},
		{"1", "icmp"},
		{"", "tcp"}, // default
		{"99", "99"}, // unknown passes through
	}
	for _, tc := range tests {
		if got := protoToName(tc.in); got != tc.want {
			t.Errorf("protoToName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
