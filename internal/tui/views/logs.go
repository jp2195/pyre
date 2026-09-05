package views

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/models"
)

// FetchLogsCmd asks the parent model to fetch one page for one log tab. The
// view owns the range and the query but has no API client, so it describes
// the fetch and the parent performs it — the same split FetchDetailCmd uses.
type FetchLogsCmd struct {
	Type   models.LogType
	Query  string
	Since  time.Time
	Skip   int
	Append bool
}

// LogPageMeta is what a completed fetch reports about the page it returned.
// It exists so the views package never imports internal/api.
type LogPageMeta struct {
	HasMore bool
	// Sent is the assembled expression the device received.
	Sent string
	// Append adds these rows to the tab instead of replacing them.
	Append bool
}

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

	// One state per log tab. The three tabs are three independent fetches,
	// so a failure, a page count, and a staleness flag all belong to one
	// tab rather than to the view.
	tabs map[models.LogType]logTabState

	// rng is the selected time-range preset, shared by all three tabs.
	rng LogRange
	// query is the last expression the device accepted.
	query string

	sortBy      LogSortField
	lastRefresh time.Time
}

// logTabState is everything the view knows about one tab that is not the rows
// themselves. The rows stay in their own typed slices because they are three
// genuinely different types.
type logTabState struct {
	// err is the last fetch error for this tab, so one failed tab does not
	// blank the other two.
	err error
	// fetched is how many rows have been loaded, and therefore the Skip
	// for the next page.
	fetched int
	// hasMore reports that the last page came back exactly full.
	hasMore bool
	// sent is the assembled expression the device received for this tab.
	sent string
	// rng and query are what these rows were fetched under. Staleness is
	// derived by comparing them against the view's current selection
	// rather than flagged when the selection changes: nothing has to
	// remember to mark the other tabs, and refresh needs no special case.
	rng   LogRange
	query string
}

func (m LogsModel) tabState(t models.LogType) logTabState {
	return m.tabs[t]
}

func (m *LogsModel) setTabState(t models.LogType, s logTabState) {
	if m.tabs == nil {
		m.tabs = make(map[models.LogType]logTabState, 3)
	}
	m.tabs[t] = s
}

// tabStale reports whether a tab's rows were fetched under a different range
// or query than the one now selected, and so must be refetched before they
// are shown again.
func (m LogsModel) tabStale(t models.LogType) bool {
	s := m.tabState(t)
	return s.rng != m.rng || s.query != m.query
}

// LogRange is a time-range preset for a log query. The zero value is
// LogRangeAll, which sends no bound at all -- the behavior the view had
// before server-side queries existed.
type LogRange int

const (
	LogRangeAll LogRange = iota
	LogRange15m
	LogRange1h
	LogRange24h
	LogRange7d
	logRangeCount
)

// Label is the short form shown in the status line.
func (r LogRange) Label() string {
	switch r {
	case LogRange15m:
		return "15m"
	case LogRange1h:
		return "1h"
	case LogRange24h:
		return "24h"
	case LogRange7d:
		return "7d"
	default:
		return "all"
	}
}

// Since is the lower bound for this preset, or the zero time for "all". The
// caller passes now so the bound is testable, and so one keystroke cannot
// produce two different bounds for two tabs.
func (r LogRange) Since(now time.Time) time.Time {
	switch r {
	case LogRange15m:
		return now.Add(-15 * time.Minute)
	case LogRange1h:
		return now.Add(-time.Hour)
	case LogRange24h:
		return now.Add(-24 * time.Hour)
	case LogRange7d:
		return now.Add(-7 * 24 * time.Hour)
	default:
		return time.Time{}
	}
}

// Range is the selected time-range preset.
func (m LogsModel) Range() LogRange { return m.rng }

// RangeSince is the lower bound implied by the selected range.
func (m LogsModel) RangeSince() time.Time { return m.rng.Since(time.Now()) }

// Query is the last expression the device accepted.
func (m LogsModel) Query() string { return m.query }

func NewLogsModel() LogsModel {
	base := NewTableBase("Filter logs...")
	base.SortAsc = false // Default to newest first
	return LogsModel{
		TableBase:     base,
		activeLogType: models.LogTypeSystem,
		tabs:          make(map[models.LogType]logTabState, 3),
	}
}

// SetActiveLogType selects which log tab is displayed. The tab is normally
// chosen with the bracket keys; this lets callers address a tab directly.
func (m LogsModel) SetActiveLogType(t models.LogType) LogsModel {
	m.activeLogType = t
	return m
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

// SetSystemLogs records the result of a system log fetch. On error the
// existing rows are left alone rather than replaced with nil: a later query
// the device rejects must not blank rows already on screen.
func (m LogsModel) SetSystemLogs(logs []models.SystemLogEntry, meta LogPageMeta, err error) LogsModel {
	s := m.tabState(models.LogTypeSystem)
	s.err = err
	if err == nil {
		if meta.Append {
			m.systemLogs = append(m.systemLogs, logs...)
		} else {
			m.systemLogs = logs
		}
		s.fetched = len(m.systemLogs)
		s.hasMore = meta.HasMore
		s.sent = meta.Sent
		s.rng = m.rng
		s.query = m.query
	}
	m.setTabState(models.LogTypeSystem, s)
	m.Loading = false
	m.lastRefresh = time.Now()
	m.applyFilter()
	m.ensureCursorValid()
	return m
}

// SetTrafficLogs records the result of a traffic log fetch. See
// SetSystemLogs for the error-preserves-rows behavior.
func (m LogsModel) SetTrafficLogs(logs []models.TrafficLogEntry, meta LogPageMeta, err error) LogsModel {
	s := m.tabState(models.LogTypeTraffic)
	s.err = err
	if err == nil {
		if meta.Append {
			m.trafficLogs = append(m.trafficLogs, logs...)
		} else {
			m.trafficLogs = logs
		}
		s.fetched = len(m.trafficLogs)
		s.hasMore = meta.HasMore
		s.sent = meta.Sent
		s.rng = m.rng
		s.query = m.query
	}
	m.setTabState(models.LogTypeTraffic, s)
	m.Loading = false
	m.lastRefresh = time.Now()
	m.applyFilter()
	m.ensureCursorValid()
	return m
}

// SetThreatLogs records the result of a threat log fetch. See
// SetSystemLogs for the error-preserves-rows behavior.
func (m LogsModel) SetThreatLogs(logs []models.ThreatLogEntry, meta LogPageMeta, err error) LogsModel {
	s := m.tabState(models.LogTypeThreat)
	s.err = err
	if err == nil {
		if meta.Append {
			m.threatLogs = append(m.threatLogs, logs...)
		} else {
			m.threatLogs = logs
		}
		s.fetched = len(m.threatLogs)
		s.hasMore = meta.HasMore
		s.sent = meta.Sent
		s.rng = m.rng
		s.query = m.query
	}
	m.setTabState(models.LogTypeThreat, s)
	m.Loading = false
	m.lastRefresh = time.Now()
	m.applyFilter()
	m.ensureCursorValid()
	return m
}

func (m LogsModel) ActiveLogType() models.LogType {
	return m.activeLogType
}

// fetchRequest builds the command that asks for one page of a tab.
func (m LogsModel) fetchRequest(t models.LogType, skip int, appendRows bool) tea.Cmd {
	req := FetchLogsCmd{
		Type:   t,
		Query:  m.query,
		Since:  m.RangeSince(),
		Skip:   skip,
		Append: appendRows,
	}
	return func() tea.Msg { return req }
}

// onTabSwitch resets the cursor and asks for the newly shown tab when it has
// no rows yet or its rows predate the current range or query.
func (m LogsModel) onTabSwitch() (LogsModel, tea.Cmd) {
	m.Cursor = 0
	m.Offset = 0
	m.Expanded = false

	if m.rowCount(m.activeLogType) == 0 || m.tabStale(m.activeLogType) {
		return m, m.fetchRequest(m.activeLogType, 0, false)
	}
	return m, nil
}

// onQueryChanged resets the active tab to its first page and refetches it.
// The other two tabs need no marking: each records the range and query its
// rows came from, so they now read as stale and are refetched when next
// shown. That is also why refresh needs no special case — it refetches the
// visible tab, and the other two go stale on their own.
func (m LogsModel) onQueryChanged() (LogsModel, tea.Cmd) {
	s := m.tabState(m.activeLogType)
	s.fetched = 0
	s.hasMore = false
	m.setTabState(m.activeLogType, s)
	m.Cursor = 0
	m.Offset = 0
	m.Expanded = false
	return m, m.fetchRequest(m.activeLogType, 0, false)
}

// rowCount is how many unfiltered rows a tab holds.
func (m LogsModel) rowCount(t models.LogType) int {
	switch t {
	case models.LogTypeTraffic:
		return len(m.trafficLogs)
	case models.LogTypeThreat:
		return len(m.threatLogs)
	default:
		return len(m.systemLogs)
	}
}

// activeErr returns the error for the tab currently on screen, so a failed
// fetch only blanks its own tab.
func (m LogsModel) activeErr() error {
	return m.tabState(m.activeLogType).err
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

// statusLine names the range, the active query and how much is loaded. It
// never claims a total: the device reports no match count, so the honest
// phrasing is "more available" rather than "500 of N".
func (m LogsModel) statusLine() string {
	s := m.tabState(m.activeLogType)
	parts := []string{m.rng.Label()}

	if m.query != "" && m.Width >= 80 {
		parts = append(parts, m.query)
	}

	switch {
	case s.hasMore && m.Width < 60:
		parts = append(parts, fmt.Sprintf("%d+", s.fetched))
	case s.hasMore:
		parts = append(parts, fmt.Sprintf("%d shown, more available", s.fetched))
	case s.fetched > 0:
		parts = append(parts, fmt.Sprintf("%d shown", s.fetched))
	}

	return ViewSubtitleStyle.Render(truncateEllipsis(strings.Join(parts, " · "), m.Width))
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
		case "t":
			m.rng = (m.rng + 1) % logRangeCount
			return m.onQueryChanged()
		case "T":
			m.rng = (m.rng + logRangeCount - 1) % logRangeCount
			return m.onQueryChanged()
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
			return m.onTabSwitch()
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
			return m.onTabSwitch()
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
	sections = append(sections, m.statusLine())

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
		{"t", "range"},
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
	tier logTier
	flex int // width of the flexible column; 0 means it is omitted entirely
}

// logTier names the column sets a log table can render, widest first.
type logTier int

const (
	logTierWide logTier = iota
	logTierNarrow
	logTierCompact
)

// wide reports whether the full column set was selected.
func (l logColumnLayout) wide() bool { return l.tier == logTierWide }

// compact reports whether the smallest column set was selected.
func (l logColumnLayout) compact() bool { return l.tier == logTierCompact }

// The flexible column is bounded at both ends: wide enough to be readable on
// a small terminal, and narrow enough that a very wide one does not stretch a
// single column across half the screen and strand the columns after it.
const (
	minFlexColumn = 12
	maxFlexColumn = 48
)

// logLayout picks the widest column set that actually fits, and sizes the
// flexible column from what the fixed columns leave behind.
//
// fixed gives the total width of each tier's fixed columns, including the
// single space between them and before the flexible column. A tier is only
// eligible when it leaves room for a readable flexible column, so a table is
// never rendered wider than the terminal it was measured against. Selecting
// a tier by threshold alone, without that check, is what let the narrow
// traffic set overflow every terminal below 88 columns.
//
// wideThreshold holds the widest tier back until the terminal is roomy
// enough to be worth the extra columns, which is a taste judgment rather
// than a fitting one.
func logLayout(width int, fixed [3]int, wideThreshold int) logColumnLayout {
	for _, tier := range []logTier{logTierWide, logTierNarrow, logTierCompact} {
		if tier == logTierWide && width < wideThreshold {
			continue
		}
		if flex := width - fixed[tier]; flex >= minFlexColumn {
			return logColumnLayout{tier: tier, flex: min(flex, maxFlexColumn)}
		}
	}
	// Narrower than even the compact set: drop its flexible column rather
	// than render it below the readable minimum and spill off the screen.
	return logColumnLayout{tier: logTierCompact, flex: 0}
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

// logTimeOnlyLayout drops the date from a log timestamp. Every row on screen
// is from the same recent window, so the date is the least informative thing
// the line carries and the first thing worth spending on a narrower terminal.
const logTimeOnlyLayout = "15:04:05"

// formatCompactRow lays out the smallest column set: a time column, some
// fixed columns, then the flexible column, which is omitted when the layout
// could not give it a readable width. Header and data rows share it so their
// columns cannot drift apart.
func formatCompactRow(l logColumnLayout, timeWidth int, timeText string, widths []int, cells []string, flexText string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%-*s", timeWidth, truncate(timeText, timeWidth))
	for i, w := range widths {
		fmt.Fprintf(&b, " %-*s", w, truncate(cells[i], w))
	}
	if l.flex > 0 {
		fmt.Fprintf(&b, " %-*s", l.flex, truncate(flexText, l.flex))
	}
	return b.String()
}
