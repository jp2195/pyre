package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
	table       table.Model
	arpTable    []models.ARPEntry
}

func NewInterfacesModel() InterfacesModel {
	base := NewTableBase("Filter interfaces...")
	base.SortAsc = true

	// Initialize table with columns
	columns := []table.Column{
		{Title: "St", Width: 2},
		{Title: "Name", Width: 16},
		{Title: "Type", Width: 10},
		{Title: "Zone", Width: 12},
		{Title: "IP", Width: 18},
		{Title: "MAC", Width: 17},
		{Title: "VR", Width: 12},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	// Apply theme-aware styles
	s := table.DefaultStyles()
	c := theme.Colors()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(c.Border).
		BorderBottom(true).
		Bold(true).
		Foreground(c.TextLabel)
	s.Selected = s.Selected.
		Foreground(c.White).
		Background(c.Primary).
		Bold(false)
	t.SetStyles(s)

	return InterfacesModel{
		TableBase: base,
		table:     t,
	}
}

func (m InterfacesModel) SetSize(width, height int) InterfacesModel {
	m.TableBase = m.TableBase.SetSize(width, height)

	// Update table dimensions
	tableHeight := height - 8 // Account for header, filter, help
	if m.Expanded {
		tableHeight -= 14
	}
	if tableHeight < 3 {
		tableHeight = 3
	}
	m.table.SetHeight(tableHeight)

	// Update column widths proportionally
	availWidth := width - 4
	columns := []table.Column{
		{Title: "St", Width: 2},
		{Title: "Name", Width: maxInt(12, availWidth*15/100)},
		{Title: "Type", Width: maxInt(8, availWidth*10/100)},
		{Title: "Zone", Width: maxInt(10, availWidth*12/100)},
		{Title: "IP", Width: maxInt(15, availWidth*18/100)},
		{Title: "MAC", Width: 17},
		{Title: "VR", Width: maxInt(10, availWidth*12/100)},
	}
	m.table.SetColumns(columns)

	// Clamp cursor to valid range after resize
	if m.table.Cursor() >= len(m.filtered) && len(m.filtered) > 0 {
		m.table.SetCursor(len(m.filtered) - 1)
	}

	return m
}

func (m InterfacesModel) SetLoading(loading bool) InterfacesModel {
	m.TableBase = m.TableBase.SetLoading(loading)
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
	m.applyFilter()
	m.updateTableRows()
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

func (m *InterfacesModel) updateTableRows() {
	rows := make([]table.Row, len(m.filtered))
	for i, iface := range m.filtered {
		status := "○"
		if iface.State == "up" {
			status = "●"
		}

		ip := cleanValue(iface.IP)
		if ip == "" {
			ip = "—"
		}

		rows[i] = table.Row{
			status,
			cleanValue(iface.Name),
			cleanValue(iface.Type),
			cleanValue(iface.Zone),
			ip,
			cleanValue(iface.MAC),
			cleanValue(iface.VirtualRouter),
		}
	}
	m.table.SetRows(rows)
}

func (m InterfacesModel) Init() tea.Cmd {
	return nil
}

func (m InterfacesModel) Update(msg tea.Msg) (InterfacesModel, tea.Cmd) {
	if m.FilterMode {
		return m.updateFilterMode(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "s":
			m.sortBy = (m.sortBy + 1) % 4
			m.applySort()
			m.updateTableRows()
			return m, nil
		case "S":
			m.SortAsc = !m.SortAsc
			m.applySort()
			m.updateTableRows()
			return m, nil
		case "/":
			m.FilterMode = true
			m.Filter.Focus()
			return m, nil
		case "enter":
			m.Expanded = !m.Expanded
			// Recalculate table height
			m = m.SetSize(m.Width, m.Height)
			return m, nil
		}
	}

	// Delegate to table for navigation
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m InterfacesModel) updateFilterMode(msg tea.Msg) (InterfacesModel, tea.Cmd) {
	base, exited, cmd := m.HandleFilterMode(msg)
	m.TableBase = base
	if exited {
		m.applyFilter()
		m.updateTableRows()
	}
	return m, cmd
}

func (m InterfacesModel) SelectedInterface() (models.Interface, bool) {
	idx := m.table.Cursor()
	if idx >= 0 && idx < len(m.filtered) {
		return m.filtered[idx], true
	}
	return models.Interface{}, false
}

// TableCursor returns the current table cursor position (for testing)
func (m InterfacesModel) TableCursor() int {
	return m.table.Cursor()
}

func (m InterfacesModel) View() string {
	if m.Width == 0 {
		return "Loading..."
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
			sections = append(sections, m.table.View())
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

	padding := m.Width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if padding < 1 {
		padding = 1
	}

	return left + strings.Repeat(" ", padding) + right
}

func (m InterfacesModel) renderError() string {
	return ErrorMsgStyle.Bold(true).Padding(1, 0).Render(fmt.Sprintf("Error: %v", m.Err))
}

func (m InterfacesModel) renderFilterBar() string {
	return FilterBorderStyle.Render(m.Filter.View())
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
		lines = append(lines, labelStyle.Render("MTU")+valueStyle.Render(fmt.Sprintf("%d", iface.MTU)))
	}
	if iface.Tag > 0 {
		lines = append(lines, labelStyle.Render("VLAN Tag")+valueStyle.Render(fmt.Sprintf("%d", iface.Tag)))
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
		maxShow := 5
		if len(arpEntries) < maxShow {
			maxShow = len(arpEntries)
		}
		for i := 0; i < maxShow; i++ {
			entry := arpEntries[i]
			statusIndicator := dimStyle.Render("?")
			if entry.Status == "complete" || entry.Status == "c" {
				statusIndicator = upStyle.Render("●")
			} else if entry.Status == "incomplete" || entry.Status == "i" {
				statusIndicator = downStyle.Render("○")
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
