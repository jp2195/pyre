package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDefault != entries[j].IsDefault {
			return entries[i].IsDefault
		}
		if !entries[i].LastConnected.IsZero() && !entries[j].LastConnected.IsZero() {
			return entries[i].LastConnected.After(entries[j].LastConnected)
		}
		if entries[i].LastConnected.IsZero() != entries[j].LastConnected.IsZero() {
			return !entries[i].LastConnected.IsZero()
		}
		return entries[i].Host < entries[j].Host
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
	case tea.KeyMsg:
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
		// Group connections by type
		firewalls := make([]ConnectionEntry, 0)
		panoramas := make([]ConnectionEntry, 0)

		for _, entry := range m.connections {
			if entry.Type == "panorama" {
				panoramas = append(panoramas, entry)
			} else {
				firewalls = append(firewalls, entry)
			}
		}

		// Track position across both groups for cursor highlighting
		pos := 0

		// Render firewalls section
		if len(firewalls) > 0 {
			b.WriteString(DetailSectionStyle.Render("FIREWALLS"))
			b.WriteString("\n")
			for _, entry := range firewalls {
				b.WriteString(m.renderEntry(entry, pos == m.cursor))
				b.WriteString("\n")
				pos++
			}
		}

		// Render panorama section
		if len(panoramas) > 0 {
			if len(firewalls) > 0 {
				b.WriteString("\n")
			}
			b.WriteString(DetailSectionStyle.Render("PANORAMA"))
			b.WriteString("\n")
			for _, entry := range panoramas {
				b.WriteString(m.renderEntry(entry, pos == m.cursor))
				b.WriteString("\n")
				pos++
			}
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

	// Format last connected time
	lastConnected := "Never connected"
	if !entry.LastConnected.IsZero() {
		lastConnected = formatConnectionTimeAgo(entry.LastConnected)
	}

	// Build the line - host is now the primary identifier
	hostStr := rowStyle.Render(fmt.Sprintf("%-40s", displayHost))
	timeStr := DetailDimStyle.Render(lastConnected)

	return indicator + hostStr + " " + timeStr
}

// formatConnectionTimeAgo returns a human-readable relative time
func formatConnectionTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}

	d := time.Since(t)

	if d < time.Minute {
		return "Just now"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", mins)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	}
	if d < 7*24*time.Hour {
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	}
	if d < 30*24*time.Hour {
		weeks := int(d.Hours() / 24 / 7)
		if weeks == 1 {
			return "1w ago"
		}
		return fmt.Sprintf("%dw ago", weeks)
	}

	return t.Format("Jan 2, 2006")
}
