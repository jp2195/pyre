package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jp2195/pyre/internal/models"
)

type SessionSortField int

const (
	SessionSortID SessionSortField = iota
	SessionSortBytes
	SessionSortAge
	SessionSortApplication
)

type SessionsModel struct {
	TableBase
	sessions []models.Session
	filtered []models.Session
	sortBy   SessionSortField

	// Detail view state
	detail        *models.SessionDetail // Cached detail for selected session
	detailLoading bool                  // Loading indicator
	detailID      int64                 // Which session the detail is for
}

func NewSessionsModel() SessionsModel {
	return SessionsModel{
		TableBase: NewTableBase("Filter sessions..."),
	}
}

func (m SessionsModel) SetSize(width, height int) SessionsModel {
	m.TableBase = m.TableBase.SetSize(width, height)

	// Clamp cursor to valid range after resize
	count := len(m.filtered)
	if m.Cursor >= count && count > 0 {
		m.Cursor = count - 1
	}

	// Adjust offset to keep cursor visible
	visibleRows := m.visibleRows()
	if visibleRows > 0 && m.Cursor >= m.Offset+visibleRows {
		m.Offset = m.Cursor - visibleRows + 1
	}

	return m
}

func (m SessionsModel) SetLoading(loading bool) SessionsModel {
	m.TableBase = m.TableBase.SetLoading(loading)
	return m
}

// HasData returns true if sessions have been loaded.
func (m SessionsModel) HasData() bool {
	return m.sessions != nil
}

func (m SessionsModel) SetSessions(sessions []models.Session, err error) SessionsModel {
	m.sessions = sessions
	m.Err = err
	m.Loading = false
	m.Cursor = 0
	m.Offset = 0
	m.detail = nil // Clear detail when sessions refresh
	m.detailID = 0
	m.detailLoading = false
	m.applyFilter()
	return m
}

// SetDetail sets the detailed session information.
func (m SessionsModel) SetDetail(detail *models.SessionDetail, err error) SessionsModel {
	m.detailLoading = false
	if err == nil && detail != nil {
		m.detail = detail
		m.detailID = detail.ID
	}
	return m
}

// SetDetailLoading sets the detail loading state for a specific session ID.
func (m SessionsModel) SetDetailLoading(id int64) SessionsModel {
	m.detailLoading = true
	m.detailID = id
	m.detail = nil
	return m
}

// SelectedSession returns the currently selected session, if any.
func (m SessionsModel) SelectedSession() *models.Session {
	if m.Cursor >= 0 && m.Cursor < len(m.filtered) {
		return &m.filtered[m.Cursor]
	}
	return nil
}

// IsDetailLoading returns true if detail is currently loading.
func (m SessionsModel) IsDetailLoading() bool {
	return m.detailLoading
}

// GetDetailID returns the session ID for which detail was requested.
func (m SessionsModel) GetDetailID() int64 {
	return m.detailID
}

func (m *SessionsModel) applyFilter() {
	if m.FilterValue() == "" {
		m.filtered = make([]models.Session, len(m.sessions))
		copy(m.filtered, m.sessions)
	} else {
		query := strings.ToLower(m.FilterValue())
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
		if m.SortAsc {
			return less
		}
		return !less
	})
}

func (m *SessionsModel) cycleSort() {
	m.sortBy = (m.sortBy + 1) % 4
	// Default to descending for bytes/age, ascending for ID/app
	m.SortAsc = m.sortBy == SessionSortID || m.sortBy == SessionSortApplication
	m.applySort()
}

func (m SessionsModel) sortLabel() string {
	dir := "↓"
	if m.SortAsc {
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

// FetchDetailCmd is returned when the user requests session detail fetch.
type FetchDetailCmd struct {
	SessionID int64
}

func (m SessionsModel) Update(msg tea.Msg) (SessionsModel, tea.Cmd) {
	if m.FilterMode {
		return m.updateFilter(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle session-specific keys first
		switch msg.String() {
		case "esc":
			if m.HandleClearFilter() {
				m.applyFilter()
			}
			return m, nil
		case "s":
			m.cycleSort()
			m.Cursor = 0
			m.Offset = 0
			return m, nil
		case "d":
			// 'd' fetches detail for selected session
			if m.Expanded && m.Cursor < len(m.filtered) && !m.detailLoading {
				session := m.filtered[m.Cursor]
				m.detailLoading = true
				m.detailID = session.ID
				return m, func() tea.Msg {
					return FetchDetailCmd{SessionID: session.ID}
				}
			}
			return m, nil
		}

		// Delegate to TableBase for common navigation
		visible := m.visibleRows()
		oldCursor := m.Cursor
		base, handled, cmd := m.HandleNavigation(msg, len(m.filtered), visible)
		if handled {
			m.TableBase = base
			// Clear cached detail if cursor moved to different session
			if m.Cursor != oldCursor {
				m.detail = nil
				m.detailID = 0
				m.detailLoading = false
			}
			return m, cmd
		}
	}

	return m, nil
}

func (m SessionsModel) updateFilter(msg tea.Msg) (SessionsModel, tea.Cmd) {
	base, exited, cmd := m.HandleFilterMode(msg)
	m.TableBase = base
	if exited {
		m.applyFilter()
	}
	return m, cmd
}

func (m SessionsModel) visibleRows() int {
	rows := m.Height - 8
	if m.Expanded {
		rows -= 8
	}
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m SessionsModel) View() string {
	if m.Width == 0 {
		return RenderLoadingInline(m.SpinnerFrame, "Loading...")
	}

	titleStyle := ViewTitleStyle.MarginBottom(1)
	panelStyle := ViewPanelStyle.Width(m.Width - 4)

	var b strings.Builder
	title := "Active Sessions"
	sortInfo := BannerInfoStyle.Render(fmt.Sprintf(" [%d sessions | Sort: %s | s: change]", len(m.filtered), m.sortLabel()))
	b.WriteString(titleStyle.Render(title) + sortInfo)
	b.WriteString("\n")

	if m.FilterMode {
		b.WriteString(FilterBorderStyle.Render(m.Filter.View()))
		b.WriteString("\n\n")
	} else if m.IsFiltered() {
		filterInfo := FilterInfoStyle.Render(fmt.Sprintf("Filter: %s (press / to edit, esc to clear)", m.FilterValue()))
		b.WriteString(filterInfo)
		b.WriteString("\n\n")
	}

	if m.Err != nil {
		b.WriteString(ErrorMsgStyle.Render("Error: " + m.Err.Error()))
		return panelStyle.Render(b.String())
	}

	if m.Loading || m.sessions == nil {
		b.WriteString(RenderLoadingInline(m.SpinnerFrame, "Loading sessions..."))
		return panelStyle.Render(b.String())
	}

	if len(m.filtered) == 0 {
		b.WriteString(EmptyMsgStyle.Render("No sessions found"))
		return panelStyle.Render(b.String())
	}

	b.WriteString(m.renderTable())

	if m.Expanded && m.Cursor < len(m.filtered) {
		b.WriteString("\n\n")
		b.WriteString(m.renderDetail(m.filtered[m.Cursor]))
	}

	return panelStyle.Render(b.String())
}

func (m SessionsModel) renderTable() string {
	headerStyle := TableHeaderStyle
	selectedStyle := TableRowSelectedStyle

	// State-based colors
	activeStyle := StatusActiveStyle
	discardStyle := StatusWarningStyle
	closedStyle := StatusMutedStyle

	var b strings.Builder

	// Header: ID, Source, Destination, Port, Proto, App, State, Zones, Age, Bytes
	header := fmt.Sprintf("%-7s %-15s %-15s %-5s %-4s %-10s %-7s %-15s %-5s %-8s",
		"ID", "Source", "Destination", "Port", "Pro", "App", "State", "Zones", "Age", "Bytes")
	b.WriteString(headerStyle.Render(header) + "\n")

	visibleRows := m.visibleRows()
	end := m.Offset + visibleRows
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	for i := m.Offset; i < end; i++ {
		s := m.filtered[i]
		isSelected := i == m.Cursor

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
	titleStyle := ViewTitleStyle
	labelStyle := DetailLabelStyle
	valueStyle := DetailValueStyle
	dimStyle := DetailDimStyle
	sectionStyle := DetailSectionStyle

	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Session Details: %d", s.ID)))

	// Show hint for fetching extended details
	if m.detail == nil && !m.detailLoading {
		b.WriteString(dimStyle.Render("  [d: fetch extended details]"))
	} else if m.detailLoading {
		b.WriteString(dimStyle.Render("  (loading...)"))
	}
	b.WriteString("\n\n")

	// Basic info (always available)
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
		b.WriteString(labelStyle.Render("Duration:      ") + valueStyle.Render(formatDuration(s.StartTime)) + "\n")
	}

	// Extended details (if fetched and for this session)
	if m.detail != nil && m.detail.ID == s.ID {
		d := m.detail

		// NAT Section
		if d.NATDestIP != "" || d.NATRule != "" {
			b.WriteString("\n")
			b.WriteString(sectionStyle.Render("NAT Details"))
			b.WriteString("\n")
			if d.NATDestIP != "" {
				b.WriteString(labelStyle.Render("NAT Dest:      ") + valueStyle.Render(fmt.Sprintf("%s:%d", d.NATDestIP, d.NATDestPort)) + "\n")
			}
			if d.NATRule != "" {
				b.WriteString(labelStyle.Render("NAT Rule:      ") + valueStyle.Render(d.NATRule) + "\n")
			}
		}

		// Security Section
		if d.URLCategory != "" || d.DecryptionRule != "" {
			b.WriteString("\n")
			b.WriteString(sectionStyle.Render("Security"))
			b.WriteString("\n")
			if d.URLCategory != "" {
				b.WriteString(labelStyle.Render("URL Category:  ") + valueStyle.Render(d.URLCategory) + "\n")
			}
			if d.URLFilteringRule != "" {
				b.WriteString(labelStyle.Render("URL Rule:      ") + valueStyle.Render(d.URLFilteringRule) + "\n")
			}
			if d.DecryptionRule != "" {
				b.WriteString(labelStyle.Render("Decrypt Rule:  ") + valueStyle.Render(d.DecryptionRule) + "\n")
			}
		}

		// Traffic Stats Section
		b.WriteString("\n")
		b.WriteString(sectionStyle.Render("Traffic Statistics"))
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("Pkts to Client:") + valueStyle.Render(fmt.Sprintf("%d", d.PacketsToClient)) + "\n")
		b.WriteString(labelStyle.Render("Pkts to Server:") + valueStyle.Render(fmt.Sprintf("%d", d.PacketsToServer)) + "\n")
		b.WriteString(labelStyle.Render("Bytes to Client:") + valueStyle.Render(formatBytes(d.BytesToClient)) + "\n")
		b.WriteString(labelStyle.Render("Bytes to Server:") + valueStyle.Render(formatBytes(d.BytesToServer)) + "\n")

		// Session Timing
		if d.Timeout > 0 || d.TimeToLive > 0 {
			b.WriteString("\n")
			b.WriteString(sectionStyle.Render("Timing"))
			b.WriteString("\n")
			if d.Timeout > 0 {
				b.WriteString(labelStyle.Render("Timeout:       ") + valueStyle.Render(fmt.Sprintf("%ds", d.Timeout)) + "\n")
			}
			if d.TimeToLive > 0 {
				b.WriteString(labelStyle.Render("TTL:           ") + valueStyle.Render(fmt.Sprintf("%ds", d.TimeToLive)) + "\n")
			}
			if d.IdleTime > 0 {
				b.WriteString(labelStyle.Render("Idle:          ") + valueStyle.Render(fmt.Sprintf("%ds", d.IdleTime)) + "\n")
			}
		}

		// Flags
		flags := []string{}
		if d.Offloaded {
			flags = append(flags, "offloaded")
		}
		if d.DecryptMirror {
			flags = append(flags, "decrypt-mirror")
		}
		if len(flags) > 0 {
			b.WriteString(labelStyle.Render("Flags:         ") + dimStyle.Render(strings.Join(flags, ", ")) + "\n")
		}
	}

	return b.String()
}
