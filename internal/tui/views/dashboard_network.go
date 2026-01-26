package views

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jp2195/pyre/internal/models"
)

// NetworkDashboardModel represents the network-focused dashboard
type NetworkDashboardModel struct {
	interfaces    []models.Interface
	arpTable      []models.ARPEntry
	routes        []models.RouteEntry
	bgpNeighbors  []models.BGPNeighbor
	ospfNeighbors []models.OSPFNeighbor

	ifaceErr error
	arpErr   error
	routeErr error
	bgpErr   error
	ospfErr  error

	width  int
	height int
}

// NewNetworkDashboardModel creates a new network dashboard model
func NewNetworkDashboardModel() NetworkDashboardModel {
	return NetworkDashboardModel{}
}

// SetSize sets the terminal dimensions
func (m NetworkDashboardModel) SetSize(width, height int) NetworkDashboardModel {
	m.width = width
	m.height = height
	return m
}

// SetInterfaces sets the interface data
func (m NetworkDashboardModel) SetInterfaces(ifaces []models.Interface, err error) NetworkDashboardModel {
	m.interfaces = ifaces
	m.ifaceErr = err
	return m
}

// SetARPTable sets the ARP table data
func (m NetworkDashboardModel) SetARPTable(entries []models.ARPEntry, err error) NetworkDashboardModel {
	m.arpTable = entries
	m.arpErr = err
	return m
}

// SetRoutingTable sets the routing table data
func (m NetworkDashboardModel) SetRoutingTable(routes []models.RouteEntry, err error) NetworkDashboardModel {
	m.routes = routes
	m.routeErr = err
	return m
}

// SetBGPNeighbors sets the BGP neighbor data
func (m NetworkDashboardModel) SetBGPNeighbors(neighbors []models.BGPNeighbor, err error) NetworkDashboardModel {
	m.bgpNeighbors = neighbors
	m.bgpErr = err
	return m
}

// SetOSPFNeighbors sets the OSPF neighbor data
func (m NetworkDashboardModel) SetOSPFNeighbors(neighbors []models.OSPFNeighbor, err error) NetworkDashboardModel {
	m.ospfNeighbors = neighbors
	m.ospfErr = err
	return m
}

// Update handles key events
func (m NetworkDashboardModel) Update(msg tea.Msg) (NetworkDashboardModel, tea.Cmd) {
	return m, nil
}

// View renders the network dashboard
func (m NetworkDashboardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	totalWidth := m.width - 4
	leftColWidth := totalWidth / 2
	rightColWidth := totalWidth - leftColWidth - 2

	if leftColWidth < 35 {
		return m.renderSingleColumn(totalWidth)
	}

	// Left column: interfaces
	leftPanels := []string{
		m.renderTopInterfaces(leftColWidth),
		m.renderInterfaceErrors(leftColWidth),
	}
	leftCol := lipgloss.JoinVertical(lipgloss.Left, leftPanels...)

	// Right column: ARP, routing, and neighbors
	rightPanels := []string{
		m.renderARPSummary(rightColWidth),
		m.renderRoutingSummary(rightColWidth),
		m.renderNeighborsSummary(rightColWidth),
	}
	rightCol := lipgloss.JoinVertical(lipgloss.Left, rightPanels...)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftCol, "  ", rightCol)
}

func (m NetworkDashboardModel) renderSingleColumn(width int) string {
	panels := []string{
		m.renderTopInterfaces(width),
		m.renderInterfaceErrors(width),
		m.renderARPSummary(width),
		m.renderRoutingSummary(width),
		m.renderNeighborsSummary(width),
	}
	return lipgloss.JoinVertical(lipgloss.Left, panels...)
}

func (m NetworkDashboardModel) renderTopInterfaces(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Top Interfaces by Traffic"))
	b.WriteString("\n")

	if m.ifaceErr != nil {
		b.WriteString(errorStyle().Render("Error: " + m.ifaceErr.Error()))
		return panelStyle().Width(width).Render(b.String())
	}
	if m.interfaces == nil {
		b.WriteString(dimStyle().Render("Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	if len(m.interfaces) == 0 {
		b.WriteString(dimStyle().Render("No interfaces"))
		return panelStyle().Width(width).Render(b.String())
	}

	// Sort interfaces by total bytes (descending)
	sorted := make([]models.Interface, len(m.interfaces))
	copy(sorted, m.interfaces)
	sort.Slice(sorted, func(i, j int) bool {
		totalI := sorted[i].BytesIn + sorted[i].BytesOut
		totalJ := sorted[j].BytesIn + sorted[j].BytesOut
		return totalI > totalJ
	})

	// Show top 8 interfaces
	maxShow := 8
	if len(sorted) < maxShow {
		maxShow = len(sorted)
	}

	nameWidth := 16
	for i := 0; i < maxShow; i++ {
		iface := sorted[i]
		total := iface.BytesIn + iface.BytesOut
		if total == 0 && i > 3 {
			continue // Skip interfaces with no traffic after top 4
		}

		stateStyle := highlightStyle()
		if iface.State != "up" {
			stateStyle = dimStyle()
		}

		name := truncateEllipsis(iface.Name, nameWidth)
		b.WriteString(fmt.Sprintf("%-*s ", nameWidth, name))
		b.WriteString(stateStyle.Render(fmt.Sprintf("%-4s", iface.State)))
		b.WriteString(" ")
		b.WriteString(dimStyle().Render("In:"))
		b.WriteString(valueStyle().Render(formatBytes(iface.BytesIn)))
		b.WriteString(dimStyle().Render(" Out:"))
		b.WriteString(valueStyle().Render(formatBytes(iface.BytesOut)))
		b.WriteString("\n")
	}

	return panelStyle().Width(width).Render(strings.TrimSuffix(b.String(), "\n"))
}

func (m NetworkDashboardModel) renderInterfaceErrors(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Interface Errors & Drops"))
	b.WriteString("\n")

	if m.ifaceErr != nil || m.interfaces == nil {
		b.WriteString(dimStyle().Render("Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	// Find interfaces with errors or drops
	var problemIfaces []models.Interface
	for _, iface := range m.interfaces {
		if iface.ErrorsIn > 0 || iface.ErrorsOut > 0 || iface.DropsIn > 0 || iface.DropsOut > 0 {
			problemIfaces = append(problemIfaces, iface)
		}
	}

	if len(problemIfaces) == 0 {
		b.WriteString(highlightStyle().Render("No errors or drops detected"))
		return panelStyle().Width(width).Render(b.String())
	}

	// Sort by total errors/drops
	sort.Slice(problemIfaces, func(i, j int) bool {
		totalI := problemIfaces[i].ErrorsIn + problemIfaces[i].ErrorsOut + problemIfaces[i].DropsIn + problemIfaces[i].DropsOut
		totalJ := problemIfaces[j].ErrorsIn + problemIfaces[j].ErrorsOut + problemIfaces[j].DropsIn + problemIfaces[j].DropsOut
		return totalI > totalJ
	})

	maxShow := 6
	if len(problemIfaces) < maxShow {
		maxShow = len(problemIfaces)
	}

	nameWidth := 16
	for i := 0; i < maxShow; i++ {
		iface := problemIfaces[i]
		name := truncateEllipsis(iface.Name, nameWidth)

		b.WriteString(fmt.Sprintf("%-*s ", nameWidth, name))

		if iface.ErrorsIn > 0 || iface.ErrorsOut > 0 {
			b.WriteString(errorStyle().Render(fmt.Sprintf("Err:%d/%d ", iface.ErrorsIn, iface.ErrorsOut)))
		}
		if iface.DropsIn > 0 || iface.DropsOut > 0 {
			b.WriteString(warningStyle().Render(fmt.Sprintf("Drop:%d/%d", iface.DropsIn, iface.DropsOut)))
		}
		b.WriteString("\n")
	}

	return panelStyle().Width(width).Render(strings.TrimSuffix(b.String(), "\n"))
}

func (m NetworkDashboardModel) renderARPSummary(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("ARP Table"))
	b.WriteString("\n")

	if m.arpErr != nil {
		b.WriteString(dimStyle().Render("Not available"))
		return panelStyle().Width(width).Render(b.String())
	}
	if m.arpTable == nil {
		b.WriteString(dimStyle().Render("Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	if len(m.arpTable) == 0 {
		b.WriteString(dimStyle().Render("Empty"))
		return panelStyle().Width(width).Render(b.String())
	}

	// Count by interface
	ifaceCounts := make(map[string]int)
	completeCount := 0
	for _, entry := range m.arpTable {
		ifaceCounts[entry.Interface]++
		if strings.ToLower(entry.Status) == "complete" || strings.ToLower(entry.Status) == "c" {
			completeCount++
		}
	}

	// Summary
	b.WriteString(valueStyle().Render(fmt.Sprintf("%d", len(m.arpTable))))
	b.WriteString(dimStyle().Render(" entries ("))
	b.WriteString(highlightStyle().Render(fmt.Sprintf("%d", completeCount)))
	b.WriteString(dimStyle().Render(" complete)"))
	b.WriteString("\n\n")

	// Per-interface breakdown
	b.WriteString(subtitleStyle().Render("By Interface:"))
	b.WriteString("\n")

	// Sort interfaces by count
	type ifaceCount struct {
		name  string
		count int
	}
	counts := make([]ifaceCount, 0, len(ifaceCounts))
	for name, count := range ifaceCounts {
		counts = append(counts, ifaceCount{name, count})
	}
	sort.Slice(counts, func(i, j int) bool {
		return counts[i].count > counts[j].count
	})

	maxShow := 6
	if len(counts) < maxShow {
		maxShow = len(counts)
	}
	for i := 0; i < maxShow; i++ {
		ic := counts[i]
		name := truncateEllipsis(ic.name, 16)
		b.WriteString(labelStyle().Render(fmt.Sprintf("  %-16s ", name)))
		b.WriteString(valueStyle().Render(fmt.Sprintf("%d", ic.count)))
		if i < maxShow-1 {
			b.WriteString("\n")
		}
	}

	return panelStyle().Width(width).Render(b.String())
}

func (m NetworkDashboardModel) renderRoutingSummary(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Routing Table"))
	b.WriteString("\n")

	if m.routeErr != nil {
		b.WriteString(dimStyle().Render("Not available"))
		return panelStyle().Width(width).Render(b.String())
	}
	if m.routes == nil {
		b.WriteString(dimStyle().Render("Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	if len(m.routes) == 0 {
		b.WriteString(dimStyle().Render("Empty"))
		return panelStyle().Width(width).Render(b.String())
	}

	// Count by protocol
	protocolCounts := make(map[string]int)
	for _, route := range m.routes {
		proto := route.Protocol
		if proto == "" {
			proto = "static"
		}
		protocolCounts[proto]++
	}

	// Total routes
	b.WriteString(valueStyle().Render(fmt.Sprintf("%d", len(m.routes))))
	b.WriteString(dimStyle().Render(" routes"))
	b.WriteString("\n\n")

	// By protocol breakdown
	b.WriteString(subtitleStyle().Render("By Protocol:"))
	b.WriteString("\n")

	protocols := []string{"connected", "static", "bgp", "ospf"}
	for _, proto := range protocols {
		if count, ok := protocolCounts[proto]; ok && count > 0 {
			protoStyle := dimStyle()
			switch proto {
			case "connected":
				protoStyle = highlightStyle()
			case "bgp":
				protoStyle = accentStyle()
			case "ospf":
				protoStyle = warningStyle()
			}
			b.WriteString(protoStyle.Render(fmt.Sprintf("  %-12s ", proto)))
			b.WriteString(valueStyle().Render(fmt.Sprintf("%d", count)))
			b.WriteString("\n")
		}
	}

	// Show any other protocols
	for proto, count := range protocolCounts {
		found := false
		for _, p := range protocols {
			if p == proto {
				found = true
				break
			}
		}
		if !found && count > 0 {
			b.WriteString(dimStyle().Render(fmt.Sprintf("  %-12s ", proto)))
			b.WriteString(valueStyle().Render(fmt.Sprintf("%d", count)))
			b.WriteString("\n")
		}
	}

	return panelStyle().Width(width).Render(strings.TrimSuffix(b.String(), "\n"))
}

func (m NetworkDashboardModel) renderNeighborsSummary(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Routing Neighbors"))
	b.WriteString("\n")

	hasBGP := m.bgpNeighbors != nil && len(m.bgpNeighbors) > 0
	hasOSPF := m.ospfNeighbors != nil && len(m.ospfNeighbors) > 0

	// Both are still loading
	if m.bgpNeighbors == nil && m.ospfNeighbors == nil && m.bgpErr == nil && m.ospfErr == nil {
		b.WriteString(dimStyle().Render("Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	if !hasBGP && !hasOSPF {
		b.WriteString(dimStyle().Render("No BGP or OSPF neighbors"))
		return panelStyle().Width(width).Render(b.String())
	}

	// BGP Neighbors
	if hasBGP {
		b.WriteString(subtitleStyle().Render("BGP Peers:"))
		b.WriteString("\n")

		// Count established
		established := 0
		for _, n := range m.bgpNeighbors {
			state := strings.ToLower(n.State)
			if state == "established" || state == "openconfirm" {
				established++
			}
		}
		b.WriteString(dimStyle().Render("  "))
		b.WriteString(highlightStyle().Render(fmt.Sprintf("%d", established)))
		b.WriteString(dimStyle().Render(fmt.Sprintf("/%d established", len(m.bgpNeighbors))))
		b.WriteString("\n")

		// Show up to 4 peers
		maxShow := 4
		if len(m.bgpNeighbors) < maxShow {
			maxShow = len(m.bgpNeighbors)
		}
		for i := 0; i < maxShow; i++ {
			n := m.bgpNeighbors[i]
			state := strings.ToLower(n.State)
			stateStyle := dimStyle()
			stateIcon := "x"
			if state == "established" || state == "openconfirm" {
				stateStyle = highlightStyle()
				stateIcon = "o"
			} else if state == "connect" || state == "active" || state == "opensent" {
				stateStyle = warningStyle()
				stateIcon = "~"
			}

			peer := truncateEllipsis(n.PeerAddress, 15)
			b.WriteString("  ")
			b.WriteString(stateStyle.Render(stateIcon))
			b.WriteString(" ")
			b.WriteString(valueStyle().Render(fmt.Sprintf("%-15s", peer)))
			b.WriteString(dimStyle().Render(fmt.Sprintf(" AS%d", n.PeerAS)))
			if n.PrefixesReceived > 0 {
				b.WriteString(dimStyle().Render(fmt.Sprintf(" (%d pfx)", n.PrefixesReceived)))
			}
			b.WriteString("\n")
		}
		if len(m.bgpNeighbors) > maxShow {
			b.WriteString(dimStyle().Render(fmt.Sprintf("  ... and %d more\n", len(m.bgpNeighbors)-maxShow)))
		}
	}

	// OSPF Neighbors
	if hasOSPF {
		if hasBGP {
			b.WriteString("\n")
		}
		b.WriteString(subtitleStyle().Render("OSPF Neighbors:"))
		b.WriteString("\n")

		// Count full adjacencies
		full := 0
		for _, n := range m.ospfNeighbors {
			state := strings.ToLower(n.State)
			if state == "full" {
				full++
			}
		}
		b.WriteString(dimStyle().Render("  "))
		b.WriteString(highlightStyle().Render(fmt.Sprintf("%d", full)))
		b.WriteString(dimStyle().Render(fmt.Sprintf("/%d full adjacency", len(m.ospfNeighbors))))
		b.WriteString("\n")

		// Show up to 4 neighbors
		maxShow := 4
		if len(m.ospfNeighbors) < maxShow {
			maxShow = len(m.ospfNeighbors)
		}
		for i := 0; i < maxShow; i++ {
			n := m.ospfNeighbors[i]
			state := strings.ToLower(n.State)
			stateStyle := dimStyle()
			stateIcon := "x"
			if state == "full" {
				stateStyle = highlightStyle()
				stateIcon = "o"
			} else if state == "2-way" || state == "exstart" || state == "exchange" || state == "loading" {
				stateStyle = warningStyle()
				stateIcon = "~"
			}

			neighborID := truncateEllipsis(n.NeighborID, 15)
			b.WriteString("  ")
			b.WriteString(stateStyle.Render(stateIcon))
			b.WriteString(" ")
			b.WriteString(valueStyle().Render(fmt.Sprintf("%-15s", neighborID)))
			if n.Interface != "" {
				b.WriteString(dimStyle().Render(fmt.Sprintf(" %s", truncateEllipsis(n.Interface, 12))))
			}
			if n.Area != "" {
				b.WriteString(dimStyle().Render(fmt.Sprintf(" area %s", n.Area)))
			}
			b.WriteString("\n")
		}
		if len(m.ospfNeighbors) > maxShow {
			b.WriteString(dimStyle().Render(fmt.Sprintf("  ... and %d more\n", len(m.ospfNeighbors)-maxShow)))
		}
	}

	return panelStyle().Width(width).Render(strings.TrimSuffix(b.String(), "\n"))
}
