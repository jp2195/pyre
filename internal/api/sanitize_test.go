package api

import "testing"

func TestSanitizeForDisplay(t *testing.T) {
	cases := map[string]string{
		// Basic cases
		"plain":                         "plain",
		"with \x1b[31mred\x1b[0m color": "with red color",
		"nul\x00byte":                   "nulbyte",
		"bell\x07":                      "bell",

		// OSC: window-title injection terminated by BEL
		"pre \x1b]0;title\x07 post": "pre  post",
		// OSC: terminated by String Terminator (ESC '\\')
		"pre \x1b]2;t\x1b\\ post": "pre  post",
		// DCS: terminated by ST
		"pre \x1bP1$r something \x1b\\ post": "pre  post",
		// Two-byte ESC (full reset)
		"pre \x1bc post": "pre  post",

		// Truncated CSI at EOF: no hang, nothing leaks
		"text\x1b[": "text",
		// Truncated OSC at EOF: consumed as OSC, dropped entirely
		"text\x1b]0;title": "text",
	}
	for in, want := range cases {
		if got := SanitizeForDisplay(in); got != want {
			t.Errorf("SanitizeForDisplay(%q) = %q, want %q", in, got, want)
		}
	}
}
