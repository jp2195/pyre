package api

import "strings"

// SanitizeForDisplay strips ANSI escape sequences and C0 control characters
// from server-supplied strings before they are surfaced as Go errors.
//
// PAN-OS responses occasionally echo operator input or embed terminal colour
// codes in msg/line elements. Letting those bytes flow into error strings
// means any downstream log writer, TUI pane, or stderr consumer can be
// tricked into moving the cursor, changing colours, or (in rare cases)
// issuing terminal commands via escape sequences. Neutralising them here
// keeps the API package the single choke point for untrusted display data.
//
// The state machine recognises the following ESC-introduced sequences:
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
// In addition:
//   - All C0 controls (< 0x20) are dropped except '\n' and '\t', which are
//     legitimate formatting characters in multi-line error text.
//   - 0x7f (DEL) is dropped.
//   - Leading and trailing whitespace is trimmed after stripping.
//   - Truncated sequences at end-of-input are dropped entirely (no leak,
//     no hang).
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
			case 0x07:
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
			if r == 0x1b {
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
