package views

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// TestWrapText_BreaksATokenThatHasNoBreakOpportunity covers the overflow that
// whitespace-only wrapping cannot fix. The strings this helper is given --
// PAN-OS expressions, URLs in a log description, file digests -- routinely
// contain one run with no space in it that is wider than the panel. Emitting
// it whole pushes the box past the terminal edge, which is the very thing the
// caller wrapped the text to avoid.
func TestWrapText_BreaksATokenThatHasNoBreakOpportunity(t *testing.T) {
	const width = 20
	text := "digest " + strings.Repeat("a", 75) + " end"

	for _, line := range wrapText(text, width) {
		if got := lipgloss.Width(line); got > width {
			t.Errorf("line is %d cells wide, want at most %d: %q", got, width, line)
		}
	}
}

// TestWrapText_MeasuresInDisplayCellsNotBytes pins the unit. Measuring with
// len() counts bytes, so any non-ASCII text -- a hostname with an accent, a
// device message that is not pure ASCII -- wraps far earlier than it needs to
// and leaves the panel visibly ragged.
func TestWrapText_MeasuresInDisplayCellsNotBytes(t *testing.T) {
	// Ten single-cell words plus nine spaces is 19 cells, but 29 bytes.
	words := make([]string, 10)
	for i := range words {
		words[i] = "é"
	}
	text := strings.Join(words, " ")

	if got := lipgloss.Width(text); got != 19 {
		t.Fatalf("precondition: text is %d cells, want 19", got)
	}
	if lines := wrapText(text, 19); len(lines) != 1 {
		t.Errorf("wrapped %d-cell text into %d lines at width 19, want 1: %q",
			lipgloss.Width(text), len(lines), lines)
	}
}

// TestWrapText_NeverLoopsOnAWidthNarrowerThanOneRune guards the break loop: a
// rune that cannot fit the width at all must still make progress.
func TestWrapText_NeverLoopsOnAWidthNarrowerThanOneRune(t *testing.T) {
	lines := wrapText("世界世界", 1)
	if len(lines) == 0 {
		t.Fatal("no output")
	}
	if joined := strings.Join(lines, ""); joined != "世界世界" {
		t.Errorf("characters were lost or duplicated: %q", joined)
	}
}
