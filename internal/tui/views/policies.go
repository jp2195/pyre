package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jp2195/pyre/internal/models"
	"github.com/jp2195/pyre/internal/tui/theme"
)

type PolicySortField int

const (
	PolicySortPosition PolicySortField = iota
	PolicySortName
	PolicySortHits
	PolicySortLastHit
)

type PoliciesModel struct {
	TableBase
	policies []models.SecurityRule
	filtered []models.SecurityRule
	sortBy   PolicySortField
}

func NewPoliciesModel() PoliciesModel {
	return PoliciesModel{
		TableBase: NewTableBase("Filter rules..."),
	}
}

func (m PoliciesModel) SetSize(width, height int) PoliciesModel {
	m.TableBase = m.TableBase.SetSize(width, height)

	// Clamp cursor to valid range after resize
	count := len(m.filtered)
	if m.Cursor >= count && count > 0 {
		m.Cursor = count - 1
	}

	// Adjust offset to keep cursor visible
	visibleRows := m.visibleRows()
	if visibleRows > 0 && m.Cursor >= m.Offset+visibleRows {
		m.Offset = m.Cursor - visibleRows + 1
	}

	return m
}

func (m PoliciesModel) SetLoading(loading bool) PoliciesModel {
	m.TableBase = m.TableBase.SetLoading(loading)
	return m
}

// HasData returns true if policies have been loaded.
func (m PoliciesModel) HasData() bool {
	return m.policies != nil
}

func (m PoliciesModel) SetPolicies(policies []models.SecurityRule, err error) PoliciesModel {
	m.policies = policies
	m.Err = err
	m.Loading = false
	m.Cursor = 0
	m.Offset = 0
	m.applyFilter()
	return m
}

func (m *PoliciesModel) applyFilter() {
	if m.FilterValue() == "" {
		m.filtered = make([]models.SecurityRule, len(m.policies))
		copy(m.filtered, m.policies)
	} else {
		query := strings.ToLower(m.FilterValue())
		m.filtered = nil

		for _, p := range m.policies {
			if strings.Contains(strings.ToLower(p.Name), query) ||
				strings.Contains(strings.ToLower(p.Description), query) ||
				containsAny(p.Tags, query) ||
				containsAny(p.SourceZones, query) ||
				containsAny(p.DestZones, query) ||
				containsAny(p.Sources, query) ||
				containsAny(p.Destinations, query) ||
				containsAny(p.Applications, query) ||
				containsAny(p.Services, query) {
				m.filtered = append(m.filtered, p)
			}
		}
	}
	m.applySort()
}

func (m *PoliciesModel) applySort() {
	sort.Slice(m.filtered, func(i, j int) bool {
		var less bool
		switch m.sortBy {
		case PolicySortName:
			less = m.filtered[i].Name < m.filtered[j].Name
		case PolicySortHits:
			less = m.filtered[i].HitCount < m.filtered[j].HitCount
		case PolicySortLastHit:
			less = m.filtered[i].LastHit.Before(m.filtered[j].LastHit)
		default: // PolicySortPosition
			less = m.filtered[i].Position < m.filtered[j].Position
		}
		if m.SortAsc {
			return less
		}
		return !less
	})
}

func (m *PoliciesModel) cycleSort() {
	m.sortBy = (m.sortBy + 1) % 4
	m.SortAsc = m.sortBy == PolicySortPosition || m.sortBy == PolicySortName
	m.applySort()
}

func (m PoliciesModel) sortLabel() string {
	dir := "↓"
	if m.SortAsc {
		dir = "↑"
	}
	switch m.sortBy {
	case PolicySortName:
		return fmt.Sprintf("Name %s", dir)
	case PolicySortHits:
		return fmt.Sprintf("Hits %s", dir)
	case PolicySortLastHit:
		return fmt.Sprintf("Last Hit %s", dir)
	default:
		return fmt.Sprintf("Position %s", dir)
	}
}

func containsAny(items []string, query string) bool {
	for _, item := range items {
		if strings.Contains(strings.ToLower(item), query) {
			return true
		}
	}
	return false
}

func (m PoliciesModel) Update(msg tea.Msg) (PoliciesModel, tea.Cmd) {
	if m.FilterMode {
		return m.updateFilter(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle policy-specific keys first
		switch msg.String() {
		case "esc":
			if m.HandleCollapseIfExpanded() {
				return m, nil
			}
			if m.HandleClearFilter() {
				m.applyFilter()
			}
			return m, nil
		case "s":
			m.cycleSort()
			m.Cursor = 0
			m.Offset = 0
			return m, nil
		}

		// Delegate to TableBase for common navigation
		visible := m.visibleRows()
		base, handled, cmd := m.HandleNavigation(msg, len(m.filtered), visible)
		if handled {
			m.TableBase = base
			return m, cmd
		}
	}

	return m, nil
}

func (m PoliciesModel) updateFilter(msg tea.Msg) (PoliciesModel, tea.Cmd) {
	base, exited, cmd := m.HandleFilterMode(msg)
	m.TableBase = base
	if exited {
		m.applyFilter()
	}
	return m, cmd
}

func (m PoliciesModel) visibleRows() int {
	rows := m.Height - 8
	if m.Expanded {
		rows -= 14
	}
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m PoliciesModel) View() string {
	if m.Width == 0 {
		return RenderLoadingInline(m.SpinnerFrame, "Loading...")
	}

	titleStyle := ViewTitleStyle.MarginBottom(1)
	panelStyle := ViewPanelStyle.Width(m.Width - 4)

	var b strings.Builder
	title := "Security Policies"
	sortInfo := BannerInfoStyle.Render(fmt.Sprintf(" [%d rules | Sort: %s | s: change | /: filter | enter: details]", len(m.filtered), m.sortLabel()))
	b.WriteString(titleStyle.Render(title) + sortInfo)
	b.WriteString("\n")

	if m.FilterMode {
		b.WriteString(FilterBorderStyle.Render(m.Filter.View()))
		b.WriteString("\n\n")
	} else if m.IsFiltered() {
		filterInfo := FilterActiveStyle.Render(fmt.Sprintf("Filtered: \"%s\"", m.FilterValue()))
		clearHint := FilterClearHintStyle.Render(" (esc to clear)")
		b.WriteString(filterInfo + clearHint)
		b.WriteString("\n\n")
	}

	if m.Err != nil {
		b.WriteString(ErrorMsgStyle.Render("Error: " + m.Err.Error()))
		return panelStyle.Render(b.String())
	}

	if m.Loading || m.policies == nil {
		b.WriteString(RenderLoadingInline(m.SpinnerFrame, "Loading policies..."))
		return panelStyle.Render(b.String())
	}

	if len(m.filtered) == 0 {
		b.WriteString(EmptyMsgStyle.Render("No policies found"))
		return panelStyle.Render(b.String())
	}

	b.WriteString(m.renderTable())

	if m.Expanded && m.Cursor < len(m.filtered) {
		b.WriteString("\n")
		b.WriteString(m.renderDetail(m.filtered[m.Cursor]))
	}

	return panelStyle.Render(b.String())
}

func (m PoliciesModel) renderTable() string {
	// Styles from centralized definitions
	headerStyle := DetailLabelStyle.Bold(true)
	selectedStyle := TableRowSelectedStyle.Bold(true)
	normalStyle := DetailValueStyle
	disabledStyle := TableRowDisabledStyle
	dimStyle := DetailDimStyle
	allowStyle := ActionAllowStyle
	denyStyle := ActionDenyStyle
	tagStyle := TagStyle

	// Calculate available width
	availableWidth := m.Width - 12 // Account for padding and borders

	var b strings.Builder

	// Header
	header := m.formatHeaderRow(availableWidth)
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", minInt(availableWidth, len(header)+10))))
	b.WriteString("\n")

	visibleRows := m.visibleRows()
	end := m.Offset + visibleRows
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	for i := m.Offset; i < end; i++ {
		p := m.filtered[i]
		isSelected := i == m.Cursor

		row := m.formatRuleRow(p, availableWidth, allowStyle, denyStyle, tagStyle, dimStyle)

		if isSelected {
			b.WriteString(selectedStyle.Render(row))
		} else if p.Disabled {
			b.WriteString(disabledStyle.Render(row))
		} else {
			b.WriteString(normalStyle.Render(row))
		}
		b.WriteString("\n")
	}

	// Show scroll indicator if needed
	if len(m.filtered) > visibleRows {
		scrollInfo := fmt.Sprintf("  Showing %d-%d of %d", m.Offset+1, end, len(m.filtered))
		b.WriteString(dimStyle.Render(scrollInfo))
	}

	return b.String()
}

func (m PoliciesModel) formatHeaderRow(width int) string {
	// Adaptive column layout based on width
	if width >= 140 {
		return fmt.Sprintf("%-4s %-24s %-8s %-20s %-18s %-16s %-10s %-10s",
			"#", "Name", "Action", "Source → Dest Zone", "Application", "Service", "Hits", "Last Hit")
	} else if width >= 110 {
		return fmt.Sprintf("%-4s %-20s %-8s %-18s %-14s %-10s %-10s",
			"#", "Name", "Action", "Zones", "Application", "Hits", "Last Hit")
	} else {
		return fmt.Sprintf("%-4s %-16s %-7s %-14s %-10s %-8s",
			"#", "Name", "Action", "Zones", "App", "Hits")
	}
}

func (m PoliciesModel) formatRuleRow(p models.SecurityRule, width int, allowStyle, denyStyle, tagStyle, dimStyle lipgloss.Style) string {
	// Format action with color
	action := strings.ToUpper(p.Action)
	if len(action) > 8 {
		action = action[:7] + "…"
	}

	// Format zones
	srcZone := formatZoneCompact(p.SourceZones)
	dstZone := formatZoneCompact(p.DestZones)
	zones := srcZone + "→" + dstZone

	// Format apps
	apps := formatListCompact(p.Applications, 14)

	// Format services
	services := formatListCompact(p.Services, 14)

	// Format hits
	hits := formatHitCount(p.HitCount)

	// Format last hit
	lastHit := formatLastHit(p.LastHit)

	// Format name with tags indicator
	name := p.Name
	if len(p.Tags) > 0 {
		name = name + " •"
	}

	if width >= 140 {
		return fmt.Sprintf("%-4d %-24s %-8s %-20s %-18s %-16s %-10s %-10s",
			p.Position, truncateStr(name, 24), action,
			truncateStr(zones, 20), truncateStr(apps, 18),
			truncateStr(services, 16), hits, lastHit)
	} else if width >= 110 {
		return fmt.Sprintf("%-4d %-20s %-8s %-18s %-14s %-10s %-10s",
			p.Position, truncateStr(name, 20), action,
			truncateStr(zones, 18), truncateStr(apps, 14), hits, lastHit)
	} else {
		return fmt.Sprintf("%-4d %-16s %-7s %-14s %-10s %-8s",
			p.Position, truncateStr(name, 16), truncateStr(action, 7),
			truncateStr(zones, 14), truncateStr(apps, 10), hits)
	}
}

func (m PoliciesModel) renderDetail(p models.SecurityRule) string {
	c := theme.Colors()
	boxStyle := ViewPanelStyle.
		BorderForeground(c.Primary).
		Width(m.Width - 10)

	titleStyle := ViewTitleStyle
	labelStyle := DetailLabelStyle.Width(16)
	valueStyle := DetailValueStyle
	dimValueStyle := DetailDimStyle
	tagStyle := TagStyle
	sectionStyle := DetailSectionStyle.Foreground(c.Primary)

	var b strings.Builder

	// Title with status indicators
	title := p.Name
	if p.Disabled {
		title += dimValueStyle.Render(" (disabled)")
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	// Tags
	if len(p.Tags) > 0 {
		b.WriteString(tagStyle.Render("Tags: " + strings.Join(p.Tags, ", ")))
		b.WriteString("\n")
	}

	// Description
	if p.Description != "" {
		b.WriteString(dimValueStyle.Render(p.Description))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Source/Destination Section
	b.WriteString(sectionStyle.Render("Traffic Match"))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Source Zones:") + " " + valueStyle.Render(formatListFull(p.SourceZones)) + "\n")
	b.WriteString(labelStyle.Render("Source Addr:") + " " + formatAddresses(p.Sources, p.NegateSource, valueStyle, dimValueStyle) + "\n")
	if len(p.SourceUsers) > 0 && (len(p.SourceUsers) != 1 || p.SourceUsers[0] != "any") {
		b.WriteString(labelStyle.Render("Source Users:") + " " + valueStyle.Render(formatListFull(p.SourceUsers)) + "\n")
	}
	b.WriteString(labelStyle.Render("Dest Zones:") + " " + valueStyle.Render(formatListFull(p.DestZones)) + "\n")
	b.WriteString(labelStyle.Render("Dest Addr:") + " " + formatAddresses(p.Destinations, p.NegateDest, valueStyle, dimValueStyle) + "\n")

	// Application/Service Section
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Application/Service"))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Applications:") + " " + valueStyle.Render(formatListFull(p.Applications)) + "\n")
	b.WriteString(labelStyle.Render("Services:") + " " + valueStyle.Render(formatListFull(p.Services)) + "\n")
	if len(p.URLCategories) > 0 && (len(p.URLCategories) != 1 || p.URLCategories[0] != "any") {
		b.WriteString(labelStyle.Render("URL Categories:") + " " + valueStyle.Render(formatListFull(p.URLCategories)) + "\n")
	}

	// Action & Profiles Section
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Action & Profiles"))
	b.WriteString("\n")
	actionStyle := ActionStyle(p.Action)
	b.WriteString(labelStyle.Render("Action:") + " " + actionStyle.Render(strings.ToUpper(p.Action)) + "\n")

	// Security profiles
	if p.Profile != "" {
		b.WriteString(labelStyle.Render("Profile Group:") + " " + valueStyle.Render(p.Profile) + "\n")
	} else if hasProfiles(p) {
		profiles := []string{}
		if p.AntivirusProfile != "" {
			profiles = append(profiles, "AV:"+p.AntivirusProfile)
		}
		if p.VulnerabilityProfile != "" {
			profiles = append(profiles, "Vuln:"+p.VulnerabilityProfile)
		}
		if p.SpywareProfile != "" {
			profiles = append(profiles, "Spy:"+p.SpywareProfile)
		}
		if p.URLFilteringProfile != "" {
			profiles = append(profiles, "URL:"+p.URLFilteringProfile)
		}
		if p.WildFireProfile != "" {
			profiles = append(profiles, "WF:"+p.WildFireProfile)
		}
		b.WriteString(labelStyle.Render("Profiles:") + " " + valueStyle.Render(strings.Join(profiles, ", ")) + "\n")
	}

	// Logging
	logSettings := []string{}
	if p.LogStart {
		logSettings = append(logSettings, "start")
	}
	if p.LogEnd {
		logSettings = append(logSettings, "end")
	}
	if len(logSettings) > 0 {
		logStr := strings.Join(logSettings, ", ")
		if p.LogForwarding != "" {
			logStr += " → " + p.LogForwarding
		}
		b.WriteString(labelStyle.Render("Logging:") + " " + valueStyle.Render(logStr) + "\n")
	}

	// Usage Stats Section
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Usage Statistics"))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Hit Count:") + " " + valueStyle.Render(formatHitCountFull(p.HitCount)) + "\n")
	b.WriteString(labelStyle.Render("Last Hit:") + " " + valueStyle.Render(formatTimestamp(p.LastHit)) + "\n")
	if !p.FirstHit.IsZero() {
		b.WriteString(labelStyle.Render("First Hit:") + " " + valueStyle.Render(formatTimestamp(p.FirstHit)) + "\n")
	}

	return boxStyle.Render(b.String())
}

// Helper functions

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

func formatListFull(items []string) string {
	if len(items) == 0 {
		return "any"
	}
	return strings.Join(items, ", ")
}

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

func hasProfiles(p models.SecurityRule) bool {
	return p.AntivirusProfile != "" || p.VulnerabilityProfile != "" ||
		p.SpywareProfile != "" || p.URLFilteringProfile != "" ||
		p.FileBlockingProfile != "" || p.WildFireProfile != ""
}

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

func formatTimestamp(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}
	return t.Format("2006-01-02 15:04:05")
}

func truncateStr(s string, maxLen int) string {
	return truncateEllipsis(s, maxLen)
}
