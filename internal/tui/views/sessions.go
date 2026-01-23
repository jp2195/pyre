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

type SessionSortField int

const (
	SessionSortID SessionSortField = iota
	SessionSortBytes
	SessionSortAge
	SessionSortApplication
)

type SessionsModel struct {
	sessions   []models.Session
	filtered   []models.Session
	err        error
	cursor     int
	offset     int
	filterMode bool
	filter     textinput.Model
	expanded   bool
	width      int
	height     int
	sortBy     SessionSortField
	sortAsc    bool
	loading    bool
}

func NewSessionsModel() SessionsModel {
	f := textinput.New()
	f.Placeholder = "Filter sessions..."
	f.CharLimit = 100
	f.Width = 40

	return SessionsModel{
		filter: f,
	}
}

func (m SessionsModel) SetSize(width, height int) SessionsModel {
	m.width = width
	m.height = height
	return m
}

func (m SessionsModel) SetLoading(loading bool) SessionsModel {
	m.loading = loading
	return m
}

func (m SessionsModel) SetSessions(sessions []models.Session, err error) SessionsModel {
	m.sessions = sessions
	m.err = err
	m.loading = false
	m.cursor = 0
	m.offset = 0
	m.applyFilter()
	return m
}

func (m *SessionsModel) applyFilter() {
	if m.filter.Value() == "" {
		m.filtered = make([]models.Session, len(m.sessions))
		copy(m.filtered, m.sessions)
	} else {
		query := strings.ToLower(m.filter.Value())
		m.filtered = nil

		for _, s := range m.sessions {
			if strings.Contains(strings.ToLower(s.Application), query) ||
				strings.Contains(s.SourceIP, query) ||
				strings.Contains(s.DestIP, query) ||
				strings.Contains(strings.ToLower(s.SourceZone), query) ||
				strings.Contains(strings.ToLower(s.DestZone), query) ||
				strings.Contains(strings.ToLower(s.Rule), query) ||
				strings.Contains(strings.ToLower(s.User), query) {
				m.filtered = append(m.filtered, s)
			}
		}
	}
	m.applySort()
}

func (m *SessionsModel) applySort() {
	sort.Slice(m.filtered, func(i, j int) bool {
		var less bool
		switch m.sortBy {
		case SessionSortBytes:
			less = (m.filtered[i].BytesIn + m.filtered[i].BytesOut) < (m.filtered[j].BytesIn + m.filtered[j].BytesOut)
		case SessionSortAge:
			less = m.filtered[i].StartTime.Before(m.filtered[j].StartTime)
		case SessionSortApplication:
			less = m.filtered[i].Application < m.filtered[j].Application
		default: // SessionSortID
			less = m.filtered[i].ID < m.filtered[j].ID
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
}

func (m *SessionsModel) cycleSort() {
	m.sortBy = (m.sortBy + 1) % 4
	// Default to descending for bytes/age, ascending for ID/app
	m.sortAsc = m.sortBy == SessionSortID || m.sortBy == SessionSortApplication
	m.applySort()
}

func (m SessionsModel) sortLabel() string {
	dir := "↓"
	if m.sortAsc {
		dir = "↑"
	}
	switch m.sortBy {
	case SessionSortBytes:
		return fmt.Sprintf("Bytes %s", dir)
	case SessionSortAge:
		return fmt.Sprintf("Age %s", dir)
	case SessionSortApplication:
		return fmt.Sprintf("App %s", dir)
	default:
		return fmt.Sprintf("ID %s", dir)
	}
}

func (m SessionsModel) Update(msg tea.Msg) (SessionsModel, tea.Cmd) {
	if m.filterMode {
		return m.updateFilter(msg)
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
			m.cursor = len(m.filtered) - 1
			m.ensureVisible()
		case "ctrl+d", "pgdown":
			m.cursor += 10
			if m.cursor >= len(m.filtered) {
				m.cursor = len(m.filtered) - 1
			}
			m.ensureVisible()
		case "ctrl+u", "pgup":
			m.cursor -= 10
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.ensureVisible()
		case "/":
			m.filterMode = true
			m.filter.Focus()
		case "enter":
			m.expanded = !m.expanded
		case "esc":
			if m.filter.Value() != "" {
				m.filter.SetValue("")
				m.applyFilter()
			}
		case "s":
			m.cycleSort()
			m.cursor = 0
			m.offset = 0
		}
	}

	return m, nil
}

func (m SessionsModel) updateFilter(msg tea.Msg) (SessionsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc":
			m.filterMode = false
			m.filter.Blur()
			m.applyFilter()
			m.cursor = 0
			m.offset = 0
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	return m, cmd
}

func (m *SessionsModel) ensureVisible() {
	visibleRows := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	} else if m.cursor >= m.offset+visibleRows {
		m.offset = m.cursor - visibleRows + 1
	}
}

func (m SessionsModel) visibleRows() int {
	rows := m.height - 8
	if m.expanded {
		rows -= 8
	}
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m SessionsModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7C3AED")).
		MarginBottom(1)

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#374151")).
		Padding(1, 2).
		Width(m.width - 4)

	var b strings.Builder
	title := fmt.Sprintf("Active Sessions (%d)", len(m.filtered))
	sortInfo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Render(fmt.Sprintf(" [Sort: %s, press 's' to change]", m.sortLabel()))
	b.WriteString(titleStyle.Render(title) + sortInfo)
	b.WriteString("\n")

	if m.filterMode {
		filterStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			Padding(0, 1)
		b.WriteString(filterStyle.Render(m.filter.View()))
		b.WriteString("\n\n")
	} else if m.filter.Value() != "" {
		filterInfo := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Render(fmt.Sprintf("Filter: %s (press / to edit, esc to clear)", m.filter.Value()))
		b.WriteString(filterInfo)
		b.WriteString("\n\n")
	}

	if m.err != nil {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Render("Error: " + m.err.Error()))
		return panelStyle.Render(b.String())
	}

	if m.loading || m.sessions == nil {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("Loading sessions..."))
		return panelStyle.Render(b.String())
	}

	if len(m.filtered) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("No sessions found"))
		return panelStyle.Render(b.String())
	}

	b.WriteString(m.renderTable())

	if m.expanded && m.cursor < len(m.filtered) {
		b.WriteString("\n\n")
		b.WriteString(m.renderDetail(m.filtered[m.cursor]))
	}

	return panelStyle.Render(b.String())
}

func (m SessionsModel) renderTable() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#374151"))

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#1F2937")).
		Foreground(lipgloss.Color("#FFFFFF"))

	// State-based colors
	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
	discardStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B"))
	closedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	var b strings.Builder

	// Header: ID, Source, Destination, Port, Proto, App, State, Zones, Age, Bytes
	header := fmt.Sprintf("%-7s %-15s %-15s %-5s %-4s %-10s %-7s %-15s %-5s %-8s",
		"ID", "Source", "Destination", "Port", "Pro", "App", "State", "Zones", "Age", "Bytes")
	b.WriteString(headerStyle.Render(header) + "\n")

	visibleRows := m.visibleRows()
	end := m.offset + visibleRows
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	for i := m.offset; i < end; i++ {
		s := m.filtered[i]
		isSelected := i == m.cursor

		// Format zone flow (e.g., "trust→untrust")
		zoneFlow := fmt.Sprintf("%s→%s", truncate(s.SourceZone, 7), truncate(s.DestZone, 7))

		// Format protocol
		proto := s.Protocol
		if proto == "" {
			proto = "tcp"
		}

		row := fmt.Sprintf("%-7d %-15s %-15s %-5d %-4s %-10s %-7s %-15s %-5s %-8s",
			s.ID,
			truncate(s.SourceIP, 15),
			truncate(s.DestIP, 15),
			s.DestPort,
			truncate(proto, 4),
			truncate(s.Application, 10),
			truncate(s.State, 7),
			truncate(zoneFlow, 15),
			formatDuration(s.StartTime),
			formatBytes(s.BytesIn+s.BytesOut))

		if isSelected {
			b.WriteString(selectedStyle.Render(row) + "\n")
		} else {
			// Color code by state - render state portion with color
			prefix := fmt.Sprintf("%-7d %-15s %-15s %-5d %-4s %-10s ",
				s.ID,
				truncate(s.SourceIP, 15),
				truncate(s.DestIP, 15),
				s.DestPort,
				truncate(proto, 4),
				truncate(s.Application, 10))
			stateStr := fmt.Sprintf("%-7s", truncate(s.State, 7))
			suffix := fmt.Sprintf(" %-15s %-5s %-8s",
				truncate(zoneFlow, 15),
				formatDuration(s.StartTime),
				formatBytes(s.BytesIn+s.BytesOut))

			switch strings.ToUpper(s.State) {
			case "ACTIVE":
				b.WriteString(prefix + activeStyle.Render(stateStr) + suffix + "\n")
			case "DISCARD", "DROP":
				b.WriteString(prefix + discardStyle.Render(stateStr) + suffix + "\n")
			case "CLOSED", "INIT":
				b.WriteString(prefix + closedStyle.Render(stateStr) + suffix + "\n")
			default:
				b.WriteString(row + "\n")
			}
		}
	}

	return b.String()
}

func formatDuration(start time.Time) string {
	if start.IsZero() {
		return "-"
	}
	d := time.Since(start)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

func (m SessionsModel) renderDetail(s models.Session) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7C3AED"))

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF"))

	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Session Details: %d", s.ID)))
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("Application:   ") + valueStyle.Render(s.Application) + "\n")
	proto := s.Protocol
	if proto == "" {
		proto = "tcp"
	}
	b.WriteString(labelStyle.Render("Protocol:      ") + valueStyle.Render(proto) + "\n")
	b.WriteString(labelStyle.Render("State:         ") + valueStyle.Render(s.State) + "\n")
	b.WriteString(labelStyle.Render("Source:        ") + valueStyle.Render(fmt.Sprintf("%s:%d (%s)", s.SourceIP, s.SourcePort, s.SourceZone)) + "\n")
	b.WriteString(labelStyle.Render("Destination:   ") + valueStyle.Render(fmt.Sprintf("%s:%d (%s)", s.DestIP, s.DestPort, s.DestZone)) + "\n")
	if s.NATSourceIP != "" {
		b.WriteString(labelStyle.Render("NAT Source:    ") + valueStyle.Render(fmt.Sprintf("%s:%d", s.NATSourceIP, s.NATSourcePort)) + "\n")
	}
	if s.User != "" {
		b.WriteString(labelStyle.Render("User:          ") + valueStyle.Render(s.User) + "\n")
	}
	b.WriteString(labelStyle.Render("Rule:          ") + valueStyle.Render(s.Rule) + "\n")
	b.WriteString(labelStyle.Render("Bytes In:      ") + valueStyle.Render(formatBytes(s.BytesIn)) + "\n")
	b.WriteString(labelStyle.Render("Bytes Out:     ") + valueStyle.Render(formatBytes(s.BytesOut)) + "\n")
	if !s.StartTime.IsZero() {
		b.WriteString(labelStyle.Render("Start Time:    ") + valueStyle.Render(s.StartTime.Format("2006-01-02 15:04:05")) + "\n")
		b.WriteString(labelStyle.Render("Duration:      ") + valueStyle.Render(formatDuration(s.StartTime)))
	}

	return b.String()
}

// formatBytes is defined in helpers.go
