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

type IPSecSortField int

const (
	IPSecSortName IPSecSortField = iota
	IPSecSortGateway
	IPSecSortState
	IPSecSortTraffic
)

type IPSecTunnelsModel struct {
	TableBase
	tunnels  []models.IPSecTunnel
	filtered []models.IPSecTunnel
	sortBy   IPSecSortField
}

func NewIPSecTunnelsModel() IPSecTunnelsModel {
	return IPSecTunnelsModel{
		TableBase: NewTableBase("Filter tunnels..."),
	}
}

func (m IPSecTunnelsModel) SetSize(width, height int) IPSecTunnelsModel {
	m.TableBase = m.TableBase.SetSize(width, height)

	count := len(m.filtered)
	if m.Cursor >= count && count > 0 {
		m.Cursor = count - 1
	}

	visibleRows := m.visibleRows()
	if visibleRows > 0 && m.Cursor >= m.Offset+visibleRows {
		m.Offset = m.Cursor - visibleRows + 1
	}

	return m
}

func (m IPSecTunnelsModel) SetLoading(loading bool) IPSecTunnelsModel {
	m.TableBase = m.TableBase.SetLoading(loading)
	return m
}

// HasData returns true if tunnel data has been loaded.
func (m IPSecTunnelsModel) HasData() bool {
	return m.tunnels != nil
}

func (m IPSecTunnelsModel) SetTunnels(tunnels []models.IPSecTunnel, err error) IPSecTunnelsModel {
	m.tunnels = tunnels
	m.Err = err
	m.Loading = false
	m.Cursor = 0
	m.Offset = 0
	m.applyFilter()
	return m
}

func (m *IPSecTunnelsModel) applyFilter() {
	if m.FilterValue() == "" {
		m.filtered = make([]models.IPSecTunnel, len(m.tunnels))
		copy(m.filtered, m.tunnels)
	} else {
		query := strings.ToLower(m.FilterValue())
		m.filtered = nil

		for _, t := range m.tunnels {
			if strings.Contains(strings.ToLower(t.Name), query) ||
				strings.Contains(strings.ToLower(t.Gateway), query) ||
				strings.Contains(strings.ToLower(t.State), query) ||
				strings.Contains(strings.ToLower(t.Protocol), query) ||
				strings.Contains(strings.ToLower(t.Encryption), query) {
				m.filtered = append(m.filtered, t)
			}
		}
	}
	m.applySort()
}

func (m *IPSecTunnelsModel) applySort() {
	sort.Slice(m.filtered, func(i, j int) bool {
		var less bool
		switch m.sortBy {
		case IPSecSortGateway:
			less = m.filtered[i].Gateway < m.filtered[j].Gateway
		case IPSecSortState:
			less = m.filtered[i].State < m.filtered[j].State
		case IPSecSortTraffic:
			totalI := m.filtered[i].BytesIn + m.filtered[i].BytesOut
			totalJ := m.filtered[j].BytesIn + m.filtered[j].BytesOut
			less = totalI < totalJ
		default: // IPSecSortName
			less = m.filtered[i].Name < m.filtered[j].Name
		}
		if m.SortAsc {
			return less
		}
		return !less
	})
}

func (m *IPSecTunnelsModel) cycleSort() {
	m.sortBy = (m.sortBy + 1) % 4
	m.SortAsc = m.sortBy == IPSecSortName || m.sortBy == IPSecSortGateway
	m.applySort()
}

func (m IPSecTunnelsModel) sortLabel() string {
	dir := "↓"
	if m.SortAsc {
		dir = "↑"
	}
	switch m.sortBy {
	case IPSecSortGateway:
		return fmt.Sprintf("Gateway %s", dir)
	case IPSecSortState:
		return fmt.Sprintf("State %s", dir)
	case IPSecSortTraffic:
		return fmt.Sprintf("Traffic %s", dir)
	default:
		return fmt.Sprintf("Name %s", dir)
	}
}

func (m IPSecTunnelsModel) Update(msg tea.Msg) (IPSecTunnelsModel, tea.Cmd) {
	if m.FilterMode {
		return m.updateFilter(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
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

		visible := m.visibleRows()
		base, handled, cmd := m.HandleNavigation(msg, len(m.filtered), visible)
		if handled {
			m.TableBase = base
			return m, cmd
		}
	}

	return m, nil
}

func (m IPSecTunnelsModel) updateFilter(msg tea.Msg) (IPSecTunnelsModel, tea.Cmd) {
	base, exited, cmd := m.HandleFilterMode(msg)
	m.TableBase = base
	if exited {
		m.applyFilter()
	}
	return m, cmd
}

func (m IPSecTunnelsModel) visibleRows() int {
	rows := m.Height - 8
	if m.Expanded {
		rows -= 14
	}
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m IPSecTunnelsModel) View() string {
	if m.Width == 0 {
		return RenderLoadingInline(m.SpinnerFrame, "Loading...")
	}

	titleStyle := ViewTitleStyle.MarginBottom(1)
	panelStyle := ViewPanelStyle.Width(m.Width - 4)

	var b strings.Builder
	title := "IPSec Tunnels"
	sortInfo := BannerInfoStyle.Render(fmt.Sprintf(" [%d tunnels | Sort: %s | s: change | /: filter | enter: details]", len(m.filtered), m.sortLabel()))
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

	if m.Loading || m.tunnels == nil {
		b.WriteString(RenderLoadingInline(m.SpinnerFrame, "Loading IPSec tunnels..."))
		return panelStyle.Render(b.String())
	}

	if len(m.filtered) == 0 {
		b.WriteString(EmptyMsgStyle.Render("No IPSec tunnels found"))
		return panelStyle.Render(b.String())
	}

	b.WriteString(m.renderTable())

	if m.Expanded && m.Cursor < len(m.filtered) {
		b.WriteString("\n")
		b.WriteString(m.renderDetail(m.filtered[m.Cursor]))
	}

	return panelStyle.Render(b.String())
}

func (m IPSecTunnelsModel) renderTable() string {
	headerStyle := DetailLabelStyle.Bold(true)
	selectedStyle := TableRowSelectedStyle.Bold(true)
	dimStyle := DetailDimStyle

	availableWidth := m.Width - 12

	var b strings.Builder

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
		t := m.filtered[i]
		isSelected := i == m.Cursor

		row := m.formatTunnelRow(t, availableWidth)

		if isSelected {
			b.WriteString(selectedStyle.Render(row))
		} else {
			b.WriteString(m.stateStyledRow(row, t.State))
		}
		b.WriteString("\n")
	}

	if len(m.filtered) > visibleRows {
		scrollInfo := fmt.Sprintf("  Showing %d-%d of %d", m.Offset+1, end, len(m.filtered))
		b.WriteString(dimStyle.Render(scrollInfo))
	}

	return b.String()
}

func (m IPSecTunnelsModel) formatHeaderRow(width int) string {
	if width >= 120 {
		return fmt.Sprintf("%-4s %-20s %-18s %-6s %-8s %-10s %-10s %-10s %-10s",
			"", "Name", "Gateway", "State", "Proto", "Encrypt", "In", "Out", "Uptime")
	} else if width >= 90 {
		return fmt.Sprintf("%-4s %-18s %-15s %-6s %-10s %-10s %-10s",
			"", "Name", "Gateway", "State", "In", "Out", "Uptime")
	}
	return fmt.Sprintf("%-4s %-16s %-15s %-6s %-10s",
		"", "Name", "Gateway", "State", "Traffic")
}

func (m IPSecTunnelsModel) formatTunnelRow(t models.IPSecTunnel, width int) string {
	stateIcon := stateIndicator(t.State)
	totalTraffic := formatBytes(t.BytesIn + t.BytesOut)

	if width >= 120 {
		return fmt.Sprintf(" %s  %-20s %-18s %-6s %-8s %-10s %-10s %-10s %-10s",
			stateIcon,
			truncateEllipsis(t.Name, 20),
			truncateEllipsis(t.Gateway, 18),
			t.State,
			truncateEllipsis(t.Protocol, 8),
			truncateEllipsis(t.Encryption, 10),
			formatBytes(t.BytesIn),
			formatBytes(t.BytesOut),
			truncateEllipsis(t.Uptime, 10))
	} else if width >= 90 {
		return fmt.Sprintf(" %s  %-18s %-15s %-6s %-10s %-10s %-10s",
			stateIcon,
			truncateEllipsis(t.Name, 18),
			truncateEllipsis(t.Gateway, 15),
			t.State,
			formatBytes(t.BytesIn),
			formatBytes(t.BytesOut),
			truncateEllipsis(t.Uptime, 10))
	}
	return fmt.Sprintf(" %s  %-16s %-15s %-6s %-10s",
		stateIcon,
		truncateEllipsis(t.Name, 16),
		truncateEllipsis(t.Gateway, 15),
		t.State,
		totalTraffic)
}

func stateIndicator(state string) string {
	switch state {
	case "up":
		return "●"
	case "init":
		return "~"
	default:
		return "○"
	}
}

func (m IPSecTunnelsModel) stateStyledRow(row, state string) string {
	c := theme.Colors()
	switch state {
	case "up":
		return lipgloss.NewStyle().Foreground(c.Success).Render(row)
	case "init":
		return lipgloss.NewStyle().Foreground(c.Warning).Render(row)
	case "down":
		return lipgloss.NewStyle().Foreground(c.Error).Render(row)
	default:
		return DetailValueStyle.Render(row)
	}
}

func (m IPSecTunnelsModel) renderDetail(t models.IPSecTunnel) string {
	c := theme.Colors()
	boxStyle := ViewPanelStyle.
		BorderForeground(c.Primary).
		Width(m.Width - 10)

	titleStyle := ViewTitleStyle
	labelStyle := DetailLabelStyle.Width(18)
	valueStyle := DetailValueStyle
	dimValueStyle := DetailDimStyle
	sectionStyle := DetailSectionStyle.Foreground(c.Primary)

	var b strings.Builder

	// Title with state indicator
	title := t.Name
	var stateStyle lipgloss.Style
	switch t.State {
	case "up":
		stateStyle = lipgloss.NewStyle().Foreground(c.Success).Bold(true)
	case "init":
		stateStyle = lipgloss.NewStyle().Foreground(c.Warning).Bold(true)
	default:
		stateStyle = lipgloss.NewStyle().Foreground(c.Error).Bold(true)
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("  ")
	b.WriteString(stateStyle.Render(strings.ToUpper(t.State)))
	b.WriteString("\n\n")

	// Connection section
	b.WriteString(sectionStyle.Render("Connection"))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Gateway:") + " " + valueStyle.Render(t.Gateway) + "\n")
	if t.LocalIP != "" {
		b.WriteString(labelStyle.Render("Local IP:") + " " + valueStyle.Render(t.LocalIP) + "\n")
	}
	if t.RemoteIP != "" {
		b.WriteString(labelStyle.Render("Remote IP:") + " " + valueStyle.Render(t.RemoteIP) + "\n")
	}

	// Security section
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Security"))
	b.WriteString("\n")
	if t.Protocol != "" {
		b.WriteString(labelStyle.Render("Protocol:") + " " + valueStyle.Render(t.Protocol) + "\n")
	}
	if t.Encryption != "" {
		b.WriteString(labelStyle.Render("Encryption:") + " " + valueStyle.Render(t.Encryption) + "\n")
	}
	if t.Auth != "" {
		b.WriteString(labelStyle.Render("Authentication:") + " " + valueStyle.Render(t.Auth) + "\n")
	}
	if t.LocalSPI != "" {
		b.WriteString(labelStyle.Render("Local SPI:") + " " + dimValueStyle.Render(t.LocalSPI) + "\n")
	}
	if t.RemoteSPI != "" {
		b.WriteString(labelStyle.Render("Remote SPI:") + " " + dimValueStyle.Render(t.RemoteSPI) + "\n")
	}

	// Traffic section
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Traffic Statistics"))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Bytes In:") + " " + valueStyle.Render(formatBytes(t.BytesIn)) + "\n")
	b.WriteString(labelStyle.Render("Bytes Out:") + " " + valueStyle.Render(formatBytes(t.BytesOut)) + "\n")
	b.WriteString(labelStyle.Render("Packets In:") + " " + valueStyle.Render(fmt.Sprintf("%d", t.PacketsIn)) + "\n")
	b.WriteString(labelStyle.Render("Packets Out:") + " " + valueStyle.Render(fmt.Sprintf("%d", t.PacketsOut)) + "\n")
	if t.Uptime != "" {
		b.WriteString(labelStyle.Render("Uptime:") + " " + valueStyle.Render(t.Uptime) + "\n")
	}
	if t.Errors > 0 {
		b.WriteString(labelStyle.Render("Errors:") + " " + lipgloss.NewStyle().Foreground(c.Error).Render(fmt.Sprintf("%d", t.Errors)) + "\n")
	}

	return boxStyle.Render(b.String())
}
