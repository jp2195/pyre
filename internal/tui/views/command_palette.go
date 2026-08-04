package views

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/tui/theme"
)

// Command represents a command in the palette
type Command struct {
	ID          string
	Label       string
	Description string
	Category    string
	Shortcut    string // Display only (e.g., "1", "r", "Tab")
	Action      func() tea.Msg
	Available   func() bool // Context-aware availability
}

// CommandPaletteModel represents the command palette modal
type CommandPaletteModel struct {
	commands  []Command
	filtered  []Command
	query     string
	cursor    int
	width     int
	height    int
	textInput textinput.Model
	focused   bool
	prefilter string // Category to pre-filter (e.g., "Connections" for ":")
}

// NewCommandPaletteModel creates a new command palette model
func NewCommandPaletteModel() CommandPaletteModel {
	ti := textinput.New()
	ti.Placeholder = "Type to search..."
	ti.Prompt = "> "
	ti.CharLimit = 64
	ti.SetWidth(50)

	return CommandPaletteModel{
		textInput: ti,
		commands:  []Command{},
		filtered:  []Command{},
	}
}

// SetCommands sets the available commands
func (m CommandPaletteModel) SetCommands(commands []Command) CommandPaletteModel {
	m.commands = commands
	m.filterCommands()
	return m
}

// SetSize sets the terminal dimensions
func (m CommandPaletteModel) SetSize(width, height int) CommandPaletteModel {
	m.width = width
	m.height = height
	return m
}

// Focus focuses the command palette and resets state
func (m CommandPaletteModel) Focus() CommandPaletteModel {
	m.focused = true
	m.query = ""
	m.cursor = 0
	m.prefilter = ""
	m.textInput.SetValue("")
	m.textInput.Focus()
	m.filterCommands()
	return m
}

// FocusWithFilter focuses the palette with a pre-filter category
func (m CommandPaletteModel) FocusWithFilter(category string) CommandPaletteModel {
	m.focused = true
	m.query = ""
	m.cursor = 0
	m.prefilter = category
	m.textInput.SetValue("")
	m.textInput.Focus()
	m.filterCommands()
	return m
}

// Blur unfocuses the command palette
func (m CommandPaletteModel) Blur() CommandPaletteModel {
	m.focused = false
	m.textInput.Blur()
	return m
}

// IsFocused returns whether the palette is focused
func (m CommandPaletteModel) IsFocused() bool {
	return m.focused
}

// SelectedCommand returns the currently selected command
func (m CommandPaletteModel) SelectedCommand() *Command {
	if m.cursor >= 0 && m.cursor < len(m.filtered) {
		return &m.filtered[m.cursor]
	}
	return nil
}

// filterCommands filters commands based on the current query and prefilter
func (m *CommandPaletteModel) filterCommands() {
	m.filtered = nil
	query := strings.ToLower(m.query)

	for _, cmd := range m.commands {
		// Skip unavailable commands
		if cmd.Available != nil && !cmd.Available() {
			continue
		}

		// Apply prefilter if set
		if m.prefilter != "" && cmd.Category != m.prefilter {
			continue
		}

		// If no query, include all available commands
		if query == "" {
			m.filtered = append(m.filtered, cmd)
			continue
		}

		// Match against label and description (fuzzy-ish)
		label := strings.ToLower(cmd.Label)
		desc := strings.ToLower(cmd.Description)
		cat := strings.ToLower(cmd.Category)

		if strings.Contains(label, query) ||
			strings.Contains(desc, query) ||
			strings.Contains(cat, query) {
			m.filtered = append(m.filtered, cmd)
		}
	}

	// Reset cursor if out of bounds
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

// Update handles events for the command palette
func (m CommandPaletteModel) Update(msg tea.Msg) (CommandPaletteModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			} else {
				// Wrap to bottom
				m.cursor = max(0, len(m.filtered)-1)
			}
			return m, nil

		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			} else {
				// Wrap to top
				m.cursor = 0
			}
			return m, nil

		case "home", "ctrl+a":
			m.cursor = 0
			return m, nil

		case "end", "ctrl+e":
			m.cursor = max(0, len(m.filtered)-1)
			return m, nil

		case "pgup":
			m.cursor = max(0, m.cursor-10)
			return m, nil

		case "pgdown":
			m.cursor = max(min(len(m.filtered)-1, m.cursor+10), 0)
			return m, nil
		}
	}

	// Handle text input changes
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	// Update query and re-filter if changed
	newQuery := m.textInput.Value()
	if newQuery != m.query {
		m.query = newQuery
		m.filterCommands()
	}

	return m, cmd
}

// View renders the command palette
func (m CommandPaletteModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	modalWidth := m.modalWidth()
	if modalWidth < 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			"Terminal too narrow for command palette")
	}

	var b strings.Builder

	// Input field
	m.textInput.SetWidth(modalWidth - 4)
	b.WriteString(ModalInputStyle.Render(m.textInput.View()))
	b.WriteString("\n")

	// Command list
	m.renderCommandList(&b, modalWidth)

	// Help text
	helpText := "Up/Down navigate  Enter select  Esc close  Type to filter"
	b.WriteString(ModalHelpStyle.Width(modalWidth - 4).Render(helpText))

	// Wrap in modal and center
	modal := ModalStyle.Width(modalWidth).Render(b.String())
	c := theme.Colors()
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Foreground(c.Overlay)),
	)
}

// modalWidth calculates the modal width, returning -1 if the terminal is too narrow.
func (m CommandPaletteModel) modalWidth() int {
	const minWidth = 30
	modalWidth := 60
	if m.width < modalWidth+4 {
		modalWidth = m.width - 4
	}
	if modalWidth < minWidth {
		modalWidth = minWidth
	}
	if m.width < minWidth+4 {
		return -1
	}
	return modalWidth
}

// renderCommandList renders the categorized, scrollable command list.
func (m CommandPaletteModel) renderCommandList(b *strings.Builder, modalWidth int) {
	categoryStyle := DetailDimStyle.Bold(true).MarginTop(1)
	itemStyle := TableRowNormalStyle
	shortcutStyle := DetailLabelStyle
	descStyle := DetailDimStyle

	categoryOrder := []string{"Monitor", "Analyze", "Tools", "Connections", "Actions", "System"}
	commandsByCategory := make(map[string][]Command)
	for _, cmd := range m.filtered {
		commandsByCategory[cmd.Category] = append(commandsByCategory[cmd.Category], cmd)
	}

	maxVisible := max(m.height-12, 5)
	currentIdx := 0
	visibleCount := 0

	scrollOffset := 0
	if m.cursor >= maxVisible {
		scrollOffset = m.cursor - maxVisible + 1
	}

	for _, cat := range categoryOrder {
		commands, ok := commandsByCategory[cat]
		if !ok || len(commands) == 0 {
			continue
		}

		if currentIdx+len(commands) <= scrollOffset {
			currentIdx += len(commands)
			continue
		}

		if visibleCount < maxVisible && currentIdx >= scrollOffset {
			b.WriteString(categoryStyle.Render("  " + cat))
			b.WriteString("\n")
		}

		for _, cmd := range commands {
			if currentIdx < scrollOffset {
				currentIdx++
				continue
			}
			if visibleCount >= maxVisible {
				currentIdx++
				continue
			}

			indicator := "  "
			style := itemStyle
			if currentIdx == m.cursor {
				indicator = "> "
				style = ModalSelectedStyle
			}

			shortcut := ""
			if cmd.Shortcut != "" {
				shortcut = shortcutStyle.Render("[" + cmd.Shortcut + "] ")
			}

			desc := ""
			if cmd.Description != "" {
				maxDescLen := modalWidth - len(cmd.Label) - len(shortcut) - 10
				if maxDescLen > 0 {
					desc = descStyle.Render("  " + truncateEllipsis(cmd.Description, maxDescLen))
				}
			}

			b.WriteString(indicator + shortcut + style.Render(cmd.Label) + desc + "\n")
			visibleCount++
			currentIdx++
		}
	}

	if len(m.filtered) > maxVisible {
		scrollInfo := StatusMutedStyle.
			Align(lipgloss.Right).
			Width(modalWidth - 4).
			Render(strings.Repeat(" ", modalWidth-20) +
				"(" + strconv.Itoa(m.cursor+1) + "/" + strconv.Itoa(len(m.filtered)) + ")")
		b.WriteString(scrollInfo + "\n")
	}
}
