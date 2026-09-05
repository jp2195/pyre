package tui

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/tui/views"
)

// ansiEscape matches the styling lipgloss emits, so a rendered line can be
// read as the text a person sees.
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// fillLogin drives the login screen the way a person does: type the host,
// tab, type the username, tab, type the password. Everything goes through
// Model.Update so the real key handler runs.
func fillLogin(t *testing.T, host, username, password string) Model {
	t.Helper()
	m := newTestModel(t, ViewLogin)
	m.width, m.height = 100, 40
	m.login = m.login.SetSize(100, 40)

	send := func(mod Model, msg tea.Msg) Model {
		next, _ := mod.Update(msg)
		return next.(Model)
	}
	typeString := func(mod Model, s string) Model {
		for _, r := range s {
			mod = send(mod, tea.KeyPressMsg{Code: r, Text: string(r)})
		}
		return mod
	}

	m = typeString(m, host)
	m = send(m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = typeString(m, username)
	m = send(m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = typeString(m, password)
	return m
}

func press(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(msg)
	return next.(Model), cmd
}

// TestLoginFlow_EnterStartsTheLogin checks the happy path reaches the point
// of talking to the device.
func TestLoginFlow_EnterStartsTheLogin(t *testing.T) {
	m := fillLogin(t, "10.0.104.50", "testuser", "test-password")
	if !m.login.CanSubmit() {
		t.Fatalf("precondition: form cannot submit with host=%q user=%q",
			m.login.Host(), m.login.Username())
	}

	after, cmd := press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter on a complete form returned no command")
	}
	if !after.loading {
		t.Error("enter did not put the model into its loading state")
	}
}

// TestLoginFlow_SubmittingSaysSomethingIsHappening checks the screen reports
// that a login is under way.
//
// Reaching a firewall takes as long as the network and the device take, and
// can sit until the request times out. The login screen renders on its own,
// without the footer that carries the spinner everywhere else, so pressing
// enter changed nothing on screen at all: no spinner, no message, no
// disabled field. The only way to tell it was working was that typing
// stopped having an effect.
func TestLoginFlow_SubmittingSaysSomethingIsHappening(t *testing.T) {
	m := fillLogin(t, "10.0.104.50", "testuser", "test-password")
	before := m.renderContent()

	after, _ := press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	during := after.renderContent()

	if during == before {
		t.Errorf("the screen is unchanged while a login is in flight:\n%s", during)
	}
}

// TestLoginFlow_IncompleteFormSaysWhy checks that enter on a form that cannot
// be submitted explains itself.
//
// The handler submits only when CanSubmit reports true and otherwise passes
// the key to the focused text input, which ignores it. Pressing enter with a
// field still empty, or with a pasted URL in the host field, did nothing
// whatsoever and said nothing about why.
func TestLoginFlow_IncompleteFormSaysWhy(t *testing.T) {
	cases := map[string]Model{
		"no password":  fillLogin(t, "10.0.104.50", "testuser", ""),
		"no username":  fillLogin(t, "10.0.104.50", "", ""),
		"no host":      fillLogin(t, "", "testuser", "test-password"),
		"pasted a URL": fillLogin(t, "https://10.0.104.50", "testuser", "test-password"),
	}

	for name, m := range cases {
		before := m.renderContent()
		after, cmd := press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})

		if after.loading || cmd != nil {
			t.Errorf("%s: enter started a login anyway", name)
			continue
		}
		if after.renderContent() == before {
			t.Errorf("%s: enter changed nothing on screen, so the operator is\n"+
				"given no reason the form did not submit", name)
		}
	}
}

// TestLoginFlow_SuccessLandsOnTheDashboardWithoutThePassword covers the whole
// point of the screen, and the credential-lifetime promise in CLAUDE.md: the
// password must not survive in the widget once it has been exchanged.
func TestLoginFlow_SuccessLandsOnTheDashboardWithoutThePassword(t *testing.T) {
	m := fillLogin(t, "10.0.104.50", "testuser", "test-password")
	m, _ = press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = press(t, m, LoginSuccessMsg{
		Host: "10.0.104.50", APIKey: "an-api-key", Username: "testuser",
	})

	if m.currentView != ViewDashboard {
		t.Errorf("after a successful login the view is %v, want the dashboard", m.currentView)
	}
	if m.loading {
		t.Error("still loading after a successful login")
	}
	if got := m.login.Password(); got != "" {
		t.Errorf("password survives in the form as %q", got)
	}
	if strings.Contains(m.login.View(), "test-password") {
		t.Error("password is still rendered by the login form")
	}
	conn := m.session.GetActiveConnection()
	if conn == nil {
		t.Fatal("no active connection after a successful login")
	}
	if conn.Host != "10.0.104.50" {
		t.Errorf("active connection is %q, want 10.0.104.50", conn.Host)
	}
}

// TestLoginFlow_FailureKeepsYouOnTheFormAndShowsTheReason checks a rejected
// login leaves the operator somewhere they can fix it.
func TestLoginFlow_FailureKeepsYouOnTheFormAndShowsTheReason(t *testing.T) {
	m := fillLogin(t, "10.0.104.50", "testuser", "wrong")
	m, _ = press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = press(t, m, LoginErrorMsg{Err: errors.New("Invalid credentials")})

	if m.currentView != ViewLogin {
		t.Errorf("a failed login moved to view %v, want to stay on the form", m.currentView)
	}
	if m.loading {
		t.Error("still loading after a failed login")
	}
	if !strings.Contains(m.login.View(), "Invalid credentials") {
		t.Errorf("the failure is not shown on the form:\n%s", m.login.View())
	}
	if got := m.login.Host(); got != "10.0.104.50" {
		t.Errorf("host was cleared to %q, so the operator must retype it", got)
	}
}

// TestLoginFlow_RetryClearsTheStaleFailure checks the previous failure stops
// being shown once another attempt is under way, so the screen does not
// report a stale error while it is busy succeeding.
func TestLoginFlow_RetryClearsTheStaleFailure(t *testing.T) {
	m := fillLogin(t, "10.0.104.50", "testuser", "wrong")
	m, _ = press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = press(t, m, LoginErrorMsg{Err: errors.New("Invalid credentials")})

	// Correct the password and try again.
	for range len("wrong") {
		m, _ = press(t, m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	}
	for _, r := range "test-password" {
		m, _ = press(t, m, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	m, _ = press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})

	if strings.Contains(m.login.View(), "Invalid credentials") {
		t.Errorf("the previous failure is still on screen during the retry:\n%s", m.login.View())
	}
}

// TestLoginFlow_SpaceTogglesOnlyTheCheckbox checks the space key reaches the
// text inputs, since a space is legal in neither a host nor a username but
// is perfectly legal in a password.
func TestLoginFlow_SpaceTogglesOnlyTheCheckbox(t *testing.T) {
	m := fillLogin(t, "10.0.104.50", "testuser", "two")
	m, _ = press(t, m, tea.KeyPressMsg{Code: ' ', Text: " "})
	m, _ = press(t, m, tea.KeyPressMsg{Code: 'w', Text: "w"})

	if got := m.login.Password(); got != "two w" {
		t.Errorf("password = %q, want %q: space must be typable into a password", got, "two w")
	}
	if m.login.Insecure() {
		t.Error("typing a space into the password toggled the insecure checkbox")
	}

	onCheckbox, _ := press(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	toggled, _ := press(t, onCheckbox, tea.KeyPressMsg{Code: ' ', Text: " "})
	if !toggled.login.Insecure() {
		t.Error("space on the checkbox did not toggle it")
	}
	if got := toggled.login.Password(); got != "two w" {
		t.Errorf("toggling the checkbox changed the password to %q", got)
	}
}

// TestLoginFlow_EscapeLeavesNoCredentialsBehind checks backing out of the
// screen drops what was typed.
func TestLoginFlow_EscapeLeavesNoCredentialsBehind(t *testing.T) {
	m := fillLogin(t, "10.0.104.50", "testuser", "test-password")
	m, _ = press(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})

	if m.currentView != ViewConnectionHub {
		t.Errorf("escape went to view %v, want the connection hub", m.currentView)
	}
	if got := m.login.Password(); got != "" {
		t.Errorf("password %q survived leaving the screen", got)
	}
	if got := m.login.Host(); got != "" {
		t.Errorf("host %q survived leaving the screen", got)
	}
}

// TestLoginFlow_FitsTheTerminal checks the screen an operator sees first
// stays inside the terminal.
func TestLoginFlow_FitsTheTerminal(t *testing.T) {
	for _, width := range []int{60, 80, 120} {
		m := newTestModel(t, ViewLogin)
		u, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: 30})
		m = u.(Model)
		m.currentView = ViewLogin
		m, _ = press(t, m, LoginErrorMsg{Err: errors.New("dial tcp 10.0.104.50:443: connect: connection refused")})

		for _, line := range strings.Split(m.renderContent(), "\n") {
			if got := lipgloss.Width(line); got > width {
				t.Errorf("width %d: a line is %d cells, %d over:\n  %q", width, got, got-width, line)
				break
			}
		}
	}
}

// TestLoginFlow_EscapeDuringLoginCancelsIt checks that backing out while a
// login is in flight actually abandons it.
//
// The request keeps running after escape, and its reply used to be acted on
// regardless: the connection was added, made active, and the view jumped to
// its dashboard, seconds after the operator had left the screen. Escape has
// to mean cancelled.
func TestLoginFlow_EscapeDuringLoginCancelsIt(t *testing.T) {
	m := fillLogin(t, "10.0.104.50", "testuser", "test-password")
	m, _ = press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !m.loading {
		t.Fatal("precondition: the login did not start")
	}

	m, _ = press(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.loading {
		t.Error("the model is still loading after escaping the login screen")
	}

	// The request was already in flight, so its reply still arrives.
	m, _ = press(t, m, LoginSuccessMsg{
		Host: "10.0.104.50", APIKey: "an-api-key", Username: "testuser",
	})

	if m.currentView != ViewConnectionHub {
		t.Errorf("an abandoned login moved the view to %v", m.currentView)
	}
	if conn := m.session.GetActiveConnection(); conn != nil {
		t.Errorf("an abandoned login connected to %q anyway", conn.Host)
	}
}

// TestLoginFlow_EnterTwiceStartsOneLogin checks a second enter while the
// first attempt is still running does not fire another key generation
// request at the device.
func TestLoginFlow_EnterTwiceStartsOneLogin(t *testing.T) {
	m := fillLogin(t, "10.0.104.50", "testuser", "test-password")

	m, first := press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if first == nil {
		t.Fatal("precondition: the first enter started nothing")
	}
	if _, second := press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter}); second != nil {
		t.Error("a second enter started another login while the first was in flight")
	}
}

// TestLoginFlow_LongErrorWrapsWithoutGaps checks a wrapped error reads as one
// block. The error style carries a top margin, so rendering the wrapped
// lines one at a time put a blank line between every one of them.
func TestLoginFlow_LongErrorWrapsWithoutGaps(t *testing.T) {
	m := fillLogin(t, "10.0.104.50", "testuser", "test-password")
	u, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 40})
	m = u.(Model)
	m, _ = press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = press(t, m, LoginErrorMsg{
		Err: errors.New("dial tcp 10.0.104.50:443: connect: connection refused"),
	})

	lines := strings.Split(stripANSI(m.login.View()), "\n")
	for i, line := range lines {
		if !strings.Contains(line, "Error: dial tcp") {
			continue
		}
		if i+1 >= len(lines) || strings.TrimSpace(strings.ReplaceAll(lines[i+1], "│", "")) == "" {
			t.Errorf("the wrapped error has a blank line inside it:\n%s", strings.Join(lines[i:i+3], "\n"))
		}
		return
	}
	t.Error("the error was not rendered at all")
}

// TestConnectionForm_SpaceTogglesItsCheckboxes is the sibling of the login
// check. Bubble Tea v2 stringifies the space bar as "space", so a handler
// comparing against a literal " " never fires; this form binds both
// spellings and must keep doing so.
func TestConnectionForm_SpaceTogglesItsCheckboxes(t *testing.T) {
	m := newTestModel(t, ViewConnectionForm)
	m.connectionForm = views.NewAddConnectionForm()

	// Tab past host and username to the type selector.
	for range 2 {
		next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		m = next.(Model)
	}
	if got := m.connectionForm.FocusedField(); got != views.FormFieldType {
		t.Fatalf("precondition: focus is %d, want the type field", got)
	}

	next, _ := m.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	m = next.(Model)
	if got := m.connectionForm.Type(); got != "panorama" {
		t.Errorf("space on the type selector left it as %q", got)
	}
}

func stripANSI(s string) string { return ansiEscape.ReplaceAllString(s, "") }
