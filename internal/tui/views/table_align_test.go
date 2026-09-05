package views

import (
	"regexp"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/models"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

func plain(s string) string { return ansiRe.ReplaceAllString(s, "") }

// displayColumnOf reports the display column at which token starts, measured from
// the left edge of the rendered line. Lines begin with the panel border, so
// counting leading spaces from index zero measures nothing.
func displayColumnOf(line, token string) int {
	i := strings.Index(line, token)
	if i < 0 {
		return -1
	}
	return lipgloss.Width(line[:i])
}

// TestRuleList_RowsShareOneLeftEdge covers a visible misalignment. Selected
// and disabled rows were rendered with styles carrying horizontal padding
// while normal rows and the header were not, so the row under the cursor sat
// one column to the right of every other row and the whole table appeared to
// shift as the cursor moved.
func TestRuleList_RowsShareOneLeftEdge(t *testing.T) {
	InitStyles()

	m := NewPoliciesModel().SetSize(200, 40)
	m = m.SetPolicies([]models.SecurityRule{
		{Name: "rule-one", Position: 1, Action: "allow"},
		{Name: "rule-two", Position: 2, Action: "allow"},
		{Name: "rule-three", Position: 3, Action: "deny", Disabled: true},
	}, nil)

	lines := strings.Split(plain(m.View()), "\n")
	var rows []string
	for _, l := range lines {
		if strings.Contains(l, "rule-one") || strings.Contains(l, "rule-two") || strings.Contains(l, "rule-three") {
			rows = append(rows, l)
		}
	}
	if len(rows) != 3 {
		t.Fatalf("found %d rule rows, want 3", len(rows))
	}

	// "local" is the rulebase column, present and identical on every row, so
	// its column is a direct measure of where each row begins.
	want := displayColumnOf(rows[0], "local")
	if want < 0 {
		t.Fatalf("could not locate the rulebase column in %q", rows[0])
	}
	for i, r := range rows[1:] {
		if got := displayColumnOf(r, "local"); got != want {
			t.Errorf("row %d has its rulebase column at %d, the first row has it at %d: rows do not share a left edge", i+2, got, want)
		}
	}
}
