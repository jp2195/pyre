package views

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

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

	// One error per log type. The three tabs are three independent fetches;
	// a single shared Err meant one failure blanked the other two tabs.
	systemErr  error
	trafficErr error
	threatErr  error

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
	m.EnsureCursorValid(m.filteredCount())
	if visibleRows := m.visibleRows(); visibleRows > 0 {
		m.EnsureVisible(visibleRows)
	}
	return m
}

func (m LogsModel) SetLoading(loading bool) LogsModel {
	m.TableBase = m.TableBase.SetLoading(loading)
	return m
}

// SetSpinnerFrame updates the current spinner animation frame.
func (m LogsModel) SetSpinnerFrame(frame string) LogsModel {
	m.TableBase = m.TableBase.SetSpinnerFrame(frame)
	return m
}

// HasData returns true if any logs have been loaded.
func (m LogsModel) HasData() bool {
	return m.systemLogs != nil || m.trafficLogs != nil || m.threatLogs != nil
}

func (m LogsModel) SetSystemLogs(logs []models.SystemLogEntry, err error) LogsModel {
	m.systemLogs = logs
	m.systemErr = err
	m.Loading = false
	m.lastRefresh = time.Now()
	m.applyFilter()
	m.ensureCursorValid()
	return m
}

func (m LogsModel) SetTrafficLogs(logs []models.TrafficLogEntry, err error) LogsModel {
	m.trafficLogs = logs
	m.trafficErr = err
	m.Loading = false
	m.lastRefresh = time.Now()
	m.applyFilter()
	m.ensureCursorValid()
	return m
}

func (m LogsModel) SetThreatLogs(logs []models.ThreatLogEntry, err error) LogsModel {
	m.threatLogs = logs
	m.threatErr = err
	m.Loading = false
	m.lastRefresh = time.Now()
	m.applyFilter()
	m.ensureCursorValid()
	return m
}

func (m LogsModel) ActiveLogType() models.LogType {
	return m.activeLogType
}

// activeErr returns the error for the tab currently on screen, so a failed
// fetch only blanks its own tab.
func (m LogsModel) activeErr() error {
	switch m.activeLogType {
	case models.LogTypeTraffic:
		return m.trafficErr
	case models.LogTypeThreat:
		return m.threatErr
	default:
		return m.systemErr
	}
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
	m.filteredSystem = filterSystemLogs(m.systemLogs, query)
	m.filteredTraffic = filterTrafficLogs(m.trafficLogs, query)
	m.filteredThreat = filterThreatLogs(m.threatLogs, query)
	m.applySort()
}

func (m *LogsModel) applySort() {
	sortSystemLogs(m.filteredSystem, m.sortBy, m.SortAsc)
	sortTrafficLogs(m.filteredTraffic, m.sortBy, m.SortAsc)
	sortThreatLogs(m.filteredThreat, m.sortBy, m.SortAsc)
}

func (m *LogsModel) cycleSort() {
	m.sortBy = (m.sortBy + 1) % 4
	m.applySort()
}

func (m LogsModel) sortLabel() string {
	dir := "↓"
	if m.SortAsc {
		dir = "↑"
	}
	switch m.sortBy {
	case LogSortSeverity:
		return fmt.Sprintf("Severity %s", dir)
	case LogSortSource:
		return fmt.Sprintf("Source %s", dir)
	case LogSortAction:
		return fmt.Sprintf("Action %s", dir)
	default:
		return fmt.Sprintf("Time %s", dir)
	}
}

func (m LogsModel) Update(msg tea.Msg) (LogsModel, tea.Cmd) {
	if m.FilterMode {
		return m.updateFilterMode(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
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
	// Detect enter before delegating: TableBase exits filter mode for both
	// enter and esc but doesn't distinguish them in its return value, and
	// logs only re-applies its derived filtered slice on enter (commit), not
	// on esc.
	committed := false
	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "enter" {
		committed = true
	}
	var cmd tea.Cmd
	m.TableBase, _, cmd = m.HandleFilterMode(msg)
	if committed {
		m.applyFilter()
	}
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
	if activeErr := m.activeErr(); activeErr != nil {
		sections = append(sections, m.renderError(activeErr))
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

	// The sort/updated block is supplementary. If the tabs and it together do
	// not fit, drop it rather than pushing the line past the terminal edge:
	// padding was clamped to a minimum of one space, so a narrow terminal
	// produced a line wider than the screen.
	rightSide := sortInfo + updateInfo
	gap := m.Width - lipgloss.Width(tabBar) - lipgloss.Width(rightSide) - 2
	if gap < 1 {
		rightSide = ""
		gap = m.Width - lipgloss.Width(tabBar)
	}
	if gap < 0 {
		gap = 0
	}

	return tabBar + strings.Repeat(" ", gap) + rightSide + "\n"
}

func (m LogsModel) renderFilterBar() string {
	return FilterBorderStyle.Render(m.Filter.View()) + "\n"
}

func (m LogsModel) renderError(err error) string {
	return ErrorMsgStyle.Bold(true).Padding(1, 0).Render(fmt.Sprintf("Error: %v", err))
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

func (m LogsModel) renderDetailPanel() string {
	switch m.activeLogType {
	case models.LogTypeSystem:
		if m.Cursor >= 0 && m.Cursor < len(m.filteredSystem) {
			return m.renderSystemDetail(m.filteredSystem[m.Cursor])
		}
	case models.LogTypeTraffic:
		if m.Cursor >= 0 && m.Cursor < len(m.filteredTraffic) {
			return m.renderTrafficDetail(m.filteredTraffic[m.Cursor])
		}
	case models.LogTypeThreat:
		if m.Cursor >= 0 && m.Cursor < len(m.filteredThreat) {
			return m.renderThreatDetail(m.filteredThreat[m.Cursor])
		}
	}
	return ""
}

func (m LogsModel) renderHelp() string {
	expandText := "details"
	if m.Expanded {
		expandText = "collapse"
	}

	keys := []struct{ key, desc string }{
		{"[", "prev type"},
		{"]", "next type"},
		{"j/k", "scroll"},
		{"enter", expandText},
		{"/", "filter"},
		{"s", "sort field"},
		{"S", "sort dir"},
		{"r", "refresh"},
	}

	// Keep only the hints that fit. The full list is wider than a narrow
	// terminal, and because the view is joined vertically, one over-long line
	// widened every other line with it and pushed the whole table off screen.
	var (
		parts []string
		used  int
	)
	for _, k := range keys {
		part := HelpKeyStyle.Render(k.key) + HelpDescStyle.Render(":"+k.desc)
		w := lipgloss.Width(part)
		if len(parts) > 0 {
			w += 2 // the separator
		}
		if m.Width > 0 && used+w > m.Width {
			break
		}
		used += w
		parts = append(parts, part)
	}

	return ViewSubtitleStyle.MarginTop(1).Render(strings.Join(parts, "  "))
}

// --- Shared helpers used by log type files ---

// renderLogRows renders the visible window of a filtered log slice. renderRow
// is called for each visible item and must return the fully styled row string
// for both selected and normal states.
func renderLogRows[T any](offset, cursor, visibleRows int, items []T, renderRow func(item T, selected bool) string) string {
	var b strings.Builder
	end := min(offset+visibleRows, len(items))
	for i := offset; i < end; i++ {
		b.WriteString(renderRow(items[i], i == cursor) + "\n")
	}
	return b.String()
}

// logColumnLayout describes how a log table divides the terminal: a set of
// fixed column widths plus one column that absorbs whatever is left.
//
// The log tables used to use a single hardcoded set of widths, so on a wide
// terminal they stopped around column 110 and truncated values there was room
// for, while on a narrow one they overflowed the screen. Several fixed widths
// were also too small for the vocabulary the device emits: an Action column
// of 7 cells cannot hold "sinkhole", and a Severity column of 9 cannot hold
// "informational".
type logColumnLayout struct {
	wide bool // the terminal is roomy enough for the full column set
	flex int  // width of the one column that takes the remaining space
}

// The flexible column is bounded at both ends: wide enough to be readable on
// a small terminal, and narrow enough that a very wide one does not stretch a
// single column across half the screen and strand the columns after it.
const (
	minFlexColumn = 12
	maxFlexColumn = 48
)

// logLayout picks the column set for the given terminal width and sizes the
// flexible column from what the fixed ones leave behind.
func logLayout(width, fixedWide, fixedNarrow int, wideThreshold int) logColumnLayout {
	clamp := func(n int) int { return min(max(n, minFlexColumn), maxFlexColumn) }
	if width >= wideThreshold {
		return logColumnLayout{wide: true, flex: clamp(width - fixedWide)}
	}
	return logColumnLayout{wide: false, flex: clamp(width - fixedNarrow)}
}

// abbreviateSeverity returns a short severity label.
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

func colorBySeverity(row, severity string) string {
	return SeverityStyle(severity).Render(row)
}

func colorByAction(row, action string) string {
	return ActionStyle(action).Render(row)
}
