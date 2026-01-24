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

type NATSortField int

const (
	NATSortPosition NATSortField = iota
	NATSortName
	NATSortHits
	NATSortLastHit
)

type NATPoliciesModel struct {
	TableBase
	rules    []models.NATRule
	filtered []models.NATRule
	sortBy   NATSortField
}

func NewNATPoliciesModel() NATPoliciesModel {
	return NATPoliciesModel{
		TableBase: NewTableBase("Filter NAT rules..."),
	}
}

func (m NATPoliciesModel) SetSize(width, height int) NATPoliciesModel {
	m.TableBase = m.TableBase.SetSize(width, height)
	return m
}

func (m NATPoliciesModel) SetLoading(loading bool) NATPoliciesModel {
	m.TableBase = m.TableBase.SetLoading(loading)
	return m
}

func (m NATPoliciesModel) SetRules(rules []models.NATRule, err error) NATPoliciesModel {
	m.rules = rules
	m.Err = err
	m.Loading = false
	m.Cursor = 0
	m.Offset = 0
	m.applyFilter()
	return m
}

func (m *NATPoliciesModel) applyFilter() {
	if m.FilterValue() == "" {
		m.filtered = make([]models.NATRule, len(m.rules))
		copy(m.filtered, m.rules)
	} else {
		query := strings.ToLower(m.FilterValue())
		m.filtered = nil

		for _, r := range m.rules {
			if strings.Contains(strings.ToLower(r.Name), query) ||
				strings.Contains(strings.ToLower(r.Description), query) ||
				containsAny(r.Tags, query) ||
				containsAny(r.SourceZones, query) ||
				containsAny(r.DestZones, query) ||
				containsAny(r.Sources, query) ||
				containsAny(r.Destinations, query) ||
				strings.Contains(strings.ToLower(r.TranslatedSource), query) ||
				strings.Contains(strings.ToLower(r.TranslatedDest), query) {
				m.filtered = append(m.filtered, r)
			}
		}
	}
	m.applySort()
}

func (m *NATPoliciesModel) applySort() {
	sort.Slice(m.filtered, func(i, j int) bool {
		var less bool
		switch m.sortBy {
		case NATSortName:
			less = m.filtered[i].Name < m.filtered[j].Name
		case NATSortHits:
			less = m.filtered[i].HitCount < m.filtered[j].HitCount
		case NATSortLastHit:
			less = m.filtered[i].LastHit.Before(m.filtered[j].LastHit)
		default: // NATSortPosition
			less = m.filtered[i].Position < m.filtered[j].Position
		}
		if m.SortAsc {
			return less
		}
		return !less
	})
}

func (m *NATPoliciesModel) cycleSort() {
	m.sortBy = (m.sortBy + 1) % 4
	m.SortAsc = m.sortBy == NATSortPosition || m.sortBy == NATSortName
	m.applySort()
}

func (m NATPoliciesModel) sortLabel() string {
	dir := "down"
	if m.SortAsc {
		dir = "up"
	}
	switch m.sortBy {
	case NATSortName:
		return fmt.Sprintf("Name %s", dir)
	case NATSortHits:
		return fmt.Sprintf("Hits %s", dir)
	case NATSortLastHit:
		return fmt.Sprintf("Last Hit %s", dir)
	default:
		return fmt.Sprintf("Position %s", dir)
	}
}

func (m NATPoliciesModel) Update(msg tea.Msg) (NATPoliciesModel, tea.Cmd) {
	if m.FilterMode {
		return m.updateFilter(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle NAT-specific keys first
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
		base, handled, cmd := m.TableBase.HandleNavigation(msg, len(m.filtered), visible)
		if handled {
			m.TableBase = base
			return m, cmd
		}
	}

	return m, nil
}

func (m NATPoliciesModel) updateFilter(msg tea.Msg) (NATPoliciesModel, tea.Cmd) {
	base, exited, cmd := m.TableBase.HandleFilterMode(msg)
	m.TableBase = base
	if exited {
		m.applyFilter()
	}
	return m, cmd
}

func (m NATPoliciesModel) visibleRows() int {
	rows := m.Height - 8
	if m.Expanded {
		rows -= 14
	}
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m NATPoliciesModel) View() string {
	if m.Width == 0 {
		return "Loading..."
	}

	titleStyle := ViewTitleStyle.MarginBottom(1)
	panelStyle := ViewPanelStyle.Width(m.Width - 4)

	var b strings.Builder
	title := fmt.Sprintf("NAT Policies (%d rules)", len(m.filtered))
	sortInfo := BannerInfoStyle.Render(fmt.Sprintf("  [Sort: %s | s: change | /: filter | enter: details]", m.sortLabel()))
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

	if m.Loading || m.rules == nil {
		b.WriteString(LoadingMsgStyle.Render("Loading NAT rules..."))
		return panelStyle.Render(b.String())
	}

	if len(m.filtered) == 0 {
		b.WriteString(EmptyMsgStyle.Render("No NAT rules found"))
		return panelStyle.Render(b.String())
	}

	b.WriteString(m.renderTable())

	if m.Expanded && m.Cursor < len(m.filtered) {
		b.WriteString("\n")
		b.WriteString(m.renderDetail(m.filtered[m.Cursor]))
	}

	return panelStyle.Render(b.String())
}

func (m NATPoliciesModel) renderTable() string {
	// Styles from centralized definitions
	headerStyle := DetailLabelStyle.Bold(true)
	selectedStyle := TableRowSelectedStyle.Bold(true)
	normalStyle := DetailValueStyle
	disabledStyle := TableRowDisabledStyle
	dimStyle := DetailDimStyle

	// Calculate available width
	availableWidth := m.Width - 12

	var b strings.Builder

	// Header
	header := m.formatHeaderRow(availableWidth)
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("-", minInt(availableWidth, len(header)+10))))
	b.WriteString("\n")

	visibleRows := m.visibleRows()
	end := m.Offset + visibleRows
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	for i := m.Offset; i < end; i++ {
		r := m.filtered[i]
		isSelected := i == m.Cursor

		row := m.formatRuleRow(r, availableWidth, dimStyle)

		if isSelected {
			b.WriteString(selectedStyle.Render(row))
		} else if r.Disabled {
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

func (m NATPoliciesModel) formatHeaderRow(width int) string {
	if width >= 140 {
		return fmt.Sprintf("%-4s %-20s %-16s %-16s %-18s %-18s %-10s %-10s",
			"#", "Name", "Src Zone", "Dst Zone", "Src NAT", "Dst NAT", "Hits", "Last Hit")
	} else if width >= 110 {
		return fmt.Sprintf("%-4s %-18s %-14s %-14s %-16s %-16s %-10s",
			"#", "Name", "Zones", "Service", "Src NAT", "Dst NAT", "Hits")
	} else {
		return fmt.Sprintf("%-4s %-14s %-12s %-14s %-14s %-8s",
			"#", "Name", "Zones", "Src NAT", "Dst NAT", "Hits")
	}
}

func (m NATPoliciesModel) formatRuleRow(r models.NATRule, width int, dimStyle lipgloss.Style) string {
	// Format zones
	srcZone := formatZoneCompact(r.SourceZones)
	dstZone := formatZoneCompact(r.DestZones)
	zones := srcZone + "->" + dstZone

	// Format source NAT
	srcNAT := formatSourceNAT(r)

	// Format destination NAT
	dstNAT := formatDestNAT(r)

	// Format service
	service := "any"
	if len(r.Services) > 0 && r.Services[0] != "any" {
		service = formatListCompact(r.Services, 14)
	}

	// Format hits
	hits := formatHitCount(r.HitCount)

	// Format last hit
	lastHit := formatNATLastHit(r.LastHit)

	// Format name with tags indicator
	name := r.Name
	if len(r.Tags) > 0 {
		name = name + " *"
	}

	if width >= 140 {
		return fmt.Sprintf("%-4d %-20s %-16s %-16s %-18s %-18s %-10s %-10s",
			r.Position, truncateEllipsis(name, 20),
			truncateEllipsis(srcZone, 16), truncateEllipsis(dstZone, 16),
			truncateEllipsis(srcNAT, 18), truncateEllipsis(dstNAT, 18),
			hits, lastHit)
	} else if width >= 110 {
		return fmt.Sprintf("%-4d %-18s %-14s %-14s %-16s %-16s %-10s",
			r.Position, truncateEllipsis(name, 18),
			truncateEllipsis(zones, 14), truncateEllipsis(service, 14),
			truncateEllipsis(srcNAT, 16), truncateEllipsis(dstNAT, 16), hits)
	} else {
		return fmt.Sprintf("%-4d %-14s %-12s %-14s %-14s %-8s",
			r.Position, truncateEllipsis(name, 14),
			truncateEllipsis(zones, 12),
			truncateEllipsis(srcNAT, 14), truncateEllipsis(dstNAT, 14), hits)
	}
}

func formatSourceNAT(r models.NATRule) string {
	switch r.SourceTransType {
	case models.SourceTransDynamicIPPort:
		if r.SourceInterfaceIP {
			return "DIPP:" + r.TranslatedSource
		}
		return "DIPP:" + truncateEllipsis(r.TranslatedSource, 12)
	case models.SourceTransDynamicIP:
		return "DIP:" + truncateEllipsis(r.TranslatedSource, 13)
	case models.SourceTransStaticIP:
		return "Static:" + truncateEllipsis(r.TranslatedSource, 10)
	default:
		return "None"
	}
}

func formatDestNAT(r models.NATRule) string {
	if r.TranslatedDest == "" {
		return "None"
	}
	result := truncateEllipsis(r.TranslatedDest, 14)
	if r.TranslatedDestPort != "" {
		result += ":" + r.TranslatedDestPort
	}
	return result
}

func formatNATLastHit(t time.Time) string {
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

func (m NATPoliciesModel) renderDetail(r models.NATRule) string {
	c := theme.Colors()
	boxStyle := ViewPanelStyle.
		BorderForeground(c.Primary).
		Width(m.Width - 10)

	titleStyle := ViewTitleStyle
	labelStyle := DetailLabelStyle.Width(18)
	valueStyle := DetailValueStyle
	dimValueStyle := DetailDimStyle
	tagStyle := TagStyle
	sectionStyle := DetailSectionStyle.Foreground(c.Primary)

	var b strings.Builder

	// Title with status indicators
	title := r.Name
	if r.Disabled {
		title += dimValueStyle.Render(" (disabled)")
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	// Tags
	if len(r.Tags) > 0 {
		b.WriteString(tagStyle.Render("Tags: " + strings.Join(r.Tags, ", ")))
		b.WriteString("\n")
	}

	// Description
	if r.Description != "" {
		b.WriteString(dimValueStyle.Render(r.Description))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Original Packet Section
	b.WriteString(sectionStyle.Render("Original Packet (Match)"))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Source Zones:") + " " + valueStyle.Render(formatListFull(r.SourceZones)) + "\n")
	b.WriteString(labelStyle.Render("Source Addresses:") + " " + valueStyle.Render(formatListFull(r.Sources)) + "\n")
	b.WriteString(labelStyle.Render("Dest Zones:") + " " + valueStyle.Render(formatListFull(r.DestZones)) + "\n")
	b.WriteString(labelStyle.Render("Dest Addresses:") + " " + valueStyle.Render(formatListFull(r.Destinations)) + "\n")
	b.WriteString(labelStyle.Render("Service:") + " " + valueStyle.Render(formatListFull(r.Services)) + "\n")
	if r.DestInterface != "" && r.DestInterface != "any" {
		b.WriteString(labelStyle.Render("Dest Interface:") + " " + valueStyle.Render(r.DestInterface) + "\n")
	}

	// Translated Packet Section
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Translated Packet"))
	b.WriteString("\n")

	// Source Translation
	srcTransLabel := "Source Translation:"
	switch r.SourceTransType {
	case models.SourceTransDynamicIPPort:
		transType := "Dynamic IP and Port"
		if r.SourceInterfaceIP {
			transType += " (Interface)"
		}
		b.WriteString(labelStyle.Render(srcTransLabel) + " " + valueStyle.Render(transType) + "\n")
		b.WriteString(labelStyle.Render("  Translated To:") + " " + valueStyle.Render(r.TranslatedSource) + "\n")
	case models.SourceTransDynamicIP:
		b.WriteString(labelStyle.Render(srcTransLabel) + " " + valueStyle.Render("Dynamic IP") + "\n")
		b.WriteString(labelStyle.Render("  Translated To:") + " " + valueStyle.Render(r.TranslatedSource) + "\n")
	case models.SourceTransStaticIP:
		b.WriteString(labelStyle.Render(srcTransLabel) + " " + valueStyle.Render("Static IP") + "\n")
		b.WriteString(labelStyle.Render("  Translated To:") + " " + valueStyle.Render(r.TranslatedSource) + "\n")
	default:
		b.WriteString(labelStyle.Render(srcTransLabel) + " " + dimValueStyle.Render("None") + "\n")
	}

	// Destination Translation
	dstTransLabel := "Dest Translation:"
	if r.TranslatedDest != "" {
		b.WriteString(labelStyle.Render(dstTransLabel) + " " + valueStyle.Render(r.TranslatedDest) + "\n")
		if r.TranslatedDestPort != "" {
			b.WriteString(labelStyle.Render("  Translated Port:") + " " + valueStyle.Render(r.TranslatedDestPort) + "\n")
		}
	} else {
		b.WriteString(labelStyle.Render(dstTransLabel) + " " + dimValueStyle.Render("None") + "\n")
	}

	// Usage Stats Section
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Usage Statistics"))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Hit Count:") + " " + valueStyle.Render(formatHitCountFull(r.HitCount)) + "\n")
	b.WriteString(labelStyle.Render("Last Hit:") + " " + valueStyle.Render(formatTimestamp(r.LastHit)) + "\n")
	if !r.FirstHit.IsZero() {
		b.WriteString(labelStyle.Render("First Hit:") + " " + valueStyle.Render(formatTimestamp(r.FirstHit)) + "\n")
	}

	return boxStyle.Render(b.String())
}
