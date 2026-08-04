package views

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/auth"
)

// LoginField represents which input field is currently focused.
type LoginField int

const (
	FieldHost LoginField = iota
	FieldUsername
	FieldPassword
	FieldInsecure
)

type LoginModel struct {
	hostInput     textinput.Model
	usernameInput textinput.Model
	passwordInput textinput.Model
	focusedField  LoginField
	err           error
	width         int
	height        int
	insecure      bool
	// submitting is true while a keygen request is in flight. It suppresses
	// repeat submissions: PAN-OS counts every keygen as a login attempt, so
	// an impatient user pressing Enter during an MFA push would otherwise
	// burn through the failed-attempt budget and lock the account out.
	submitting bool
	// spinner is the frame rendered next to the in-flight message, supplied
	// by the parent model so it animates with the app-wide spinner tick.
	spinner string
}

func NewLoginModel(creds *auth.Credentials) LoginModel {
	host := textinput.New()
	host.Placeholder = "firewall.example.com"
	host.CharLimit = 255
	host.SetWidth(40)
	if creds.Host != "" {
		host.SetValue(creds.Host)
	}

	username := textinput.New()
	username.Placeholder = "admin"
	username.CharLimit = 64
	username.SetWidth(40)
	if creds.Username != "" {
		username.SetValue(creds.Username)
	}

	password := textinput.New()
	password.Placeholder = "password"
	password.CharLimit = 128
	password.SetWidth(40)
	password.EchoMode = textinput.EchoPassword
	password.EchoCharacter = '•'

	m := LoginModel{
		hostInput:     host,
		usernameInput: username,
		passwordInput: password,
		focusedField:  FieldHost,
		insecure:      creds.Insecure,
	}

	// Set initial focus based on what's already filled
	if creds.Host != "" && creds.Username != "" {
		// Both host and username provided, focus on password
		m.focusedField = FieldPassword
	} else if creds.Host != "" {
		// Only host provided, focus on username
		m.focusedField = FieldUsername
	}

	m.updateFocus()
	return m
}

func (m *LoginModel) updateFocus() {
	m.hostInput.Blur()
	m.usernameInput.Blur()
	m.passwordInput.Blur()

	switch m.focusedField {
	case FieldHost:
		m.hostInput.Focus()
	case FieldUsername:
		m.usernameInput.Focus()
	case FieldPassword:
		m.passwordInput.Focus()
		// FieldInsecure doesn't need focus - it's a checkbox
	}
}

func (m LoginModel) SetSize(width, height int) LoginModel {
	m.width = width
	m.height = height
	return m
}

func (m LoginModel) SetError(err error) LoginModel {
	m.err = err
	// A failed attempt ends the in-flight request; re-arm the form so the
	// user can correct their credentials and retry.
	m.submitting = false
	return m
}

// SetSubmitting marks the form as having a keygen request in flight.
func (m LoginModel) SetSubmitting(v bool) LoginModel {
	m.submitting = v
	if v {
		// Clear any stale error so the previous failure isn't shown
		// alongside the "authenticating" message.
		m.err = nil
	}
	return m
}

// Submitting reports whether a keygen request is currently in flight.
func (m LoginModel) Submitting() bool {
	return m.submitting
}

// SetSpinner supplies the current spinner frame for the in-flight message.
func (m LoginModel) SetSpinner(frame string) LoginModel {
	m.spinner = frame
	return m
}

func (m LoginModel) NextField() LoginModel {
	m.focusedField = (m.focusedField + 1) % 4
	m.updateFocus()
	return m
}

// PrevField moves focus to the previous input field (for Shift+Tab).
func (m LoginModel) PrevField() LoginModel {
	m.focusedField = (m.focusedField + 3) % 4 // +3 is equivalent to -1 mod 4
	m.updateFocus()
	return m
}

// ToggleInsecure toggles the insecure checkbox value.
func (m LoginModel) ToggleInsecure() LoginModel {
	m.insecure = !m.insecure
	return m
}

// FocusedField returns the currently focused field.
func (m LoginModel) FocusedField() LoginField {
	return m.focusedField
}

func (m LoginModel) Host() string {
	return strings.TrimSpace(m.hostInput.Value())
}

func (m LoginModel) Username() string {
	return strings.TrimSpace(m.usernameInput.Value())
}

func (m LoginModel) Password() string {
	return m.passwordInput.Value()
}

func (m LoginModel) Insecure() bool {
	return m.insecure
}

// ClearPassword clears the password from memory after successful login.
// This is a security measure to minimize the time credentials are in memory.
func (m LoginModel) ClearPassword() LoginModel {
	m.passwordInput.SetValue("")
	return m
}

func (m LoginModel) CanSubmit() bool {
	return m.Host() != "" && validateHost(m.Host()) == "" && m.Username() != "" && m.Password() != ""
}

func (m LoginModel) Update(msg tea.Msg) (LoginModel, tea.Cmd) {
	var cmd tea.Cmd

	switch m.focusedField {
	case FieldHost:
		m.hostInput, cmd = m.hostInput.Update(msg)
	case FieldUsername:
		m.usernameInput, cmd = m.usernameInput.Update(msg)
	case FieldPassword:
		m.passwordInput, cmd = m.passwordInput.Update(msg)
		// FieldInsecure doesn't need text input updates
	}

	return m, cmd
}

// loginContentWidth is the login form's content width, matching the widest
// idle line ("Tab: next  Space: toggle  Enter: connect  Ctrl+C: quit"). The
// status region is pinned to it so no state can widen the box.
const loginContentWidth = 54

// loginStatusRows is the height reserved for the status region — enough for
// the three-line in-flight message, so shorter states pad rather than shrink.
const loginStatusRows = 3

// loginStatusStyle is the fixed-size region holding the help text, the
// in-flight message, or an error. Constant width and height keep the centered
// box from moving when the state changes.
func loginStatusStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Width(loginContentWidth).
		Height(loginStatusRows).
		MarginTop(1)
}

func (m LoginModel) View() string {
	titleStyle := ViewTitleStyle.MarginBottom(2)
	labelStyle := DetailLabelStyle.MarginBottom(1)

	inputStyle := InputStyle
	focusedInputStyle := InputFocusedStyle

	errorStyle := ErrorMsgStyle.Bold(true).MarginTop(1)
	helpStyle := HelpDescStyle.MarginTop(2)

	var b strings.Builder

	b.WriteString(titleStyle.Render("pyre - Palo Alto Firewall TUI"))
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("Firewall Host"))
	b.WriteString("\n")
	if m.focusedField == FieldHost {
		b.WriteString(focusedInputStyle.Render(m.hostInput.View()))
	} else {
		b.WriteString(inputStyle.Render(m.hostInput.View()))
	}
	if hostErr := validateHost(m.Host()); hostErr != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(hostErr))
	}
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Username"))
	b.WriteString("\n")
	if m.focusedField == FieldUsername {
		b.WriteString(focusedInputStyle.Render(m.usernameInput.View()))
	} else {
		b.WriteString(inputStyle.Render(m.usernameInput.View()))
	}
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Password"))
	b.WriteString("\n")
	if m.focusedField == FieldPassword {
		b.WriteString(focusedInputStyle.Render(m.passwordInput.View()))
	} else {
		b.WriteString(inputStyle.Render(m.passwordInput.View()))
	}
	b.WriteString("\n")

	// Insecure checkbox
	checkboxChar := "[ ]"
	if m.insecure {
		checkboxChar = "[x]"
	}
	checkboxLabel := checkboxChar + " Skip TLS verification (insecure)"
	if m.focusedField == FieldInsecure {
		b.WriteString(focusedInputStyle.Render(checkboxLabel))
	} else {
		b.WriteString(inputStyle.Render(checkboxLabel))
	}

	// Status region: idle help, in-flight progress, and errors all render
	// here at a fixed width and height.
	//
	// These used to be written inline at their natural size. The box is
	// centered with lipgloss.Place, so the in-flight text — the widest line
	// in the form — grew the box and shoved it left and up the instant Enter
	// was pressed. The form appeared to jump precisely when the user was
	// looking for feedback, and the spinner landed somewhere other than where
	// they were watching. A constant-size region keeps the box still.
	var status string
	switch {
	case m.submitting:
		spin := m.spinner
		if spin == "" {
			spin = "•"
		}
		status = StatusWarningStyle.Render(spin+" Authenticating…") + "\n" +
			helpStyle.MarginTop(0).Render("Approve the MFA prompt if one was sent.") + "\n" +
			helpStyle.MarginTop(0).Render("Enter is ignored · Esc cancels")
	case m.err != nil:
		status = ErrorMsgStyle.Bold(true).Render("Error: " + m.err.Error())
	default:
		status = helpStyle.MarginTop(0).Render("Tab: next  Space: toggle  Enter: connect  Ctrl+C: quit")
	}

	b.WriteString("\n")
	b.WriteString(loginStatusStyle().Render(status))

	content := b.String()

	boxStyle := ViewPanelStyle.Padding(2, 4)

	box := boxStyle.Render(content)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}
