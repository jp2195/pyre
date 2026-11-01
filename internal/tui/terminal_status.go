package tui

import tea "charm.land/bubbletea/v2"

// statusApp is the machine-readable program name sent with every Program
// Status report. It must match [A-Za-z0-9_.+-]{1,32}.
const statusApp = "pyre"

// terminalProgress reports pyre's state through the terminal's taskbar or tab
// progress indicator (OSC 9;4), so a long fetch or a failure is visible while
// the operator is in another tab. Terminals without support ignore it.
//
// An error wins over loading: the error is the thing the operator needs to
// come back for, and it clears itself after errorDismissTimeout. Bubble Tea
// resets the indicator when the program exits, so no quit path has to.
func (m Model) terminalProgress() *tea.ProgressBar {
	switch {
	case m.err != nil:
		return tea.NewProgressBar(tea.ProgressBarError, 100)
	case m.anyLoading():
		return tea.NewProgressBar(tea.ProgressBarIndeterminate, 0)
	}
	return nil
}

// programStatus reports pyre's state through the Program Status Protocol
// (OSC 7501). Terminals without support ignore it.
//
// It deliberately reports only working and blocked. Bubble Tea leaves the
// last status in place when the program exits so that done and error reports
// outlive it, while terminals drop working and blocked records themselves.
// Reporting an error here would therefore leave a stale failure in the tab
// after a normal quit, unless every quit path cleared it first. Errors go to
// terminalProgress instead, which the renderer does reset on exit.
//
// The text is generic on purpose: no hostname, serial, or error detail. It
// leaves the TUI for the terminal's own UI, where pyre controls neither where
// it is shown nor how long it is kept.
func (m Model) programStatus() *tea.ProgramStatus {
	switch {
	case m.anyLoading():
		return &tea.ProgramStatus{
			State:   tea.ProgramStateWorking,
			App:     statusApp,
			Message: "Fetching from the firewall",
		}
	case m.currentView == ViewLogin:
		return &tea.ProgramStatus{
			State:   tea.ProgramStateBlocked,
			App:     statusApp,
			Kind:    tea.ProgramStatusKindAuth,
			Message: "Waiting for credentials",
		}
	}
	return nil
}
