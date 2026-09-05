package views

import (
	"strings"
	"testing"

	"github.com/jp2195/pyre/internal/models"
)

// TestRuleListBanner_StaysOnOneLine checks the banner that follows a rule
// list's title.
//
// The banner is a fixed string listing the item count, the sort field, and
// every key binding the view offers. Together with the title it runs to about
// 90 cells, so on an 80-column terminal lipgloss wrapped it and the tail of
// the key list landed on its own line above the table. It does not overflow
// the screen, which is why a width check does not catch it, but it costs a
// row of the table and reads as a rendering fault.
func TestRuleListBanner_StaysOnOneLine(t *testing.T) {
	m := NewPoliciesModel().SetPolicies([]models.SecurityRule{
		{Name: "allow-outbound-web", Position: 1, Action: "allow", HitCount: 4211},
	}, nil)

	for width := 60; width <= 200; width += 5 {
		view := m.SetSize(width, 30).View()
		for _, line := range strings.Split(view, "\n") {
			if !strings.Contains(line, "Sort:") {
				continue
			}
			if !strings.Contains(line, "]") {
				t.Errorf("banner wrapped at width %d:\n  %q", width, line)
			}
			break
		}
	}
}

// TestTabbedViews_TitleAndTabsShareALine covers the same defect in the two
// views that carry a tab indicator beside the title. The title style set a
// bottom margin, which makes lipgloss render the title as a two-line block,
// so whatever is appended to it lands on the second line indented by the
// width of the title rather than beside it.
func TestTabbedViews_TitleAndTabsShareALine(t *testing.T) {
	routes := NewRoutesModel().SetRoutes([]models.RouteEntry{
		{Destination: "0.0.0.0/0", Nexthop: "10.0.0.1", Protocol: "static", Interface: "ethernet1/1"},
	}, nil)
	objects := NewObjectsModel().SetAddresses([]models.AddressObject{
		{Name: "web-server", Type: "ip-netmask", Value: "10.0.0.10/32"},
	}, nil)

	cases := []struct {
		name  string
		view  func(int) string
		title string
		tail  string
	}{
		{"routes", func(w int) string { return routes.SetSize(w, 30).View() }, "Routes", "to switch)"},
		{"objects", func(w int) string { return objects.SetSize(w, 30).View() }, "Objects", "to switch)"},
	}

	for _, tc := range cases {
		for width := 60; width <= 200; width += 5 {
			for _, line := range strings.Split(tc.view(width), "\n") {
				if !strings.Contains(line, tc.title) {
					continue
				}
				if !strings.Contains(line, tc.tail) {
					t.Errorf("%s at width %d: tabs are not on the title line:\n  %q",
						tc.name, width, line)
				}
				break
			}
		}
	}
}

// TestRoutesBanner_StaysOnOneLine is the routes view's version of the
// rule-list banner check.
func TestRoutesBanner_StaysOnOneLine(t *testing.T) {
	m := NewRoutesModel().SetRoutes([]models.RouteEntry{
		{Destination: "0.0.0.0/0", Nexthop: "10.0.0.1", Protocol: "static", Interface: "ethernet1/1"},
	}, nil)
	for width := 60; width <= 200; width += 5 {
		for _, line := range strings.Split(m.SetSize(width, 30).View(), "\n") {
			if !strings.Contains(line, "routes |") {
				continue
			}
			if !strings.Contains(line, "]") {
				t.Errorf("routes banner wrapped at width %d:\n  %q", width, line)
			}
			break
		}
	}
}

// TestListViews_RuleIsOneLine checks that the horizontal rule under a table
// header occupies a single line.
//
// The rule spans the table's available width, and each view derives that
// width by subtracting a chrome allowance from the terminal width. Routes
// subtracted six where the panel actually spends eight, so its rule was four
// cells wider than the content area and wrapped, leaving a stub of rule on
// the line below the header.
func TestListViews_RuleIsOneLine(t *testing.T) {
	routes := NewRoutesModel().SetRoutes([]models.RouteEntry{
		{Destination: "0.0.0.0/0", Nexthop: "10.0.0.1", Protocol: "static", Interface: "ethernet1/1"},
	}, nil)
	policies := NewPoliciesModel().SetPolicies([]models.SecurityRule{
		{Name: "allow-outbound-web", Position: 1, Action: "allow", HitCount: 4211},
	}, nil)
	objects := NewObjectsModel().SetAddresses([]models.AddressObject{
		{Name: "web-server", Type: "ip-netmask", Value: "10.0.0.10/32"},
	}, nil)

	views := []struct {
		name string
		view func(int) string
	}{
		{"routes", func(w int) string { return routes.SetSize(w, 30).View() }},
		{"policies", func(w int) string { return policies.SetSize(w, 30).View() }},
		{"objects", func(w int) string { return objects.SetSize(w, 30).View() }},
	}

	for _, v := range views {
		for width := 60; width <= 200; width += 5 {
			rules := 0
			for _, line := range strings.Split(v.view(width), "\n") {
				if strings.Contains(line, strings.Repeat("─", 4)) && !strings.Contains(line, "╭") && !strings.Contains(line, "╰") {
					rules++
				}
			}
			if rules != 1 {
				t.Errorf("%s at width %d: header rule occupies %d lines, want 1", v.name, width, rules)
			}
		}
	}
}
