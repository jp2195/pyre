package views

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/auth"
)

// truncate truncates a string to maxLen runes, adding ... if needed.
func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}

// truncateEllipsis truncates to maxLen runes with a Unicode ellipsis.
func truncateEllipsis(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}

// formatNumberWithCommas formats a number with thousand separators
func formatNumberWithCommas(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}

	var result strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}
	return result.String()
}

// formatBytes formats bytes into human readable format (KB, MB, GB, etc)
func formatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatPackets formats packet counts into human readable format (K, M, B)
func formatPackets(packets int64) string {
	if packets == 0 {
		return "0"
	}
	if packets < 1000 {
		return strconv.FormatInt(packets, 10)
	}
	if packets < 1000000 {
		return fmt.Sprintf("%.1fK", float64(packets)/1000)
	}
	if packets < 1000000000 {
		return fmt.Sprintf("%.1fM", float64(packets)/1000000)
	}
	return fmt.Sprintf("%.1fB", float64(packets)/1000000000)
}

// validateHost delegates to the app-wide host validator in internal/auth.
func validateHost(host string) string {
	return auth.ValidateHost(host)
}

// cleanValue normalizes ugly API values like "N/A", "ukn", "[n/a]" to empty
func cleanValue(s string) string {
	lower := strings.ToLower(strings.TrimSpace(s))
	if lower == "" || lower == "n/a" || lower == "ukn" || lower == "[n/a]" || lower == "unknown" {
		return ""
	}
	return s
}

// wrapText wraps text to the specified width.
func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}

	currentLine := words[0]
	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	lines = append(lines, currentLine)
	return lines
}

// tableSeparator renders the horizontal rule under a table header. It spans
// the table, not the header text: the last column is often wider than its
// label, so measuring the header left the rule ending at an arbitrary point
// part-way across the data.
//
// The available width is derived by subtracting a fixed chrome allowance from
// the terminal width, so it goes negative once the terminal is narrower than
// that allowance. strings.Repeat panics on a negative count, and a panic
// inside a Bubble Tea View leaves the terminal in the alternate screen with
// echo off, so clamp before repeating. Dragging a pane narrow is a routine
// thing to do; it must not take the program down.
func tableSeparator(availableWidth int) string {
	return strings.Repeat("─", max(availableWidth, 0))
}

// modalChromeWidth is what a centered modal box spends on its border and
// horizontal padding, so the content has terminal width minus this to work in.
const modalChromeWidth = 10

// fitHints joins hints in order, keeping only those that still fit in width.
// The first hint is always kept, so the line is never empty.
//
// A modal box has no explicit width: it sizes itself to its widest line. A
// fixed help string therefore sets the width of the whole box, and on a
// narrow terminal it pushes the box wider than the screen. Dropping the
// least important hints keeps the box inside the terminal instead.
func fitHints(width int, sep string, hints ...string) string {
	if len(hints) == 0 {
		return ""
	}
	out := hints[0]
	for _, h := range hints[1:] {
		if width > 0 && lipgloss.Width(out)+lipgloss.Width(sep)+lipgloss.Width(h) > width {
			break
		}
		out += sep + h
	}
	return out
}

// modalInputWidth sizes a text input inside a centered modal. The inputs were
// a fixed 40 cells, which is wider than the content area of a terminal under
// about 54 columns.
func modalInputWidth(termWidth int) int {
	const preferred = 40
	// A text input renders three cells wider than the width set on it, for
	// the prompt and the cursor cell. InputStyle frames that with a border
	// and a cell of padding either side, and the modal box adds its own
	// border and four cells of padding. Seventeen cells go before any text.
	const overhead = 17
	if termWidth <= 0 {
		return preferred
	}
	return max(min(preferred, termWidth-overhead), 12)
}

// tableChromeWidth is what a framed list view spends on chrome before any
// table column: two cells of panel border, four of horizontal padding, and
// two of slack so a full-width row does not sit flush against the frame.
//
// The views used to each subtract their own number. Routes subtracted six,
// which is less than the panel actually spends, so its header rule ran past
// the content area and wrapped.
const tableChromeWidth = 12

// modalBoxWidth sizes a centered modal box: the preferred width when the
// terminal has room for it, otherwise whatever the terminal leaves after its
// margins, and never wider than the terminal itself.
//
// The screens each did this by hand. Two shrank the box with the terminal but
// set no lower bound, so the width went negative on a tiny one; the third
// clamped to a floor wider than a small terminal and overflowed it.
func modalBoxWidth(termWidth, preferred int) int {
	if termWidth <= 0 {
		return preferred
	}
	return max(min(preferred, termWidth-modalChromeWidth), 1)
}

// modalContentWidth is the width of the text area inside a modal box of the
// given outer width. ViewPanelStyle draws a one-cell border and pads two
// cells either side, and Width sets the outer width, border included.
func modalContentWidth(boxWidth int) int {
	return max(boxWidth-6, 1)
}

// inputChromeWidth is what a framed line inside a modal form spends before
// its text: InputStyle's border and padding, plus the modal box's own border
// and horizontal padding.
const inputChromeWidth = 14
