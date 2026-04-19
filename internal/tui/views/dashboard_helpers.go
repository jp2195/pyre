package views

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
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
	s := fmt.Sprintf("%d", n)
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

func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	case d < 7*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	default:
		return t.Format("Jan 2")
	}
}
