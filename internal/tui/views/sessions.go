package views

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
)

// FetchDetailCmd is returned when the user requests session detail fetch.
type FetchDetailCmd struct {
	SessionID int64
}

type SessionsModel struct {
	list RuleListModel[models.Session]

	// Detail view state
	detail        *models.SessionDetail // Cached detail for selected session
	detailLoading bool                  // Loading indicator
	detailID      int64                 // Which session the detail is for
}

func NewSessionsModel() SessionsModel {
	config := RuleListConfig[models.Session]{
		Title:             "Active Sessions",
		ItemNoun:          "sessions",
		LoadingMsg:        "Loading sessions...",
		EmptyMsg:          "No sessions found",
		FilterPlaceholder: "Filter sessions...",
		SortLabels:        []string{"ID", "Bytes", "Age", "App"},
		DefaultSortAsc:    func(idx int) bool { return idx == 0 || idx == 3 },
		MatchFilter:       matchSession,
		CompareItems:      compareSession,
		FormatHeaderRow:   formatSessionHeader,
		FormatRow:         formatSessionRow,
		StyleRow:          styleSessionRow,
		// RenderDetail is bound per-render in View so it can see the
		// current cached detail / loading state.
	}
	return SessionsModel{list: NewRuleListModel(config)}
}

func (m SessionsModel) SetSize(width, height int) SessionsModel {
	m.list = m.list.SetSize(width, height)
	return m
}

func (m SessionsModel) SetLoading(loading bool) SessionsModel {
	m.list = m.list.SetLoading(loading)
	return m
}

// SetSpinnerFrame updates the current spinner animation frame.
func (m SessionsModel) SetSpinnerFrame(frame string) SessionsModel {
	m.list.SpinnerFrame = frame
	return m
}

// HasData returns true if sessions have been loaded.
func (m SessionsModel) HasData() bool {
	return m.list.HasData()
}

// IsFilterMode returns true while the filter text input is focused.
func (m SessionsModel) IsFilterMode() bool {
	return m.list.IsFilterMode()
}

func (m SessionsModel) SetSessions(sessions []models.Session, err error) SessionsModel {
	m.list = m.list.SetItems(sessions, err)
	m.detail = nil // Clear detail when sessions refresh
	m.detailID = 0
	m.detailLoading = false
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

// SetExpanded sets the detail-panel expansion state. Used by tests that need
// to pre-expand the detail panel before dispatching keys.
func (m SessionsModel) SetExpanded(expanded bool) SessionsModel {
	m.list.Expanded = expanded
	return m
}

// IsDetailLoading returns true if detail is currently loading.
func (m SessionsModel) IsDetailLoading() bool {
	return m.detailLoading
}

// GetDetailID returns the session ID for which detail was requested.
func (m SessionsModel) GetDetailID() int64 {
	return m.detailID
}

func (m SessionsModel) Update(msg tea.Msg) (SessionsModel, tea.Cmd) {
	// 'd' fetches extended detail for the selected session; handled here
	// (not in RuleListModel) because the loading guard and cache live on
	// this wrapper.
	if key, ok := msg.(tea.KeyPressMsg); ok && !m.list.FilterMode && key.String() == "d" {
		filtered := m.list.Filtered()
		if m.list.Expanded && m.list.Cursor < len(filtered) && !m.detailLoading {
			session := filtered[m.list.Cursor]
			m.detailLoading = true
			m.detailID = session.ID
			return m, func() tea.Msg {
				return FetchDetailCmd{SessionID: session.ID}
			}
		}
		return m, nil
	}

	oldCursor := m.list.Cursor
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	// Clear cached detail if cursor moved to a different session.
	if m.list.Cursor != oldCursor {
		m.detail = nil
		m.detailID = 0
		m.detailLoading = false
	}
	return m, cmd
}

func (m SessionsModel) View() string {
	detail, loading := m.detail, m.detailLoading
	m.list.config.RenderDetail = func(s models.Session, width int) string {
		return renderSessionDetail(s, detail, loading)
	}
	return m.list.View()
}

// --- Type-specific functions ---

func matchSession(s models.Session, query string) bool {
	return strings.Contains(strings.ToLower(s.Application), query) ||
		strings.Contains(s.SourceIP, query) ||
		strings.Contains(s.DestIP, query) ||
		strings.Contains(strings.ToLower(s.SourceZone), query) ||
		strings.Contains(strings.ToLower(s.DestZone), query) ||
		strings.Contains(strings.ToLower(s.Rule), query) ||
		strings.Contains(strings.ToLower(s.User), query)
}

func compareSession(a, b models.Session, sortIdx int) bool {
	switch sortIdx {
	case 1: // Bytes
		return a.BytesIn+a.BytesOut < b.BytesIn+b.BytesOut
	case 2: // Age
		return a.StartTime.Before(b.StartTime)
	case 3: // App
		return a.Application < b.Application
	default: // ID
		return a.ID < b.ID
	}
}

func formatSessionHeader(width int) string {
	return fmt.Sprintf("%-7s %-15s %-15s %-5s %-4s %-10s %-7s %-15s %-5s %-8s",
		"ID", "Source", "Destination", "Port", "Pro", "App", "State", "Zones", "Age", "Bytes")
}

// sessionRowParts splits a row into prefix, state cell, and suffix so the
// state cell can be color-coded independently for non-selected rows.
func sessionRowParts(s models.Session) (prefix, state, suffix string) {
	zoneFlow := fmt.Sprintf("%s→%s", truncate(s.SourceZone, 7), truncate(s.DestZone, 7))
	proto := s.Protocol
	if proto == "" {
		proto = "tcp"
	}
	prefix = fmt.Sprintf("%-7d %-15s %-15s %-5d %-4s %-10s ",
		s.ID,
		truncate(s.SourceIP, 15),
		truncate(s.DestIP, 15),
		s.DestPort,
		truncate(proto, 4),
		truncate(s.Application, 10))
	state = fmt.Sprintf("%-7s", truncate(s.State, 7))
	suffix = fmt.Sprintf(" %-15s %-5s %-8s",
		truncate(zoneFlow, 15),
		formatDuration(s.StartTime),
		formatBytes(s.BytesIn+s.BytesOut))
	return prefix, state, suffix
}

func formatSessionRow(s models.Session, width int) string {
	prefix, state, suffix := sessionRowParts(s)
	return prefix + state + suffix
}

// styleSessionRow renders a non-selected row with the state cell color-coded.
func styleSessionRow(s models.Session, width int) string {
	prefix, state, suffix := sessionRowParts(s)
	switch strings.ToUpper(s.State) {
	case "ACTIVE":
		return prefix + StatusActiveStyle.Render(state) + suffix
	case "DISCARD", "DROP":
		return prefix + StatusWarningStyle.Render(state) + suffix
	case "CLOSED", "INIT":
		return prefix + StatusMutedStyle.Render(state) + suffix
	default:
		return prefix + state + suffix
	}
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

func renderSessionDetail(s models.Session, detail *models.SessionDetail, detailLoading bool) string {
	titleStyle := ViewTitleStyle
	labelStyle := DetailLabelStyle
	valueStyle := DetailValueStyle
	dimStyle := DetailDimStyle
	sectionStyle := DetailSectionStyle

	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Session Details: %d", s.ID)))

	// Show hint for fetching extended details
	if detail == nil && !detailLoading {
		b.WriteString(dimStyle.Render("  [d: fetch extended details]"))
	} else if detailLoading {
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
	if detail != nil && detail.ID == s.ID {
		d := detail

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
		b.WriteString(labelStyle.Render("Pkts to Client:") + valueStyle.Render(strconv.FormatInt(d.PacketsToClient, 10)) + "\n")
		b.WriteString(labelStyle.Render("Pkts to Server:") + valueStyle.Render(strconv.FormatInt(d.PacketsToServer, 10)) + "\n")
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
