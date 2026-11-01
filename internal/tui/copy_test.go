package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
)

// clipboardOf runs cmd and returns what it would put on the clipboard. Bubble
// Tea's message type for it is unexported but is a string type holding
// exactly the copied value, so its type name and printed form are checked.
func clipboardOf(t *testing.T, cmd tea.Cmd) string {
	t.Helper()
	if cmd == nil {
		t.Fatal("copy returned no command")
	}
	msg := cmd()
	if typ := fmt.Sprintf("%T", msg); typ != "tea.setClipboardMsg" {
		t.Fatalf("copy produced %s, want a clipboard write", typ)
	}
	return fmt.Sprint(msg)
}

// noticeOf is the longest form of the footer notice, or "" when none is up.
func noticeOf(m Model, now time.Time) string {
	if forms := m.activeNotice(now); len(forms) > 0 {
		return forms[0]
	}
	return ""
}

func TestCopy_SessionID(t *testing.T) {
	m := connectedSessionsModel(t, 3)
	m = pressKey(t, m, "j") // second row, ID 2

	now := time.Now()
	next, cmd := m.copySelected(now)
	if got := clipboardOf(t, cmd); got != "2" {
		t.Errorf("clipboard = %q, want the selected session's ID %q", got, "2")
	}
	if n := noticeOf(next, now); n != "copied session ID: 2" {
		t.Errorf("notice = %q", n)
	}
	if n := noticeOf(next, now.Add(noticeTimeout)); n != "" {
		t.Errorf("notice still up after %v: %q", noticeTimeout, n)
	}
}

func TestCopy_PolicyName(t *testing.T) {
	m := newTestModel(t, ViewPolicies)
	m.policies = m.policies.SetPolicies([]models.SecurityRule{{Name: "allow-web"}, {Name: "deny-all"}}, nil)
	_, cmd := m.copySelected(time.Now())
	if got := clipboardOf(t, cmd); got != "allow-web" {
		t.Errorf("clipboard = %q, want %q", got, "allow-web")
	}
}

func TestCopy_LogLineIsPlainText(t *testing.T) {
	m := newTestModel(t, ViewLogs)
	line, ok := m.logs.SelectedLine()
	if ok || line != "" {
		t.Fatalf("empty logs view selected %q", line)
	}
	next, cmd := m.copySelected(time.Now())
	if cmd != nil {
		t.Error("copied with no row selected")
	}
	if n := noticeOf(next, time.Now()); n != "nothing selected to copy" {
		t.Errorf("notice = %q", n)
	}
}

func TestCopy_NothingToCopyOutsideTables(t *testing.T) {
	m := newTestModel(t, ViewDashboard)
	next, cmd := m.copySelected(time.Now())
	if cmd != nil {
		t.Error("copy on a dashboard issued a clipboard write")
	}
	if n := noticeOf(next, time.Now()); n != "nothing to copy here" {
		t.Errorf("notice = %q", n)
	}
}

// TestCopy_LongValueIsNotEchoed keeps a log line from flooding the footer:
// the notice names what was copied, the clipboard gets the whole value.
func TestCopy_LongValueIsNotEchoed(t *testing.T) {
	m := newTestModel(t, ViewPolicies)
	long := strings.Repeat("x", noticeValueMax+1)
	m.policies = m.policies.SetPolicies([]models.SecurityRule{{Name: long}}, nil)
	next, cmd := m.copySelected(time.Now())
	if got := clipboardOf(t, cmd); got != long {
		t.Errorf("clipboard truncated: %d chars", len(got))
	}
	if n := noticeOf(next, time.Now()); n != "copied rule name" {
		t.Errorf("notice = %q, want the label alone", n)
	}
}

func TestCopy_KeyAndPalette(t *testing.T) {
	m := connectedSessionsModel(t, 3)
	if _, cmd := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"}); cmd == nil {
		t.Error("y did not copy")
	}
	if _, cmd := m.Update(copySelectedMsg{}); cmd == nil {
		t.Error("palette copy did not copy")
	}
}

func TestViewSlots_EveryTableCopies(t *testing.T) {
	for _, s := range viewSlots() {
		if s.isLoading != nil && s.copyValue == nil {
			t.Errorf("view %d: table view has no copyValue", s.refreshFor)
		}
	}
}
