package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/jp2195/pyre/internal/models"
)

// containsAny returns true if any item in the list contains the query (case-insensitive).
func containsAny(items []string, query string) bool {
	for _, item := range items {
		if strings.Contains(strings.ToLower(item), query) {
			return true
		}
	}
	return false
}

// formatRuleBase returns a short label for the rulebase type.
func formatRuleBase(rb models.RuleBase) string {
	switch rb {
	case models.RuleBasePre:
		return "pre"
	case models.RuleBasePost:
		return "post"
	case models.RuleBaseLocal:
		return "local"
	default:
		return "local"
	}
}

// formatRuleBaseFull returns a descriptive label for the rulebase type.
func formatRuleBaseFull(rb models.RuleBase) string {
	switch rb {
	case models.RuleBasePre:
		return "Pre-Rulebase (Panorama)"
	case models.RuleBasePost:
		return "Post-Rulebase (Panorama)"
	case models.RuleBaseLocal:
		return "Local Rulebase"
	default:
		return "Local Rulebase"
	}
}

// formatZoneCompact formats zone list for compact table display.
func formatZoneCompact(zones []string) string {
	if len(zones) == 0 || (len(zones) == 1 && zones[0] == "any") {
		return "any"
	}
	if len(zones) == 1 {
		z := zones[0]
		if len(z) > 8 {
			return z[:6] + "…"
		}
		return z
	}
	first := zones[0]
	if len(first) > 5 {
		first = first[:4] + "…"
	}
	return fmt.Sprintf("%s+%d", first, len(zones)-1)
}

// formatListCompact formats a string list for compact table display.
func formatListCompact(items []string, maxLen int) string {
	if len(items) == 0 || (len(items) == 1 && items[0] == "any") {
		return "any"
	}
	if len(items) == 1 {
		return truncateStr(items[0], maxLen)
	}
	first := truncateStr(items[0], maxLen-4)
	return fmt.Sprintf("%s+%d", first, len(items)-1)
}

// formatListFull formats a string list with comma separation.
func formatListFull(items []string) string {
	if len(items) == 0 {
		return "any"
	}
	return strings.Join(items, ", ")
}

// formatAddresses formats address list with optional negation.
func formatAddresses(addrs []string, negate bool, valueStyle, dimStyle lipgloss.Style) string {
	if len(addrs) == 0 || (len(addrs) == 1 && addrs[0] == "any") {
		return dimStyle.Render("any")
	}
	result := valueStyle.Render(strings.Join(addrs, ", "))
	if negate {
		result = valueStyle.Render("NOT ") + result
	}
	return result
}

// formatHitCount formats a hit count for compact table display.
func formatHitCount(count int64) string {
	if count == 0 {
		return "0"
	}
	if count >= 1_000_000_000 {
		return fmt.Sprintf("%.1fB", float64(count)/1_000_000_000)
	}
	if count >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(count)/1_000_000)
	}
	if count >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(count)/1_000)
	}
	return fmt.Sprintf("%d", count)
}

// formatHitCountFull formats a hit count with thousand separators.
func formatHitCountFull(count int64) string {
	if count == 0 {
		return "0"
	}
	s := fmt.Sprintf("%d", count)
	n := len(s)
	if n <= 3 {
		return s
	}
	var result strings.Builder
	for i, c := range s {
		if i > 0 && (n-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}
	return result.String()
}

// formatLastHit formats a time as a relative duration for table display.
func formatLastHit(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	if d < 7*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
	return t.Format("Jan 2")
}

// formatTimestamp formats a time as a full timestamp.
func formatTimestamp(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}
	return t.Format("2006-01-02 15:04:05")
}

// truncateStr truncates a string using unicode ellipsis.
func truncateStr(s string, maxLen int) string {
	return truncateEllipsis(s, maxLen)
}

// hasProfiles returns true if the security rule has any individual profiles set.
func hasProfiles(p models.SecurityRule) bool {
	return p.AntivirusProfile != "" || p.VulnerabilityProfile != "" ||
		p.SpywareProfile != "" || p.URLFilteringProfile != "" ||
		p.FileBlockingProfile != "" || p.WildFireProfile != ""
}
