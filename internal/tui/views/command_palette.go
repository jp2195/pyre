package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
	ti.Width = 50

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
	case tea.KeyMsg:
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
			m.cursor = min(len(m.filtered)-1, m.cursor+10)
			if m.cursor < 0 {
				m.cursor = 0
			}
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

	// Styles
	categoryStyle := DetailDimStyle.Bold(true).MarginTop(1)
	itemStyle := TableRowNormalStyle
	shortcutStyle := DetailLabelStyle
	descStyle := DetailDimStyle

	// Calculate modal width
	modalWidth := 60
	if m.width < modalWidth+10 {
		modalWidth = m.width - 10
	}
	if modalWidth < 40 {
		modalWidth = 40
	}

	// Build content
	var b strings.Builder

	// Input field
	m.textInput.Width = modalWidth - 4
	b.WriteString(ModalInputStyle.Render(m.textInput.View()))
	b.WriteString("\n")

	// Group commands by category
	categoryOrder := []string{"Monitor", "Analyze", "Tools", "Connections", "Actions", "System"}
	commandsByCategory := make(map[string][]Command)
	for _, cmd := range m.filtered {
		commandsByCategory[cmd.Category] = append(commandsByCategory[cmd.Category], cmd)
	}

	// Calculate visible area (reserve space for input, help, borders)
	maxVisible := m.height - 12
	if maxVisible < 5 {
		maxVisible = 5
	}

	// Track current index for cursor highlighting
	currentIdx := 0
	visibleCount := 0

	// Calculate scroll offset
	scrollOffset := 0
	if m.cursor >= maxVisible {
		scrollOffset = m.cursor - maxVisible + 1
	}

	// Render categories and items
	for _, cat := range categoryOrder {
		commands, ok := commandsByCategory[cat]
		if !ok || len(commands) == 0 {
			continue
		}

		// Calculate if this category header should be visible
		categoryStartIdx := currentIdx

		// Skip items before scroll offset
		if currentIdx+len(commands) <= scrollOffset {
			currentIdx += len(commands)
			continue
		}

		// Only show category header if at least one item is visible
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

			// Build item line
			indicator := "  "
			style := itemStyle

			if currentIdx == m.cursor {
				indicator = "> "
				style = ModalSelectedStyle
			}

			// Shortcut display
			shortcut := ""
			if cmd.Shortcut != "" {
				shortcut = shortcutStyle.Render("[" + cmd.Shortcut + "] ")
			}

			// Label and description
			label := cmd.Label
			desc := ""
			if cmd.Description != "" {
				// Truncate description to fit
				maxDescLen := modalWidth - len(label) - len(shortcut) - 10
				if maxDescLen > 0 && len(cmd.Description) > 0 {
					d := cmd.Description
					if len(d) > maxDescLen {
						d = d[:maxDescLen-3] + "..."
					}
					desc = descStyle.Render("  " + d)
				}
			}

			line := indicator + shortcut + style.Render(label) + desc
			b.WriteString(line + "\n")

			visibleCount++
			currentIdx++
		}

		// Don't show category header if we started after scroll offset
		_ = categoryStartIdx
	}

	// Show scroll indicator if needed
	if len(m.filtered) > maxVisible {
		scrollInfo := StatusMutedStyle.
			Align(lipgloss.Right).
			Width(modalWidth - 4).
			Render(strings.Repeat(" ", modalWidth-20) +
				"(" + itoa(m.cursor+1) + "/" + itoa(len(m.filtered)) + ")")
		b.WriteString(scrollInfo + "\n")
	}

	// Help text
	helpText := "Up/Down navigate  Enter select  Esc close  Type to filter"
	b.WriteString(ModalHelpStyle.Width(modalWidth - 4).Render(helpText))

	// Wrap in modal
	content := b.String()
	modal := ModalStyle.Width(modalWidth).Render(content)

	// Center in terminal
	c := theme.Colors()
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(c.Overlay),
	)
}

// Helper function to convert int to string without importing strconv
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
