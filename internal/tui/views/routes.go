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

// RoutesTab represents which tab is active in the routes view
type RoutesTab int

const (
	RoutesTabRoutes RoutesTab = iota
	RoutesTabNeighbors
)

type RouteSortField int

const (
	RouteSortDestination RouteSortField = iota
	RouteSortProtocol
	RouteSortNexthop
	RouteSortInterface
)

type RoutesModel struct {
	TableBase
	routes        []models.RouteEntry
	filtered      []models.RouteEntry
	bgpNeighbors  []models.BGPNeighbor
	ospfNeighbors []models.OSPFNeighbor

	routeErr error
	bgpErr   error
	ospfErr  error

	sortBy         RouteSortField
	lastRefresh    time.Time
	neighborCursor int
	neighborOffset int

	activeTab      RoutesTab
	protocolFilter string // Filter by protocol: "", "connected", "static", "bgp", "ospf"
}

func NewRoutesModel() RoutesModel {
	base := NewTableBase("Filter routes...")
	base.SortAsc = true

	return RoutesModel{
		TableBase: base,
		activeTab: RoutesTabRoutes,
	}
}

func (m RoutesModel) SetSize(width, height int) RoutesModel {
	m.TableBase = m.TableBase.SetSize(width, height)

	// Clamp route cursor
	count := len(m.filtered)
	if m.Cursor >= count && count > 0 {
		m.Cursor = count - 1
	}

	// Clamp neighbor cursor
	neighborCount := len(m.bgpNeighbors) + len(m.ospfNeighbors)
	if m.neighborCursor >= neighborCount && neighborCount > 0 {
		m.neighborCursor = neighborCount - 1
	}

	return m
}

func (m RoutesModel) SetLoading(loading bool) RoutesModel {
	m.TableBase = m.TableBase.SetLoading(loading)
	return m
}

// HasData returns true if routes have been loaded.
func (m RoutesModel) HasData() bool {
	return m.routes != nil || m.routeErr != nil
}

func (m RoutesModel) SetRoutes(routes []models.RouteEntry, err error) RoutesModel {
	m.routes = routes
	m.routeErr = err
	m.Loading = false
	m.lastRefresh = time.Now()
	m.Cursor = 0
	m.Offset = 0
	m.applyFilter()
	return m
}

func (m RoutesModel) SetBGPNeighbors(neighbors []models.BGPNeighbor, err error) RoutesModel {
	m.bgpNeighbors = neighbors
	m.bgpErr = err
	return m
}

func (m RoutesModel) SetOSPFNeighbors(neighbors []models.OSPFNeighbor, err error) RoutesModel {
	m.ospfNeighbors = neighbors
	m.ospfErr = err
	return m
}

func (m *RoutesModel) applyFilter() {
	filterText := strings.ToLower(m.FilterValue())

	m.filtered = nil
	for _, route := range m.routes {
		// Apply protocol filter
		if m.protocolFilter != "" && route.Protocol != m.protocolFilter {
			continue
		}

		// Apply text filter
		if filterText != "" {
			searchable := strings.ToLower(
				route.Destination + " " +
					route.Nexthop + " " +
					route.Interface + " " +
					route.Protocol + " " +
					route.VirtualRouter,
			)
			if !strings.Contains(searchable, filterText) {
				continue
			}
		}
		m.filtered = append(m.filtered, route)
	}

	// Sort the filtered routes
	m.sortRoutes()
}

func (m *RoutesModel) sortRoutes() {
	sort.SliceStable(m.filtered, func(i, j int) bool {
		var less bool
		switch m.sortBy {
		case RouteSortProtocol:
			less = m.filtered[i].Protocol < m.filtered[j].Protocol
		case RouteSortNexthop:
			less = m.filtered[i].Nexthop < m.filtered[j].Nexthop
		case RouteSortInterface:
			less = m.filtered[i].Interface < m.filtered[j].Interface
		default: // RouteSortDestination
			less = m.filtered[i].Destination < m.filtered[j].Destination
		}
		if !m.SortAsc {
			return !less
		}
		return less
	})
}

func (m RoutesModel) visibleRows() int {
	rows := m.Height - 10
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m RoutesModel) Update(msg tea.Msg) (RoutesModel, tea.Cmd) {
	// Handle filter mode first
	if m.FilterMode {
		return m.updateFilterMode(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle tab switching with [ and ]
		if msg.String() == "[" || msg.String() == "]" {
			if m.activeTab == RoutesTabRoutes {
				m.activeTab = RoutesTabNeighbors
			} else {
				m.activeTab = RoutesTabRoutes
			}
			return m, nil
		}

		// Handle protocol filter shortcuts (only on routes tab)
		if m.activeTab == RoutesTabRoutes {
			switch msg.String() {
			case "a": // All protocols
				m.protocolFilter = ""
				m.applyFilter()
				return m, nil
			case "c": // Connected
				m.protocolFilter = "connected"
				m.applyFilter()
				return m, nil
			case "s": // Static
				m.protocolFilter = "static"
				m.applyFilter()
				return m, nil
			case "b": // BGP
				m.protocolFilter = "bgp"
				m.applyFilter()
				return m, nil
			case "o": // OSPF
				m.protocolFilter = "ospf"
				m.applyFilter()
				return m, nil
			}

			// Delegate to TableBase for common navigation
			visible := m.visibleRows()
			base, handled, cmd := m.HandleNavigation(msg, len(m.filtered), visible)
			if handled {
				m.TableBase = base
				return m, cmd
			}
		} else {
			// Neighbors tab navigation
			neighborCount := len(m.bgpNeighbors) + len(m.ospfNeighbors)
			visible := m.visibleRows()
			switch msg.String() {
			case "j", "down":
				if m.neighborCursor < neighborCount-1 {
					m.neighborCursor++
					if m.neighborCursor >= m.neighborOffset+visible {
						m.neighborOffset = m.neighborCursor - visible + 1
					}
				}
				return m, nil
			case "k", "up":
				if m.neighborCursor > 0 {
					m.neighborCursor--
					if m.neighborCursor < m.neighborOffset {
						m.neighborOffset = m.neighborCursor
					}
				}
				return m, nil
			case "g", "home":
				m.neighborCursor = 0
				m.neighborOffset = 0
				return m, nil
			case "G", "end":
				if neighborCount > 0 {
					m.neighborCursor = neighborCount - 1
					if m.neighborCursor >= m.neighborOffset+visible {
						m.neighborOffset = m.neighborCursor - visible + 1
					}
				}
				return m, nil
			}
		}
	}

	return m, nil
}

func (m RoutesModel) updateFilterMode(msg tea.Msg) (RoutesModel, tea.Cmd) {
	base, exited, cmd := m.HandleFilterMode(msg)
	m.TableBase = base
	if exited {
		m.applyFilter()
	}
	return m, cmd
}

func (m RoutesModel) View() string {
	if m.Width == 0 {
		return "Loading..."
	}

	var b strings.Builder
	c := theme.Colors()

	// Render tabs
	tabStyle := lipgloss.NewStyle().Padding(0, 2)
	activeTabStyle := tabStyle.Background(c.Primary).Foreground(c.White)
	inactiveTabStyle := tabStyle.Foreground(c.TextMuted)

	routesTab := inactiveTabStyle.Render("Routes")
	neighborsTab := inactiveTabStyle.Render("Neighbors")
	if m.activeTab == RoutesTabRoutes {
		routesTab = activeTabStyle.Render("Routes")
	} else {
		neighborsTab = activeTabStyle.Render("Neighbors")
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, routesTab, " ", neighborsTab)
	b.WriteString(tabBar)
	b.WriteString("\n\n")

	if m.activeTab == RoutesTabRoutes {
		b.WriteString(m.renderRoutesTab())
	} else {
		b.WriteString(m.renderNeighborsTab())
	}

	// Help text
	b.WriteString("\n")
	if m.activeTab == RoutesTabRoutes {
		filterInfo := ""
		if m.protocolFilter != "" {
			filterInfo = fmt.Sprintf(" [%s]", m.protocolFilter)
		}
		b.WriteString(lipgloss.NewStyle().Foreground(c.TextMuted).Render(
			fmt.Sprintf("[/] switch  /filter  a/c/s/b/o protocol%s  r refresh", filterInfo)))
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(c.TextMuted).Render(
			"[/] switch  r refresh"))
	}

	return b.String()
}

func (m RoutesModel) renderRoutesTab() string {
	var b strings.Builder
	c := theme.Colors()

	// Show loading or error state
	if m.Loading {
		return RenderLoadingInline(m.SpinnerFrame, "Loading routes...")
	}
	if m.routeErr != nil {
		return lipgloss.NewStyle().Foreground(c.Error).Render("Error: " + m.routeErr.Error())
	}
	if m.routes == nil {
		return RenderLoadingInline(m.SpinnerFrame, "Loading routes...")
	}

	// Summary line
	total := len(m.routes)
	showing := len(m.filtered)
	summaryText := fmt.Sprintf("%d routes", total)
	if showing != total {
		summaryText = fmt.Sprintf("%d of %d routes", showing, total)
	}
	if m.protocolFilter != "" {
		summaryText += fmt.Sprintf(" (filter: %s)", m.protocolFilter)
	}
	b.WriteString(lipgloss.NewStyle().Foreground(c.TextLabel).Render(summaryText))
	b.WriteString("\n")

	// Filter input if active
	if m.FilterMode {
		b.WriteString(FilterBorderStyle.Render(m.Filter.View()))
		b.WriteString("\n")
	}

	// Routes table
	b.WriteString(m.renderRoutesTable())

	return b.String()
}

func (m RoutesModel) renderRoutesTable() string {
	headerStyle := DetailLabelStyle.Bold(true)
	selectedStyle := TableRowSelectedStyle.Bold(true)
	normalStyle := DetailValueStyle
	dimStyle := DetailDimStyle

	availableWidth := m.Width - 6

	var b strings.Builder

	// Header
	header := m.formatRouteHeaderRow(availableWidth)
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", minInt(availableWidth, len(header)+10))))
	b.WriteString("\n")

	visibleRows := m.visibleRows()
	end := min(m.Offset+visibleRows, len(m.filtered))

	for i := m.Offset; i < end; i++ {
		route := m.filtered[i]
		isSelected := i == m.Cursor

		row := m.formatRouteRow(route, availableWidth)

		if isSelected {
			b.WriteString(selectedStyle.Render(row))
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

func (m RoutesModel) formatRouteHeaderRow(width int) string {
	if width >= 100 {
		return fmt.Sprintf("%-6s %-22s %-18s %-14s %-6s %-12s",
			"Proto", "Destination", "Next Hop", "Interface", "Metric", "VR")
	} else if width >= 70 {
		return fmt.Sprintf("%-6s %-20s %-16s %-12s %-6s",
			"Proto", "Destination", "Next Hop", "Interface", "Metric")
	}
	return fmt.Sprintf("%-6s %-18s %-16s",
		"Proto", "Destination", "Next Hop")
}

func (m RoutesModel) formatRouteRow(route models.RouteEntry, width int) string {
	proto := route.Protocol
	if proto == "" {
		proto = "static"
	}
	// Abbreviate protocol names
	switch proto {
	case "connected":
		proto = "C"
	case "static":
		proto = "S"
	case "local":
		proto = "L"
	case "bgp":
		proto = "B"
	case "ospf":
		proto = "O"
	}

	nexthop := route.Nexthop
	if nexthop == "" || nexthop == "directly connected" {
		nexthop = "direct"
	}

	metric := ""
	if route.Metric > 0 {
		metric = fmt.Sprintf("%d", route.Metric)
	}

	if width >= 100 {
		return fmt.Sprintf("%-6s %-22s %-18s %-14s %-6s %-12s",
			proto, truncateStr(route.Destination, 22),
			truncateStr(nexthop, 18), truncateStr(route.Interface, 14),
			metric, truncateStr(route.VirtualRouter, 12))
	} else if width >= 70 {
		return fmt.Sprintf("%-6s %-20s %-16s %-12s %-6s",
			proto, truncateStr(route.Destination, 20),
			truncateStr(nexthop, 16), truncateStr(route.Interface, 12),
			metric)
	}
	return fmt.Sprintf("%-6s %-18s %-16s",
		proto, truncateStr(route.Destination, 18),
		truncateStr(nexthop, 16))
}

func (m RoutesModel) renderNeighborsTab() string {
	var b strings.Builder
	c := theme.Colors()

	// Count neighbors
	bgpCount := len(m.bgpNeighbors)
	ospfCount := len(m.ospfNeighbors)
	totalCount := bgpCount + ospfCount

	if totalCount == 0 && m.bgpErr == nil && m.ospfErr == nil && m.bgpNeighbors == nil && m.ospfNeighbors == nil {
		return RenderLoadingInline(m.SpinnerFrame, "Loading neighbors...")
	}

	if totalCount == 0 {
		return lipgloss.NewStyle().Foreground(c.TextMuted).Render("No BGP or OSPF neighbors configured")
	}

	// Summary line
	var parts []string
	if bgpCount > 0 {
		parts = append(parts, fmt.Sprintf("%d BGP peers", bgpCount))
	}
	if ospfCount > 0 {
		parts = append(parts, fmt.Sprintf("%d OSPF neighbors", ospfCount))
	}
	b.WriteString(lipgloss.NewStyle().Foreground(c.TextLabel).Render(strings.Join(parts, ", ")))
	b.WriteString("\n")

	// Neighbors table
	b.WriteString(m.renderNeighborsTable())

	return b.String()
}

func (m RoutesModel) renderNeighborsTable() string {
	headerStyle := DetailLabelStyle.Bold(true)
	selectedStyle := TableRowSelectedStyle.Bold(true)
	normalStyle := DetailValueStyle
	dimStyle := DetailDimStyle

	availableWidth := m.Width - 6

	var b strings.Builder

	// Header
	header := m.formatNeighborHeaderRow(availableWidth)
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", minInt(availableWidth, len(header)+10))))
	b.WriteString("\n")

	// Build combined neighbor list
	type neighborRow struct {
		nType   string
		peer    string
		state   string
		asArea  string
		prefix  string
		uptime  string
		vr      string
	}

	var rows []neighborRow
	for _, n := range m.bgpNeighbors {
		state := n.State
		if len(state) > 12 {
			state = state[:12]
		}
		prefixes := ""
		if n.PrefixesReceived > 0 {
			prefixes = fmt.Sprintf("%d", n.PrefixesReceived)
		}
		rows = append(rows, neighborRow{
			nType:  "BGP",
			peer:   n.PeerAddress,
			state:  state,
			asArea: fmt.Sprintf("AS%d", n.PeerAS),
			prefix: prefixes,
			uptime: n.Uptime,
			vr:     n.VirtualRouter,
		})
	}
	for _, n := range m.ospfNeighbors {
		rows = append(rows, neighborRow{
			nType:  "OSPF",
			peer:   n.NeighborID,
			state:  n.State,
			asArea: n.Area,
			prefix: "-",
			uptime: n.DeadTime,
			vr:     n.VirtualRouter,
		})
	}

	visibleRows := m.visibleRows()
	end := min(m.neighborOffset+visibleRows, len(rows))

	for i := m.neighborOffset; i < end; i++ {
		r := rows[i]
		isSelected := i == m.neighborCursor

		row := m.formatNeighborRow(r.nType, r.peer, r.state, r.asArea, r.prefix, r.uptime, r.vr, availableWidth)

		if isSelected {
			b.WriteString(selectedStyle.Render(row))
		} else {
			b.WriteString(normalStyle.Render(row))
		}
		b.WriteString("\n")
	}

	// Show scroll indicator if needed
	if len(rows) > visibleRows {
		scrollInfo := fmt.Sprintf("  Showing %d-%d of %d", m.neighborOffset+1, end, len(rows))
		b.WriteString(dimStyle.Render(scrollInfo))
	}

	return b.String()
}

func (m RoutesModel) formatNeighborHeaderRow(width int) string {
	if width >= 100 {
		return fmt.Sprintf("%-5s %-16s %-12s %-10s %-8s %-12s %-12s",
			"Type", "Peer/Neighbor", "State", "AS/Area", "Prefixes", "Uptime", "VR")
	} else if width >= 70 {
		return fmt.Sprintf("%-5s %-16s %-12s %-10s %-8s %-12s",
			"Type", "Peer/Neighbor", "State", "AS/Area", "Prefixes", "Uptime")
	}
	return fmt.Sprintf("%-5s %-16s %-12s %-10s",
		"Type", "Peer/Neighbor", "State", "AS/Area")
}

func (m RoutesModel) formatNeighborRow(nType, peer, state, asArea, prefix, uptime, vr string, width int) string {
	if width >= 100 {
		return fmt.Sprintf("%-5s %-16s %-12s %-10s %-8s %-12s %-12s",
			nType, truncateStr(peer, 16), truncateStr(state, 12),
			truncateStr(asArea, 10), prefix, truncateStr(uptime, 12),
			truncateStr(vr, 12))
	} else if width >= 70 {
		return fmt.Sprintf("%-5s %-16s %-12s %-10s %-8s %-12s",
			nType, truncateStr(peer, 16), truncateStr(state, 12),
			truncateStr(asArea, 10), prefix, truncateStr(uptime, 12))
	}
	return fmt.Sprintf("%-5s %-16s %-12s %-10s",
		nType, truncateStr(peer, 16), truncateStr(state, 12),
		truncateStr(asArea, 10))
}

// IsFilterMode returns true if the filter input is active
func (m RoutesModel) IsFilterMode() bool {
	return m.FilterMode
}
