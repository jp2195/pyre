package views

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/models"
	"github.com/jp2195/pyre/internal/tui/theme"
)

type InterfaceSortField int

const (
	InterfaceSortName InterfaceSortField = iota
	InterfaceSortZone
	InterfaceSortState
	InterfaceSortIP
)

type InterfacesModel struct {
	TableBase
	interfaces  []models.Interface
	filtered    []models.Interface
	sortBy      InterfaceSortField
	lastRefresh time.Time
	arpTable    []models.ARPEntry
}

func NewInterfacesModel() InterfacesModel {
	base := NewTableBase("Filter interfaces...")
	base.SortAsc = true

	return InterfacesModel{
		TableBase: base,
	}
}

func (m InterfacesModel) SetSize(width, height int) InterfacesModel {
	m.TableBase = m.TableBase.SetSize(width, height)
	m.EnsureCursorValid(len(m.filtered))
	if visibleRows := m.visibleRows(); visibleRows > 0 {
		m.EnsureVisible(visibleRows)
	}
	return m
}

func (m InterfacesModel) SetLoading(loading bool) InterfacesModel {
	m.TableBase = m.TableBase.SetLoading(loading)
	return m
}

// SetSpinnerFrame updates the current spinner animation frame.
func (m InterfacesModel) SetSpinnerFrame(frame string) InterfacesModel {
	m.TableBase = m.TableBase.SetSpinnerFrame(frame)
	return m
}

// HasData returns true if interfaces have been loaded.
func (m InterfacesModel) HasData() bool {
	return m.interfaces != nil
}

func (m InterfacesModel) SetInterfaces(interfaces []models.Interface, err error) InterfacesModel {
	m.interfaces = interfaces
	m.Err = err
	m.Loading = false
	m.lastRefresh = time.Now()
	m.Cursor = 0
	m.Offset = 0
	m.applyFilter()
	return m
}

// SetARPTable sets the ARP table entries for display in the detail panel.
func (m InterfacesModel) SetARPTable(entries []models.ARPEntry) InterfacesModel {
	m.arpTable = entries
	return m
}

// getARPEntriesForInterface returns ARP entries for a specific interface.
func (m InterfacesModel) getARPEntriesForInterface(ifaceName string) []models.ARPEntry {
	var result []models.ARPEntry
	for _, entry := range m.arpTable {
		if entry.Interface == ifaceName {
			result = append(result, entry)
		}
	}
	return result
}

func (m *InterfacesModel) applyFilter() {
	if m.FilterValue() == "" {
		m.filtered = make([]models.Interface, len(m.interfaces))
		copy(m.filtered, m.interfaces)
	} else {
		query := strings.ToLower(m.FilterValue())
		m.filtered = nil

		for _, iface := range m.interfaces {
			if strings.Contains(strings.ToLower(iface.Name), query) ||
				strings.Contains(strings.ToLower(iface.Zone), query) ||
				strings.Contains(strings.ToLower(iface.IP), query) ||
				strings.Contains(strings.ToLower(iface.State), query) ||
				strings.Contains(strings.ToLower(iface.Type), query) ||
				strings.Contains(strings.ToLower(iface.VirtualRouter), query) {
				m.filtered = append(m.filtered, iface)
			}
		}
	}
	m.applySort()
}

func (m *InterfacesModel) applySort() {
	sort.Slice(m.filtered, func(i, j int) bool {
		var less bool
		switch m.sortBy {
		case InterfaceSortZone:
			less = m.filtered[i].Zone < m.filtered[j].Zone
		case InterfaceSortState:
			iUp := m.filtered[i].State == "up"
			jUp := m.filtered[j].State == "up"
			if iUp != jUp {
				less = iUp
			} else {
				less = m.filtered[i].Name < m.filtered[j].Name
			}
		case InterfaceSortIP:
			less = m.filtered[i].IP < m.filtered[j].IP
		default:
			less = m.filtered[i].Name < m.filtered[j].Name
		}
		if !m.SortAsc {
			less = !less
		}
		return less
	})
}

func (m InterfacesModel) visibleRows() int {
	return m.VisibleRows(8, 14)
}

func (m InterfacesModel) Init() tea.Cmd {
	return nil
}

func (m InterfacesModel) Update(msg tea.Msg) (InterfacesModel, tea.Cmd) {
	if m.FilterMode {
		return m.updateFilterMode(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
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
			m.sortBy = (m.sortBy + 1) % 4
			m.applySort()
			return m, nil
		case "S":
			m.SortAsc = !m.SortAsc
			m.applySort()
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

func (m InterfacesModel) updateFilterMode(msg tea.Msg) (InterfacesModel, tea.Cmd) {
	base, exited, cmd := m.HandleFilterMode(msg)
	m.TableBase = base
	if exited {
		m.applyFilter()
	}
	return m, cmd
}

func (m InterfacesModel) SelectedInterface() (models.Interface, bool) {
	if m.Cursor >= 0 && m.Cursor < len(m.filtered) {
		return m.filtered[m.Cursor], true
	}
	return models.Interface{}, false
}

func (m InterfacesModel) View() string {
	if m.Width == 0 {
		return RenderLoadingInline(m.SpinnerFrame, "Loading...")
	}

	var sections []string

	// Loading banner (prominent)
	if m.Loading {
		sections = append(sections, m.renderLoadingBanner())
	}

	// Header with summary
	sections = append(sections, m.renderHeader())

	// Filter bar
	if m.FilterMode {
		sections = append(sections, m.renderFilterBar())
	} else if m.IsFiltered() {
		filterInfo := FilterInfoStyle.Render(fmt.Sprintf("Filter: %s (%d/%d)  [esc to clear]", m.FilterValue(), len(m.filtered), len(m.interfaces)))
		sections = append(sections, filterInfo)
	}

	// Error or content
	if m.Err != nil {
		sections = append(sections, m.renderError())
	} else if !m.Loading {
		if len(m.filtered) == 0 {
			sections = append(sections, EmptyMsgStyle.Padding(1, 0).Render("No interfaces found"))
		} else {
			sections = append(sections, m.renderTable())
		}

		// Expanded detail panel
		if m.Expanded {
			if iface, ok := m.SelectedInterface(); ok {
				sections = append(sections, m.renderDetailPanel(iface))
			}
		}
	}

	// Help
	sections = append(sections, m.renderHelp())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m InterfacesModel) renderLoadingBanner() string {
	return RenderLoadingBanner(m.SpinnerFrame, "Refreshing interfaces...", m.Width)
}

func (m InterfacesModel) renderHeader() string {
	var upCount, downCount int
	var totalBytesIn, totalBytesOut int64
	zoneMap := make(map[string]bool)

	for _, iface := range m.interfaces {
		if iface.State == "up" {
			upCount++
		} else {
			downCount++
		}
		if iface.Zone != "" {
			zoneMap[iface.Zone] = true
		}
		totalBytesIn += iface.BytesIn
		totalBytesOut += iface.BytesOut
	}

	upStyle := StatusActiveStyle
	downStyle := StatusInactiveStyle
	labelStyle := DetailLabelStyle
	valueStyle := DetailValueStyle

	stats := []string{
		fmt.Sprintf("%d interfaces", len(m.interfaces)),
		upStyle.Render(fmt.Sprintf("%d up", upCount)),
		downStyle.Render(fmt.Sprintf("%d down", downCount)),
		fmt.Sprintf("%d zones", len(zoneMap)),
	}

	if totalBytesIn > 0 || totalBytesOut > 0 {
		stats = append(stats,
			labelStyle.Render("In: ")+valueStyle.Render(formatBytes(totalBytesIn)),
			labelStyle.Render("Out: ")+valueStyle.Render(formatBytes(totalBytesOut)),
		)
	}

	left := strings.Join(stats, "  ")

	// Right side - last update time
	var right string
	if !m.lastRefresh.IsZero() {
		ago := time.Since(m.lastRefresh).Truncate(time.Second)
		right = labelStyle.Render(fmt.Sprintf("Updated %s ago", ago))
	}

	padding := max(m.Width-lipgloss.Width(left)-lipgloss.Width(right)-2, 1)

	return left + strings.Repeat(" ", padding) + right
}

func (m InterfacesModel) renderError() string {
	return ErrorMsgStyle.Bold(true).Padding(1, 0).Render(fmt.Sprintf("Error: %v", m.Err))
}

func (m InterfacesModel) renderFilterBar() string {
	return FilterBorderStyle.Render(m.Filter.View())
}

func (m InterfacesModel) renderTable() string {
	headerStyle := DetailLabelStyle.Bold(true)
	selectedStyle := TableRowSelectedStyle.Bold(true)
	normalStyle := DetailValueStyle
	dimStyle := DetailDimStyle
	c := theme.Colors()

	availableWidth := m.Width - 6

	var b strings.Builder

	// Header
	header := m.formatHeaderRow(availableWidth)
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", min(availableWidth, len(header)+10))))
	b.WriteString("\n")

	visibleRows := m.visibleRows()
	end := min(m.Offset+visibleRows, len(m.filtered))

	upStyle := lipgloss.NewStyle().Foreground(c.Success)
	downStyle := lipgloss.NewStyle().Foreground(c.Error)

	for i := m.Offset; i < end; i++ {
		iface := m.filtered[i]
		isSelected := i == m.Cursor

		bullet := "●"
		bulletStyle := upStyle
		if iface.State != "up" {
			bullet = "○"
			bulletStyle = downStyle
		}
		content := m.formatInterfaceRow(iface, availableWidth)

		if isSelected {
			b.WriteString(selectedStyle.Render(bullet + " " + content))
		} else {
			b.WriteString(bulletStyle.Render(bullet) + " " + normalStyle.Render(content))
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

func (m InterfacesModel) formatHeaderRow(width int) string {
	if width >= 120 {
		return fmt.Sprintf("St %-16s %-10s %-12s %-18s %-17s %-12s",
			"Name", "Type", "Zone", "IP", "MAC", "VR")
	} else if width >= 90 {
		return fmt.Sprintf("St %-14s %-8s %-10s %-16s %-12s",
			"Name", "Type", "Zone", "IP", "VR")
	}
	return fmt.Sprintf("St %-14s %-10s %-16s",
		"Name", "Zone", "IP")
}

func (m InterfacesModel) formatInterfaceRow(iface models.Interface, width int) string {
	ip := cleanValue(iface.IP)
	if ip == "" {
		ip = "—"
	}

	name := cleanValue(iface.Name)
	ifType := cleanValue(iface.Type)
	zone := cleanValue(iface.Zone)
	mac := cleanValue(iface.MAC)
	vr := cleanValue(iface.VirtualRouter)

	if width >= 120 {
		return fmt.Sprintf("%-16s %-10s %-12s %-18s %-17s %-12s",
			truncateStr(name, 16), truncateStr(ifType, 10),
			truncateStr(zone, 12), truncateStr(ip, 18), truncateStr(mac, 17),
			truncateStr(vr, 12))
	} else if width >= 90 {
		return fmt.Sprintf("%-14s %-8s %-10s %-16s %-12s",
			truncateStr(name, 14), truncateStr(ifType, 8),
			truncateStr(zone, 10), truncateStr(ip, 16), truncateStr(vr, 12))
	}
	return fmt.Sprintf("%-14s %-10s %-16s",
		truncateStr(name, 14), truncateStr(zone, 10), truncateStr(ip, 16))
}

func (m InterfacesModel) renderDetailPanel(iface models.Interface) string {
	panelStyle := DetailPanelStyle.Width(m.Width - 2)
	titleStyle := ViewTitleStyle
	sectionStyle := DetailSectionStyle
	labelStyle := DetailLabelStyle.Width(16)
	valueStyle := DetailValueStyle
	dimStyle := DetailDimStyle
	upStyle := StatusActiveStyle
	downStyle := StatusInactiveStyle

	row := func(label, value string) string {
		v := cleanValue(value)
		if v == "" {
			return labelStyle.Render(label) + dimStyle.Render("—")
		}
		return labelStyle.Render(label) + valueStyle.Render(v)
	}

	var lines []string
	lines = append(lines, titleStyle.Render("Interface Details: "+iface.Name))

	// Basic Info
	lines = append(lines, sectionStyle.Render("Basic Information"))
	if iface.State == "up" {
		lines = append(lines, labelStyle.Render("State")+upStyle.Render("● UP"))
	} else {
		lines = append(lines, labelStyle.Render("State")+downStyle.Render("○ DOWN"))
	}
	lines = append(lines, row("Type", iface.Type))
	lines = append(lines, row("Zone", iface.Zone))
	lines = append(lines, row("Mode", iface.Mode))
	if iface.Vsys != "" {
		lines = append(lines, row("Vsys", iface.Vsys))
	}

	// Network
	lines = append(lines, sectionStyle.Render("Network"))
	lines = append(lines, row("IP Address", iface.IP))
	lines = append(lines, row("MAC Address", iface.MAC))
	lines = append(lines, row("Virtual Router", iface.VirtualRouter))
	if iface.MTU > 0 {
		lines = append(lines, labelStyle.Render("MTU")+valueStyle.Render(strconv.Itoa(iface.MTU)))
	}
	if iface.Tag > 0 {
		lines = append(lines, labelStyle.Render("VLAN Tag")+valueStyle.Render(strconv.Itoa(iface.Tag)))
	}

	// Physical
	lines = append(lines, sectionStyle.Render("Physical"))
	lines = append(lines, row("Speed", iface.Speed))
	lines = append(lines, row("Duplex", iface.Duplex))

	// Traffic Stats (if available)
	if iface.BytesIn > 0 || iface.BytesOut > 0 || iface.PacketsIn > 0 || iface.PacketsOut > 0 {
		lines = append(lines, sectionStyle.Render("Traffic Statistics"))
		lines = append(lines, labelStyle.Render("Bytes In")+valueStyle.Render(formatBytes(iface.BytesIn)))
		lines = append(lines, labelStyle.Render("Bytes Out")+valueStyle.Render(formatBytes(iface.BytesOut)))
		lines = append(lines, labelStyle.Render("Packets In")+valueStyle.Render(formatPackets(iface.PacketsIn)))
		lines = append(lines, labelStyle.Render("Packets Out")+valueStyle.Render(formatPackets(iface.PacketsOut)))

		if iface.ErrorsIn > 0 || iface.ErrorsOut > 0 {
			lines = append(lines, labelStyle.Render("Errors")+StatusWarningStyle.Render(fmt.Sprintf("%d in / %d out", iface.ErrorsIn, iface.ErrorsOut)))
		}
		if iface.DropsIn > 0 || iface.DropsOut > 0 {
			lines = append(lines, labelStyle.Render("Drops")+StatusInactiveStyle.Render(fmt.Sprintf("%d in / %d out", iface.DropsIn, iface.DropsOut)))
		}
	}

	// ARP Entries for this interface
	arpEntries := m.getARPEntriesForInterface(iface.Name)
	if len(arpEntries) > 0 {
		lines = append(lines, sectionStyle.Render("ARP Entries"))
		maxShow := min(len(arpEntries), 5)
		for i := range maxShow {
			entry := arpEntries[i]
			var statusIndicator string
			switch entry.Status {
			case "complete", "c":
				statusIndicator = upStyle.Render("●")
			case "incomplete", "i":
				statusIndicator = downStyle.Render("○")
			default:
				statusIndicator = dimStyle.Render("?")
			}
			lines = append(lines, fmt.Sprintf("  %s %-15s  %s", statusIndicator, entry.IP, dimStyle.Render(entry.MAC)))
		}
		if len(arpEntries) > maxShow {
			lines = append(lines, dimStyle.Render(fmt.Sprintf("  ... and %d more", len(arpEntries)-maxShow)))
		}
	}

	// Use two-column layout if wide enough
	if m.Width >= 100 {
		// Split lines into two columns
		leftLines := lines[:len(lines)/2+1]
		rightLines := lines[len(lines)/2+1:]

		colWidth := (m.Width - 8) / 2
		colStyle := lipgloss.NewStyle().Width(colWidth)
		leftCol := colStyle.Render(lipgloss.JoinVertical(lipgloss.Left, leftLines...))
		rightCol := colStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rightLines...))

		return panelStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, leftCol, "  ", rightCol))
	}

	return panelStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m InterfacesModel) renderHelp() string {
	keyStyle := HelpKeyStyle
	descStyle := HelpDescStyle

	sortNames := []string{"name", "zone", "state", "ip"}
	sortDir := "asc"
	if !m.SortAsc {
		sortDir = "desc"
	}

	expandText := "details"
	if m.Expanded {
		expandText = "collapse"
	}

	keys := []struct{ key, desc string }{
		{"j/k", "scroll"},
		{"enter", expandText},
		{"/", "filter"},
		{"s", fmt.Sprintf("sort (%s)", sortNames[m.sortBy])},
		{"S", sortDir},
		{"r", "refresh"},
	}

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, keyStyle.Render(k.key)+descStyle.Render(":"+k.desc))
	}

	return ViewSubtitleStyle.MarginTop(1).Render(strings.Join(parts, "  "))
}
