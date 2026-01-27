package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jp2195/pyre/internal/config"
)

// FormMode represents the mode of the connection form
type FormMode int

const (
	FormModeQuickConnect FormMode = iota // Quick connect, save unchecked by default
	FormModeAdd                          // Adding new connection, save checked, name required
	FormModeEdit                         // Editing existing connection
)

// ConnectionFormField represents which field is focused
type ConnectionFormField int

const (
	FormFieldHost ConnectionFormField = iota
	FormFieldUsername
	FormFieldType
	FormFieldInsecure
	FormFieldSave
)

// ConnectionFormModel is the model for the connection form view
type ConnectionFormModel struct {
	mode          FormMode
	editingHost   string // Original host when editing (for detecting changes)
	hostInput     textinput.Model
	usernameInput textinput.Model
	connType      string // "firewall" or "panorama"
	insecure      bool
	saveToConfig  bool
	focusedField  ConnectionFormField
	err           error
	width         int
	height        int
}

// NewQuickConnectForm creates a form for quick connect (no save by default)
func NewQuickConnectForm() ConnectionFormModel {
	m := newBaseForm()
	m.mode = FormModeQuickConnect
	m.saveToConfig = false
	m.focusedField = FormFieldHost
	m.updateFocus()
	return m
}

// NewAddConnectionForm creates a form for adding a new connection
func NewAddConnectionForm() ConnectionFormModel {
	m := newBaseForm()
	m.mode = FormModeAdd
	m.saveToConfig = true
	m.focusedField = FormFieldHost
	m.updateFocus()
	return m
}

// NewEditConnectionForm creates a form for editing an existing connection
func NewEditConnectionForm(host string, conn config.ConnectionConfig) ConnectionFormModel {
	m := newBaseForm()
	m.mode = FormModeEdit
	m.editingHost = host
	m.hostInput.SetValue(host)
	m.usernameInput.SetValue(conn.Username)
	m.connType = conn.Type
	if m.connType == "" {
		m.connType = "firewall"
	}
	m.insecure = conn.Insecure
	m.saveToConfig = true // Always save when editing
	m.focusedField = FormFieldHost
	m.updateFocus()
	return m
}

func newBaseForm() ConnectionFormModel {
	hostInput := textinput.New()
	hostInput.Placeholder = "10.1.1.1 or firewall.example.com"
	hostInput.CharLimit = 255
	hostInput.Width = 40

	usernameInput := textinput.New()
	usernameInput.Placeholder = "admin"
	usernameInput.CharLimit = 64
	usernameInput.Width = 40

	return ConnectionFormModel{
		hostInput:     hostInput,
		usernameInput: usernameInput,
		connType:      "firewall",
		insecure:      false,
		saveToConfig:  false,
	}
}

func (m *ConnectionFormModel) updateFocus() {
	m.hostInput.Blur()
	m.usernameInput.Blur()

	switch m.focusedField {
	case FormFieldHost:
		m.hostInput.Focus()
	case FormFieldUsername:
		m.usernameInput.Focus()
	}
}

// SetSize sets the dimensions for the view
func (m ConnectionFormModel) SetSize(width, height int) ConnectionFormModel {
	m.width = width
	m.height = height
	return m
}

// SetError sets an error message to display
func (m ConnectionFormModel) SetError(err error) ConnectionFormModel {
	m.err = err
	return m
}

// ClearError clears any error message
func (m ConnectionFormModel) ClearError() ConnectionFormModel {
	m.err = nil
	return m
}

// NextField moves focus to the next field
func (m ConnectionFormModel) NextField() ConnectionFormModel {
	maxField := FormFieldSave
	if m.mode == FormModeEdit {
		maxField = FormFieldInsecure // Hide save checkbox in edit mode
	}

	if int(m.focusedField) < int(maxField) {
		m.focusedField++
	} else {
		// Wrap around to host
		m.focusedField = FormFieldHost
	}
	m.updateFocus()
	return m
}

// PrevField moves focus to the previous field
func (m ConnectionFormModel) PrevField() ConnectionFormModel {
	maxField := FormFieldSave
	if m.mode == FormModeEdit {
		maxField = FormFieldInsecure
	}

	if m.focusedField > FormFieldHost {
		m.focusedField--
	} else {
		// Wrap around
		m.focusedField = maxField
	}
	m.updateFocus()
	return m
}

// ToggleType toggles between firewall and panorama
func (m ConnectionFormModel) ToggleType() ConnectionFormModel {
	if m.connType == "firewall" {
		m.connType = "panorama"
	} else {
		m.connType = "firewall"
	}
	return m
}

// ToggleInsecure toggles the insecure checkbox
func (m ConnectionFormModel) ToggleInsecure() ConnectionFormModel {
	m.insecure = !m.insecure
	return m
}

// ToggleSave toggles the save to config checkbox
func (m ConnectionFormModel) ToggleSave() ConnectionFormModel {
	m.saveToConfig = !m.saveToConfig
	return m
}

// FocusedField returns the currently focused field
func (m ConnectionFormModel) FocusedField() ConnectionFormField {
	return m.focusedField
}

// Mode returns the form mode
func (m ConnectionFormModel) Mode() FormMode {
	return m.mode
}

// Host returns the host value
func (m ConnectionFormModel) Host() string {
	return strings.TrimSpace(m.hostInput.Value())
}

// Username returns the username value
func (m ConnectionFormModel) Username() string {
	return strings.TrimSpace(m.usernameInput.Value())
}

// Type returns the connection type
func (m ConnectionFormModel) Type() string {
	return m.connType
}

// Insecure returns the insecure value
func (m ConnectionFormModel) Insecure() bool {
	return m.insecure
}

// SaveToConfig returns whether to save to config
func (m ConnectionFormModel) SaveToConfig() bool {
	return m.saveToConfig
}

// EditingHost returns the original host when editing
func (m ConnectionFormModel) EditingHost() string {
	return m.editingHost
}

// CanSubmit returns true if the form has valid data
func (m ConnectionFormModel) CanSubmit() bool {
	return m.Host() != ""
}

// GetConfig returns the connection config from form values
func (m ConnectionFormModel) GetConfig() config.ConnectionConfig {
	return config.ConnectionConfig{
		Username: m.Username(),
		Type:     m.connType,
		Insecure: m.insecure,
	}
}

// Update handles input updates
func (m ConnectionFormModel) Update(msg tea.Msg) (ConnectionFormModel, tea.Cmd) {
	var cmd tea.Cmd

	switch m.focusedField {
	case FormFieldHost:
		m.hostInput, cmd = m.hostInput.Update(msg)
	case FormFieldUsername:
		m.usernameInput, cmd = m.usernameInput.Update(msg)
	}

	return m, cmd
}

// View renders the connection form
func (m ConnectionFormModel) View() string {
	titleStyle := ViewTitleStyle.MarginBottom(2)
	labelStyle := DetailLabelStyle.MarginBottom(1)
	inputStyle := InputStyle
	focusedInputStyle := InputFocusedStyle
	errorStyle := ErrorMsgStyle.Bold(true).MarginTop(1)
	helpStyle := HelpDescStyle.MarginTop(2)

	var b strings.Builder

	// Title based on mode
	var title string
	switch m.mode {
	case FormModeQuickConnect:
		title = "Quick Connect"
	case FormModeAdd:
		title = "Add Connection"
	case FormModeEdit:
		title = "Edit Connection"
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	// Host field
	b.WriteString(labelStyle.Render("Host"))
	b.WriteString("\n")
	if m.focusedField == FormFieldHost {
		b.WriteString(focusedInputStyle.Render(m.hostInput.View()))
	} else {
		b.WriteString(inputStyle.Render(m.hostInput.View()))
	}
	b.WriteString("\n")

	// Username field
	b.WriteString(labelStyle.Render("Username (optional)"))
	b.WriteString("\n")
	if m.focusedField == FormFieldUsername {
		b.WriteString(focusedInputStyle.Render(m.usernameInput.View()))
	} else {
		b.WriteString(inputStyle.Render(m.usernameInput.View()))
	}
	b.WriteString("\n")

	// Type selector (radio buttons)
	b.WriteString(labelStyle.Render("Type"))
	b.WriteString("\n")

	fwRadio := "( ) Firewall"
	panRadio := "( ) Panorama"
	if m.connType == "firewall" {
		fwRadio = "(*) Firewall"
	} else {
		panRadio = "(*) Panorama"
	}

	typeRow := fwRadio + "    " + panRadio
	if m.focusedField == FormFieldType {
		b.WriteString(focusedInputStyle.Render(typeRow))
	} else {
		b.WriteString(inputStyle.Render(typeRow))
	}
	b.WriteString("\n")

	// Insecure checkbox
	insecureCheck := "[ ] Skip TLS verification (insecure)"
	if m.insecure {
		insecureCheck = "[x] Skip TLS verification (insecure)"
	}
	if m.focusedField == FormFieldInsecure {
		b.WriteString(focusedInputStyle.Render(insecureCheck))
	} else {
		b.WriteString(inputStyle.Render(insecureCheck))
	}
	b.WriteString("\n")

	// Save to config checkbox (only show for quick connect and add modes)
	if m.mode != FormModeEdit {
		saveCheck := "[ ] Save to ~/.pyre.yaml"
		if m.saveToConfig {
			saveCheck = "[x] Save to ~/.pyre.yaml"
		}
		if m.focusedField == FormFieldSave {
			b.WriteString(focusedInputStyle.Render(saveCheck))
		} else {
			b.WriteString(inputStyle.Render(saveCheck))
		}
	}

	// Error message
	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("Error: " + m.err.Error()))
	}

	// Help text
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("[Tab] Next field  [Space] Toggle  [Enter] Submit  [Esc] Cancel"))

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
