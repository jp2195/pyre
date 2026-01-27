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
	table          table.Model
	neighborsTable table.Model

	activeTab      RoutesTab
	protocolFilter string // Filter by protocol: "", "connected", "static", "bgp", "ospf"
}

func NewRoutesModel() RoutesModel {
	base := NewTableBase("Filter routes...")
	base.SortAsc = true

	// Initialize routes table with columns
	routeColumns := []table.Column{
		{Title: "Proto", Width: 5},
		{Title: "Destination", Width: 20},
		{Title: "Next Hop", Width: 18},
		{Title: "Interface", Width: 14},
		{Title: "Metric", Width: 6},
		{Title: "VR", Width: 12},
	}

	t := table.New(
		table.WithColumns(routeColumns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	// Initialize neighbors table
	neighborColumns := []table.Column{
		{Title: "Type", Width: 5},
		{Title: "Peer/Neighbor", Width: 16},
		{Title: "State", Width: 12},
		{Title: "AS/Area", Width: 10},
		{Title: "Prefixes", Width: 8},
		{Title: "Uptime", Width: 12},
		{Title: "VR", Width: 12},
	}

	nt := table.New(
		table.WithColumns(neighborColumns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	// Apply theme-aware styles to both tables
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
	nt.SetStyles(s)

	return RoutesModel{
		TableBase:      base,
		table:          t,
		neighborsTable: nt,
		activeTab:      RoutesTabRoutes,
	}
}

func (m RoutesModel) SetSize(width, height int) RoutesModel {
	m.TableBase = m.TableBase.SetSize(width, height)

	// Update table dimensions
	tableHeight := height - 10 // Account for header, tabs, filter, help
	if tableHeight < 3 {
		tableHeight = 3
	}
	m.table.SetHeight(tableHeight)
	m.neighborsTable.SetHeight(tableHeight)

	// Update column widths proportionally
	availWidth := width - 4

	routeColumns := []table.Column{
		{Title: "Proto", Width: 6},
		{Title: "Destination", Width: maxInt(18, availWidth*22/100)},
		{Title: "Next Hop", Width: maxInt(15, availWidth*20/100)},
		{Title: "Interface", Width: maxInt(12, availWidth*15/100)},
		{Title: "Metric", Width: 6},
		{Title: "VR", Width: maxInt(10, availWidth*12/100)},
	}
	m.table.SetColumns(routeColumns)

	neighborColumns := []table.Column{
		{Title: "Type", Width: 5},
		{Title: "Peer/Neighbor", Width: maxInt(15, availWidth*18/100)},
		{Title: "State", Width: maxInt(10, availWidth*14/100)},
		{Title: "AS/Area", Width: maxInt(10, availWidth*12/100)},
		{Title: "Prefixes", Width: 8},
		{Title: "Uptime", Width: maxInt(10, availWidth*14/100)},
		{Title: "VR", Width: maxInt(10, availWidth*12/100)},
	}
	m.neighborsTable.SetColumns(neighborColumns)

	// Clamp cursor to valid range after resize
	if m.table.Cursor() >= len(m.filtered) && len(m.filtered) > 0 {
		m.table.SetCursor(len(m.filtered) - 1)
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
	m.applyFilter()
	m.updateTableRows()
	return m
}

func (m RoutesModel) SetBGPNeighbors(neighbors []models.BGPNeighbor, err error) RoutesModel {
	m.bgpNeighbors = neighbors
	m.bgpErr = err
	m.updateNeighborsTable()
	return m
}

func (m RoutesModel) SetOSPFNeighbors(neighbors []models.OSPFNeighbor, err error) RoutesModel {
	m.ospfNeighbors = neighbors
	m.ospfErr = err
	m.updateNeighborsTable()
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

func (m *RoutesModel) updateTableRows() {
	rows := make([]table.Row, len(m.filtered))
	for i, route := range m.filtered {
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

		rows[i] = table.Row{
			proto,
			route.Destination,
			nexthop,
			route.Interface,
			metric,
			route.VirtualRouter,
		}
	}
	m.table.SetRows(rows)
}

func (m *RoutesModel) updateNeighborsTable() {
	rows := make([]table.Row, 0, len(m.bgpNeighbors)+len(m.ospfNeighbors))

	// Add BGP neighbors
	for _, n := range m.bgpNeighbors {
		state := n.State
		stateAbbr := state
		if len(stateAbbr) > 12 {
			stateAbbr = stateAbbr[:12]
		}

		prefixes := ""
		if n.PrefixesReceived > 0 {
			prefixes = fmt.Sprintf("%d", n.PrefixesReceived)
		}

		rows = append(rows, table.Row{
			"BGP",
			n.PeerAddress,
			stateAbbr,
			fmt.Sprintf("AS%d", n.PeerAS),
			prefixes,
			n.Uptime,
			n.VirtualRouter,
		})
	}

	// Add OSPF neighbors
	for _, n := range m.ospfNeighbors {
		rows = append(rows, table.Row{
			"OSPF",
			n.NeighborID,
			n.State,
			n.Area,
			"-",
			n.DeadTime,
			n.VirtualRouter,
		})
	}

	m.neighborsTable.SetRows(rows)
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
				m.updateTableRows()
				return m, nil
			case "c": // Connected
				m.protocolFilter = "connected"
				m.applyFilter()
				m.updateTableRows()
				return m, nil
			case "s": // Static
				m.protocolFilter = "static"
				m.applyFilter()
				m.updateTableRows()
				return m, nil
			case "b": // BGP
				m.protocolFilter = "bgp"
				m.applyFilter()
				m.updateTableRows()
				return m, nil
			case "o": // OSPF
				m.protocolFilter = "ospf"
				m.applyFilter()
				m.updateTableRows()
				return m, nil
			case "/":
				m.FilterMode = true
				m.Filter.Focus()
				return m, nil
			}
		}

		// Handle table navigation
		if m.activeTab == RoutesTabRoutes {
			m.table, _ = m.table.Update(msg)
		} else {
			m.neighborsTable, _ = m.neighborsTable.Update(msg)
		}
	}

	return m, nil
}

func (m RoutesModel) updateFilterMode(msg tea.Msg) (RoutesModel, tea.Cmd) {
	base, exited, cmd := m.HandleFilterMode(msg)
	m.TableBase = base
	if exited {
		m.applyFilter()
		m.updateTableRows()
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
	b.WriteString(m.table.View())

	return b.String()
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
	b.WriteString(m.neighborsTable.View())

	return b.String()
}

// IsFilterMode returns true if the filter input is active
func (m RoutesModel) IsFilterMode() bool {
	return m.FilterMode
}
