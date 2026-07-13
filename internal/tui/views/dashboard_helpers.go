package views

import (
	"fmt"
	"image/color"
	"sort"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/models"
)

// DashboardBase provides shared layout fields and helpers for all dashboard models.
type DashboardBase struct {
	Width        int
	Height       int
	SpinnerFrame string
}

// ColumnWidths returns the total, left, and right column widths for two-column layout.
func (d DashboardBase) ColumnWidths() (totalWidth, leftColWidth, rightColWidth int) {
	totalWidth = d.Width - 4
	leftColWidth = totalWidth / 2
	rightColWidth = totalWidth - leftColWidth - 2
	return
}

// IsNarrow returns true if the terminal is too narrow for two-column layout.
func (d DashboardBase) IsNarrow() bool {
	totalWidth := d.Width - 4
	return totalWidth/2 < 35
}

// RenderTwoColumn joins left and right panel slices into a two-column layout.
func (d DashboardBase) RenderTwoColumn(leftPanels, rightPanels []string) string {
	leftCol := lipgloss.JoinVertical(lipgloss.Left, leftPanels...)
	rightCol := lipgloss.JoinVertical(lipgloss.Left, rightPanels...)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftCol, "  ", rightCol)
}

// RenderSingleColumn joins panels into a single vertical column.
func (d DashboardBase) RenderSingleColumn(panels []string) string {
	return lipgloss.JoinVertical(lipgloss.Left, panels...)
}

// panelStyle returns the panel style with reduced padding for dashboard
func panelStyle() lipgloss.Style {
	return ViewPanelStyle.Padding(0, 1)
}

// Style accessor functions - these must be functions (not variables) because
// styles are initialized at runtime via InitStyles(), not at package load time
func titleStyle() lipgloss.Style     { return ViewTitleStyle }
func subtitleStyle() lipgloss.Style  { return SubtitleBoldStyle }
func labelStyle() lipgloss.Style     { return DetailLabelStyle }
func valueStyle() lipgloss.Style     { return DetailValueStyle }
func dimStyle() lipgloss.Style       { return DetailDimStyle }
func highlightStyle() lipgloss.Style { return StatusActiveStyle }
func warningStyle() lipgloss.Style   { return StatusWarningStyle }
func errorStyle() lipgloss.Style     { return ErrorMsgStyle }
func accentStyle() lipgloss.Style    { return TagStyle }

func renderBar(percent float64, width int, c color.Color) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filled := min(int(percent/100*float64(width)), width)

	filledStyle := lipgloss.NewStyle().Foreground(c)
	emptyStyle := StatusMutedStyle

	bar := strings.Builder{}
	for i := range width {
		if i < filled {
			bar.WriteString(filledStyle.Render("█"))
		} else {
			bar.WriteString(emptyStyle.Render("░"))
		}
	}
	return bar.String()
}

func formatNumber(n int64) string {
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

func formatThroughput(kbps int64) string {
	if kbps == 0 {
		return "0 Kbps"
	}
	if kbps >= 1_000_000 {
		return fmt.Sprintf("%.1f Gbps", float64(kbps)/1_000_000)
	}
	if kbps >= 1_000 {
		return fmt.Sprintf("%.1f Mbps", float64(kbps)/1_000)
	}
	return fmt.Sprintf("%d Kbps", kbps)
}

// relTimeOpts controls the cosmetic differences between the app's
// relative-time displays; the bucketing algorithm is shared.
type relTimeOpts struct {
	zeroLabel    string // returned for the zero time
	justNowLabel string // returned for < 1 minute
	weeks        bool   // include a weeks bucket before falling back to a date
	dateFormat   string // final fallback format
}

func formatRelativeTime(t time.Time, o relTimeOpts) string {
	if t.IsZero() {
		return o.zeroLabel
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return o.justNowLabel
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case o.weeks && d < 30*24*time.Hour:
		return fmt.Sprintf("%dw ago", int(d.Hours()/24/7))
	default:
		return t.Format(o.dateFormat)
	}
}

func formatTimeAgo(t time.Time) string {
	return formatRelativeTime(t, relTimeOpts{zeroLabel: "", justNowLabel: "just now", dateFormat: "Jan 2"})
}

// renderZeroHitRulesPanel renders the "Zero-Hit Rules" panel shared by the
// security and config dashboards. maxShow and nameWidthSub are the two
// knobs the dashboards historically differed on.
func renderZeroHitRulesPanel(policies []models.SecurityRule, policyErr error, spinnerFrame string, width, maxShow, nameWidthSub int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Zero-Hit Rules"))
	b.WriteString("\n")

	if policyErr != nil {
		b.WriteString(dimStyle().Render("Not available"))
		return panelStyle().Width(width).Render(b.String())
	}
	if policies == nil {
		b.WriteString(RenderLoadingInline(spinnerFrame, "Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	var zeroHitRules []models.SecurityRule
	for _, rule := range policies {
		if rule.HitCount == 0 && !rule.Disabled {
			zeroHitRules = append(zeroHitRules, rule)
		}
	}

	if len(zeroHitRules) == 0 {
		b.WriteString(highlightStyle().Render("All active rules have hits"))
		return panelStyle().Width(width).Render(b.String())
	}

	totalActive := 0
	for _, rule := range policies {
		if !rule.Disabled {
			totalActive++
		}
	}

	pct := float64(len(zeroHitRules)) / float64(totalActive) * 100
	b.WriteString(warningStyle().Render(strconv.Itoa(len(zeroHitRules))))
	b.WriteString(dimStyle().Render(fmt.Sprintf(" of %d rules (%.0f%%)", totalActive, pct)))
	b.WriteString("\n\n")

	show := min(len(zeroHitRules), maxShow)
	nameWidth := min(width-nameWidthSub, 30)

	for i := range show {
		rule := zeroHitRules[i]
		name := truncateEllipsis(rule.Name, nameWidth)
		b.WriteString(labelStyle().Render(fmt.Sprintf("%3d. ", rule.Position)))
		b.WriteString(valueStyle().Render(fmt.Sprintf("%-*s ", nameWidth, name)))
		b.WriteString(ruleActionStyle(rule.Action).Render(rule.Action))
		if i < show-1 {
			b.WriteString("\n")
		}
	}

	if len(zeroHitRules) > show {
		b.WriteString("\n")
		b.WriteString(dimStyle().Render(fmt.Sprintf("... and %d more", len(zeroHitRules)-show)))
	}

	return panelStyle().Width(width).Render(b.String())
}

// renderMostHitRulesPanel renders the "Most-Hit Rules" panel. policiesByHits
// must already be sorted by hit count descending (cached at SetPolicies time
// so View() does no per-frame sorting).
func renderMostHitRulesPanel(policiesByHits []models.SecurityRule, policyErr error, spinnerFrame string, width, maxShow int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Most-Hit Rules"))
	b.WriteString("\n")

	if policyErr != nil {
		b.WriteString(dimStyle().Render("Not available"))
		return panelStyle().Width(width).Render(b.String())
	}
	if policiesByHits == nil {
		b.WriteString(RenderLoadingInline(spinnerFrame, "Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}
	if len(policiesByHits) == 0 {
		b.WriteString(dimStyle().Render("No rules"))
		return panelStyle().Width(width).Render(b.String())
	}

	shown := 0
	totalWithHits := 0
	nameWidth := min(width-20, 25)

	for _, rule := range policiesByHits {
		if rule.HitCount == 0 {
			continue
		}
		totalWithHits++
		if shown >= maxShow {
			continue
		}

		name := truncateEllipsis(rule.Name, nameWidth)
		b.WriteString(valueStyle().Render(fmt.Sprintf("%-*s ", nameWidth, name)))
		b.WriteString(ruleActionStyle(rule.Action).Render(fmt.Sprintf("%-5s ", rule.Action)))
		b.WriteString(accentStyle().Render(formatNumber(rule.HitCount)))
		b.WriteString("\n")
		shown++
	}

	if shown == 0 {
		b.WriteString(dimStyle().Render("No rules have been hit"))
	}

	result := strings.TrimSuffix(b.String(), "\n")
	if totalWithHits > maxShow {
		result += "\n" + dimStyle().Render(fmt.Sprintf("... and %d more rules", totalWithHits-maxShow))
	}

	return panelStyle().Width(width).Render(result)
}

// ruleActionStyle maps a rule action to its display style (shared by the
// two panels above; previously copy-pasted four times).
func ruleActionStyle(action string) lipgloss.Style {
	switch action {
	case "allow":
		return highlightStyle()
	case "deny", "drop":
		return errorStyle()
	default:
		return dimStyle()
	}
}

// sortPoliciesByHits returns a copy of policies sorted by hit count
// descending. Call from SetPolicies, never from View().
func sortPoliciesByHits(policies []models.SecurityRule) []models.SecurityRule {
	sorted := make([]models.SecurityRule, len(policies))
	copy(sorted, policies)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].HitCount > sorted[j].HitCount
	})
	return sorted
}
