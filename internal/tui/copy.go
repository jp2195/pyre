package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// noticeTimeout is how long a footer notice ("copied ...") stays up. The app
// clock re-renders every second, so it disappears without a dismiss message.
const noticeTimeout = 3 * time.Second

// noticeValueMax caps how much of a copied value the notice echoes back. A
// log line is far longer than the footer; the label alone says what went.
const noticeValueMax = 40

// copySelectedMsg copies the selected row, from the command palette.
type copySelectedMsg struct{}

// setNotice shows a message in the footer until noticeTimeout passes. Give
// it longest form first: the footer shows the first one that fits the
// terminal, so a narrow screen still gets the short form instead of nothing.
func (m Model) setNotice(now time.Time, forms ...string) Model {
	m.notice = forms
	m.noticeUntil = now.Add(noticeTimeout)
	return m
}

// activeNotice is the footer notice's forms, or nil once it has expired.
func (m Model) activeNotice(now time.Time) []string {
	if len(m.notice) == 0 || !now.Before(m.noticeUntil) {
		return nil
	}
	return m.notice
}

// copySelected puts the selected row's identifier on the clipboard through
// OSC 52 and says what it copied. OSC 52 is write-only from pyre's side: the
// terminal never confirms, so a terminal that ignores it (or tmux without
// set-clipboard) shows the notice and copies nothing. The docs say so.
func (m Model) copySelected(now time.Time) (Model, tea.Cmd) {
	s, ok := m.currentSlot()
	if !ok || s.copyValue == nil {
		return m.setNotice(now, "nothing to copy here"), nil
	}
	label, value, ok := s.copyValue(&m)
	if !ok || value == "" {
		return m.setNotice(now, "nothing selected to copy"), nil
	}
	short := "copied " + label
	if len(value) > noticeValueMax {
		return m.setNotice(now, short), tea.SetClipboard(value)
	}
	return m.setNotice(now, short+": "+value, short), tea.SetClipboard(value)
}
