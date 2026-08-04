package views

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/models"
	"github.com/jp2195/pyre/internal/tui/theme"
)

// SecurityDashboardModel represents the security-focused dashboard
type SecurityDashboardModel struct {
	DashboardBase

	threatSummary  *models.ThreatSummary
	policies       []models.SecurityRule
	policiesByHits []models.SecurityRule

	threatErr error
	policyErr error
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
	m.Width = width
	m.Height = height
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
	if policies == nil {
		m.policiesByHits = nil
	} else {
		m.policiesByHits = sortPoliciesByHits(policies)
	}
	return m
}

// Update handles key events
func (m SecurityDashboardModel) Update(msg tea.Msg) (SecurityDashboardModel, tea.Cmd) {
	return m, nil
}

// HasData reports whether every panel has settled — data received or the
// fetch failed. It gates the app-wide spinner tick chain (Model.anyLoading),
// so returning true while a source is still in flight freezes that panel's
// spinner on "Loading..." forever.
func (m SecurityDashboardModel) HasData() bool {
	hasThreat := m.threatSummary != nil || m.threatErr != nil
	hasPolicies := m.policies != nil || m.policyErr != nil
	return hasThreat && hasPolicies
}

// View renders the dashboard, trimmed to the visible height. The panel stack
// is frequently taller than the terminal, so ClampToHeight windows it and
// appends a scroll indicator.
func (m SecurityDashboardModel) View() string {
	return m.ClampToHeight(m.content())
}

// ContentHeight is the untrimmed height of the panel stack, used to clamp the
// scroll offset.
func (m SecurityDashboardModel) ContentHeight() int {
	return lipgloss.Height(m.content())
}

func (m SecurityDashboardModel) content() string {
	if m.Width == 0 {
		return RenderLoadingInline(m.SpinnerFrame, "Loading...")
	}

	totalWidth, leftColWidth, rightColWidth := m.ColumnWidths()

	if m.IsNarrow() {
		return m.renderSingleColumn(totalWidth)
	}

	// Left column: threat info
	leftPanels := []string{
		m.renderThreatBreakdown(leftColWidth),
		m.renderThreatSeverity(leftColWidth),
	}

	// Right column: policy analysis
	rightPanels := []string{
		m.renderZeroHitRules(rightColWidth),
		m.renderMostHitRules(rightColWidth),
	}

	return m.RenderTwoColumn(leftPanels, rightPanels)
}

func (m SecurityDashboardModel) renderSingleColumn(width int) string {
	return m.RenderSingleColumn([]string{
		m.renderThreatBreakdown(width),
		m.renderThreatSeverity(width),
		m.renderZeroHitRules(width),
		m.renderMostHitRules(width),
	})
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
	b.WriteString(valueStyle().Render(strconv.FormatInt(ts.TotalThreats, 10)))
	b.WriteString("\n\n")

	// Action breakdown
	b.WriteString(subtitleStyle().Render("Actions:"))
	b.WriteString("\n")

	barWidth := max(width-20, 10)

	c := theme.Colors()

	// Blocked
	if ts.BlockedCount > 0 {
		blockedPct := float64(ts.BlockedCount) / float64(ts.TotalThreats) * 100
		b.WriteString(labelStyle().Render("Blocked  "))
		b.WriteString(renderBar(blockedPct, barWidth, c.Success))
		b.WriteString(highlightStyle().Render(fmt.Sprintf(" %d", ts.BlockedCount)))
		b.WriteString("\n")
	}

	// Alerted
	if ts.AlertedCount > 0 {
		alertedPct := float64(ts.AlertedCount) / float64(ts.TotalThreats) * 100
		b.WriteString(labelStyle().Render("Alerted  "))
		b.WriteString(renderBar(alertedPct, barWidth, c.Warning))
		b.WriteString(warningStyle().Render(fmt.Sprintf(" %d", ts.AlertedCount)))
	}

	return panelStyle().Width(width).Render(b.String())
}

func (m SecurityDashboardModel) renderThreatSeverity(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Threats by Severity"))
	b.WriteString("\n")

	// Distinguish "still fetching" from "fetch failed" — mirroring the
	// Threat Summary panel above. Rendering an error as a spinner left the
	// panel loading forever with no way to tell it had failed.
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
		b.WriteString(dimStyle().Render("None"))
		return panelStyle().Width(width).Render(b.String())
	}

	c := theme.Colors()
	criticalStyle := SeverityCriticalStyle
	highStyle := SeverityHighStyle
	mediumStyle := SeverityMediumStyle
	lowStyle := SeverityLowStyle

	barWidth := max(width-20, 10)

	// Critical
	if ts.CriticalCount > 0 {
		pct := float64(ts.CriticalCount) / float64(ts.TotalThreats) * 100
		b.WriteString(criticalStyle.Render("Critical "))
		b.WriteString(renderBar(pct, barWidth, c.Critical))
		b.WriteString(criticalStyle.Render(fmt.Sprintf(" %d", ts.CriticalCount)))
		b.WriteString("\n")
	}

	// High
	if ts.HighCount > 0 {
		pct := float64(ts.HighCount) / float64(ts.TotalThreats) * 100
		b.WriteString(highStyle.Render("High     "))
		b.WriteString(renderBar(pct, barWidth, c.High))
		b.WriteString(highStyle.Render(fmt.Sprintf(" %d", ts.HighCount)))
		b.WriteString("\n")
	}

	// Medium
	if ts.MediumCount > 0 {
		pct := float64(ts.MediumCount) / float64(ts.TotalThreats) * 100
		b.WriteString(mediumStyle.Render("Medium   "))
		b.WriteString(renderBar(pct, barWidth, c.Medium))
		b.WriteString(mediumStyle.Render(fmt.Sprintf(" %d", ts.MediumCount)))
		b.WriteString("\n")
	}

	// Low
	if ts.LowCount > 0 {
		pct := float64(ts.LowCount) / float64(ts.TotalThreats) * 100
		b.WriteString(lowStyle.Render("Low      "))
		b.WriteString(renderBar(pct, barWidth, c.Low))
		b.WriteString(lowStyle.Render(fmt.Sprintf(" %d", ts.LowCount)))
	}

	return panelStyle().Width(width).Render(strings.TrimSuffix(b.String(), "\n"))
}

func (m SecurityDashboardModel) renderZeroHitRules(width int) string {
	return renderZeroHitRulesPanel(m.policies, m.policyErr, m.SpinnerFrame, width, 6, 12)
}

func (m SecurityDashboardModel) renderMostHitRules(width int) string {
	return renderMostHitRulesPanel(m.policiesByHits, m.policyErr, m.SpinnerFrame, width, 8)
}
