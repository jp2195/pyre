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
		{"", ""},     // absent stays absent rather than being reported as TCP
		{"99", "99"}, // unknown passes through
	}
	for _, tc := range tests {
		if got := protoToName(tc.in); got != tc.want {
			t.Errorf("protoToName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestProtoToName_DoesNotInventTCP covers a small piece of fabricated data.
// An absent protocol was rendered as "tcp", so a session or log entry whose
// protocol the device did not report looked like a definite TCP flow. In a
// tool whose job is to report what the firewall actually saw, an unknown
// value has to look unknown.
func TestProtoToName_DoesNotInventTCP(t *testing.T) {
	if got := protoToName(""); got == "tcp" {
		t.Errorf("protoToName(\"\") = %q; an absent protocol must not be reported as TCP", got)
	}
	// Known mappings must still work.
	for in, want := range map[string]string{
		"6": "tcp", "17": "udp", "1": "icmp", "58": "icmp6",
		// PAN-OS logs already send names; those pass through.
		"tcp": "tcp", "udp": "udp",
		// An unrecognized number is reported as given rather than guessed at.
		"253": "253",
	} {
		if got := protoToName(in); got != want {
			t.Errorf("protoToName(%q) = %q, want %q", in, got, want)
		}
	}
}
