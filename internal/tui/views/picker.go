package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/joshuamontgomery/pyre/internal/auth"
)

type PickerModel struct {
	connections []*auth.Connection
	active      string
	cursor      int
	width       int
	height      int
}

func NewPickerModel(session *auth.Session) PickerModel {
	m := PickerModel{}
	m = m.UpdateConnections(session)
	return m
}

func (m PickerModel) UpdateConnections(session *auth.Session) PickerModel {
	m.connections = session.ListConnections()
	m.active = session.ActiveFirewall
	m.cursor = 0

	for i, c := range m.connections {
		if c.Name == m.active {
			m.cursor = i
			break
		}
	}

	return m
}

func (m PickerModel) SetSize(width, height int) PickerModel {
	m.width = width
	m.height = height
	return m
}

func (m PickerModel) Selected() string {
	if m.cursor >= 0 && m.cursor < len(m.connections) {
		return m.connections[m.cursor].Name
	}
	return ""
}

func (m PickerModel) Update(msg tea.Msg) (PickerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.connections)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		}
	}
	return m, nil
}

func (m PickerModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7C3AED")).
		MarginBottom(1)

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#374151")).
		Padding(1, 2)

	rowStyle := lipgloss.NewStyle().
		Padding(0, 1)

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#1F2937")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1)

	activeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981"))

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		MarginTop(1)

	var b strings.Builder
	b.WriteString(titleStyle.Render("Firewall Connections"))
	b.WriteString("\n\n")

	panoramaStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A855F7"))

	targetStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#60A5FA"))

	if len(m.connections) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("No connections. Press 'a' to add a new firewall."))
	} else {
		for i, conn := range m.connections {
			style := rowStyle
			if i == m.cursor {
				style = selectedStyle
			}

			indicator := "  "
			if conn.Name == m.active {
				indicator = activeStyle.Render("● ")
			}

			status := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render("connected")
			if !conn.Connected {
				status = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Render("disconnected")
			}

			line := indicator + style.Render(conn.Name) + " " +
				lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("("+conn.Config.Host+")")

			// Add Panorama indicator with device count
			if conn.IsPanorama {
				connCount := conn.ConnectedDeviceCount()
				totalCount := len(conn.ManagedDevices)
				line += " " + panoramaStyle.Render(fmt.Sprintf("[Panorama: %d/%d devices]", connCount, totalCount))

				// Show current target if set
				if target := conn.GetTargetDevice(); target != nil {
					hostname := target.Hostname
					if hostname == "" {
						hostname = target.Serial
					}
					line += " " + targetStyle.Render("→ "+hostname)
				}
			}

			line += " - " + status

			b.WriteString(line + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k: navigate  enter: select  a: add new  esc: back"))

	content := b.String()

	boxWidth := 60
	if m.width < boxWidth+10 {
		boxWidth = m.width - 10
	}

	box := panelStyle.Width(boxWidth).Render(content)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}
