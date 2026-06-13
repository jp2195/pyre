package views

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/config"
)

// ConnectionEntry represents a connection displayed in the hub
type ConnectionEntry struct {
	Host          string // Host is the primary identifier
	Type          string // "firewall" or "panorama"
	Insecure      bool
	LastConnected time.Time
	LastUser      string
	IsDefault     bool
}

// ConnectionHubModel is the model for the connection hub view
type ConnectionHubModel struct {
	connections   []ConnectionEntry
	cursor        int
	width         int
	height        int
	showConfirm   bool   // delete confirmation
	confirmTarget string // name of connection to delete
}

// NewConnectionHubModel creates a new connection hub model
func NewConnectionHubModel() ConnectionHubModel {
	return ConnectionHubModel{
		connections: []ConnectionEntry{},
		cursor:      0,
	}
}

// SetConnections updates the list of connections from config and state
func (m ConnectionHubModel) SetConnections(cfg *config.Config, state *config.State) ConnectionHubModel {
	entries := make([]ConnectionEntry, 0, len(cfg.Connections))

	for host, conn := range cfg.Connections {
		entry := ConnectionEntry{
			Host:     host,
			Type:     conn.Type,
			Insecure: conn.Insecure,
		}

		// Default to "firewall" if type is not set
		if entry.Type == "" {
			entry.Type = "firewall"
		}

		// Set default marker
		if host == cfg.Default {
			entry.IsDefault = true
		}

		// Load state if available
		if state != nil {
			if connState := state.GetConnection(host); connState != nil {
				entry.LastConnected = connState.LastConnected
				entry.LastUser = connState.LastUser
			}
		}

		entries = append(entries, entry)
	}

	// Sort: default first, then by last connected, then alphabetically by host
	slices.SortFunc(entries, func(a, b ConnectionEntry) int {
		if a.IsDefault != b.IsDefault {
			if a.IsDefault {
				return -1
			}
			return 1
		}
		if !a.LastConnected.IsZero() && !b.LastConnected.IsZero() {
			return b.LastConnected.Compare(a.LastConnected)
		}
		if a.LastConnected.IsZero() != b.LastConnected.IsZero() {
			if !a.LastConnected.IsZero() {
				return -1
			}
			return 1
		}
		return cmp.Compare(a.Host, b.Host)
	})

	m.connections = entries

	// Reset cursor if out of bounds
	if m.cursor >= len(entries) {
		m.cursor = len(entries) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}

	return m
}

// SetSize sets the dimensions for the view
func (m ConnectionHubModel) SetSize(width, height int) ConnectionHubModel {
	m.width = width
	m.height = height
	return m
}

// Selected returns the currently selected connection entry
func (m ConnectionHubModel) Selected() *ConnectionEntry {
	if m.cursor >= 0 && m.cursor < len(m.connections) {
		return &m.connections[m.cursor]
	}
	return nil
}

// SelectedHost returns the host of the selected connection
func (m ConnectionHubModel) SelectedHost() string {
	if entry := m.Selected(); entry != nil {
		return entry.Host
	}
	return ""
}

// ShowDeleteConfirm shows the delete confirmation dialog
func (m ConnectionHubModel) ShowDeleteConfirm(name string) ConnectionHubModel {
	m.showConfirm = true
	m.confirmTarget = name
	return m
}

// HideDeleteConfirm hides the delete confirmation dialog
func (m ConnectionHubModel) HideDeleteConfirm() ConnectionHubModel {
	m.showConfirm = false
	m.confirmTarget = ""
	return m
}

// IsConfirming returns true if showing delete confirmation
func (m ConnectionHubModel) IsConfirming() bool {
	return m.showConfirm
}

// ConfirmTarget returns the name of the connection being confirmed for deletion
func (m ConnectionHubModel) ConfirmTarget() string {
	return m.confirmTarget
}

// HasConnections returns true if there are any connections
func (m ConnectionHubModel) HasConnections() bool {
	return len(m.connections) > 0
}

// Update handles keyboard input for navigation
func (m ConnectionHubModel) Update(msg tea.Msg) (ConnectionHubModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.showConfirm {
			// In confirmation mode, only handle y/n/esc
			return m, nil
		}

		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.connections)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "g", "home":
			m.cursor = 0
		case "G", "end":
			if len(m.connections) > 0 {
				m.cursor = len(m.connections) - 1
			}
		}
	}
	return m, nil
}

// View renders the connection hub
func (m ConnectionHubModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	titleStyle := ViewTitleStyle.MarginBottom(1)
	panelStyle := ViewPanelStyle
	helpStyle := HelpDescStyle.MarginTop(1)

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("PYRE - Connections"))
	b.WriteString("\n\n")

	if len(m.connections) == 0 {
		b.WriteString(EmptyMsgStyle.Render("No connections configured."))
		b.WriteString("\n\n")
		b.WriteString(HelpDescStyle.Render("Press [n] to add a new connection or [q] for quick connect"))
	} else {
		// Render in original sorted order (recency-based) so the cursor
		// index maps directly to a row. An inline type tag distinguishes
		// firewalls from Panoramas without splitting the list — splitting
		// previously caused the cursor to highlight the wrong row when a
		// Panorama was interleaved between firewalls.
		b.WriteString(DetailSectionStyle.Render("CONNECTIONS"))
		b.WriteString("\n")
		for i, entry := range m.connections {
			b.WriteString(m.renderEntry(entry, i == m.cursor))
			b.WriteString("\n")
		}

		b.WriteString("\n")

		if m.showConfirm {
			confirmMsg := fmt.Sprintf("Delete %q? [y/n]", m.confirmTarget)
			b.WriteString(WarningMsgStyle.Render(confirmMsg))
		} else {
			b.WriteString(helpStyle.Render("[Enter] Connect  [n] New  [e] Edit  [d] Delete  [q] Quick Connect"))
		}
	}

	content := b.String()

	// Calculate box width based on content
	boxWidth := 70
	if m.width < boxWidth+10 {
		boxWidth = m.width - 10
	}
	if boxWidth < 40 {
		boxWidth = 40
	}

	box := panelStyle.Width(boxWidth).Render(content)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}

// renderEntry renders a single connection entry
func (m ConnectionHubModel) renderEntry(entry ConnectionEntry, selected bool) string {
	rowStyle := TableRowNormalStyle
	if selected {
		rowStyle = TableRowSelectedStyle
	}

	// Indicator for default connection
	indicator := "  "
	if entry.IsDefault {
		indicator = StatusActiveStyle.Render("* ")
	}

	// Format host, truncating if needed
	host := entry.Host
	displayHost := host
	if len(displayHost) > 40 {
		displayHost = displayHost[:37] + "..."
	}

	// Inline type tag distinguishes firewalls from Panoramas now that the
	// list is rendered in a single section.
	typeTag := "[Firewall]"
	if entry.Type == "panorama" {
		typeTag = "[Panorama]"
	}

	// Format last connected time
	lastConnected := "Never connected"
	if !entry.LastConnected.IsZero() {
		lastConnected = formatTimeAgo(entry.LastConnected)
	}

	// Build the line - host is the primary identifier; type tag is rendered
	// inside rowStyle so the selection highlight covers the whole row.
	rowText := fmt.Sprintf("%-10s %-40s", typeTag, displayHost)
	hostStr := rowStyle.Render(rowText)
	timeStr := DetailDimStyle.Render(lastConnected)

	return indicator + hostStr + " " + timeStr
}
