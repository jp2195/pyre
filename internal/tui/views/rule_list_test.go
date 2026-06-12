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
