package views

import (
	"regexp"
	"strings"
	"testing"
)

// TestCommandPalette_InputFitsTheModal covers a layout overflow. The search
// input was sized to the modal's inner width, but the bubbles text input
// renders its prompt in addition to that width, so the field came out wider
// than the box and its underline wrapped onto a second line as a stray
// fragment.
func TestCommandPalette_InputFitsTheModal(t *testing.T) {
	InitStyles()

	runs := regexp.MustCompile(`─+`)
	for _, w := range []int{80, 100, 140, 200} {
		m := NewCommandPaletteModel().SetSize(w, 40)
		m = m.SetCommands([]Command{
			{ID: "a", Label: "Overview", Description: "System health, resources", Category: "Monitor"},
			{ID: "b", Label: "Security Policies", Description: "Security rules", Category: "Analyze"},
		}).Focus()

		out := m.View()
		var shortest = -1
		for _, line := range strings.Split(out, "\n") {
			for _, run := range runs.FindAllString(line, -1) {
				n := len([]rune(run))
				if shortest < 0 || n < shortest {
					shortest = n
				}
			}
		}
		if shortest >= 0 && shortest < 10 {
			t.Errorf("width %d: found a %d-cell horizontal rule; the input underline has wrapped", w, shortest)
		}
	}
}

// TestCommandPalette_DescriptionsAlign checks the list reads as a table
// rather than a ragged column: every description should start at the same
// place regardless of how long its label is.
func TestCommandPalette_DescriptionsAlign(t *testing.T) {
	InitStyles()

	m := NewCommandPaletteModel().SetSize(120, 40)
	m = m.SetCommands([]Command{
		{ID: "a", Label: "VPN", Description: "IPSec, GlobalProtect", Category: "Monitor"},
		{ID: "b", Label: "Overview", Description: "System health, resources", Category: "Monitor"},
	}).Focus()

	lines := strings.Split(plain(m.View()), "\n")
	col := func(needle string) int {
		for _, l := range lines {
			if i := strings.Index(l, needle); i >= 0 {
				return i
			}
		}
		return -1
	}
	a, b := col("IPSec, GlobalProtect"), col("System health, resources")
	if a < 0 || b < 0 {
		t.Fatalf("descriptions not rendered (a=%d b=%d)", a, b)
	}
	if a != b {
		t.Errorf("descriptions start at columns %d and %d; the list is ragged", a, b)
	}
}
