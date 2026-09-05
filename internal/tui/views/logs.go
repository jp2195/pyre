package views

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/models"
)

// FetchLogsCmd asks the parent model to fetch one page for one log tab. The
// view owns the range and the query but has no API client, so it describes
// the fetch and the parent performs it — the same split FetchDetailCmd uses.
//
// A request is also the fetch's identity. It travels to the device and comes
// back on the page, so a response can be matched against what the tab wants
// by then. Without that, a page issued 1–4 seconds ago is applied to whatever
// the tab holds now: two presses of m append the same rows twice, and a page
// from one time range gets appended to rows from another and then labeled
// with the current selection.
type FetchLogsCmd struct {
	Type   models.LogType
	Query  string
	Since  time.Time
	Skip   int
	Append bool
	// Range is the preset this fetch was issued under. Rows are only
	// comparable within one range, so a page whose range no longer matches
	// the selection is dropped rather than merged.
	Range LogRange
	// ReqID identifies this fetch among the ones issued for its tab. It
	// increases monotonically per tab, so a response carrying any id but
	// the one the tab is waiting on has been superseded.
	ReqID int
}

// LogPageMeta is what a completed fetch reports about the page it returned.
// It exists so the views package never imports internal/api.
type LogPageMeta struct {
	// Req is the request this page answers, echoed back unchanged. Append
	// lives on it, along with the identity the page is matched against.
	Req     FetchLogsCmd
	HasMore bool
	// Sent is the assembled expression the device received.
	Sent string
	// Warning is a non-fatal problem with the fetch the operator needs to
	// see -- today, that the time bound had to be written without knowing
	// the device's UTC offset. Empty when there is nothing to say.
	Warning string
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
	// query is the expression the view is asking the device for right now.
	// It is what a fetch sends, what tab staleness is keyed to, and what
	// the status line names.
	query string
	// accepted is the last expression the device answered without an
	// error. A query the device refuses is never stored: query reverts to
	// this when a fetch carrying it fails, so a refresh does not keep
	// retrying an expression the device has already turned down and the
	// other tabs do not read stale against one.
	accepted string
	// committed is the text the operator last put in the query bar with
	// enter. It is kept even when the device refuses it, so reopening the
	// bar offers the expression for correction rather than making them
	// type it again.
	committed string

	// queryInput edits the device-side expression. It is separate from
	// TableBase.Filter, which filters rows already on screen.
	queryInput textinput.Model
	queryMode  bool

	sortBy LogSortField
}

// logTabState is everything the view knows about one tab that is not the rows
// themselves. The rows stay in their own typed slices because they are three
// genuinely different types.
type logTabState struct {
	// err is the last fetch error for this tab, so one failed tab does not
	// blank the other two.
	err error
	// errQuery is the expression whose fetch produced err, kept so the
	// device's message and the query that caused it are read together. The
	// message alone does not name what was asked, and a refused query is
	// deliberately not stored anywhere else.
	errQuery string
	// fetched is how many rows have been loaded, and therefore the Skip
	// for the next page.
	fetched int
	// hasMore reports that the last page came back exactly full.
	hasMore bool
	// sent is the assembled expression the device received for this tab.
	sent string
	// warning is the last non-fatal complaint about this tab's fetch. It
	// is shown rather than logged: a bound written in a guessed zone
	// produces a plausible-looking table for the wrong window, which is
	// exactly the kind of wrong answer nobody thinks to question.
	warning string
	// rng and query are what these rows were fetched under. Staleness is
	// derived by comparing them against the view's current selection
	// rather than flagged when the selection changes: nothing has to
	// remember to mark the other tabs, and refresh needs no special case.
	rng   LogRange
	query string
	// seq counts the fetches issued for this tab, so every one of them can
	// be told apart from the one before it.
	seq int
	// pending is the fetch this tab is waiting on, or the zero value when
	// it is waiting on nothing (ReqID 0 means idle; ids start at 1). It
	// carries the identity every response is matched against, and lets a
	// tab switch see that the page it would ask for is already on its way.
	pending FetchLogsCmd
	// loading is whether this tab has work in flight. It is per tab
	// because the three tabs are three independent fetches: a System
	// response landing while Traffic is still fetching must not tell
	// Traffic it is finished and let it render "No traffic logs found".
	loading bool
	// since is the lower bound the current pagination started under. Every
	// page of one pagination reuses it: recomputing it per page moves the
	// window forward over a newest-first list that also grows at the head,
	// so a skip counted from the rows already shown starts repeating them.
	since time.Time
	// lastRefresh is when this tab's rows arrived, so the tab bar's
	// "Updated Ns ago" describes the rows on screen rather than whichever
	// tab answered most recently.
	lastRefresh time.Time
}

// inflightMatches reports whether the fetch this tab is waiting on was issued
// under the given range and query, and so will return rows the view still
// wants.
func (s logTabState) inflightMatches(rng LogRange, query string) bool {
	return s.pending.ReqID != 0 && s.pending.Range == rng && s.pending.Query == query
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

// syncLoading mirrors the active tab's in-flight state onto TableBase.Loading,
// which drives the loading banner and the parent's spinner gate.
func (m *LogsModel) syncLoading() {
	m.Loading = m.tabState(m.activeLogType).loading
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

// Query is the last expression the device accepted. An expression the device
// refused is never reported here.
func (m LogsModel) Query() string { return m.accepted }

func NewLogsModel() LogsModel {
	base := NewTableBase("Filter logs...")
	base.SortAsc = false // Default to newest first

	qi := textinput.New()
	qi.Placeholder = "PAN-OS query, e.g. (addr.src in 203.0.113.5)"
	qi.CharLimit = 512

	return LogsModel{
		TableBase:     base,
		activeLogType: models.LogTypeSystem,
		tabs:          make(map[models.LogType]logTabState, 3),
		queryInput:    qi,
	}
}

// SetActiveLogType selects which log tab is displayed. The tab is normally
// chosen with the bracket keys; this lets callers address a tab directly.
func (m LogsModel) SetActiveLogType(t models.LogType) LogsModel {
	m.activeLogType = t
	m.syncLoading()
	return m
}

func (m LogsModel) SetSize(width, height int) LogsModel {
	m.TableBase = m.TableBase.SetSize(width, height)
	m.queryInput.SetWidth(queryInputWidth(width, m.queryInput.Prompt))
	m.EnsureCursorValid(m.filteredCount())
	if visibleRows := m.visibleRows(); visibleRows > 0 {
		m.EnsureVisible(visibleRows)
	}
	return m
}

// queryBarLabel prefixes the query input in View(). Its width counts toward
// the input's own width budget below.
const queryBarLabel = "device query: "

// minQueryInputWidth is a floor so a very narrow terminal cannot drive the
// input's width to zero or negative. table_base.go was once bitten by a
// strings.Repeat panic from exactly that below 12 columns.
const minQueryInputWidth = 10

// queryInputWidth sizes the query bar's text input to fit next to its label
// inside the terminal. It is computed from termWidth rather than set once in
// NewLogsModel because the terminal can be resized after construction, and
// SetSize is where the view learns about that.
//
// prompt and a reserved cursor cell also count against the budget: bubbles'
// textinput renders promptWidth+Width()+1 cells wide, not Width() cells --
// the trailing +1 is the cursor cell at the end of the visible window.
// command_palette.go's own sizing carries the same reservation.
func queryInputWidth(termWidth int, prompt string) int {
	overhead := lipgloss.Width(queryBarLabel) + lipgloss.Width(prompt) + 1
	return max(termWidth-overhead, minQueryInputWidth)
}

// SetLoading marks the tab on screen as loading or idle. Loading is per tab
// rather than per view: the three tabs are three independent fetches, so a
// response for one of them must not decide whether the others are still
// waiting.
func (m LogsModel) SetLoading(loading bool) LogsModel {
	s := m.tabState(m.activeLogType)
	s.loading = loading
	m.setTabState(m.activeLogType, s)
	m.syncLoading()
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
	t := models.LogTypeSystem
	m, wanted := m.beginPage(t, meta)
	if !wanted {
		return m
	}
	if err == nil {
		if meta.Req.Append {
			m.systemLogs = append(m.systemLogs, logs...)
		} else {
			m.systemLogs = logs
		}
	}
	return m.finishPage(t, meta, err, len(m.systemLogs))
}

// SetTrafficLogs records the result of a traffic log fetch. See
// SetSystemLogs for the error-preserves-rows behavior.
func (m LogsModel) SetTrafficLogs(logs []models.TrafficLogEntry, meta LogPageMeta, err error) LogsModel {
	t := models.LogTypeTraffic
	m, wanted := m.beginPage(t, meta)
	if !wanted {
		return m
	}
	if err == nil {
		if meta.Req.Append {
			m.trafficLogs = append(m.trafficLogs, logs...)
		} else {
			m.trafficLogs = logs
		}
	}
	return m.finishPage(t, meta, err, len(m.trafficLogs))
}

// SetThreatLogs records the result of a threat log fetch. See
// SetSystemLogs for the error-preserves-rows behavior.
func (m LogsModel) SetThreatLogs(logs []models.ThreatLogEntry, meta LogPageMeta, err error) LogsModel {
	t := models.LogTypeThreat
	m, wanted := m.beginPage(t, meta)
	if !wanted {
		return m
	}
	if err == nil {
		if meta.Req.Append {
			m.threatLogs = append(m.threatLogs, logs...)
		} else {
			m.threatLogs = logs
		}
	}
	return m.finishPage(t, meta, err, len(m.threatLogs))
}

// beginPage decides what to do with a completed fetch before its rows are
// touched. The second result reports whether the page is still wanted; when
// it is not, the returned model has already recorded whatever the tab's
// in-flight state should become.
//
// A page takes 1–4 seconds to arrive, and two things can happen in that time.
// The tab may have asked again — another page, a refresh, a revisit — in
// which case a newer fetch is still on its way and this one is simply late.
// Or the range or query may have moved on, which every tab's rows are keyed
// to; that tab is then idle but stale, and is refetched when next shown.
// Applying a page in either state appends rows gathered under one selection
// onto rows gathered under another and then labels the mixture with the
// current one.
func (m LogsModel) beginPage(t models.LogType, meta LogPageMeta) (LogsModel, bool) {
	s := m.tabState(t)
	if meta.Req.ReqID != s.pending.ReqID {
		// Superseded. The newer request is still outstanding, so the tab
		// stays loading and keeps waiting for it.
		return m, false
	}
	if meta.Req.Range != m.rng || meta.Req.Query != m.query {
		s.pending = FetchLogsCmd{}
		s.loading = false
		m.setTabState(t, s)
		m.syncLoading()
		return m, false
	}
	return m, true
}

// finishPage records the outcome of a page the tab was waiting for. rows is
// how many rows the tab holds now that the caller has stored them.
func (m LogsModel) finishPage(t models.LogType, meta LogPageMeta, err error, rows int) LogsModel {
	s := m.tabState(t)
	s.err = err
	s.pending = FetchLogsCmd{}
	s.loading = false
	s.lastRefresh = time.Now()
	s.since = meta.Req.Since
	s.warning = meta.Warning
	if err == nil {
		s.errQuery = ""
		s.fetched = rows
		s.hasMore = meta.HasMore
		s.sent = meta.Sent
		// Stamped from the request rather than from the current
		// selection, so these rows are labeled with what actually
		// produced them.
		s.rng = meta.Req.Range
		s.query = meta.Req.Query
		m.accepted = meta.Req.Query
	} else {
		s.errQuery = meta.Req.Query
		// The device would not answer this expression, so it is not the
		// one to keep asking with. Reverting means a refresh re-sends the
		// last accepted query instead of retrying a refused one, and the
		// other tabs stop reading stale against something that never
		// produced a row.
		if m.query != m.accepted && meta.Req.Query == m.query {
			m.query = m.accepted
		}
	}
	m.setTabState(t, s)
	m.syncLoading()
	m.applyFilter()
	m.ensureCursorValid()
	return m
}

func (m LogsModel) ActiveLogType() models.LogType {
	return m.activeLogType
}

// fetchRequest mints one page request for a tab and records it as the fetch
// that tab is now waiting on. Every response is matched back against that
// record, so a page the view has stopped wanting is dropped rather than
// merged into rows it does not belong with.
//
// It has a pointer receiver because minting a request is a state change:
// callers must keep the model it returns through, or the tab will not know
// what it asked for.
func (m *LogsModel) fetchRequest(t models.LogType, skip int, appendRows bool) tea.Cmd {
	s := m.tabState(t)
	s.seq++
	if !appendRows {
		// A pagination fixes its lower bound when it starts and every
		// later page reuses it. Recomputing the bound per page walks the
		// window forward across a newest-first list that is also growing
		// at the head, so a skip counted from the rows already on screen
		// stops pointing past them and rows repeat.
		s.since = m.rng.Since(time.Now())
	}
	s.pending = FetchLogsCmd{
		Type:   t,
		Query:  m.query,
		Since:  s.since,
		Skip:   skip,
		Append: appendRows,
		Range:  m.rng,
		ReqID:  s.seq,
	}
	s.loading = true
	m.setTabState(t, s)
	m.syncLoading()

	req := s.pending
	return func() tea.Msg { return req }
}

// needsFetch reports whether a tab has to be asked for a first page: it holds
// nothing, or its rows predate the current range or query. A fetch for
// exactly that selection already on its way counts as covered — asking again
// would spend another couple of megabytes and another device job to receive
// the same page twice.
func (m LogsModel) needsFetch(t models.LogType) bool {
	if m.rowCount(t) > 0 && !m.tabStale(t) {
		return false
	}
	return !m.tabState(t).inflightMatches(m.rng, m.query)
}

// onTabSwitch resets the cursor and asks for the newly shown tab when it has
// no rows yet or its rows predate the current range or query.
func (m LogsModel) onTabSwitch() (LogsModel, tea.Cmd) {
	m.Cursor = 0
	m.Offset = 0
	m.Expanded = false
	// The tab on screen changed, so the banner and the spinner gate now
	// follow a different tab's in-flight state.
	m.syncLoading()

	if m.needsFetch(m.activeLogType) {
		// Without the loading state, the first visit to a tab renders its
		// empty state for however long the fetch takes — an empty table
		// asserting there are no logs when it simply has not asked yet.
		cmd := m.fetchRequest(m.activeLogType, 0, false)
		return m, cmd
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
	cmd := m.fetchRequest(m.activeLogType, 0, false)
	return m, cmd
}

// RefreshActiveTab resets the tab on screen to its first page and asks for it
// again under the current range and query. Pages loaded with m are dropped:
// after paging back through 2,000 rows, a refresh should return the newest
// page, which is what refresh means. The parent calls this instead of
// building a request of its own, so every fetch carries an identity and no
// path can issue one the view does not know it is waiting for.
func (m LogsModel) RefreshActiveTab() (LogsModel, tea.Cmd) {
	return m.onQueryChanged()
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

// IsQueryMode reports whether the device-query bar has focus. The parent
// routes every key to the input while it does, exactly as it does for the
// filter.
func (m LogsModel) IsQueryMode() bool { return m.queryMode }

// QueryValue is the in-progress text in the query bar.
func (m LogsModel) QueryValue() string { return m.queryInput.Value() }

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
	if m.queryMode {
		return m.updateQueryMode(msg)
	}
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
		case "f":
			m.queryMode = true
			// Opened with the last committed text rather than the last
			// accepted one, so an expression the device refused can be
			// corrected instead of retyped.
			m.queryInput.SetValue(m.committed)
			m.queryInput.Focus()
			m.queryInput.CursorEnd()
			return m, textinput.Blink
		case "m":
			s := m.tabState(m.activeLogType)
			if !s.hasMore || s.loading {
				// A page already on its way carries the skip this
				// keypress would reuse: fetched only moves when a page
				// lands, so a second m asks for the same rows again,
				// appends them twice, and leaves the page after that
				// starting past a gap of rows nobody ever sees.
				return m, nil
			}
			// Deliberately not automatic on reaching the last row: a
			// page is a couple of megabytes and a second or more, which
			// is not what a j keypress should cost.
			cmd := m.fetchRequest(m.activeLogType, s.fetched, true)
			return m, cmd
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

// updateQueryMode edits the device query. Enter sends it and refetches; esc
// abandons the edit and leaves the last committed expression in the bar.
func (m LogsModel) updateQueryMode(msg tea.Msg) (LogsModel, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "enter":
			m.queryMode = false
			m.queryInput.Blur()
			// query is what the view now asks for; accepted only moves
			// once the device answers it without an error, so a refused
			// expression is never stored.
			m.committed = strings.TrimSpace(m.queryInput.Value())
			m.query = m.committed
			return m.onQueryChanged()
		case "esc":
			m.queryMode = false
			m.queryInput.Blur()
			m.queryInput.SetValue(m.committed)
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.queryInput, cmd = m.queryInput.Update(msg)
	return m, cmd
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

	if warning := m.tabState(m.activeLogType).warning; warning != "" {
		sections = append(sections, WarningMsgStyle.Render(
			strings.Join(wrapText(warning, m.Width), "\n")))
	}

	// Device query bar
	if m.queryMode {
		sections = append(sections, ViewSubtitleStyle.Render(queryBarLabel)+m.queryInput.View())
	}

	// Filter bar
	if m.FilterMode {
		sections = append(sections, m.renderFilterBar())
	} else if m.IsFiltered() {
		// Wrapped rather than rendered as-is: the filter text comes from
		// the user (up to Filter.CharLimit) and an unwrapped long value
		// would widen this line past the terminal, the same defect class
		// emptyStateFor below is fixed for.
		info := fmt.Sprintf("Filter: %s (%d results)  [esc to clear]", m.FilterValue(), m.filteredCount())
		sections = append(sections, FilterInfoStyle.Render(strings.Join(wrapText(info, m.Width), "\n")))
	}

	// Error and content. A failed fetch is shown above the rows it could
	// not replace rather than instead of them: the device refusing a query
	// is no reason to take away the results the operator was reading, and
	// those rows are what they compare the message against. With nothing
	// loaded there is no table worth rendering, so the error stands alone.
	activeErr := m.activeErr()
	if activeErr != nil {
		sections = append(sections, m.renderError(activeErr))
	}
	if activeErr == nil || m.rowCount(m.activeLogType) > 0 {
		if !m.Loading || m.filteredCount() > 0 {
			sections = append(sections, m.renderTable())

			if m.Expanded && m.filteredCount() > 0 {
				sections = append(sections, m.renderDetailPanel())
			}
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
	if last := m.tabState(m.activeLogType).lastRefresh; !last.IsZero() {
		ago := time.Since(last).Truncate(time.Second)
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

// renderError shows the device's own message, and the expression that
// produced it when there was one. The message alone rarely names what was
// asked -- "syntax error at 14:50:57" says nothing about which query -- and a
// refused expression is deliberately not stored anywhere else, so this is the
// only place the operator can read the two together.
func (m LogsModel) renderError(err error) string {
	lines := wrapText(fmt.Sprintf("Error: %v", err), m.Width)
	if q := m.tabState(m.activeLogType).errQuery; q != "" {
		lines = append(lines, wrapText("Query: "+q, m.Width)...)
	}
	return ErrorMsgStyle.Bold(true).Padding(1, 0).Render(strings.Join(lines, "\n"))
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
		{"f", "device query"},
		{"m", "more"},
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

// emptyStateFor renders the message shown when a tab has no rows. With an
// expression in play it names what the device was actually asked, because a
// bound the device accepted but did not honor returns zero rows with no
// error, and a blank table reads as "nothing happened".
//
// That attribution only holds when the device is actually the reason the
// table is empty. The local `/` filter (TableBase, independent of the
// device query) can also empty a tab that the device filled: rowCount gives
// the raw, unfiltered count, so rowCount>0 with IsFiltered means these rows
// exist and it is the local filter hiding them, not the sent expression.
// Blaming the device query there would misattribute a client-side result to
// the server.
func (m LogsModel) emptyStateFor(kind string) string {
	if m.IsFiltered() && m.rowCount(m.activeLogType) > 0 {
		msg := fmt.Sprintf("No %s logs match the filter %q (%d loaded)", kind, m.FilterValue(), m.rowCount(m.activeLogType))
		return EmptyMsgStyle.Padding(1, 0).Render(strings.Join(wrapText(msg, m.Width), "\n"))
	}
	sent := m.tabState(m.activeLogType).sent
	if sent == "" {
		return EmptyMsgStyle.Padding(1, 0).Render(fmt.Sprintf("No %s logs found", kind))
	}
	return EmptyMsgStyle.Padding(1, 0).Render(
		"0 rows matched\n" + strings.Join(wrapText(sent, m.Width), "\n"))
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
