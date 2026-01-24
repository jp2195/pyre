package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jp2195/pyre/internal/models"
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
	return m
}

func (m InterfacesModel) SetLoading(loading bool) InterfacesModel {
	m.TableBase = m.TableBase.SetLoading(loading)
	return m
}

func (m InterfacesModel) SetInterfaces(interfaces []models.Interface, err error) InterfacesModel {
	m.interfaces = interfaces
	m.Err = err
	m.Loading = false
	m.lastRefresh = time.Now()
	m.applyFilter()
	if m.Cursor >= len(m.filtered) && len(m.filtered) > 0 {
		m.Cursor = len(m.filtered) - 1
	}
	return m
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

func (m InterfacesModel) Init() tea.Cmd {
	return nil
}

func (m InterfacesModel) Update(msg tea.Msg) (InterfacesModel, tea.Cmd) {
	if m.FilterMode {
		return m.updateFilterMode(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle interface-specific keys first
		switch msg.String() {
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
		visible := m.visibleCards()
		base, handled, cmd := m.TableBase.HandleNavigation(msg, len(m.filtered), visible)
		if handled {
			m.TableBase = base
			return m, cmd
		}
	}
	return m, nil
}

func (m InterfacesModel) updateFilterMode(msg tea.Msg) (InterfacesModel, tea.Cmd) {
	base, exited, cmd := m.TableBase.HandleFilterMode(msg)
	m.TableBase = base
	if exited {
		m.applyFilter()
	}
	return m, cmd
}

func (m InterfacesModel) visibleCards() int {
	cardHeight := 4
	available := m.Height - 6
	if m.Expanded {
		available -= 14 // Reserve space for detail panel
	}
	result := available / cardHeight
	if result < 1 {
		result = 1
	}
	return result
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
		sections = append(sections, m.renderCards())

		// Expanded detail panel
		if m.Expanded && len(m.filtered) > 0 && m.Cursor < len(m.filtered) {
			sections = append(sections, m.renderDetailPanel())
		}
	}

	// Help
	sections = append(sections, m.renderHelp())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m InterfacesModel) renderLoadingBanner() string {
	banner := LoadingBannerStyle.Render(" Refreshing interfaces... ")

	// Center it
	padding := (m.Width - lipgloss.Width(banner)) / 2
	if padding < 0 {
		padding = 0
	}
	return strings.Repeat(" ", padding) + banner
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

func (m InterfacesModel) renderCards() string {
	if len(m.filtered) == 0 {
		return EmptyMsgStyle.Padding(1, 0).Render("No interfaces found")
	}

	visible := m.visibleCards()
	end := minInt(m.Offset+visible, len(m.filtered))

	var cards []string
	for i := m.Offset; i < end; i++ {
		cards = append(cards, m.renderCard(m.filtered[i], i == m.Cursor))
	}

	if len(m.filtered) > visible {
		scrollInfo := DetailDimStyle.Render(fmt.Sprintf("  Showing %d-%d of %d", m.Offset+1, end, len(m.filtered)))
		cards = append(cards, scrollInfo)
	}

	return lipgloss.JoinVertical(lipgloss.Left, cards...)
}

func (m InterfacesModel) renderCard(iface models.Interface, selected bool) string {
	upStyle := StatusActiveStyle
	downStyle := StatusInactiveStyle
	labelStyle := DetailLabelStyle
	valueStyle := DetailValueStyle
	dimStyle := DetailDimStyle

	cardStyle := CardStyle.Width(m.Width - 2)
	if selected {
		cardStyle = CardSelectedStyle.Width(m.Width - 2)
	}

	// Line 1: Status, Name, Type, Zone, Mode
	var line1Parts []string
	if iface.State == "up" {
		line1Parts = append(line1Parts, upStyle.Render("●"))
	} else {
		line1Parts = append(line1Parts, downStyle.Render("○"))
	}
	line1Parts = append(line1Parts, TextBoldStyle.Render(iface.Name))

	if t := cleanValue(iface.Type); t != "" {
		line1Parts = append(line1Parts, TagStyle.Render("["+t+"]"))
	}
	if z := cleanValue(iface.Zone); z != "" {
		line1Parts = append(line1Parts, labelStyle.Render("zone:")+valueStyle.Render(z))
	}
	if mode := cleanValue(iface.Mode); mode != "" {
		line1Parts = append(line1Parts, labelStyle.Render("mode:")+valueStyle.Render(mode))
	}

	line1 := strings.Join(line1Parts, "  ")

	// Line 2: IP, MAC, VR, MTU, Speed
	var line2Parts []string
	if ip := cleanValue(iface.IP); ip != "" {
		line2Parts = append(line2Parts, labelStyle.Render("IP: ")+valueStyle.Render(ip))
	} else {
		line2Parts = append(line2Parts, labelStyle.Render("IP: ")+dimStyle.Render("none"))
	}
	if mac := cleanValue(iface.MAC); mac != "" {
		line2Parts = append(line2Parts, labelStyle.Render("MAC: ")+valueStyle.Render(mac))
	}
	if vr := cleanValue(iface.VirtualRouter); vr != "" {
		line2Parts = append(line2Parts, labelStyle.Render("VR: ")+valueStyle.Render(vr))
	}
	if iface.MTU > 0 {
		line2Parts = append(line2Parts, labelStyle.Render("MTU: ")+valueStyle.Render(fmt.Sprintf("%d", iface.MTU)))
	}
	if speed := cleanValue(iface.Speed); speed != "" {
		line2Parts = append(line2Parts, labelStyle.Render("Speed: ")+valueStyle.Render(speed))
	}

	line2 := strings.Join(line2Parts, "  ")

	// Line 3: Traffic (only if data exists)
	var line3 string
	if iface.BytesIn > 0 || iface.BytesOut > 0 {
		var trafficParts []string
		trafficParts = append(trafficParts, labelStyle.Render("Traffic:")+
			valueStyle.Render(fmt.Sprintf(" %s in / %s out", formatBytes(iface.BytesIn), formatBytes(iface.BytesOut))))
		trafficParts = append(trafficParts, labelStyle.Render("Pkts:")+
			valueStyle.Render(fmt.Sprintf(" %s / %s", formatPackets(iface.PacketsIn), formatPackets(iface.PacketsOut))))

		if iface.ErrorsIn > 0 || iface.ErrorsOut > 0 {
			trafficParts = append(trafficParts, StatusWarningStyle.
				Render(fmt.Sprintf("Err: %d/%d", iface.ErrorsIn, iface.ErrorsOut)))
		}
		if iface.DropsIn > 0 || iface.DropsOut > 0 {
			trafficParts = append(trafficParts, StatusInactiveStyle.
				Render(fmt.Sprintf("Drop: %d/%d", iface.DropsIn, iface.DropsOut)))
		}
		line3 = strings.Join(trafficParts, "  ")
	}

	var lines []string
	lines = append(lines, line1)
	if len(line2Parts) > 0 {
		lines = append(lines, line2)
	}
	if line3 != "" {
		lines = append(lines, line3)
	}

	return cardStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m InterfacesModel) renderDetailPanel() string {
	if m.Cursor >= len(m.filtered) {
		return ""
	}

	iface := m.filtered[m.Cursor]

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

	var parts []string
	for _, k := range keys {
		parts = append(parts, keyStyle.Render(k.key)+descStyle.Render(":"+k.desc))
	}

	return ViewSubtitleStyle.MarginTop(1).Render(strings.Join(parts, "  "))
}
