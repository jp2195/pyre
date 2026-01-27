package views

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jp2195/pyre/internal/models"
	"github.com/jp2195/pyre/internal/tui/theme"
)

// SecurityDashboardModel represents the security-focused dashboard
type SecurityDashboardModel struct {
	threatSummary *models.ThreatSummary
	policies      []models.SecurityRule

	threatErr error
	policyErr error

	width        int
	height       int
	SpinnerFrame string
}

// NewSecurityDashboardModel creates a new security dashboard model
func NewSecurityDashboardModel() SecurityDashboardModel {
	return SecurityDashboardModel{}
}

// SetSpinnerFrame sets the current spinner animation frame
func (m SecurityDashboardModel) SetSpinnerFrame(frame string) SecurityDashboardModel {
	m.SpinnerFrame = frame
	return m
}

// SetSize sets the terminal dimensions
func (m SecurityDashboardModel) SetSize(width, height int) SecurityDashboardModel {
	m.width = width
	m.height = height
	return m
}

// SetThreatSummary sets the threat summary data
func (m SecurityDashboardModel) SetThreatSummary(summary *models.ThreatSummary, err error) SecurityDashboardModel {
	m.threatSummary = summary
	m.threatErr = err
	return m
}

// SetPolicies sets the security policies data
func (m SecurityDashboardModel) SetPolicies(policies []models.SecurityRule, err error) SecurityDashboardModel {
	m.policies = policies
	m.policyErr = err
	return m
}

// Update handles key events
func (m SecurityDashboardModel) Update(msg tea.Msg) (SecurityDashboardModel, tea.Cmd) {
	return m, nil
}

// HasData returns true if the dashboard has already loaded its data
func (m SecurityDashboardModel) HasData() bool {
	return m.threatSummary != nil
}

// View renders the security dashboard
func (m SecurityDashboardModel) View() string {
	if m.width == 0 {
		return RenderLoadingInline(m.SpinnerFrame, "Loading...")
	}

	totalWidth := m.width - 4
	leftColWidth := totalWidth / 2
	rightColWidth := totalWidth - leftColWidth - 2

	if leftColWidth < 35 {
		return m.renderSingleColumn(totalWidth)
	}

	// Left column: threat info
	leftPanels := []string{
		m.renderThreatBreakdown(leftColWidth),
		m.renderThreatSeverity(leftColWidth),
	}
	leftCol := lipgloss.JoinVertical(lipgloss.Left, leftPanels...)

	// Right column: policy analysis
	rightPanels := []string{
		m.renderZeroHitRules(rightColWidth),
		m.renderMostHitRules(rightColWidth),
	}
	rightCol := lipgloss.JoinVertical(lipgloss.Left, rightPanels...)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftCol, "  ", rightCol)
}

func (m SecurityDashboardModel) renderSingleColumn(width int) string {
	panels := []string{
		m.renderThreatBreakdown(width),
		m.renderThreatSeverity(width),
		m.renderZeroHitRules(width),
		m.renderMostHitRules(width),
	}
	return lipgloss.JoinVertical(lipgloss.Left, panels...)
}

func (m SecurityDashboardModel) renderThreatBreakdown(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Threat Summary"))
	b.WriteString("\n")

	if m.threatErr != nil {
		b.WriteString(dimStyle().Render("Not available"))
		return panelStyle().Width(width).Render(b.String())
	}
	if m.threatSummary == nil {
		b.WriteString(RenderLoadingInline(m.SpinnerFrame, "Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	ts := m.threatSummary

	if ts.TotalThreats == 0 {
		b.WriteString(highlightStyle().Render("No threats detected"))
		b.WriteString("\n\n")
		b.WriteString(dimStyle().Render("System is operating normally"))
		return panelStyle().Width(width).Render(b.String())
	}

	// Total count
	b.WriteString(dimStyle().Render("Total: "))
	b.WriteString(valueStyle().Render(fmt.Sprintf("%d", ts.TotalThreats)))
	b.WriteString("\n\n")

	// Action breakdown
	b.WriteString(subtitleStyle().Render("Actions:"))
	b.WriteString("\n")

	barWidth := width - 20
	if barWidth < 10 {
		barWidth = 10
	}

	c := theme.Colors()

	// Blocked
	if ts.BlockedCount > 0 {
		blockedPct := float64(ts.BlockedCount) / float64(ts.TotalThreats) * 100
		b.WriteString(labelStyle().Render("Blocked  "))
		b.WriteString(renderBar(blockedPct, barWidth, string(c.Success)))
		b.WriteString(highlightStyle().Render(fmt.Sprintf(" %d", ts.BlockedCount)))
		b.WriteString("\n")
	}

	// Alerted
	if ts.AlertedCount > 0 {
		alertedPct := float64(ts.AlertedCount) / float64(ts.TotalThreats) * 100
		b.WriteString(labelStyle().Render("Alerted  "))
		b.WriteString(renderBar(alertedPct, barWidth, string(c.Warning)))
		b.WriteString(warningStyle().Render(fmt.Sprintf(" %d", ts.AlertedCount)))
	}

	return panelStyle().Width(width).Render(b.String())
}

func (m SecurityDashboardModel) renderThreatSeverity(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Threats by Severity"))
	b.WriteString("\n")

	if m.threatErr != nil || m.threatSummary == nil {
		b.WriteString(RenderLoadingInline(m.SpinnerFrame, "Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	ts := m.threatSummary

	if ts.TotalThreats == 0 {
		b.WriteString(dimStyle().Render("None"))
		return panelStyle().Width(width).Render(b.String())
	}

	c := theme.Colors()
	criticalStyle := SeverityCriticalStyle
	highStyle := SeverityHighStyle
	mediumStyle := SeverityMediumStyle
	lowStyle := SeverityLowStyle

	barWidth := width - 20
	if barWidth < 10 {
		barWidth = 10
	}

	// Critical
	if ts.CriticalCount > 0 {
		pct := float64(ts.CriticalCount) / float64(ts.TotalThreats) * 100
		b.WriteString(criticalStyle.Render("Critical "))
		b.WriteString(renderBar(pct, barWidth, string(c.Critical)))
		b.WriteString(criticalStyle.Render(fmt.Sprintf(" %d", ts.CriticalCount)))
		b.WriteString("\n")
	}

	// High
	if ts.HighCount > 0 {
		pct := float64(ts.HighCount) / float64(ts.TotalThreats) * 100
		b.WriteString(highStyle.Render("High     "))
		b.WriteString(renderBar(pct, barWidth, string(c.High)))
		b.WriteString(highStyle.Render(fmt.Sprintf(" %d", ts.HighCount)))
		b.WriteString("\n")
	}

	// Medium
	if ts.MediumCount > 0 {
		pct := float64(ts.MediumCount) / float64(ts.TotalThreats) * 100
		b.WriteString(mediumStyle.Render("Medium   "))
		b.WriteString(renderBar(pct, barWidth, string(c.Medium)))
		b.WriteString(mediumStyle.Render(fmt.Sprintf(" %d", ts.MediumCount)))
		b.WriteString("\n")
	}

	// Low
	if ts.LowCount > 0 {
		pct := float64(ts.LowCount) / float64(ts.TotalThreats) * 100
		b.WriteString(lowStyle.Render("Low      "))
		b.WriteString(renderBar(pct, barWidth, string(c.Low)))
		b.WriteString(lowStyle.Render(fmt.Sprintf(" %d", ts.LowCount)))
	}

	return panelStyle().Width(width).Render(strings.TrimSuffix(b.String(), "\n"))
}

func (m SecurityDashboardModel) renderZeroHitRules(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Zero-Hit Rules"))
	b.WriteString("\n")

	if m.policyErr != nil {
		b.WriteString(dimStyle().Render("Not available"))
		return panelStyle().Width(width).Render(b.String())
	}
	if m.policies == nil {
		b.WriteString(RenderLoadingInline(m.SpinnerFrame, "Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	// Find rules with zero hits
	var zeroHitRules []models.SecurityRule
	for _, rule := range m.policies {
		if rule.HitCount == 0 && !rule.Disabled {
			zeroHitRules = append(zeroHitRules, rule)
		}
	}

	if len(zeroHitRules) == 0 {
		b.WriteString(highlightStyle().Render("All active rules have hits"))
		return panelStyle().Width(width).Render(b.String())
	}

	// Summary
	totalActive := 0
	for _, rule := range m.policies {
		if !rule.Disabled {
			totalActive++
		}
	}

	pct := float64(len(zeroHitRules)) / float64(totalActive) * 100
	b.WriteString(warningStyle().Render(fmt.Sprintf("%d", len(zeroHitRules))))
	b.WriteString(dimStyle().Render(fmt.Sprintf(" of %d rules (%.0f%%)", totalActive, pct)))
	b.WriteString("\n\n")

	// List first few zero-hit rules
	maxShow := 6
	if len(zeroHitRules) < maxShow {
		maxShow = len(zeroHitRules)
	}

	nameWidth := width - 12
	if nameWidth > 30 {
		nameWidth = 30
	}

	for i := 0; i < maxShow; i++ {
		rule := zeroHitRules[i]
		name := truncateEllipsis(rule.Name, nameWidth)

		var actionStyle lipgloss.Style
		switch rule.Action {
		case "allow":
			actionStyle = highlightStyle()
		case "deny", "drop":
			actionStyle = errorStyle()
		default:
			actionStyle = dimStyle()
		}

		b.WriteString(labelStyle().Render(fmt.Sprintf("%3d. ", rule.Position)))
		b.WriteString(valueStyle().Render(fmt.Sprintf("%-*s ", nameWidth, name)))
		b.WriteString(actionStyle.Render(rule.Action))
		if i < maxShow-1 {
			b.WriteString("\n")
		}
	}

	if len(zeroHitRules) > maxShow {
		b.WriteString("\n")
		b.WriteString(dimStyle().Render(fmt.Sprintf("... and %d more", len(zeroHitRules)-maxShow)))
	}

	return panelStyle().Width(width).Render(b.String())
}

func (m SecurityDashboardModel) renderMostHitRules(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Most-Hit Rules"))
	b.WriteString("\n")

	if m.policyErr != nil {
		b.WriteString(dimStyle().Render("Not available"))
		return panelStyle().Width(width).Render(b.String())
	}
	if m.policies == nil {
		b.WriteString(RenderLoadingInline(m.SpinnerFrame, "Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	if len(m.policies) == 0 {
		b.WriteString(dimStyle().Render("No rules"))
		return panelStyle().Width(width).Render(b.String())
	}

	// Sort by hit count (descending)
	sorted := make([]models.SecurityRule, len(m.policies))
	copy(sorted, m.policies)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].HitCount > sorted[j].HitCount
	})

	// Show top rules with hits
	maxShow := 8
	shown := 0
	nameWidth := width - 20
	if nameWidth > 25 {
		nameWidth = 25
	}

	for _, rule := range sorted {
		if shown >= maxShow {
			break
		}
		if rule.HitCount == 0 {
			continue
		}

		name := truncateEllipsis(rule.Name, nameWidth)

		var actionStyle lipgloss.Style
		switch rule.Action {
		case "allow":
			actionStyle = highlightStyle()
		case "deny", "drop":
			actionStyle = errorStyle()
		default:
			actionStyle = dimStyle()
		}

		b.WriteString(valueStyle().Render(fmt.Sprintf("%-*s ", nameWidth, name)))
		b.WriteString(actionStyle.Render(fmt.Sprintf("%-5s ", rule.Action)))
		b.WriteString(accentStyle().Render(formatNumber(rule.HitCount)))
		b.WriteString("\n")
		shown++
	}

	if shown == 0 {
		b.WriteString(dimStyle().Render("No rules have been hit"))
	}

	return panelStyle().Width(width).Render(strings.TrimSuffix(b.String(), "\n"))
}
