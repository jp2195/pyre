package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/joshuamontgomery/pyre/internal/models"
)

type InterfaceSortField int

const (
	InterfaceSortName InterfaceSortField = iota
	InterfaceSortZone
	InterfaceSortState
	InterfaceSortIP
)

type InterfacesModel struct {
	interfaces  []models.Interface
	filtered    []models.Interface
	err         error
	cursor      int
	offset      int
	filterMode  bool
	filter      textinput.Model
	expanded    bool
	width       int
	height      int
	sortBy      InterfaceSortField
	sortAsc     bool
	loading     bool
	lastRefresh time.Time
}

func NewInterfacesModel() InterfacesModel {
	f := textinput.New()
	f.Placeholder = "Filter interfaces..."
	f.CharLimit = 100
	f.Width = 40

	return InterfacesModel{
		filter:  f,
		sortAsc: true,
	}
}

func (m InterfacesModel) SetSize(width, height int) InterfacesModel {
	m.width = width
	m.height = height
	return m
}

func (m InterfacesModel) SetLoading(loading bool) InterfacesModel {
	m.loading = loading
	return m
}

func (m InterfacesModel) SetInterfaces(interfaces []models.Interface, err error) InterfacesModel {
	m.interfaces = interfaces
	m.err = err
	m.loading = false
	m.lastRefresh = time.Now()
	m.applyFilter()
	if m.cursor >= len(m.filtered) && len(m.filtered) > 0 {
		m.cursor = len(m.filtered) - 1
	}
	return m
}

func (m *InterfacesModel) applyFilter() {
	if m.filter.Value() == "" {
		m.filtered = make([]models.Interface, len(m.interfaces))
		copy(m.filtered, m.interfaces)
	} else {
		query := strings.ToLower(m.filter.Value())
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
		if !m.sortAsc {
			less = !less
		}
		return less
	})
}

func (m InterfacesModel) Init() tea.Cmd {
	return nil
}

func (m InterfacesModel) Update(msg tea.Msg) (InterfacesModel, tea.Cmd) {
	if m.filterMode {
		return m.updateFilterMode(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
				m.ensureVisible()
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
				m.ensureVisible()
			}
		case "g", "home":
			m.cursor = 0
			m.offset = 0
		case "G", "end":
			if len(m.filtered) > 0 {
				m.cursor = len(m.filtered) - 1
				m.ensureVisible()
			}
		case "ctrl+d", "pgdown":
			visible := m.visibleCards()
			m.cursor = minInt(m.cursor+visible, len(m.filtered)-1)
			m.ensureVisible()
		case "ctrl+u", "pgup":
			visible := m.visibleCards()
			m.cursor = maxInt(m.cursor-visible, 0)
			m.ensureVisible()
		case "/":
			m.filterMode = true
			m.filter.Focus()
			return m, textinput.Blink
		case "enter":
			m.expanded = !m.expanded
		case "s":
			m.sortBy = (m.sortBy + 1) % 4
			m.applySort()
		case "S":
			m.sortAsc = !m.sortAsc
			m.applySort()
		}
	}
	return m, nil
}

func (m InterfacesModel) updateFilterMode(msg tea.Msg) (InterfacesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc":
			m.filterMode = false
			m.filter.Blur()
			if msg.String() == "enter" {
				m.cursor = 0
				m.offset = 0
				m.applyFilter()
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	return m, cmd
}

func (m *InterfacesModel) ensureVisible() {
	visible := m.visibleCards()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
}

func (m InterfacesModel) visibleCards() int {
	cardHeight := 4
	available := m.height - 6
	if m.expanded {
		available -= 14 // Reserve space for detail panel
	}
	result := available / cardHeight
	if result < 1 {
		result = 1
	}
	return result
}

// cleanValue, formatBytes, formatPackets are defined in helpers.go

func (m InterfacesModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var sections []string

	// Loading banner (prominent)
	if m.loading {
		sections = append(sections, m.renderLoadingBanner())
	}

	// Header with summary
	sections = append(sections, m.renderHeader())

	// Filter bar
	if m.filterMode {
		sections = append(sections, m.renderFilterBar())
	} else if m.filter.Value() != "" {
		filterInfo := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Render(fmt.Sprintf("Filter: %s (%d/%d)  [esc to clear]", m.filter.Value(), len(m.filtered), len(m.interfaces)))
		sections = append(sections, filterInfo)
	}

	// Error or content
	if m.err != nil {
		sections = append(sections, m.renderError())
	} else if !m.loading {
		sections = append(sections, m.renderCards())

		// Expanded detail panel
		if m.expanded && len(m.filtered) > 0 && m.cursor < len(m.filtered) {
			sections = append(sections, m.renderDetailPanel())
		}
	}

	// Help
	sections = append(sections, m.renderHelp())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m InterfacesModel) renderLoadingBanner() string {
	banner := lipgloss.NewStyle().
		Background(lipgloss.Color("#F59E0B")).
		Foreground(lipgloss.Color("#000000")).
		Bold(true).
		Padding(0, 2).
		Render(" Refreshing interfaces... ")

	// Center it
	padding := (m.width - lipgloss.Width(banner)) / 2
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

	upStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Bold(true)
	downStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB"))

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

	padding := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if padding < 1 {
		padding = 1
	}

	return left + strings.Repeat(" ", padding) + right
}

func (m InterfacesModel) renderError() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#EF4444")).
		Bold(true).
		Padding(1, 0).
		Render(fmt.Sprintf("Error: %v", m.err))
}

func (m InterfacesModel) renderFilterBar() string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6366F1")).
		Padding(0, 1).
		Render(m.filter.View())
}

func (m InterfacesModel) renderCards() string {
	if len(m.filtered) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Padding(1, 0).
			Render("No interfaces found")
	}

	visible := m.visibleCards()
	end := minInt(m.offset+visible, len(m.filtered))

	var cards []string
	for i := m.offset; i < end; i++ {
		cards = append(cards, m.renderCard(m.filtered[i], i == m.cursor))
	}

	if len(m.filtered) > visible {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
		scrollInfo := dimStyle.Render(fmt.Sprintf("  Showing %d-%d of %d", m.offset+1, end, len(m.filtered)))
		cards = append(cards, scrollInfo)
	}

	return lipgloss.JoinVertical(lipgloss.Left, cards...)
}

func (m InterfacesModel) renderCard(iface models.Interface, selected bool) string {
	upStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
	downStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E5E7EB"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	borderColor := lipgloss.Color("#374151")
	if selected {
		borderColor = lipgloss.Color("#6366F1")
	}

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(m.width - 2)

	// Line 1: Status, Name, Type, Zone, Mode
	var line1Parts []string
	if iface.State == "up" {
		line1Parts = append(line1Parts, upStyle.Render("●"))
	} else {
		line1Parts = append(line1Parts, downStyle.Render("○"))
	}
	line1Parts = append(line1Parts, nameStyle.Render(iface.Name))

	if t := cleanValue(iface.Type); t != "" {
		line1Parts = append(line1Parts, lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Render("["+t+"]"))
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
			trafficParts = append(trafficParts, lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).
				Render(fmt.Sprintf("Err: %d/%d", iface.ErrorsIn, iface.ErrorsOut)))
		}
		if iface.DropsIn > 0 || iface.DropsOut > 0 {
			trafficParts = append(trafficParts, lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).
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
	if m.cursor >= len(m.filtered) {
		return ""
	}

	iface := m.filtered[m.cursor]

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("#6366F1")).
		Padding(0, 2).
		Width(m.width - 2).
		MarginTop(1)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A78BFA"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#9CA3AF")).MarginTop(1)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Width(16)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F3F4F6"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	upStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Bold(true)
	downStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true)

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
			errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B"))
			lines = append(lines, labelStyle.Render("Errors")+errStyle.Render(fmt.Sprintf("%d in / %d out", iface.ErrorsIn, iface.ErrorsOut)))
		}
		if iface.DropsIn > 0 || iface.DropsOut > 0 {
			dropStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
			lines = append(lines, labelStyle.Render("Drops")+dropStyle.Render(fmt.Sprintf("%d in / %d out", iface.DropsIn, iface.DropsOut)))
		}
	}

	// Use two-column layout if wide enough
	if m.width >= 100 {
		// Split lines into two columns
		leftLines := lines[:len(lines)/2+1]
		rightLines := lines[len(lines)/2+1:]

		colWidth := (m.width - 8) / 2
		leftCol := lipgloss.NewStyle().Width(colWidth).Render(lipgloss.JoinVertical(lipgloss.Left, leftLines...))
		rightCol := lipgloss.NewStyle().Width(colWidth).Render(lipgloss.JoinVertical(lipgloss.Left, rightLines...))

		return panelStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, leftCol, "  ", rightCol))
	}

	return panelStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m InterfacesModel) renderHelp() string {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	sortNames := []string{"name", "zone", "state", "ip"}
	sortDir := "asc"
	if !m.sortAsc {
		sortDir = "desc"
	}

	expandText := "details"
	if m.expanded {
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

	return lipgloss.NewStyle().MarginTop(1).Render(strings.Join(parts, "  "))
}
