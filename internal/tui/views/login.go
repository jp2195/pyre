package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
}

func NewLoginModel(creds *auth.Credentials) LoginModel {
	host := textinput.New()
	host.Placeholder = "firewall.example.com"
	host.CharLimit = 255
	host.Width = 40
	if creds.Host != "" {
		host.SetValue(creds.Host)
	}

	username := textinput.New()
	username.Placeholder = "admin"
	username.CharLimit = 64
	username.Width = 40
	if creds.Username != "" {
		username.SetValue(creds.Username)
	}

	password := textinput.New()
	password.Placeholder = "password"
	password.CharLimit = 128
	password.Width = 40
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
	return m.Host() != "" && m.Username() != "" && m.Password() != ""
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

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("Error: " + m.err.Error()))
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Tab: next  Space: toggle  Enter: connect  Ctrl+C: quit"))

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
