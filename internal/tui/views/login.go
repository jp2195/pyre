package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jp2195/pyre/internal/auth"
)

type loginField int

const (
	fieldHost loginField = iota
	fieldUsername
	fieldPassword
)

type LoginModel struct {
	hostInput     textinput.Model
	usernameInput textinput.Model
	passwordInput textinput.Model
	focusedField  loginField
	err           error
	width         int
	height        int
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
		focusedField:  fieldHost,
	}

	if creds.Host != "" {
		m.focusedField = fieldUsername
	}

	m.updateFocus()
	return m
}

func (m *LoginModel) updateFocus() {
	m.hostInput.Blur()
	m.usernameInput.Blur()
	m.passwordInput.Blur()

	switch m.focusedField {
	case fieldHost:
		m.hostInput.Focus()
	case fieldUsername:
		m.usernameInput.Focus()
	case fieldPassword:
		m.passwordInput.Focus()
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
	m.focusedField = (m.focusedField + 1) % 3
	m.updateFocus()
	return m
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

func (m LoginModel) CanSubmit() bool {
	return m.Host() != "" && m.Username() != "" && m.Password() != ""
}

func (m LoginModel) Update(msg tea.Msg) (LoginModel, tea.Cmd) {
	var cmd tea.Cmd

	switch m.focusedField {
	case fieldHost:
		m.hostInput, cmd = m.hostInput.Update(msg)
	case fieldUsername:
		m.usernameInput, cmd = m.usernameInput.Update(msg)
	case fieldPassword:
		m.passwordInput, cmd = m.passwordInput.Update(msg)
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
	if m.focusedField == fieldHost {
		b.WriteString(focusedInputStyle.Render(m.hostInput.View()))
	} else {
		b.WriteString(inputStyle.Render(m.hostInput.View()))
	}
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Username"))
	b.WriteString("\n")
	if m.focusedField == fieldUsername {
		b.WriteString(focusedInputStyle.Render(m.usernameInput.View()))
	} else {
		b.WriteString(inputStyle.Render(m.usernameInput.View()))
	}
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Password"))
	b.WriteString("\n")
	if m.focusedField == fieldPassword {
		b.WriteString(focusedInputStyle.Render(m.passwordInput.View()))
	} else {
		b.WriteString(inputStyle.Render(m.passwordInput.View()))
	}

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("Error: " + m.err.Error()))
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Tab: next field  Enter: connect  Ctrl+C: quit"))

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
