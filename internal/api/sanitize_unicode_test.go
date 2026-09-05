package api

import "testing"

// Go's XML decoder rejects raw C0 escape bytes outright: a PAN-OS response
// cannot carry an ESC through XML at all, as a raw byte or as a character
// reference. What it carries unchanged is the C1 control block -- including
// U+009B, the single-character CSI introducer many terminals honor exactly
// like ESC '[' -- plus the Unicode bidi and zero-width formatting
// characters. Those are the categories the sanitizer used to pass straight
// through to the screen, so they are what these tests pin.
//
// Every dangerous character is written as an escape on purpose. A test about
// invisible characters that contains invisible characters cannot be reviewed.

func TestSanitizeForDisplay_StripsC1ControlSequences(t *testing.T) {
	cases := map[string]string{
		// CSI introduces a parameterized sequence ending in a final byte, so
		// the parameters have to be consumed too, not left as visible junk.
		"pre \u009b31mred\u009b0m post": "pre red post",
		// OSC is window-title injection with no ESC in sight. It ends at BEL
		// or at the C1 String Terminator.
		"pre \u009d0;title\u0007 post": "pre  post",
		"pre \u009d0;title\u009c post": "pre  post",
		// DCS ends at ST.
		"pre \u0090q something \u009c post": "pre  post",
		// A lone C1 with no sequence is still a control character.
		"pre \u0080 post": "pre  post",
		// Truncated at end of input: consumed, nothing leaks.
		"text\u009b31": "text",
	}
	for in, want := range cases {
		if got := SanitizeForDisplay(in); got != want {
			t.Errorf("SanitizeForDisplay(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSanitizeForDisplay_StripsBidiAndZeroWidth(t *testing.T) {
	cases := map[string]string{
		// Right-to-left override, the trojan-source trick: a rule named this
		// way renders in an order the configuration does not have.
		"allow-\u202egnp-yned": "allow-gnp-yned",
		// The rest of the explicit bidi embedding/override set.
		"a\u202ab\u202bc\u202cd\u202de": "abcde",
		// Directional isolates.
		"a\u2066b\u2069c\u2067d\u2068e": "abcde",
		// Directional marks.
		"a\u200eb\u200fc": "abc",
		// Zero-width characters and BOM: two different names can otherwise
		// render identically.
		"a\u200bb\u200cc\u200dd\ufeffe": "abcde",
	}
	for in, want := range cases {
		if got := SanitizeForDisplay(in); got != want {
			t.Errorf("SanitizeForDisplay(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSanitizeForDisplay_KeepsOrdinaryText(t *testing.T) {
	// Text that is merely non-ASCII must survive. Hostnames, descriptions,
	// and usernames are not always ASCII, and mangling them would be its own
	// kind of lie about the configuration.
	cases := map[string]string{
		"M\u00fcnchen-fw":    "M\u00fcnchen-fw",
		"\u65e5\u672c\u8a9e": "\u65e5\u672c\u8a9e",
		"tab\there":          "tab\there",
		"line\nbreak":        "line\nbreak",
		"  trimmed  ":        "trimmed",
	}
	for in, want := range cases {
		if got := SanitizeForDisplay(in); got != want {
			t.Errorf("SanitizeForDisplay(%q) = %q, want %q", in, got, want)
		}
	}
}
