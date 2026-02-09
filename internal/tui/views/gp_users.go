package views

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jp2195/pyre/internal/models"
	"github.com/jp2195/pyre/internal/tui/theme"
)

type GPSortField int

const (
	GPSortUsername GPSortField = iota
	GPSortGateway
	GPSortLoginTime
	GPSortDuration
)

type GPUsersModel struct {
	TableBase
	users    []models.GlobalProtectUser
	filtered []models.GlobalProtectUser
	sortBy   GPSortField
}

func NewGPUsersModel() GPUsersModel {
	return GPUsersModel{
		TableBase: NewTableBase("Filter users..."),
	}
}

func (m GPUsersModel) SetSize(width, height int) GPUsersModel {
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

func (m GPUsersModel) SetLoading(loading bool) GPUsersModel {
	m.TableBase = m.TableBase.SetLoading(loading)
	return m
}

// HasData returns true if user data has been loaded.
func (m GPUsersModel) HasData() bool {
	return m.users != nil
}

func (m GPUsersModel) SetUsers(users []models.GlobalProtectUser, err error) GPUsersModel {
	m.users = users
	m.Err = err
	m.Loading = false
	m.Cursor = 0
	m.Offset = 0
	m.applyFilter()
	return m
}

func (m *GPUsersModel) applyFilter() {
	if m.FilterValue() == "" {
		m.filtered = make([]models.GlobalProtectUser, len(m.users))
		copy(m.filtered, m.users)
	} else {
		query := strings.ToLower(m.FilterValue())
		m.filtered = nil

		for _, u := range m.users {
			if strings.Contains(strings.ToLower(u.Username), query) ||
				strings.Contains(strings.ToLower(u.Domain), query) ||
				strings.Contains(strings.ToLower(u.Computer), query) ||
				strings.Contains(strings.ToLower(u.Gateway), query) ||
				strings.Contains(strings.ToLower(u.ClientIP), query) ||
				strings.Contains(strings.ToLower(u.VirtualIP), query) ||
				strings.Contains(strings.ToLower(u.SourceRegion), query) {
				m.filtered = append(m.filtered, u)
			}
		}
	}
	m.applySort()
}

func (m *GPUsersModel) applySort() {
	sort.Slice(m.filtered, func(i, j int) bool {
		var less bool
		switch m.sortBy {
		case GPSortGateway:
			less = m.filtered[i].Gateway < m.filtered[j].Gateway
		case GPSortLoginTime:
			less = m.filtered[i].LoginTime.Before(m.filtered[j].LoginTime)
		case GPSortDuration:
			less = m.filtered[i].Duration < m.filtered[j].Duration
		default: // GPSortUsername
			less = m.filtered[i].Username < m.filtered[j].Username
		}
		if m.SortAsc {
			return less
		}
		return !less
	})
}

func (m *GPUsersModel) cycleSort() {
	m.sortBy = (m.sortBy + 1) % 4
	m.SortAsc = m.sortBy == GPSortUsername || m.sortBy == GPSortGateway
	m.applySort()
}

func (m GPUsersModel) sortLabel() string {
	dir := "↓"
	if m.SortAsc {
		dir = "↑"
	}
	switch m.sortBy {
	case GPSortGateway:
		return fmt.Sprintf("Gateway %s", dir)
	case GPSortLoginTime:
		return fmt.Sprintf("Login Time %s", dir)
	case GPSortDuration:
		return fmt.Sprintf("Duration %s", dir)
	default:
		return fmt.Sprintf("Username %s", dir)
	}
}

func (m GPUsersModel) Update(msg tea.Msg) (GPUsersModel, tea.Cmd) {
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

func (m GPUsersModel) updateFilter(msg tea.Msg) (GPUsersModel, tea.Cmd) {
	base, exited, cmd := m.HandleFilterMode(msg)
	m.TableBase = base
	if exited {
		m.applyFilter()
	}
	return m, cmd
}

func (m GPUsersModel) visibleRows() int {
	rows := m.Height - 8
	if m.Expanded {
		rows -= 14
	}
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m GPUsersModel) View() string {
	if m.Width == 0 {
		return RenderLoadingInline(m.SpinnerFrame, "Loading...")
	}

	titleStyle := ViewTitleStyle.MarginBottom(1)
	panelStyle := ViewPanelStyle.Width(m.Width - 4)

	var b strings.Builder
	title := "GlobalProtect Users"
	sortInfo := BannerInfoStyle.Render(fmt.Sprintf(" [%d users | Sort: %s | s: change | /: filter | enter: details]", len(m.filtered), m.sortLabel()))
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

	if m.Loading || m.users == nil {
		b.WriteString(RenderLoadingInline(m.SpinnerFrame, "Loading GlobalProtect users..."))
		return panelStyle.Render(b.String())
	}

	if len(m.filtered) == 0 {
		b.WriteString(EmptyMsgStyle.Render("No GlobalProtect users found"))
		return panelStyle.Render(b.String())
	}

	b.WriteString(m.renderTable())

	if m.Expanded && m.Cursor < len(m.filtered) {
		b.WriteString("\n")
		b.WriteString(m.renderDetail(m.filtered[m.Cursor]))
	}

	return panelStyle.Render(b.String())
}

func (m GPUsersModel) renderTable() string {
	headerStyle := DetailLabelStyle.Bold(true)
	selectedStyle := TableRowSelectedStyle.Bold(true)
	normalStyle := DetailValueStyle
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
		u := m.filtered[i]
		isSelected := i == m.Cursor

		row := m.formatUserRow(u, availableWidth)

		if isSelected {
			b.WriteString(selectedStyle.Render(row))
		} else {
			b.WriteString(normalStyle.Render(row))
		}
		b.WriteString("\n")
	}

	if len(m.filtered) > visibleRows {
		scrollInfo := fmt.Sprintf("  Showing %d-%d of %d", m.Offset+1, end, len(m.filtered))
		b.WriteString(dimStyle.Render(scrollInfo))
	}

	return b.String()
}

func (m GPUsersModel) formatHeaderRow(width int) string {
	if width >= 140 {
		return fmt.Sprintf("%-18s %-15s %-15s %-15s %-15s %-10s %-12s %-10s",
			"Username", "Domain", "Gateway", "Virtual IP", "Client IP", "Duration", "Region", "Traffic")
	} else if width >= 100 {
		return fmt.Sprintf("%-18s %-15s %-15s %-15s %-10s %-10s",
			"Username", "Gateway", "Virtual IP", "Client IP", "Duration", "Traffic")
	}
	return fmt.Sprintf("%-16s %-15s %-15s %-10s",
		"Username", "Gateway", "Virtual IP", "Duration")
}

func (m GPUsersModel) formatUserRow(u models.GlobalProtectUser, width int) string {
	duration := u.Duration
	if duration == "" && !u.LoginTime.IsZero() {
		duration = formatTimeAgo(u.LoginTime)
	}
	traffic := formatBytes(u.BytesIn + u.BytesOut)

	if width >= 140 {
		return fmt.Sprintf("%-18s %-15s %-15s %-15s %-15s %-10s %-12s %-10s",
			truncateEllipsis(u.Username, 18),
			truncateEllipsis(u.Domain, 15),
			truncateEllipsis(u.Gateway, 15),
			truncateEllipsis(u.VirtualIP, 15),
			truncateEllipsis(u.ClientIP, 15),
			truncateEllipsis(duration, 10),
			truncateEllipsis(u.SourceRegion, 12),
			traffic)
	} else if width >= 100 {
		return fmt.Sprintf("%-18s %-15s %-15s %-15s %-10s %-10s",
			truncateEllipsis(u.Username, 18),
			truncateEllipsis(u.Gateway, 15),
			truncateEllipsis(u.VirtualIP, 15),
			truncateEllipsis(u.ClientIP, 15),
			truncateEllipsis(duration, 10),
			traffic)
	}
	return fmt.Sprintf("%-16s %-15s %-15s %-10s",
		truncateEllipsis(u.Username, 16),
		truncateEllipsis(u.Gateway, 15),
		truncateEllipsis(u.VirtualIP, 15),
		truncateEllipsis(duration, 10))
}

func (m GPUsersModel) renderDetail(u models.GlobalProtectUser) string {
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

	b.WriteString(titleStyle.Render(u.Username))
	b.WriteString("\n\n")

	// User Info section
	b.WriteString(sectionStyle.Render("User Information"))
	b.WriteString("\n")
	if u.Domain != "" {
		b.WriteString(labelStyle.Render("Domain:") + " " + valueStyle.Render(u.Domain) + "\n")
	}
	if u.Computer != "" {
		b.WriteString(labelStyle.Render("Computer:") + " " + valueStyle.Render(u.Computer) + "\n")
	}
	if u.Client != "" {
		b.WriteString(labelStyle.Render("Client Version:") + " " + valueStyle.Render(u.Client) + "\n")
	}

	// Connection section
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Connection"))
	b.WriteString("\n")
	if u.Gateway != "" {
		b.WriteString(labelStyle.Render("Gateway:") + " " + valueStyle.Render(u.Gateway) + "\n")
	}
	if u.VirtualIP != "" {
		b.WriteString(labelStyle.Render("Virtual IP:") + " " + valueStyle.Render(u.VirtualIP) + "\n")
	}
	if u.ClientIP != "" {
		b.WriteString(labelStyle.Render("Public IP:") + " " + valueStyle.Render(u.ClientIP) + "\n")
	}
	if u.SourceRegion != "" {
		b.WriteString(labelStyle.Render("Source Region:") + " " + valueStyle.Render(u.SourceRegion) + "\n")
	}

	// Session section
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Session"))
	b.WriteString("\n")
	if !u.LoginTime.IsZero() {
		b.WriteString(labelStyle.Render("Login Time:") + " " + valueStyle.Render(u.LoginTime.Format("2006-01-02 15:04:05")) + "\n")
	}
	if u.Duration != "" {
		b.WriteString(labelStyle.Render("Duration:") + " " + valueStyle.Render(u.Duration) + "\n")
	}

	// Traffic section
	if u.BytesIn > 0 || u.BytesOut > 0 {
		b.WriteString("\n")
		b.WriteString(sectionStyle.Render("Traffic"))
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("Bytes In:") + " " + valueStyle.Render(formatBytes(u.BytesIn)) + "\n")
		b.WriteString(labelStyle.Render("Bytes Out:") + " " + valueStyle.Render(formatBytes(u.BytesOut)) + "\n")
	}

	_ = dimValueStyle // keep import clean

	return boxStyle.Render(b.String())
}
