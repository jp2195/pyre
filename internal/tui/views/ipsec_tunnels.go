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

type IPSecTunnelsModel struct {
	list RuleListModel[models.IPSecTunnel]
}

func NewIPSecTunnelsModel() IPSecTunnelsModel {
	config := RuleListConfig[models.IPSecTunnel]{
		Title:             "IPSec Tunnels",
		ItemNoun:          "tunnels",
		LoadingMsg:        "Loading IPSec tunnels...",
		EmptyMsg:          "No IPSec tunnels found",
		FilterPlaceholder: "Filter tunnels...",
		SortLabels:        []string{"Name", "Gateway", "State", "Traffic"},
		DefaultSortAsc:    func(idx int) bool { return idx == 0 || idx == 1 },
		MatchFilter:       matchIPSecTunnel,
		CompareItems:      compareIPSecTunnel,
		FormatHeaderRow:   formatIPSecHeader,
		FormatRow:         formatTunnelRow,
		RenderDetail:      renderIPSecDetail,
		StyleRow:          styleTunnelRow,
	}
	return IPSecTunnelsModel{list: NewRuleListModel(config)}
}

func (m IPSecTunnelsModel) SetSize(width, height int) IPSecTunnelsModel {
	m.list = m.list.SetSize(width, height)
	return m
}

func (m IPSecTunnelsModel) SetLoading(loading bool) IPSecTunnelsModel {
	m.list = m.list.SetLoading(loading)
	return m
}

// SetSpinnerFrame updates the current spinner animation frame.
func (m IPSecTunnelsModel) SetSpinnerFrame(frame string) IPSecTunnelsModel {
	m.list.SpinnerFrame = frame
	return m
}

// HasData returns true if tunnel data has been loaded.
func (m IPSecTunnelsModel) HasData() bool {
	return m.list.HasData()
}

// IsFilterMode returns true while the filter text input is focused.
func (m IPSecTunnelsModel) IsFilterMode() bool {
	return m.list.IsFilterMode()
}

func (m IPSecTunnelsModel) SetTunnels(tunnels []models.IPSecTunnel, err error) IPSecTunnelsModel {
	m.list = m.list.SetItems(tunnels, err)
	return m
}

func (m IPSecTunnelsModel) Update(msg tea.Msg) (IPSecTunnelsModel, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m IPSecTunnelsModel) View() string {
	return m.list.View()
}

// --- Type-specific functions ---

func matchIPSecTunnel(t models.IPSecTunnel, query string) bool {
	return strings.Contains(strings.ToLower(t.Name), query) ||
		strings.Contains(strings.ToLower(t.Gateway), query) ||
		strings.Contains(strings.ToLower(t.State), query) ||
		strings.Contains(strings.ToLower(t.Protocol), query) ||
		strings.Contains(strings.ToLower(t.Encryption), query)
}

func compareIPSecTunnel(a, b models.IPSecTunnel, sortIdx int) bool {
	switch sortIdx {
	case 1: // Gateway
		return a.Gateway < b.Gateway
	case 2: // State
		return a.State < b.State
	case 3: // Traffic
		return a.BytesIn+a.BytesOut < b.BytesIn+b.BytesOut
	default: // Name
		return a.Name < b.Name
	}
}

// styleTunnelRow colors a non-selected row by tunnel state.
func styleTunnelRow(t models.IPSecTunnel, width int) string {
	row := formatTunnelRow(t, width)
	c := theme.Colors()
	switch t.State {
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

func formatIPSecHeader(width int) string {
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

func formatTunnelRow(t models.IPSecTunnel, width int) string {
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

func renderIPSecDetail(t models.IPSecTunnel, width int) string {
	c := theme.Colors()
	dr := NewDetailRenderer(width, 18)

	// Title with state indicator
	var stateStyle lipgloss.Style
	switch t.State {
	case "up":
		stateStyle = lipgloss.NewStyle().Foreground(c.Success).Bold(true)
	case "init":
		stateStyle = lipgloss.NewStyle().Foreground(c.Warning).Bold(true)
	default:
		stateStyle = lipgloss.NewStyle().Foreground(c.Error).Bold(true)
	}
	dr.Raw(ViewTitleStyle.Render(t.Name) + "  " + stateStyle.Render(strings.ToUpper(t.State)) + "\n")
	dr.Newline()

	dr.Section("Connection")
	dr.Field("Gateway:", t.Gateway)
	dr.FieldIf("Local IP:", t.LocalIP)
	dr.FieldIf("Remote IP:", t.RemoteIP)

	dr.Section("Security")
	dr.FieldIf("Protocol:", t.Protocol)
	dr.FieldIf("Encryption:", t.Encryption)
	dr.FieldIf("Authentication:", t.Auth)
	if t.LocalSPI != "" {
		dr.FieldDim("Local SPI:", t.LocalSPI)
	}
	if t.RemoteSPI != "" {
		dr.FieldDim("Remote SPI:", t.RemoteSPI)
	}

	dr.Section("Traffic Statistics")
	dr.Field("Bytes In:", formatBytes(t.BytesIn))
	dr.Field("Bytes Out:", formatBytes(t.BytesOut))
	dr.Field("Packets In:", strconv.FormatInt(t.PacketsIn, 10))
	dr.Field("Packets Out:", strconv.FormatInt(t.PacketsOut, 10))
	dr.FieldIf("Uptime:", t.Uptime)
	if t.Errors > 0 {
		dr.FieldStyled("Errors:", lipgloss.NewStyle().Foreground(c.Error).Render(strconv.Itoa(t.Errors)))
	}

	return dr.Render()
}
