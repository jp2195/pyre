package api

import (
	"testing"
)

func TestParsePANTime(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"slash layout", "2026/04/18 12:30:45", false},
		{"dash layout", "2026-04-18 12:30:45", false},
		{"ansic-ish single-digit day", "Sat Apr 18 12:30:45 2026", false},
		{"ansic-ish two-digit day", "Sat Apr 02 12:30:45 2026", false},
		{"US slash layout", "04/18/2026 12:30:45", false},
		{"cert layout with TZ", "Apr 18 12:30:45 2026 UTC", false},
		{"license expiration layout", "April 18, 2026", false},

		{"empty string", "", true},
		{"garbage", "not a timestamp", true},
		{"wrong separator", "2026.04.18 12:30:45", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parsePANTime(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Errorf("parsePANTime(%q) = %v, want error", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Errorf("parsePANTime(%q) unexpected error: %v", tc.in, err)
			}
			if got.IsZero() {
				t.Errorf("parsePANTime(%q) returned zero time", tc.in)
			}
		})
	}
}
