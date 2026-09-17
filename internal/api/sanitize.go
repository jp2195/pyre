package api

import (
	"reflect"
	"strings"
)

// SanitizeForDisplay strips ANSI escape sequences and C0 control characters
// from server-supplied strings before they are surfaced as Go errors.
//
// PAN-OS responses occasionally echo operator input or embed terminal color
// codes in msg/line elements. Letting those bytes flow into error strings
// means any downstream log writer, TUI pane, or stderr consumer can be
// tricked into moving the cursor, changing colors, or (in rare cases)
// issuing terminal commands via escape sequences. Neutralizing them here
// keeps the API package the single choke point for untrusted display data.
//
// The state machine recognizes the following ESC-introduced sequences:
//
//   - CSI (ESC '['): parameter/intermediate bytes consumed until a final
//     byte in 0x40-0x7e ('@' through '~'). Matches "\x1b[31m", "\x1b[0m", …
//   - OSC (ESC ']'): consumed until BEL (0x07) or String Terminator
//     (ESC '\\'). Covers window-title injection: "\x1b]0;title\x07".
//   - DCS (ESC 'P'), PM (ESC '^'), APC (ESC '_'), SOS (ESC 'X'): consumed
//     until String Terminator (ESC '\\').
//   - Any other two-byte ESC sequence (e.g. ESC 'c' full reset, ESC '(B'
//     charset select): both bytes dropped.
//
// It also handles the C1 forms of those introducers, which matter more here
// than the ESC forms do. Go's XML decoder rejects a raw ESC outright, as a
// byte or as a character reference, so an ESC cannot reach us through a
// parsed PAN-OS response at all — but the whole C1 block passes through
// untouched, and U+009B is a single-character CSI that many terminals honor
// exactly like ESC '['. The ESC handling above therefore guards the
// plain-text response bodies; the C1 handling guards everything else:
//
//   - U+009B (CSI), U+009D (OSC), U+0090 (DCS), U+0098 (SOS), U+009E (PM),
//     U+009F (APC) enter the same states as their ESC equivalents, and
//     U+009C (ST) terminates a string sequence.
//   - Any other C1 (U+0080-U+009F) is dropped as a bare control character.
//
// In addition:
//   - All C0 controls (< 0x20) are dropped except '\n' and '\t', which are
//     legitimate formatting characters in multi-line error text.
//   - 0x7f (DEL) is dropped.
//   - Unicode bidi and zero-width formatting characters are dropped; see
//     isDisplayDeceptive.
//   - Leading and trailing whitespace is trimmed after stripping.
//   - Truncated sequences at end-of-input are dropped entirely (no leak,
//     no hang).
//
// Text that is merely non-ASCII is left alone. The goal is that displayed
// text cannot misrepresent its own content or drive the terminal, not that
// it be ASCII.
func SanitizeForDisplay(s string) string {
	const (
		stateNormal = iota
		stateEsc    // just saw ESC (0x1b); waiting on the introducer byte
		stateCSI    // inside CSI (ESC '['); drop until final byte 0x40-0x7e
		stateOSC    // inside OSC (ESC ']'); drop until BEL or ST
		stateOSCEsc // saw ESC inside OSC; '\\' closes it, else resume OSC
		stateStr    // inside DCS/PM/APC/SOS; drop until ST (ESC '\\')
		stateStrEsc // saw ESC inside string; '\\' closes it, else resume
	)
	var b strings.Builder
	b.Grow(len(s))
	state := stateNormal
	for _, r := range s {
		switch state {
		case stateNormal:
			if r == 0x1b {
				state = stateEsc
				continue
			}
			if r < 0x20 && r != '\n' && r != '\t' {
				continue
			}
			if r == 0x7f {
				continue
			}
			// C1 controls. These reach us through XML where ESC cannot, and
			// the introducers below drive a terminal on their own.
			if r >= 0x80 && r <= 0x9f {
				switch r {
				case 0x9b: // CSI
					state = stateCSI
				case 0x9d: // OSC
					state = stateOSC
				case 0x90, 0x98, 0x9e, 0x9f: // DCS, SOS, PM, APC
					state = stateStr
				}
				// Every other C1, including a stray ST, is simply dropped.
				continue
			}
			if isDisplayDeceptive(r) {
				continue
			}
			b.WriteRune(r)
		case stateEsc:
			switch r {
			case '[':
				state = stateCSI
			case ']':
				state = stateOSC
			case 'P', '^', '_', 'X':
				state = stateStr
			default:
				// Two-byte ESC sequence (charset select, reset, etc.):
				// both ESC and this byte are dropped.
				state = stateNormal
			}
		case stateCSI:
			// Parameter bytes are 0x30-0x3f, intermediate bytes 0x20-0x2f;
			// the sequence ends on the first final byte in 0x40-0x7e.
			if r >= 0x40 && r <= 0x7e {
				state = stateNormal
			}
		case stateOSC:
			switch r {
			case 0x07, 0x9c: // BEL or C1 String Terminator
				state = stateNormal
			case 0x1b:
				state = stateOSCEsc
			}
		case stateOSCEsc:
			// ESC inside OSC: '\\' finishes the ST terminator; any other
			// byte is not a legal terminator but we drop it and resume
			// consuming the OSC body for safety.
			if r == '\\' {
				state = stateNormal
			} else {
				state = stateOSC
			}
		case stateStr:
			switch r {
			case 0x9c: // C1 String Terminator
				state = stateNormal
			case 0x1b:
				state = stateStrEsc
			}
		case stateStrEsc:
			if r == '\\' {
				state = stateNormal
			} else {
				state = stateStr
			}
		}
	}
	return strings.TrimSpace(b.String())
}

// isDisplayDeceptive reports whether r is a Unicode formatting character that
// can make text render as something other than what it says.
//
// These pass through XML untouched and are invisible by construction, so they
// are the practical way to spoof a rule or object name in a viewer like this
// one. A name carrying U+202E renders reversed; one carrying a zero-width
// character can render identically to a different name.
//
// Deliberate trade-off: U+200C and U+200D (ZWNJ, ZWJ) carry meaning in some
// scripts, so dropping them can alter legitimate text. In a tool whose job is
// to report what a firewall is actually configured to do, a name that cannot
// lie about itself is worth more than perfect rendering of those scripts.
// Stripping rather than escaping means two names differing only in these
// characters collapse to the same display string; that is the lesser evil
// against a name that reads as its own opposite.
func isDisplayDeceptive(r rune) bool {
	switch {
	case r >= 0x202a && r <= 0x202e: // LRE, RLE, PDF, LRO, RLO
		return true
	case r >= 0x2066 && r <= 0x2069: // LRI, RLI, FSI, PDI
		return true
	case r == 0x200e || r == 0x200f: // LRM, RLM
		return true
	case r >= 0x200b && r <= 0x200d: // ZWSP, ZWNJ, ZWJ
		return true
	case r == 0xfeff: // BOM / zero-width no-break space
		return true
	}
	return false
}

// sanitizeAllStrings applies SanitizeForDisplay to every settable string
// field reachable from v — recursively through pointers, structs, slices,
// and arrays. Fetchers call it on parsed models so the API package stays
// the single choke point for untrusted display data (see SanitizeForDisplay).
//
// Unexported fields and non-string kinds are skipped. time.Time and other
// opaque structs are safe: their fields are unexported and therefore not
// settable.
func sanitizeAllStrings(v any) {
	sanitizeValue(reflect.ValueOf(v))
}

func sanitizeValue(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer:
		// Models contain no interface, map, or chan fields; if one ever
		// appears, add its kind here deliberately (IsNil panics on
		// non-nullable kinds, so don't blanket-extend this case).
		if !v.IsNil() {
			sanitizeValue(v.Elem())
		}
	case reflect.Struct:
		for _, f := range v.Fields() {
			if f.CanSet() {
				sanitizeValue(f)
			}
		}
	case reflect.Slice, reflect.Array:
		// Byte slices hold raw XML (XMLResponse.Result.Inner), not display
		// text. Walking one element-by-element through reflection would mean
		// millions of calls for a large response, for no benefit: those bytes
		// are re-parsed through decodeXML, and the strings that come out of
		// that parse are sanitized then.
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return
		}
		for i := range v.Len() {
			sanitizeValue(v.Index(i))
		}
	case reflect.String:
		if v.CanSet() {
			v.SetString(SanitizeForDisplay(v.String()))
		}
	}
}
