package views

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jp2195/pyre/internal/models"
)

// ConfigDashboardModel represents the configuration-focused dashboard
type ConfigDashboardModel struct {
	policies       []models.SecurityRule
	pendingChanges []models.PendingChange

	policyErr  error
	changesErr error

	width  int
	height int
}

// NewConfigDashboardModel creates a new config dashboard model
func NewConfigDashboardModel() ConfigDashboardModel {
	return ConfigDashboardModel{}
}

// SetSize sets the terminal dimensions
func (m ConfigDashboardModel) SetSize(width, height int) ConfigDashboardModel {
	m.width = width
	m.height = height
	return m
}

// SetPolicies sets the security policies data
func (m ConfigDashboardModel) SetPolicies(policies []models.SecurityRule, err error) ConfigDashboardModel {
	m.policies = policies
	m.policyErr = err
	return m
}

// SetPendingChanges sets the pending changes data
func (m ConfigDashboardModel) SetPendingChanges(changes []models.PendingChange, err error) ConfigDashboardModel {
	m.pendingChanges = changes
	m.changesErr = err
	return m
}

// Update handles key events
func (m ConfigDashboardModel) Update(msg tea.Msg) (ConfigDashboardModel, tea.Cmd) {
	return m, nil
}

// View renders the config dashboard
func (m ConfigDashboardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	totalWidth := m.width - 4
	leftColWidth := totalWidth / 2
	rightColWidth := totalWidth - leftColWidth - 2

	if leftColWidth < 35 {
		return m.renderSingleColumn(totalWidth)
	}

	// Left column: object counts and pending changes
	leftPanels := []string{
		m.renderPolicyStats(leftColWidth),
		m.renderPendingChanges(leftColWidth),
	}
	leftCol := lipgloss.JoinVertical(lipgloss.Left, leftPanels...)

	// Right column: rule analysis
	rightPanels := []string{
		m.renderZeroHitRules(rightColWidth),
		m.renderMostHitRules(rightColWidth),
	}
	rightCol := lipgloss.JoinVertical(lipgloss.Left, rightPanels...)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftCol, "  ", rightCol)
}

func (m ConfigDashboardModel) renderSingleColumn(width int) string {
	panels := []string{
		m.renderPolicyStats(width),
		m.renderPendingChanges(width),
		m.renderZeroHitRules(width),
		m.renderMostHitRules(width),
	}
	return lipgloss.JoinVertical(lipgloss.Left, panels...)
}

func (m ConfigDashboardModel) renderPolicyStats(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Policy Statistics"))
	b.WriteString("\n")

	if m.policyErr != nil {
		b.WriteString(dimStyle().Render("Not available"))
		return panelStyle().Width(width).Render(b.String())
	}
	if m.policies == nil {
		b.WriteString(dimStyle().Render("Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	if len(m.policies) == 0 {
		b.WriteString(dimStyle().Render("No security rules"))
		return panelStyle().Width(width).Render(b.String())
	}

	// Count stats
	totalRules := len(m.policies)
	enabledRules := 0
	allowRules := 0
	denyRules := 0
	zeroHitRules := 0
	var totalHits int64 = 0

	for _, rule := range m.policies {
		if !rule.Disabled {
			enabledRules++
		}
		switch rule.Action {
		case "allow":
			allowRules++
		case "deny", "drop":
			denyRules++
		}
		if rule.HitCount == 0 && !rule.Disabled {
			zeroHitRules++
		}
		totalHits += rule.HitCount
	}

	// Summary
	b.WriteString(valueStyle().Render(fmt.Sprintf("%d", totalRules)))
	b.WriteString(dimStyle().Render(" total rules ("))
	b.WriteString(highlightStyle().Render(fmt.Sprintf("%d", enabledRules)))
	b.WriteString(dimStyle().Render(" enabled)"))
	b.WriteString("\n\n")

	// Breakdown
	labelWidth := 12
	b.WriteString(labelStyle().Render(fmt.Sprintf("%-*s", labelWidth, "Allow:")))
	b.WriteString(highlightStyle().Render(fmt.Sprintf("%d", allowRules)))
	b.WriteString("\n")

	b.WriteString(labelStyle().Render(fmt.Sprintf("%-*s", labelWidth, "Deny/Drop:")))
	b.WriteString(errorStyle().Render(fmt.Sprintf("%d", denyRules)))
	b.WriteString("\n")

	b.WriteString(labelStyle().Render(fmt.Sprintf("%-*s", labelWidth, "Zero-hit:")))
	if zeroHitRules > 0 {
		b.WriteString(warningStyle().Render(fmt.Sprintf("%d", zeroHitRules)))
	} else {
		b.WriteString(highlightStyle().Render("0"))
	}
	b.WriteString("\n")

	b.WriteString(labelStyle().Render(fmt.Sprintf("%-*s", labelWidth, "Total hits:")))
	b.WriteString(accentStyle().Render(formatNumber(totalHits)))

	return panelStyle().Width(width).Render(b.String())
}

func (m ConfigDashboardModel) renderPendingChanges(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Pending Changes"))
	b.WriteString("\n")

	if m.changesErr != nil {
		b.WriteString(dimStyle().Render("Not available"))
		return panelStyle().Width(width).Render(b.String())
	}
	if m.pendingChanges == nil {
		b.WriteString(dimStyle().Render("Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	if len(m.pendingChanges) == 0 {
		b.WriteString(highlightStyle().Render("No pending changes"))
		b.WriteString("\n")
		b.WriteString(dimStyle().Render("Configuration is committed"))
		return panelStyle().Width(width).Render(b.String())
	}

	// Count
	b.WriteString(warningStyle().Render(fmt.Sprintf("%d", len(m.pendingChanges))))
	b.WriteString(dimStyle().Render(" uncommitted changes"))
	b.WriteString("\n\n")

	// Group by user
	userChanges := make(map[string]int)
	for _, change := range m.pendingChanges {
		user := change.User
		if user == "" {
			user = "unknown"
		}
		userChanges[user]++
	}

	if len(userChanges) > 1 {
		b.WriteString(subtitleStyle().Render("By User:"))
		b.WriteString("\n")
		for user, count := range userChanges {
			userName := truncateEllipsis(user, 15)
			b.WriteString(labelStyle().Render(fmt.Sprintf("  %-15s ", userName)))
			b.WriteString(valueStyle().Render(fmt.Sprintf("%d", count)))
			b.WriteString("\n")
		}
	}

	// Show recent changes
	maxShow := 4
	if len(m.pendingChanges) < maxShow {
		maxShow = len(m.pendingChanges)
	}

	if maxShow > 0 {
		b.WriteString("\n")
		b.WriteString(subtitleStyle().Render("Recent:"))
		b.WriteString("\n")

		for i := 0; i < maxShow; i++ {
			change := m.pendingChanges[i]

			typeStyle := dimStyle()
			switch change.Type {
			case "add":
				typeStyle = highlightStyle()
			case "edit":
				typeStyle = accentStyle()
			case "delete":
				typeStyle = errorStyle()
			}

			desc := change.Description
			if desc == "" {
				desc = change.Location
			}
			desc = truncateEllipsis(desc, width-15)

			b.WriteString(typeStyle.Render(fmt.Sprintf("  %-6s ", change.Type)))
			b.WriteString(labelStyle().Render(desc))
			if i < maxShow-1 {
				b.WriteString("\n")
			}
		}
	}

	return panelStyle().Width(width).Render(b.String())
}

func (m ConfigDashboardModel) renderZeroHitRules(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Zero-Hit Rules"))
	b.WriteString("\n")

	if m.policyErr != nil {
		b.WriteString(dimStyle().Render("Not available"))
		return panelStyle().Width(width).Render(b.String())
	}
	if m.policies == nil {
		b.WriteString(dimStyle().Render("Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	// Find zero-hit rules
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

	// List rules
	maxShow := 8
	if len(zeroHitRules) < maxShow {
		maxShow = len(zeroHitRules)
	}

	nameWidth := width - 15
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

func (m ConfigDashboardModel) renderMostHitRules(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Most-Hit Rules"))
	b.WriteString("\n")

	if m.policyErr != nil {
		b.WriteString(dimStyle().Render("Not available"))
		return panelStyle().Width(width).Render(b.String())
	}
	if m.policies == nil {
		b.WriteString(dimStyle().Render("Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	if len(m.policies) == 0 {
		b.WriteString(dimStyle().Render("No rules"))
		return panelStyle().Width(width).Render(b.String())
	}

	// Sort by hit count
	sorted := make([]models.SecurityRule, len(m.policies))
	copy(sorted, m.policies)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].HitCount > sorted[j].HitCount
	})

	// Show top rules
	maxShow := 10
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
