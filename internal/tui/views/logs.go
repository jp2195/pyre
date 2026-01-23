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

type LogSortField int

const (
	LogSortTime LogSortField = iota
	LogSortSeverity
	LogSortSource
	LogSortAction
)

type LogsModel struct {
	activeLogType models.LogType

	systemLogs  []models.SystemLogEntry
	trafficLogs []models.TrafficLogEntry
	threatLogs  []models.ThreatLogEntry

	filteredSystem  []models.SystemLogEntry
	filteredTraffic []models.TrafficLogEntry
	filteredThreat  []models.ThreatLogEntry

	cursor     int
	offset     int
	filter     textinput.Model
	filterMode bool
	width      int
	height     int
	loading    bool
	err        error
	sortBy     LogSortField
	sortAsc    bool
	expanded   bool

	lastRefresh time.Time
}

func NewLogsModel() LogsModel {
	f := textinput.New()
	f.Placeholder = "Filter logs..."
	f.CharLimit = 100
	f.Width = 40

	return LogsModel{
		activeLogType: models.LogTypeSystem,
		filter:        f,
		sortAsc:       false, // Default to newest first
	}
}

func (m LogsModel) SetSize(width, height int) LogsModel {
	m.width = width
	m.height = height
	return m
}

func (m LogsModel) SetLoading(loading bool) LogsModel {
	m.loading = loading
	return m
}

func (m LogsModel) SetSystemLogs(logs []models.SystemLogEntry, err error) LogsModel {
	m.systemLogs = logs
	if err != nil {
		m.err = err
	}
	m.loading = false
	m.lastRefresh = time.Now()
	m.applyFilter()
	m.ensureCursorValid()
	return m
}

func (m LogsModel) SetTrafficLogs(logs []models.TrafficLogEntry, err error) LogsModel {
	m.trafficLogs = logs
	if err != nil {
		m.err = err
	}
	m.loading = false
	m.lastRefresh = time.Now()
	m.applyFilter()
	m.ensureCursorValid()
	return m
}

func (m LogsModel) SetThreatLogs(logs []models.ThreatLogEntry, err error) LogsModel {
	m.threatLogs = logs
	if err != nil {
		m.err = err
	}
	m.loading = false
	m.lastRefresh = time.Now()
	m.applyFilter()
	m.ensureCursorValid()
	return m
}

func (m LogsModel) SetError(err error) LogsModel {
	m.err = err
	m.loading = false
	return m
}

func (m LogsModel) ActiveLogType() models.LogType {
	return m.activeLogType
}

func (m LogsModel) IsFilterMode() bool {
	return m.filterMode
}

func (m *LogsModel) ensureCursorValid() {
	count := m.filteredCount()
	if m.cursor >= count && count > 0 {
		m.cursor = count - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
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
	query := strings.ToLower(m.filter.Value())

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
		if !m.sortAsc {
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
		if !m.sortAsc {
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
		if !m.sortAsc {
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
	if m.sortAsc {
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
	if m.filterMode {
		return m.updateFilterMode(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < m.filteredCount()-1 {
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
			m.cursor = m.filteredCount() - 1
			m.ensureVisible()
		case "ctrl+d", "pgdown":
			m.cursor += 10
			if m.cursor >= m.filteredCount() {
				m.cursor = m.filteredCount() - 1
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
			return m, textinput.Blink
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
		case "S":
			m.sortAsc = !m.sortAsc
			m.applySort()
		case "tab":
			// Cycle through log types
			switch m.activeLogType {
			case models.LogTypeSystem:
				m.activeLogType = models.LogTypeTraffic
			case models.LogTypeTraffic:
				m.activeLogType = models.LogTypeThreat
			case models.LogTypeThreat:
				m.activeLogType = models.LogTypeSystem
			}
			m.cursor = 0
			m.offset = 0
			m.expanded = false
		}
	}

	return m, nil
}

func (m LogsModel) updateFilterMode(msg tea.Msg) (LogsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc":
			m.filterMode = false
			m.filter.Blur()
			if msg.String() == "enter" {
				m.cursor = 0
				m.offset = 0
				m.applyFilter()
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	return m, cmd
}

func (m *LogsModel) ensureVisible() {
	visibleRows := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	} else if m.cursor >= m.offset+visibleRows {
		m.offset = m.cursor - visibleRows + 1
	}
}

func (m LogsModel) visibleRows() int {
	rows := m.height - 10 // Account for header, tabs, help
	if m.expanded {
		rows -= 12 // Reserve space for detail panel
	}
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m LogsModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var sections []string

	// Loading banner
	if m.loading {
		sections = append(sections, m.renderLoadingBanner())
	}

	// Tab bar for log types
	sections = append(sections, m.renderTabBar())

	// Filter bar
	if m.filterMode {
		sections = append(sections, m.renderFilterBar())
	} else if m.filter.Value() != "" {
		filterInfo := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Render(fmt.Sprintf("Filter: %s (%d results)  [esc to clear]", m.filter.Value(), m.filteredCount()))
		sections = append(sections, filterInfo)
	}

	// Error or content
	if m.err != nil {
		sections = append(sections, m.renderError())
	} else if !m.loading || m.filteredCount() > 0 {
		sections = append(sections, m.renderTable())

		if m.expanded && m.filteredCount() > 0 {
			sections = append(sections, m.renderDetailPanel())
		}
	}

	// Help
	sections = append(sections, m.renderHelp())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m LogsModel) renderLoadingBanner() string {
	banner := lipgloss.NewStyle().
		Background(lipgloss.Color("#F59E0B")).
		Foreground(lipgloss.Color("#000000")).
		Bold(true).
		Padding(0, 2).
		Render(" Loading logs... ")

	padding := (m.width - lipgloss.Width(banner)) / 2
	if padding < 0 {
		padding = 0
	}
	return strings.Repeat(" ", padding) + banner
}

func (m LogsModel) renderTabBar() string {
	activeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#6366F1")).
		Padding(0, 2)

	inactiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		Padding(0, 2)

	var tabs []string

	if m.activeLogType == models.LogTypeSystem {
		tabs = append(tabs, activeStyle.Render(fmt.Sprintf("System (%d)", len(m.filteredSystem))))
	} else {
		tabs = append(tabs, inactiveStyle.Render(fmt.Sprintf("System (%d)", len(m.filteredSystem))))
	}

	if m.activeLogType == models.LogTypeTraffic {
		tabs = append(tabs, activeStyle.Render(fmt.Sprintf("Traffic (%d)", len(m.filteredTraffic))))
	} else {
		tabs = append(tabs, inactiveStyle.Render(fmt.Sprintf("Traffic (%d)", len(m.filteredTraffic))))
	}

	if m.activeLogType == models.LogTypeThreat {
		tabs = append(tabs, activeStyle.Render(fmt.Sprintf("Threat (%d)", len(m.filteredThreat))))
	} else {
		tabs = append(tabs, inactiveStyle.Render(fmt.Sprintf("Threat (%d)", len(m.filteredThreat))))
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Center, tabs...)

	// Right side - sort info and last update
	sortInfo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Render(fmt.Sprintf("Sort: %s", m.sortLabel()))

	var updateInfo string
	if !m.lastRefresh.IsZero() {
		ago := time.Since(m.lastRefresh).Truncate(time.Second)
		updateInfo = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Render(fmt.Sprintf("  |  Updated %s ago", ago))
	}

	rightSide := sortInfo + updateInfo
	padding := m.width - lipgloss.Width(tabBar) - lipgloss.Width(rightSide) - 2
	if padding < 1 {
		padding = 1
	}

	return tabBar + strings.Repeat(" ", padding) + rightSide + "\n"
}

func (m LogsModel) renderFilterBar() string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6366F1")).
		Padding(0, 1).
		Render(m.filter.View()) + "\n"
}

func (m LogsModel) renderError() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#EF4444")).
		Bold(true).
		Padding(1, 0).
		Render(fmt.Sprintf("Error: %v", m.err))
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
	if m.loading && len(m.systemLogs) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Padding(1, 0).
			Render("Loading system logs...")
	}
	if len(m.filteredSystem) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Padding(1, 0).
			Render("No system logs found")
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#374151"))

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#1F2937")).
		Foreground(lipgloss.Color("#FFFFFF"))

	var b strings.Builder

	// Header - compact severity, more space for description
	header := fmt.Sprintf("%-19s %-4s %-18s %s",
		"Time", "Sev", "Type", "Description")
	b.WriteString(headerStyle.Render(header) + "\n")

	visibleRows := m.visibleRows()
	end := m.offset + visibleRows
	if end > len(m.filteredSystem) {
		end = len(m.filteredSystem)
	}

	for i := m.offset; i < end; i++ {
		log := m.filteredSystem[i]
		isSelected := i == m.cursor

		timeStr := log.Time.Format("2006-01-02 15:04:05")
		sevAbbrev := abbreviateSeverity(log.Severity)
		desc := truncate(log.Description, m.width-46)

		if isSelected {
			row := fmt.Sprintf("%-19s %-4s %-18s %s",
				timeStr,
				sevAbbrev,
				truncate(log.Type, 18),
				desc)
			b.WriteString(selectedStyle.Render(row) + "\n")
		} else {
			// Build row with colored severity indicator
			timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
			typeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
			descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB"))

			sevStyle := severityStyle(log.Severity)

			row := timeStyle.Render(fmt.Sprintf("%-19s", timeStr)) + " " +
				sevStyle.Render(fmt.Sprintf("%-4s", sevAbbrev)) + " " +
				typeStyle.Render(fmt.Sprintf("%-18s", truncate(log.Type, 18))) + " " +
				descStyle.Render(desc)
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
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true)
	case "high":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#F97316")).Bold(true)
	case "medium":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B"))
	case "low":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
	default: // informational
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	}
}

func (m LogsModel) renderTrafficTable() string {
	if m.loading && len(m.trafficLogs) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Padding(1, 0).
			Render("Loading traffic logs...")
	}
	if len(m.filteredTraffic) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Padding(1, 0).
			Render("No traffic logs found")
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#374151"))

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#1F2937")).
		Foreground(lipgloss.Color("#FFFFFF"))

	var b strings.Builder

	// Header
	header := fmt.Sprintf("%-19s %-7s %-15s %-15s %-12s %-15s %-10s",
		"Time", "Action", "Source", "Dest", "App", "Rule", "Bytes")
	b.WriteString(headerStyle.Render(header) + "\n")

	visibleRows := m.visibleRows()
	end := m.offset + visibleRows
	if end > len(m.filteredTraffic) {
		end = len(m.filteredTraffic)
	}

	for i := m.offset; i < end; i++ {
		log := m.filteredTraffic[i]
		isSelected := i == m.cursor

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
			b.WriteString(selectedStyle.Render(row) + "\n")
		} else {
			// Color code by action
			row = m.colorByAction(row, log.Action)
			b.WriteString(row + "\n")
		}
	}

	return b.String()
}

func (m LogsModel) renderThreatTable() string {
	if m.loading && len(m.threatLogs) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Padding(1, 0).
			Render("Loading threat logs...")
	}
	if len(m.filteredThreat) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Padding(1, 0).
			Render("No threat logs found")
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#374151"))

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#1F2937")).
		Foreground(lipgloss.Color("#FFFFFF"))

	var b strings.Builder

	// Header
	header := fmt.Sprintf("%-19s %-9s %-20s %-15s %-7s %-15s",
		"Time", "Severity", "Threat", "Source", "Action", "Category")
	b.WriteString(headerStyle.Render(header) + "\n")

	visibleRows := m.visibleRows()
	end := m.offset + visibleRows
	if end > len(m.filteredThreat) {
		end = len(m.filteredThreat)
	}

	for i := m.offset; i < end; i++ {
		log := m.filteredThreat[i]
		isSelected := i == m.cursor

		timeStr := log.Time.Format("2006-01-02 15:04:05")

		row := fmt.Sprintf("%-19s %-9s %-20s %-15s %-7s %-15s",
			timeStr,
			truncate(log.Severity, 9),
			truncate(log.ThreatName, 20),
			truncate(log.SourceIP, 15),
			truncate(log.Action, 7),
			truncate(log.ThreatCategory, 15))

		if isSelected {
			b.WriteString(selectedStyle.Render(row) + "\n")
		} else {
			// Color code by severity
			row = m.colorBySeverity(row, log.Severity)
			b.WriteString(row + "\n")
		}
	}

	return b.String()
}

func (m LogsModel) colorBySeverity(row, severity string) string {
	var style lipgloss.Style
	switch strings.ToLower(severity) {
	case "critical":
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
	case "high":
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("#F97316"))
	case "medium":
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B"))
	case "low":
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
	default:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	}
	return style.Render(row)
}

func (m LogsModel) colorByAction(row, action string) string {
	var style lipgloss.Style
	switch strings.ToLower(action) {
	case "allow":
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
	case "deny", "drop", "reset-client", "reset-server", "reset-both":
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
	default:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	}
	return style.Render(row)
}

func (m LogsModel) renderDetailPanel() string {
	switch m.activeLogType {
	case models.LogTypeSystem:
		if m.cursor < len(m.filteredSystem) {
			return m.renderSystemDetail(m.filteredSystem[m.cursor])
		}
	case models.LogTypeTraffic:
		if m.cursor < len(m.filteredTraffic) {
			return m.renderTrafficDetail(m.filteredTraffic[m.cursor])
		}
	case models.LogTypeThreat:
		if m.cursor < len(m.filteredThreat) {
			return m.renderThreatDetail(m.filteredThreat[m.cursor])
		}
	}
	return ""
}

func (m LogsModel) renderSystemDetail(log models.SystemLogEntry) string {
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("#6366F1")).
		Padding(0, 2).
		Width(m.width - 2).
		MarginTop(1)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A78BFA"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Width(12)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F3F4F6"))

	var lines []string
	lines = append(lines, titleStyle.Render("System Log Details"))
	lines = append(lines, "")
	lines = append(lines, labelStyle.Render("Time")+valueStyle.Render(log.Time.Format("2006-01-02 15:04:05")))
	lines = append(lines, labelStyle.Render("Severity")+severityStyle(log.Severity).Render(log.Severity))
	lines = append(lines, labelStyle.Render("Type")+valueStyle.Render(log.Type))
	lines = append(lines, "")
	lines = append(lines, titleStyle.Render("Message"))

	// Word wrap the description for better readability
	descWidth := m.width - 10
	if descWidth > 100 {
		descWidth = 100
	}
	wrapped := wrapText(log.Description, descWidth)
	for _, line := range wrapped {
		lines = append(lines, valueStyle.Render(line))
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
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("#6366F1")).
		Padding(0, 2).
		Width(m.width - 2).
		MarginTop(1)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A78BFA"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#9CA3AF")).MarginTop(1)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Width(14)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F3F4F6"))

	var lines []string
	lines = append(lines, titleStyle.Render("Traffic Log Details"))

	lines = append(lines, sectionStyle.Render("Session"))
	lines = append(lines, labelStyle.Render("Time")+valueStyle.Render(log.Time.Format("2006-01-02 15:04:05")))
	lines = append(lines, labelStyle.Render("Action")+m.colorByAction(log.Action, log.Action))
	lines = append(lines, labelStyle.Render("Session ID")+valueStyle.Render(fmt.Sprintf("%d", log.SessionID)))
	lines = append(lines, labelStyle.Render("Duration")+valueStyle.Render(fmt.Sprintf("%ds", log.Duration)))

	lines = append(lines, sectionStyle.Render("Source / Destination"))
	lines = append(lines, labelStyle.Render("Source")+valueStyle.Render(fmt.Sprintf("%s:%d (%s)", log.SourceIP, log.SourcePort, log.SourceZone)))
	lines = append(lines, labelStyle.Render("Destination")+valueStyle.Render(fmt.Sprintf("%s:%d (%s)", log.DestIP, log.DestPort, log.DestZone)))
	if log.NATSourceIP != "" {
		lines = append(lines, labelStyle.Render("NAT Source")+valueStyle.Render(fmt.Sprintf("%s:%d", log.NATSourceIP, log.NATSourcePort)))
	}
	if log.NATDestIP != "" {
		lines = append(lines, labelStyle.Render("NAT Dest")+valueStyle.Render(fmt.Sprintf("%s:%d", log.NATDestIP, log.NATDestPort)))
	}

	lines = append(lines, sectionStyle.Render("Application"))
	lines = append(lines, labelStyle.Render("Application")+valueStyle.Render(log.Application))
	lines = append(lines, labelStyle.Render("Protocol")+valueStyle.Render(log.Protocol))
	lines = append(lines, labelStyle.Render("Rule")+valueStyle.Render(log.Rule))
	if log.User != "" {
		lines = append(lines, labelStyle.Render("User")+valueStyle.Render(log.User))
	}

	lines = append(lines, sectionStyle.Render("Traffic"))
	lines = append(lines, labelStyle.Render("Bytes")+valueStyle.Render(fmt.Sprintf("%s (sent: %s, recv: %s)", formatBytes(log.Bytes), formatBytes(log.BytesSent), formatBytes(log.BytesRecv))))
	lines = append(lines, labelStyle.Render("Packets")+valueStyle.Render(fmt.Sprintf("%d (sent: %d, recv: %d)", log.Packets, log.PacketsSent, log.PacketsRecv)))

	return panelStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m LogsModel) renderThreatDetail(log models.ThreatLogEntry) string {
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("#6366F1")).
		Padding(0, 2).
		Width(m.width - 2).
		MarginTop(1)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A78BFA"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#9CA3AF")).MarginTop(1)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Width(14)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F3F4F6"))

	var lines []string
	lines = append(lines, titleStyle.Render("Threat Log Details"))

	lines = append(lines, sectionStyle.Render("Threat"))
	lines = append(lines, labelStyle.Render("Time")+valueStyle.Render(log.Time.Format("2006-01-02 15:04:05")))
	lines = append(lines, labelStyle.Render("Severity")+m.colorBySeverity(log.Severity, log.Severity))
	lines = append(lines, labelStyle.Render("Threat Name")+valueStyle.Render(log.ThreatName))
	lines = append(lines, labelStyle.Render("Threat ID")+valueStyle.Render(fmt.Sprintf("%d", log.ThreatID)))
	lines = append(lines, labelStyle.Render("Category")+valueStyle.Render(log.ThreatCategory))
	lines = append(lines, labelStyle.Render("Subtype")+valueStyle.Render(log.Subtype))
	lines = append(lines, labelStyle.Render("Action")+m.colorByAction(log.Action, log.Action))
	lines = append(lines, labelStyle.Render("Direction")+valueStyle.Render(log.Direction))

	lines = append(lines, sectionStyle.Render("Source / Destination"))
	lines = append(lines, labelStyle.Render("Source")+valueStyle.Render(fmt.Sprintf("%s:%d (%s)", log.SourceIP, log.SourcePort, log.SourceZone)))
	lines = append(lines, labelStyle.Render("Destination")+valueStyle.Render(fmt.Sprintf("%s:%d (%s)", log.DestIP, log.DestPort, log.DestZone)))

	lines = append(lines, sectionStyle.Render("Context"))
	lines = append(lines, labelStyle.Render("Application")+valueStyle.Render(log.Application))
	lines = append(lines, labelStyle.Render("Rule")+valueStyle.Render(log.Rule))
	if log.User != "" {
		lines = append(lines, labelStyle.Render("User")+valueStyle.Render(log.User))
	}
	if log.URL != "" {
		lines = append(lines, labelStyle.Render("URL")+valueStyle.Render(truncate(log.URL, m.width-20)))
	}
	if log.Filename != "" {
		lines = append(lines, labelStyle.Render("Filename")+valueStyle.Render(log.Filename))
	}

	return panelStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m LogsModel) renderHelp() string {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	expandText := "details"
	if m.expanded {
		expandText = "collapse"
	}

	keys := []struct{ key, desc string }{
		{"tab", "log type"},
		{"j/k", "scroll"},
		{"enter", expandText},
		{"/", "filter"},
		{"s", "sort field"},
		{"S", "sort dir"},
		{"r", "refresh"},
	}

	var parts []string
	for _, k := range keys {
		parts = append(parts, keyStyle.Render(k.key)+descStyle.Render(":"+k.desc))
	}

	return lipgloss.NewStyle().MarginTop(1).Render(strings.Join(parts, "  "))
}
