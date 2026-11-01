package tui

import (
	"fmt"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/config"
	"github.com/jp2195/pyre/internal/tui/views"
)

// clockInterval is how often the app clock ticks. It drives two things: the
// auto-refresh schedule, and re-rendering so "updated 12s ago" keeps counting
// instead of freezing at whatever it said when the last message arrived.
const clockInterval = time.Second

// clockTickMsg is the app clock. Init starts the one chain and only its
// handler schedules the next tick, so there is never more than one.
type clockTickMsg struct{ now time.Time }

func clockTick() tea.Cmd {
	return tea.Tick(clockInterval, func(t time.Time) tea.Msg { return clockTickMsg{now: t} })
}

// toggleAutoRefreshMsg turns auto-refresh on or off, from the command palette.
type toggleAutoRefreshMsg struct{}

// viewKey names one screen's dataset: a view, plus which dashboard when the
// view is ViewDashboard.
type viewKey struct {
	view      ViewState
	dashboard views.DashboardType
}

func (m Model) currentKey() viewKey {
	k := viewKey{view: m.currentView}
	if m.currentView == ViewDashboard {
		k.dashboard = m.currentDashboard
	}
	return k
}

// autoRefreshState is the auto-refresh schedule. It is session-only: R and
// the palette pause and resume it without touching the config file.
type autoRefreshState struct {
	enabled  bool
	interval time.Duration
	// nextDue is when the next refresh may fire. A refresh that is due but
	// held fires on the first tick after the hold lifts.
	nextDue time.Time
	// pending identifies the tracked refresh in flight; 0 means none.
	// Only one is in flight at a time, and a result carrying any other
	// token is dropped, so a superseded refresh cannot land on top of the
	// one that replaced it.
	pending int
	seq     int
}

// newAutoRefreshState reads the configured interval. Problems with the
// setting were already reported at load time, so here they just mean off.
func newAutoRefreshState(cfg *config.Config) autoRefreshState {
	d, _, err := cfg.Settings.RefreshInterval()
	if err != nil || d == 0 {
		return autoRefreshState{interval: config.DefaultRefreshInterval}
	}
	return autoRefreshState{enabled: true, interval: d}
}

// autoRefreshes reports whether the screen shows live device state that
// auto-refresh should keep current. Configuration (policies, NAT, objects,
// the config dashboard) is left alone: it changes when someone commits, not
// every few seconds, and the rulebase is one of the largest fetches pyre
// makes, so polling it would load the management plane for nothing.
func (k viewKey) autoRefreshes() bool {
	switch k.view {
	case ViewDashboard:
		return k.dashboard != views.DashboardConfig
	case ViewSessions, ViewInterfaces, ViewRoutes, ViewIPSecTunnels, ViewGPUsers, ViewLogs:
		return true
	}
	return false
}

// autoRefreshHold returns why a due refresh must wait for the operator, or
// "" when nothing they are doing is in the way. A refresh replaces the rows
// on screen, so it waits while someone is typing a filter, reading a detail
// panel, or (in logs) looking at anything but the newest rows.
func (m Model) autoRefreshHold() string {
	if m.currentViewFiltering() {
		return "filtering"
	}
	if m.currentView == ViewLogs && !m.logs.Following() {
		return "not at newest"
	}
	if s, ok := m.currentSlot(); ok && s.isExpanded != nil && s.isExpanded(&m) {
		return "detail open"
	}
	return ""
}

// handleClockTick fires a due auto-refresh when nothing stands in its way.
func (m Model) handleClockTick(now time.Time) (Model, tea.Cmd) {
	a := m.autoRefresh
	if !a.enabled || now.Before(a.nextDue) {
		return m, nil
	}
	if m.session.GetActiveConnection() == nil || !m.currentKey().autoRefreshes() {
		return m, nil
	}
	// A fetch already in flight, manual or automatic, is the refresh. A
	// second one on top would only make the device answer twice.
	if a.pending != 0 || m.anyLoading() {
		return m, nil
	}
	if m.autoRefreshHold() != "" {
		return m, nil
	}
	return m.startTrackedRefresh(now)
}

// startTrackedRefresh re-fetches the screen and reports back once every
// response has arrived. It does not set any view's loading state: an
// automatic refresh keeps the current rows on screen until the new ones
// replace them, where a loading flag would blank the table every interval.
func (m Model) startTrackedRefresh(now time.Time) (Model, tea.Cmd) {
	m.autoRefresh.nextDue = now.Add(m.autoRefresh.interval)
	key := m.currentKey()
	// Hoisted for the same reason as in handleRefresh: refreshCurrentView
	// mutates m (the logs view records the request it mints).
	cmd := m.refreshCurrentView()
	if cmd == nil {
		return m, nil
	}
	m.autoRefresh.seq++
	m.autoRefresh.pending = m.autoRefresh.seq
	return m, trackRefresh(m.autoRefresh.seq, m.connEpoch, key, cmd)
}

// refreshDoneMsg carries every message a tracked refresh produced, in the
// order its commands finished.
type refreshDoneMsg struct {
	token int
	epoch int
	key   viewKey
	msgs  []tea.Msg
}

// trackRefresh runs cmd, and every command nested in it through tea.Batch,
// and delivers all their messages together. Replaying them as one message is
// what lets the app know a refresh is finished, so it can stamp the time and
// let the next one fire: a Batch on its own never says when it is done.
func trackRefresh(token, epoch int, key viewKey, cmd tea.Cmd) tea.Cmd {
	return func() tea.Msg {
		return refreshDoneMsg{token: token, epoch: epoch, key: key, msgs: runCmdTree(cmd)}
	}
}

func runCmdTree(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		if msg == nil {
			return nil
		}
		return []tea.Msg{msg}
	}
	results := make([][]tea.Msg, len(batch))
	var wg sync.WaitGroup
	for i, c := range batch {
		wg.Go(func() { results[i] = runCmdTree(c) })
	}
	wg.Wait()
	var out []tea.Msg
	for _, r := range results {
		out = append(out, r...)
	}
	return out
}

// handleRefreshDone applies a finished tracked refresh. A result for another
// connection, or one that a newer refresh superseded, is dropped whole.
func (m Model) handleRefreshDone(msg refreshDoneMsg) (tea.Model, tea.Cmd) {
	if msg.epoch != m.connEpoch || msg.token != m.autoRefresh.pending {
		return m, nil
	}
	m.autoRefresh.pending = 0
	cmds := make([]tea.Cmd, 0, len(msg.msgs))
	for _, inner := range msg.msgs {
		next, cmd := m.Update(inner)
		if nm, ok := next.(Model); ok {
			m = nm
		}
		cmds = append(cmds, cmd)
	}
	if target, ok := targetFor(msg.key); ok && target.hasData(&m) {
		m.updatedAt[msg.key] = time.Now()
	}
	return m, tea.Batch(cmds...)
}

// toggleAutoRefresh pauses or resumes auto-refresh for this session.
func (m Model) toggleAutoRefresh() Model {
	m.autoRefresh.enabled = !m.autoRefresh.enabled
	if m.autoRefresh.enabled {
		m.autoRefresh.nextDue = time.Now().Add(m.autoRefresh.interval)
	}
	return m
}

// stampLoaded records when each screen's data first appeared, and forgets
// screens whose data is gone. It runs after every Update, so the stamp lands
// in the same Update as the response that filled the screen. Logs is skipped:
// its tab bar already reports each tab's own age.
func (m *Model) stampLoaded(now time.Time) {
	for _, t := range navTargets {
		if t.view == ViewLogs || t.hasData == nil {
			continue
		}
		k := viewKey{view: t.view}
		if t.view == ViewDashboard {
			k.dashboard = t.dashboard
		}
		has := t.hasData(m)
		_, stamped := m.updatedAt[k]
		switch {
		case !has && stamped:
			delete(m.updatedAt, k)
		case has && !stamped:
			m.updatedAt[k] = now
		}
	}
}

// refreshStatusParts are the footer's refresh facts for the screen: how old
// its data is, and what auto-refresh is doing about it.
func (m Model) refreshStatusParts(now time.Time) (age, auto string) {
	if at, ok := m.updatedAt[m.currentKey()]; ok {
		age = "updated " + formatAge(now.Sub(at)) + " ago"
	}
	if m.autoRefresh.enabled && m.currentKey().autoRefreshes() {
		if hold := m.autoRefreshHold(); hold != "" {
			auto = "auto held (" + hold + ")"
		} else {
			auto = "auto " + formatInterval(m.autoRefresh.interval)
		}
	}
	return age, auto
}

// renderRefreshStatus is the footer's "updated 12s ago · auto 30s" segment,
// or "" when there is nothing to say.
func (m Model) renderRefreshStatus(now time.Time) string {
	forms := m.refreshStatusForms(now)
	if len(forms) == 0 {
		return ""
	}
	return forms[0]
}

// refreshStatusForms lists the status from longest to shortest, so the
// footer can take the first that fits. The age goes last because the auto
// state is the one that explains why the age keeps growing.
func (m Model) refreshStatusForms(now time.Time) []string {
	age, auto := m.refreshStatusParts(now)
	switch {
	case age != "" && auto != "":
		return []string{age + " · " + auto, auto}
	case age != "":
		return []string{age}
	case auto != "":
		return []string{auto}
	}
	return nil
}

// formatAge renders a duration at the coarsest unit that still reads
// precisely enough: "12s", "4m", "2h".
func formatAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(max(d, 0).Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
}

// formatInterval renders a configured interval exactly: "30s", "2m", "90s".
func formatInterval(d time.Duration) string {
	if d >= time.Minute && d%time.Minute == 0 {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}
