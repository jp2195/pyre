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

type InterfacesModel struct {
	list     RuleListModel[models.Interface]
	arpTable []models.ARPEntry
}

func NewInterfacesModel() InterfacesModel {
	config := RuleListConfig[models.Interface]{
		Title:             "Interfaces",
		ItemNoun:          "interfaces",
		LoadingMsg:        "Loading interfaces...",
		EmptyMsg:          "No interfaces found",
		FilterPlaceholder: "Filter interfaces...",
		SortLabels:        []string{"Name", "Zone", "State", "IP"},
		DefaultSortAsc:    func(idx int) bool { return true },
		MatchFilter:       matchInterface,
		CompareItems:      compareInterface,
		FormatHeaderRow:   formatInterfaceHeader,
		FormatRow:         formatInterfaceListRow,
		StyleRow:          styleInterfaceRow,
		// RenderDetail is bound per-render in View so it can see the
		// current ARP table (config closures capture construction-time
		// state; the ARP table arrives later via SetARPTable).
	}
	list := NewRuleListModel(config)
	list.SortAsc = true
	return InterfacesModel{list: list}
}

func (m InterfacesModel) SetSize(width, height int) InterfacesModel {
	m.list = m.list.SetSize(width, height)
	return m
}

func (m InterfacesModel) SetLoading(loading bool) InterfacesModel {
	m.list = m.list.SetLoading(loading)
	return m
}

// SetSpinnerFrame updates the current spinner animation frame.
func (m InterfacesModel) SetSpinnerFrame(frame string) InterfacesModel {
	m.list.SpinnerFrame = frame
	return m
}

// HasData returns true if interfaces have been loaded.
func (m InterfacesModel) HasData() bool {
	return m.list.HasData()
}

func (m InterfacesModel) SetInterfaces(interfaces []models.Interface, err error) InterfacesModel {
	m.list = m.list.SetItems(interfaces, err)
	return m
}

// SetARPTable sets the ARP table entries for display in the detail panel.
func (m InterfacesModel) SetARPTable(entries []models.ARPEntry) InterfacesModel {
	m.arpTable = entries
	return m
}

func (m InterfacesModel) Update(msg tea.Msg) (InterfacesModel, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m InterfacesModel) View() string {
	arp := m.arpTable
	m.list.config.RenderDetail = func(iface models.Interface, width int) string {
		return renderInterfaceDetail(iface, width, arp)
	}
	return m.list.View()
}

// --- Type-specific functions ---

func matchInterface(iface models.Interface, query string) bool {
	return strings.Contains(strings.ToLower(iface.Name), query) ||
		strings.Contains(strings.ToLower(iface.Zone), query) ||
		strings.Contains(strings.ToLower(iface.IP), query) ||
		strings.Contains(strings.ToLower(iface.State), query) ||
		strings.Contains(strings.ToLower(iface.Type), query) ||
		strings.Contains(strings.ToLower(iface.VirtualRouter), query)
}

func compareInterface(a, b models.Interface, sortIdx int) bool {
	switch sortIdx {
	case 1: // Zone
		return a.Zone < b.Zone
	case 2: // State ("up" sorts first, ties broken by name)
		aUp := a.State == "up"
		bUp := b.State == "up"
		if aUp != bUp {
			return aUp
		}
		return a.Name < b.Name
	case 3: // IP
		return a.IP < b.IP
	default: // Name
		return a.Name < b.Name
	}
}

func interfaceBullet(iface models.Interface) string {
	if iface.State == "up" {
		return "●"
	}
	return "○"
}

// formatInterfaceListRow renders bullet + row content; used for the selected
// row (the whole string gets the selected style) and for width sizing.
func formatInterfaceListRow(iface models.Interface, width int) string {
	return interfaceBullet(iface) + " " + formatInterfaceRow(iface, width)
}

// styleInterfaceRow renders a non-selected row: colored state bullet plus
// normally-styled content.
func styleInterfaceRow(iface models.Interface, width int) string {
	c := theme.Colors()
	bulletStyle := lipgloss.NewStyle().Foreground(c.Success)
	if iface.State != "up" {
		bulletStyle = lipgloss.NewStyle().Foreground(c.Error)
	}
	return bulletStyle.Render(interfaceBullet(iface)) + " " + DetailValueStyle.Render(formatInterfaceRow(iface, width))
}

// arpEntriesForInterface returns ARP entries for a specific interface.
func arpEntriesForInterface(arpTable []models.ARPEntry, ifaceName string) []models.ARPEntry {
	var result []models.ARPEntry
	for _, entry := range arpTable {
		if entry.Interface == ifaceName {
			result = append(result, entry)
		}
	}
	return result
}

func formatInterfaceHeader(width int) string {
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

func formatInterfaceRow(iface models.Interface, width int) string {
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
			truncateEllipsis(name, 16), truncateEllipsis(ifType, 10),
			truncateEllipsis(zone, 12), truncateEllipsis(ip, 18), truncateEllipsis(mac, 17),
			truncateEllipsis(vr, 12))
	} else if width >= 90 {
		return fmt.Sprintf("%-14s %-8s %-10s %-16s %-12s",
			truncateEllipsis(name, 14), truncateEllipsis(ifType, 8),
			truncateEllipsis(zone, 10), truncateEllipsis(ip, 16), truncateEllipsis(vr, 12))
	}
	return fmt.Sprintf("%-14s %-10s %-16s",
		truncateEllipsis(name, 14), truncateEllipsis(zone, 10), truncateEllipsis(ip, 16))
}

func renderInterfaceDetail(iface models.Interface, width int, arpTable []models.ARPEntry) string {
	panelStyle := DetailPanelStyle.Width(width - 2)
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
	arpEntries := arpEntriesForInterface(arpTable, iface.Name)
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
	if width >= 100 {
		// Split lines into two columns
		leftLines := lines[:len(lines)/2+1]
		rightLines := lines[len(lines)/2+1:]

		colWidth := (width - 8) / 2
		colStyle := lipgloss.NewStyle().Width(colWidth)
		leftCol := colStyle.Render(lipgloss.JoinVertical(lipgloss.Left, leftLines...))
		rightCol := colStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rightLines...))

		return panelStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, leftCol, "  ", rightCol))
	}

	return panelStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}
