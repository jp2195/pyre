package api

import (
	"testing"
	"time"
)

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

func TestSanitizeAllStrings_WalksNestedStructures(t *testing.T) {
	type inner struct {
		Name string
		Num  int
	}
	type outer struct {
		Title string
		Items []inner
		Ptr   *inner
	}

	v := outer{
		Title: "a\x1b[31mred\x1b[0mb",
		Items: []inner{{Name: "x\x1b]0;evil\x07y", Num: 7}},
		Ptr:   &inner{Name: "p\x1b[2Jq"},
	}

	sanitizeAllStrings(&v)

	if v.Title != "aredb" {
		t.Errorf("Title = %q, want %q", v.Title, "aredb")
	}
	if v.Items[0].Name != "xy" {
		t.Errorf("Items[0].Name = %q, want %q", v.Items[0].Name, "xy")
	}
	if v.Items[0].Num != 7 {
		t.Errorf("Num clobbered: %d", v.Items[0].Num)
	}
	if v.Ptr.Name != "pq" {
		t.Errorf("Ptr.Name = %q, want %q", v.Ptr.Name, "pq")
	}
}

func TestSanitizeAllStrings_SkipsUnexportedAndOpaqueStructs(t *testing.T) {
	type withTime struct {
		Label string
		When  time.Time
	}
	when := time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC)
	v := withTime{Label: "\x1b[1mbold\x1b[0m", When: when}

	sanitizeAllStrings(&v) // must not panic on time.Time's unexported fields

	if v.Label != "bold" {
		t.Errorf("Label = %q, want %q", v.Label, "bold")
	}
	if !v.When.Equal(when) {
		t.Errorf("When mutated: %v", v.When)
	}
}

func TestSanitizeAllStrings_SliceArgumentAndNils(t *testing.T) {
	type entry struct{ Desc string }
	entries := []entry{{Desc: "\x1b[1mbold\x1b[0m"}, {Desc: "clean"}}

	sanitizeAllStrings(&entries)

	if entries[0].Desc != "bold" {
		t.Errorf("entries[0].Desc = %q, want %q", entries[0].Desc, "bold")
	}
	if entries[1].Desc != "clean" {
		t.Errorf("entries[1].Desc = %q, want %q", entries[1].Desc, "clean")
	}

	// Must not panic on nil pointer or nil slice.
	var p *entry
	sanitizeAllStrings(p)
	var s []entry
	sanitizeAllStrings(&s)
}
