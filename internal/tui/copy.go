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

// setNotice shows a short message in the footer until noticeTimeout passes.
func (m Model) setNotice(text string, now time.Time) Model {
	m.notice = text
	m.noticeUntil = now.Add(noticeTimeout)
	return m
}

// activeNotice is the footer notice, or "" once it has expired.
func (m Model) activeNotice(now time.Time) string {
	if m.notice == "" || !now.Before(m.noticeUntil) {
		return ""
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
		return m.setNotice("nothing to copy here", now), nil
	}
	label, value, ok := s.copyValue(&m)
	if !ok || value == "" {
		return m.setNotice("nothing selected to copy", now), nil
	}
	text := "copied " + label
	if len(value) <= noticeValueMax {
		text += ": " + value
	}
	return m.setNotice(text, now), tea.SetClipboard(value)
}
