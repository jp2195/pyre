package views

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

type rlItem struct {
	Name  string
	State string
}

func testRuleListConfig() RuleListConfig[rlItem] {
	return RuleListConfig[rlItem]{
		Title:             "Widgets",
		ItemNoun:          "widgets",
		LoadingMsg:        "Loading widgets...",
		EmptyMsg:          "No widgets found",
		FilterPlaceholder: "Filter widgets...",
		SortLabels:        []string{"Name", "State"},
		DefaultSortAsc:    func(idx int) bool { return idx == 0 },
		MatchFilter: func(it rlItem, q string) bool {
			return strings.Contains(strings.ToLower(it.Name), q)
		},
		CompareItems: func(a, b rlItem, idx int) bool {
			if idx == 1 {
				return a.State < b.State
			}
			return a.Name < b.Name
		},
		FormatHeaderRow: func(width int) string { return "NAME STATE" },
		FormatRow:       func(it rlItem, width int) string { return it.Name + " " + it.State },
		RenderDetail:    func(it rlItem, width int) string { return "detail:" + it.Name },
		// IsDisabled intentionally nil: renderTable must not panic.
	}
}

func TestRuleList_BannerUsesItemNoun(t *testing.T) {
	InitStyles()
	m := NewRuleListModel(testRuleListConfig())
	m = m.SetSize(100, 30)
	m = m.SetItems([]rlItem{{Name: "alpha"}}, nil)

	out := m.View()
	if !strings.Contains(out, "1 widgets") {
		t.Errorf("expected banner to contain %q, got:\n%s", "1 widgets", out)
	}
}

func TestRuleList_BannerFallsBackToRules(t *testing.T) {
	InitStyles()
	config := testRuleListConfig()
	config.ItemNoun = ""
	m := NewRuleListModel(config)
	m = m.SetSize(100, 30)
	m = m.SetItems([]rlItem{{Name: "alpha"}}, nil)

	if out := m.View(); !strings.Contains(out, "1 rules") {
		t.Errorf("expected banner fallback %q, got:\n%s", "1 rules", out)
	}
}

func TestRuleList_NilIsDisabledDoesNotPanic(t *testing.T) {
	InitStyles()
	m := NewRuleListModel(testRuleListConfig())
	m = m.SetSize(100, 30)
	m = m.SetItems([]rlItem{{Name: "alpha"}, {Name: "beta"}}, nil)

	out := m.View() // panics before the nil-guard fix
	if !strings.Contains(out, "alpha") {
		t.Errorf("expected row 'alpha' in output:\n%s", out)
	}
}

func TestRuleList_StyleRowUsedForNonSelectedRows(t *testing.T) {
	InitStyles()
	config := testRuleListConfig()
	config.StyleRow = func(it rlItem, width int) string { return "styled<" + it.Name + ">" }
	m := NewRuleListModel(config)
	m = m.SetSize(100, 30)
	m = m.SetItems([]rlItem{{Name: "alpha"}, {Name: "beta"}}, nil)

	out := m.View()
	// Cursor is on row 0 (beta, after sort desc): selected row must NOT use StyleRow.
	// Only alpha (non-selected) should use StyleRow.
	if strings.Contains(out, "styled<beta>") {
		t.Errorf("selected row should not use StyleRow:\n%s", out)
	}
	if !strings.Contains(out, "styled<alpha>") {
		t.Errorf("non-selected row should use StyleRow:\n%s", out)
	}
}

func TestRuleList_ShiftSTogglesSortDirection(t *testing.T) {
	InitStyles()
	m := NewRuleListModel(testRuleListConfig())
	m = m.SetSize(100, 30)
	m = m.SetItems([]rlItem{{Name: "alpha"}, {Name: "zeta"}}, nil)

	// Default SortAsc=false: zeta sorts before alpha.
	out := m.View()
	if strings.Index(out, "zeta") > strings.Index(out, "alpha") {
		t.Fatalf("precondition: expected descending order (zeta first):\n%s", out)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
	out = m.View()
	if strings.Index(out, "alpha") > strings.Index(out, "zeta") {
		t.Errorf("expected ascending order after S (alpha first):\n%s", out)
	}
}

// stripANSI removes SGR escape sequences so column positions can be measured
// on the visible text alone.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// columnOf returns the column at which token appears on the line containing it.
func columnOf(t *testing.T, rendered, token string) int {
	t.Helper()
	for line := range strings.SplitSeq(stripANSI(rendered), "\n") {
		if idx := strings.Index(line, token); idx >= 0 {
			return idx
		}
	}
	t.Fatalf("token %q not found in:\n%s", token, stripANSI(rendered))
	return -1
}

// TestRuleList_ColumnsDoNotShiftWithCursor pins the fix for a misalignment
// where the selected row carried horizontal padding the header and unselected
// rows did not. Every row visibly jumped one column right as the cursor landed
// on it and snapped back when it left, so the whole table twitched while
// scrolling. Column positions must not depend on which row is selected.
func TestRuleList_ColumnsDoNotShiftWithCursor(t *testing.T) {
	items := []rlItem{
		{Name: "alpha", State: "up"},
		{Name: "bravo", State: "down"},
		{Name: "charlie", State: "up"},
	}

	m := NewRuleListModel(testRuleListConfig())
	m = m.SetSize(120, 40)
	m = m.SetItems(items, nil)

	// Column of each row's State field, measured with the cursor on row 0.
	m.Cursor = 0
	base := map[string]int{}
	out := m.View()
	for _, it := range items {
		base[it.Name] = columnOf(t, out, it.Name)
	}

	// Moving the cursor must not move any column.
	for cursor := 1; cursor < len(items); cursor++ {
		m.Cursor = cursor
		out = m.View()
		for _, it := range items {
			got := columnOf(t, out, it.Name)
			if got != base[it.Name] {
				t.Errorf("cursor=%d: row %q moved from column %d to %d; "+
					"columns must not shift as the selection moves",
					cursor, it.Name, base[it.Name], got)
			}
		}
	}
}
