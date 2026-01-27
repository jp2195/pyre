package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jp2195/pyre/internal/models"
)

type LogSortField int

const (
	LogSortTime LogSortField = iota
	LogSortSeverity
	LogSortSource
	LogSortAction
)

type LogsModel struct {
	TableBase
	activeLogType models.LogType

	systemLogs  []models.SystemLogEntry
	trafficLogs []models.TrafficLogEntry
	threatLogs  []models.ThreatLogEntry

	filteredSystem  []models.SystemLogEntry
	filteredTraffic []models.TrafficLogEntry
	filteredThreat  []models.ThreatLogEntry

	sortBy      LogSortField
	lastRefresh time.Time
}

func NewLogsModel() LogsModel {
	base := NewTableBase("Filter logs...")
	base.SortAsc = false // Default to newest first
	return LogsModel{
		TableBase:     base,
		activeLogType: models.LogTypeSystem,
	}
}

func (m LogsModel) SetSize(width, height int) LogsModel {
	m.TableBase = m.TableBase.SetSize(width, height)

	// Clamp cursor to valid range after resize
	count := m.filteredCount()
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

func (m LogsModel) SetLoading(loading bool) LogsModel {
	m.TableBase = m.TableBase.SetLoading(loading)
	return m
}

// HasData returns true if any logs have been loaded.
func (m LogsModel) HasData() bool {
	return m.systemLogs != nil || m.trafficLogs != nil || m.threatLogs != nil
}

func (m LogsModel) SetSystemLogs(logs []models.SystemLogEntry, err error) LogsModel {
	m.systemLogs = logs
	if err != nil {
		m.Err = err
	}
	m.Loading = false
	m.lastRefresh = time.Now()
	m.applyFilter()
	m.ensureCursorValid()
	return m
}

func (m LogsModel) SetTrafficLogs(logs []models.TrafficLogEntry, err error) LogsModel {
	m.trafficLogs = logs
	if err != nil {
		m.Err = err
	}
	m.Loading = false
	m.lastRefresh = time.Now()
	m.applyFilter()
	m.ensureCursorValid()
	return m
}

func (m LogsModel) SetThreatLogs(logs []models.ThreatLogEntry, err error) LogsModel {
	m.threatLogs = logs
	if err != nil {
		m.Err = err
	}
	m.Loading = false
	m.lastRefresh = time.Now()
	m.applyFilter()
	m.ensureCursorValid()
	return m
}

func (m LogsModel) SetError(err error) LogsModel {
	m.Err = err
	m.Loading = false
	return m
}

func (m LogsModel) ActiveLogType() models.LogType {
	return m.activeLogType
}

func (m LogsModel) IsFilterMode() bool {
	return m.FilterMode
}

func (m *LogsModel) ensureCursorValid() {
	count := m.filteredCount()
	if m.Cursor >= count && count > 0 {
		m.Cursor = count - 1
	}
	if m.Cursor < 0 {
		m.Cursor = 0
	}
}

func (m LogsModel) filteredCount() int {
	switch m.activeLogType {
	case models.LogTypeSystem:
		return len(m.filteredSystem)
	case models.LogTypeTraffic:
		return len(m.filteredTraffic)
	case models.LogTypeThreat:
		return len(m.filteredThreat)
	}
	return 0
}

func (m *LogsModel) applyFilter() {
	query := strings.ToLower(m.FilterValue())

	// Filter system logs
	if query == "" {
		m.filteredSystem = make([]models.SystemLogEntry, len(m.systemLogs))
		copy(m.filteredSystem, m.systemLogs)
	} else {
		m.filteredSystem = nil
		for _, log := range m.systemLogs {
			if strings.Contains(strings.ToLower(log.Description), query) ||
				strings.Contains(strings.ToLower(log.Type), query) ||
				strings.Contains(strings.ToLower(log.Severity), query) {
				m.filteredSystem = append(m.filteredSystem, log)
			}
		}
	}

	// Filter traffic logs
	if query == "" {
		m.filteredTraffic = make([]models.TrafficLogEntry, len(m.trafficLogs))
		copy(m.filteredTraffic, m.trafficLogs)
	} else {
		m.filteredTraffic = nil
		for _, log := range m.trafficLogs {
			if strings.Contains(strings.ToLower(log.SourceIP), query) ||
				strings.Contains(strings.ToLower(log.DestIP), query) ||
				strings.Contains(strings.ToLower(log.Application), query) ||
				strings.Contains(strings.ToLower(log.Rule), query) ||
				strings.Contains(strings.ToLower(log.Action), query) ||
				strings.Contains(strings.ToLower(log.User), query) {
				m.filteredTraffic = append(m.filteredTraffic, log)
			}
		}
	}

	// Filter threat logs
	if query == "" {
		m.filteredThreat = make([]models.ThreatLogEntry, len(m.threatLogs))
		copy(m.filteredThreat, m.threatLogs)
	} else {
		m.filteredThreat = nil
		for _, log := range m.threatLogs {
			if strings.Contains(strings.ToLower(log.SourceIP), query) ||
				strings.Contains(strings.ToLower(log.DestIP), query) ||
				strings.Contains(strings.ToLower(log.ThreatName), query) ||
				strings.Contains(strings.ToLower(log.Severity), query) ||
				strings.Contains(strings.ToLower(log.Action), query) ||
				strings.Contains(strings.ToLower(log.ThreatCategory), query) {
				m.filteredThreat = append(m.filteredThreat, log)
			}
		}
	}

	m.applySort()
}

func (m *LogsModel) applySort() {
	// Sort system logs
	sort.Slice(m.filteredSystem, func(i, j int) bool {
		var less bool
		switch m.sortBy {
		case LogSortSeverity:
			less = severityRank(m.filteredSystem[i].Severity) < severityRank(m.filteredSystem[j].Severity)
		default: // LogSortTime
			less = m.filteredSystem[i].Time.Before(m.filteredSystem[j].Time)
		}
		if !m.SortAsc {
			return !less
		}
		return less
	})

	// Sort traffic logs
	sort.Slice(m.filteredTraffic, func(i, j int) bool {
		var less bool
		switch m.sortBy {
		case LogSortSource:
			less = m.filteredTraffic[i].SourceIP < m.filteredTraffic[j].SourceIP
		case LogSortAction:
			less = m.filteredTraffic[i].Action < m.filteredTraffic[j].Action
		default: // LogSortTime
			less = m.filteredTraffic[i].Time.Before(m.filteredTraffic[j].Time)
		}
		if !m.SortAsc {
			return !less
		}
		return less
	})

	// Sort threat logs
	sort.Slice(m.filteredThreat, func(i, j int) bool {
		var less bool
		switch m.sortBy {
		case LogSortSeverity:
			less = severityRank(m.filteredThreat[i].Severity) < severityRank(m.filteredThreat[j].Severity)
		case LogSortSource:
			less = m.filteredThreat[i].SourceIP < m.filteredThreat[j].SourceIP
		case LogSortAction:
			less = m.filteredThreat[i].Action < m.filteredThreat[j].Action
		default: // LogSortTime
			less = m.filteredThreat[i].Time.Before(m.filteredThreat[j].Time)
		}
		if !m.SortAsc {
			return !less
		}
		return less
	})
}

func severityRank(severity string) int {
	switch strings.ToLower(severity) {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	case "informational":
		return 1
	default:
		return 0
	}
}

func (m *LogsModel) cycleSort() {
	m.sortBy = (m.sortBy + 1) % 4
	m.applySort()
}

func (m LogsModel) sortLabel() string {
	dir := "desc"
	if m.SortAsc {
		dir = "asc"
	}
	switch m.sortBy {
	case LogSortSeverity:
		return fmt.Sprintf("severity %s", dir)
	case LogSortSource:
		return fmt.Sprintf("source %s", dir)
	case LogSortAction:
		return fmt.Sprintf("action %s", dir)
	default:
		return fmt.Sprintf("time %s", dir)
	}
}

func (m LogsModel) Update(msg tea.Msg) (LogsModel, tea.Cmd) {
	if m.FilterMode {
		return m.updateFilterMode(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle logs-specific keys first
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
		case "S":
			m.SortAsc = !m.SortAsc
			m.applySort()
			return m, nil
		case "]":
			// Cycle forward through log types: System -> Traffic -> Threat -> System
			switch m.activeLogType {
			case models.LogTypeSystem:
				m.activeLogType = models.LogTypeTraffic
			case models.LogTypeTraffic:
				m.activeLogType = models.LogTypeThreat
			case models.LogTypeThreat:
				m.activeLogType = models.LogTypeSystem
			}
			m.Cursor = 0
			m.Offset = 0
			m.Expanded = false
			return m, nil
		case "[":
			// Cycle backward through log types: System -> Threat -> Traffic -> System
			switch m.activeLogType {
			case models.LogTypeSystem:
				m.activeLogType = models.LogTypeThreat
			case models.LogTypeTraffic:
				m.activeLogType = models.LogTypeSystem
			case models.LogTypeThreat:
				m.activeLogType = models.LogTypeTraffic
			}
			m.Cursor = 0
			m.Offset = 0
			m.Expanded = false
			return m, nil
		}

		// Delegate to TableBase for common navigation
		visible := m.visibleRows()
		base, handled, cmd := m.HandleNavigation(msg, m.filteredCount(), visible)
		if handled {
			m.TableBase = base
			return m, cmd
		}
	}

	return m, nil
}

func (m LogsModel) updateFilterMode(msg tea.Msg) (LogsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc":
			m.FilterMode = false
			m.Filter.Blur()
			if msg.String() == "enter" {
				m.Cursor = 0
				m.Offset = 0
				m.applyFilter()
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.Filter, cmd = m.Filter.Update(msg)
	return m, cmd
}

func (m LogsModel) visibleRows() int {
	rows := m.Height - 10 // Account for header, tabs, help
	if m.Expanded {
		rows -= 12 // Reserve space for detail panel
	}
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m LogsModel) View() string {
	if m.Width == 0 {
		return RenderLoadingInline(m.SpinnerFrame, "Loading...")
	}

	var sections []string

	// Loading banner
	if m.Loading {
		sections = append(sections, m.renderLoadingBanner())
	}

	// Tab bar for log types
	sections = append(sections, m.renderTabBar())

	// Filter bar
	if m.FilterMode {
		sections = append(sections, m.renderFilterBar())
	} else if m.IsFiltered() {
		filterInfo := FilterInfoStyle.Render(fmt.Sprintf("Filter: %s (%d results)  [esc to clear]", m.FilterValue(), m.filteredCount()))
		sections = append(sections, filterInfo)
	}

	// Error or content
	if m.Err != nil {
		sections = append(sections, m.renderError())
	} else if !m.Loading || m.filteredCount() > 0 {
		sections = append(sections, m.renderTable())

		if m.Expanded && m.filteredCount() > 0 {
			sections = append(sections, m.renderDetailPanel())
		}
	}

	// Help
	sections = append(sections, m.renderHelp())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m LogsModel) renderLoadingBanner() string {
	return RenderLoadingBanner(m.SpinnerFrame, "Loading logs...", m.Width)
}

func (m LogsModel) renderTabBar() string {
	var tabs []string

	if m.activeLogType == models.LogTypeSystem {
		tabs = append(tabs, TabActiveStyle.Render(fmt.Sprintf("System (%d)", len(m.filteredSystem))))
	} else {
		tabs = append(tabs, TabInactiveStyle.Render(fmt.Sprintf("System (%d)", len(m.filteredSystem))))
	}

	if m.activeLogType == models.LogTypeTraffic {
		tabs = append(tabs, TabActiveStyle.Render(fmt.Sprintf("Traffic (%d)", len(m.filteredTraffic))))
	} else {
		tabs = append(tabs, TabInactiveStyle.Render(fmt.Sprintf("Traffic (%d)", len(m.filteredTraffic))))
	}

	if m.activeLogType == models.LogTypeThreat {
		tabs = append(tabs, TabActiveStyle.Render(fmt.Sprintf("Threat (%d)", len(m.filteredThreat))))
	} else {
		tabs = append(tabs, TabInactiveStyle.Render(fmt.Sprintf("Threat (%d)", len(m.filteredThreat))))
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Center, tabs...)

	// Right side - sort info and last update
	sortInfo := StatusMutedStyle.Render(fmt.Sprintf("Sort: %s", m.sortLabel()))

	var updateInfo string
	if !m.lastRefresh.IsZero() {
		ago := time.Since(m.lastRefresh).Truncate(time.Second)
		updateInfo = StatusMutedStyle.Render(fmt.Sprintf("  |  Updated %s ago", ago))
	}

	rightSide := sortInfo + updateInfo
	padding := m.Width - lipgloss.Width(tabBar) - lipgloss.Width(rightSide) - 2
	if padding < 1 {
		padding = 1
	}

	return tabBar + strings.Repeat(" ", padding) + rightSide + "\n"
}

func (m LogsModel) renderFilterBar() string {
	return FilterBorderStyle.Render(m.Filter.View()) + "\n"
}

func (m LogsModel) renderError() string {
	return ErrorMsgStyle.Bold(true).Padding(1, 0).Render(fmt.Sprintf("Error: %v", m.Err))
}

func (m LogsModel) renderTable() string {
	switch m.activeLogType {
	case models.LogTypeSystem:
		return m.renderSystemTable()
	case models.LogTypeTraffic:
		return m.renderTrafficTable()
	case models.LogTypeThreat:
		return m.renderThreatTable()
	}
	return ""
}

func (m LogsModel) renderSystemTable() string {
	if m.Loading && len(m.systemLogs) == 0 {
		return LoadingMsgStyle.Padding(1, 0).Render("Loading system logs...")
	}
	if len(m.filteredSystem) == 0 {
		return EmptyMsgStyle.Padding(1, 0).Render("No system logs found")
	}

	var b strings.Builder

	// Header - compact severity, more space for description
	header := fmt.Sprintf("%-19s %-4s %-18s %s",
		"Time", "Sev", "Type", "Description")
	b.WriteString(TableHeaderStyle.Render(header) + "\n")

	visibleRows := m.visibleRows()
	end := m.Offset + visibleRows
	if end > len(m.filteredSystem) {
		end = len(m.filteredSystem)
	}

	for i := m.Offset; i < end; i++ {
		log := m.filteredSystem[i]
		isSelected := i == m.Cursor

		timeStr := log.Time.Format("2006-01-02 15:04:05")
		sevAbbrev := abbreviateSeverity(log.Severity)
		desc := truncate(log.Description, m.Width-46)

		if isSelected {
			row := fmt.Sprintf("%-19s %-4s %-18s %s",
				timeStr,
				sevAbbrev,
				truncate(log.Type, 18),
				desc)
			b.WriteString(TableRowSelectedStyle.Render(row) + "\n")
		} else {
			// Build row with colored severity indicator
			sevStyle := severityStyle(log.Severity)

			row := DetailLabelStyle.Render(fmt.Sprintf("%-19s", timeStr)) + " " +
				sevStyle.Render(fmt.Sprintf("%-4s", sevAbbrev)) + " " +
				StatusMutedStyle.Render(fmt.Sprintf("%-18s", truncate(log.Type, 18))) + " " +
				DetailValueStyle.Render(desc)
			b.WriteString(row + "\n")
		}
	}

	return b.String()
}

// abbreviateSeverity returns a short severity label
func abbreviateSeverity(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return "CRIT"
	case "high":
		return "HIGH"
	case "medium":
		return "MED"
	case "low":
		return "LOW"
	case "informational":
		return "INFO"
	default:
		return truncate(severity, 4)
	}
}

// severityStyle returns the lipgloss style for a severity level
func severityStyle(severity string) lipgloss.Style {
	switch strings.ToLower(severity) {
	case "critical":
		return SeverityCriticalStyle
	case "high":
		return SeverityHighStyle
	case "medium":
		return SeverityMediumStyle
	case "low":
		return SeverityLowStyle
	default: // informational
		return StatusMutedStyle
	}
}

func (m LogsModel) renderTrafficTable() string {
	if m.Loading && len(m.trafficLogs) == 0 {
		return LoadingMsgStyle.Padding(1, 0).Render("Loading traffic logs...")
	}
	if len(m.filteredTraffic) == 0 {
		return EmptyMsgStyle.Padding(1, 0).Render("No traffic logs found")
	}

	var b strings.Builder

	// Header
	header := fmt.Sprintf("%-19s %-7s %-15s %-15s %-12s %-15s %-10s",
		"Time", "Action", "Source", "Dest", "App", "Rule", "Bytes")
	b.WriteString(TableHeaderStyle.Render(header) + "\n")

	visibleRows := m.visibleRows()
	end := m.Offset + visibleRows
	if end > len(m.filteredTraffic) {
		end = len(m.filteredTraffic)
	}

	for i := m.Offset; i < end; i++ {
		log := m.filteredTraffic[i]
		isSelected := i == m.Cursor

		timeStr := log.Time.Format("2006-01-02 15:04:05")

		row := fmt.Sprintf("%-19s %-7s %-15s %-15s %-12s %-15s %-10s",
			timeStr,
			truncate(log.Action, 7),
			truncate(log.SourceIP, 15),
			truncate(log.DestIP, 15),
			truncate(log.Application, 12),
			truncate(log.Rule, 15),
			formatBytes(log.Bytes))

		if isSelected {
			b.WriteString(TableRowSelectedStyle.Render(row) + "\n")
		} else {
			// Color code by action
			row = m.colorByAction(row, log.Action)
			b.WriteString(row + "\n")
		}
	}

	return b.String()
}

func (m LogsModel) renderThreatTable() string {
	if m.Loading && len(m.threatLogs) == 0 {
		return LoadingMsgStyle.Padding(1, 0).Render("Loading threat logs...")
	}
	if len(m.filteredThreat) == 0 {
		return EmptyMsgStyle.Padding(1, 0).Render("No threat logs found")
	}

	var b strings.Builder

	// Header
	header := fmt.Sprintf("%-19s %-9s %-20s %-15s %-7s %-15s",
		"Time", "Severity", "Threat", "Source", "Action", "Category")
	b.WriteString(TableHeaderStyle.Render(header) + "\n")

	visibleRows := m.visibleRows()
	end := m.Offset + visibleRows
	if end > len(m.filteredThreat) {
		end = len(m.filteredThreat)
	}

	for i := m.Offset; i < end; i++ {
		log := m.filteredThreat[i]
		isSelected := i == m.Cursor

		timeStr := log.Time.Format("2006-01-02 15:04:05")

		row := fmt.Sprintf("%-19s %-9s %-20s %-15s %-7s %-15s",
			timeStr,
			truncate(log.Severity, 9),
			truncate(log.ThreatName, 20),
			truncate(log.SourceIP, 15),
			truncate(log.Action, 7),
			truncate(log.ThreatCategory, 15))

		if isSelected {
			b.WriteString(TableRowSelectedStyle.Render(row) + "\n")
		} else {
			// Color code by severity
			row = m.colorBySeverity(row, log.Severity)
			b.WriteString(row + "\n")
		}
	}

	return b.String()
}

func (m LogsModel) colorBySeverity(row, severity string) string {
	return SeverityStyle(severity).Render(row)
}

func (m LogsModel) colorByAction(row, action string) string {
	return ActionStyle(action).Render(row)
}

func (m LogsModel) renderDetailPanel() string {
	switch m.activeLogType {
	case models.LogTypeSystem:
		if m.Cursor < len(m.filteredSystem) {
			return m.renderSystemDetail(m.filteredSystem[m.Cursor])
		}
	case models.LogTypeTraffic:
		if m.Cursor < len(m.filteredTraffic) {
			return m.renderTrafficDetail(m.filteredTraffic[m.Cursor])
		}
	case models.LogTypeThreat:
		if m.Cursor < len(m.filteredThreat) {
			return m.renderThreatDetail(m.filteredThreat[m.Cursor])
		}
	}
	return ""
}

func (m LogsModel) renderSystemDetail(log models.SystemLogEntry) string {
	panelStyle := DetailPanelStyle.Width(m.Width - 2)
	labelStyle := DetailLabelStyle.Width(12)

	// Word wrap the description for better readability
	descWidth := m.Width - 10
	if descWidth > 100 {
		descWidth = 100
	}
	wrapped := wrapText(log.Description, descWidth)

	lines := make([]string, 0, 7+len(wrapped))
	lines = append(lines, ViewTitleStyle.Render("System Log Details"))
	lines = append(lines, "")
	lines = append(lines, labelStyle.Render("Time")+DetailValueStyle.Render(log.Time.Format("2006-01-02 15:04:05")))
	lines = append(lines, labelStyle.Render("Severity")+severityStyle(log.Severity).Render(log.Severity))
	lines = append(lines, labelStyle.Render("Type")+DetailValueStyle.Render(log.Type))
	lines = append(lines, "")
	lines = append(lines, ViewTitleStyle.Render("Message"))

	for _, line := range wrapped {
		lines = append(lines, DetailValueStyle.Render(line))
	}

	return panelStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

// wrapText wraps text to the specified width
func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}

	currentLine := words[0]
	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	lines = append(lines, currentLine)
	return lines
}

func (m LogsModel) renderTrafficDetail(log models.TrafficLogEntry) string {
	panelStyle := DetailPanelStyle.Width(m.Width - 2)
	labelStyle := DetailLabelStyle.Width(14)

	var lines []string
	lines = append(lines, ViewTitleStyle.Render("Traffic Log Details"))

	lines = append(lines, DetailSectionStyle.Render("Session"))
	lines = append(lines, labelStyle.Render("Time")+DetailValueStyle.Render(log.Time.Format("2006-01-02 15:04:05")))
	lines = append(lines, labelStyle.Render("Action")+m.colorByAction(log.Action, log.Action))
	lines = append(lines, labelStyle.Render("Session ID")+DetailValueStyle.Render(fmt.Sprintf("%d", log.SessionID)))
	lines = append(lines, labelStyle.Render("Duration")+DetailValueStyle.Render(fmt.Sprintf("%ds", log.Duration)))

	lines = append(lines, DetailSectionStyle.Render("Source / Destination"))
	lines = append(lines, labelStyle.Render("Source")+DetailValueStyle.Render(fmt.Sprintf("%s:%d (%s)", log.SourceIP, log.SourcePort, log.SourceZone)))
	lines = append(lines, labelStyle.Render("Destination")+DetailValueStyle.Render(fmt.Sprintf("%s:%d (%s)", log.DestIP, log.DestPort, log.DestZone)))
	if log.NATSourceIP != "" {
		lines = append(lines, labelStyle.Render("NAT Source")+DetailValueStyle.Render(fmt.Sprintf("%s:%d", log.NATSourceIP, log.NATSourcePort)))
	}
	if log.NATDestIP != "" {
		lines = append(lines, labelStyle.Render("NAT Dest")+DetailValueStyle.Render(fmt.Sprintf("%s:%d", log.NATDestIP, log.NATDestPort)))
	}

	lines = append(lines, DetailSectionStyle.Render("Application"))
	lines = append(lines, labelStyle.Render("Application")+DetailValueStyle.Render(log.Application))
	lines = append(lines, labelStyle.Render("Protocol")+DetailValueStyle.Render(log.Protocol))
	lines = append(lines, labelStyle.Render("Rule")+DetailValueStyle.Render(log.Rule))
	if log.User != "" {
		lines = append(lines, labelStyle.Render("User")+DetailValueStyle.Render(log.User))
	}

	lines = append(lines, DetailSectionStyle.Render("Traffic"))
	lines = append(lines, labelStyle.Render("Bytes")+DetailValueStyle.Render(fmt.Sprintf("%s (sent: %s, recv: %s)", formatBytes(log.Bytes), formatBytes(log.BytesSent), formatBytes(log.BytesRecv))))
	lines = append(lines, labelStyle.Render("Packets")+DetailValueStyle.Render(fmt.Sprintf("%d (sent: %d, recv: %d)", log.Packets, log.PacketsSent, log.PacketsRecv)))

	return panelStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m LogsModel) renderThreatDetail(log models.ThreatLogEntry) string {
	panelStyle := DetailPanelStyle.Width(m.Width - 2)
	labelStyle := DetailLabelStyle.Width(14)

	var lines []string
	lines = append(lines, ViewTitleStyle.Render("Threat Log Details"))

	lines = append(lines, DetailSectionStyle.Render("Threat"))
	lines = append(lines, labelStyle.Render("Time")+DetailValueStyle.Render(log.Time.Format("2006-01-02 15:04:05")))
	lines = append(lines, labelStyle.Render("Severity")+m.colorBySeverity(log.Severity, log.Severity))
	lines = append(lines, labelStyle.Render("Threat Name")+DetailValueStyle.Render(log.ThreatName))
	lines = append(lines, labelStyle.Render("Threat ID")+DetailValueStyle.Render(fmt.Sprintf("%d", log.ThreatID)))
	lines = append(lines, labelStyle.Render("Category")+DetailValueStyle.Render(log.ThreatCategory))
	lines = append(lines, labelStyle.Render("Subtype")+DetailValueStyle.Render(log.Subtype))
	lines = append(lines, labelStyle.Render("Action")+m.colorByAction(log.Action, log.Action))
	lines = append(lines, labelStyle.Render("Direction")+DetailValueStyle.Render(log.Direction))

	lines = append(lines, DetailSectionStyle.Render("Source / Destination"))
	lines = append(lines, labelStyle.Render("Source")+DetailValueStyle.Render(fmt.Sprintf("%s:%d (%s)", log.SourceIP, log.SourcePort, log.SourceZone)))
	lines = append(lines, labelStyle.Render("Destination")+DetailValueStyle.Render(fmt.Sprintf("%s:%d (%s)", log.DestIP, log.DestPort, log.DestZone)))

	lines = append(lines, DetailSectionStyle.Render("Context"))
	lines = append(lines, labelStyle.Render("Application")+DetailValueStyle.Render(log.Application))
	lines = append(lines, labelStyle.Render("Rule")+DetailValueStyle.Render(log.Rule))
	if log.User != "" {
		lines = append(lines, labelStyle.Render("User")+DetailValueStyle.Render(log.User))
	}
	if log.URL != "" {
		lines = append(lines, labelStyle.Render("URL")+DetailValueStyle.Render(truncate(log.URL, m.Width-20)))
	}
	if log.Filename != "" {
		lines = append(lines, labelStyle.Render("Filename")+DetailValueStyle.Render(log.Filename))
	}

	return panelStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m LogsModel) renderHelp() string {
	expandText := "details"
	if m.Expanded {
		expandText = "collapse"
	}

	keys := []struct{ key, desc string }{
		{"[/]", "log type"},
		{"j/k", "scroll"},
		{"enter", expandText},
		{"/", "filter"},
		{"s", "sort field"},
		{"S", "sort dir"},
		{"r", "refresh"},
	}

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, HelpKeyStyle.Render(k.key)+HelpDescStyle.Render(":"+k.desc))
	}

	return ViewSubtitleStyle.MarginTop(1).Render(strings.Join(parts, "  "))
}
