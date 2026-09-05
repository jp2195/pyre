package views

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/models"
)

type DevicePickerModel struct {
	devices      []models.ManagedDevice
	targetSerial string // Currently targeted device (empty = Panorama)
	cursor       int    // 0 = Panorama, 1+ = devices
	width        int
	height       int
	panoramaName string
}

func NewDevicePickerModel() DevicePickerModel {
	return DevicePickerModel{
		panoramaName: "Panorama",
	}
}

func (m DevicePickerModel) SetDevices(devices []models.ManagedDevice, currentTarget string, panoramaName string) DevicePickerModel {
	m.devices = devices
	m.targetSerial = currentTarget
	m.panoramaName = panoramaName
	m.cursor = 0

	// Set cursor to current target
	if currentTarget != "" {
		for i, d := range devices {
			if d.Serial == currentTarget {
				m.cursor = i + 1 // +1 because index 0 is Panorama
				break
			}
		}
	}

	return m
}

func (m DevicePickerModel) SetSize(width, height int) DevicePickerModel {
	m.width = width
	m.height = height
	return m
}

// SelectedDevice returns the selected device, or nil for Panorama.
func (m DevicePickerModel) SelectedDevice() *models.ManagedDevice {
	if m.cursor == 0 {
		return nil
	}
	idx := m.cursor - 1
	if idx >= 0 && idx < len(m.devices) {
		return &m.devices[idx]
	}
	return nil
}

func (m DevicePickerModel) Update(msg tea.Msg) (DevicePickerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		totalItems := len(m.devices) + 1 // +1 for Panorama
		switch msg.String() {
		case "j", "down":
			if m.cursor < totalItems-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "g", "home":
			m.cursor = 0
		case "G", "end":
			m.cursor = totalItems - 1
		}
	}
	return m, nil
}

func (m DevicePickerModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	titleStyle := ViewTitleStyle.MarginBottom(1)
	panelStyle := ViewPanelStyle
	rowStyle := TableRowNormalStyle
	selectedStyle := TableRowSelectedStyle
	activeStyle := StatusActiveStyle
	dimStyle := DetailDimStyle
	connectedStyle := StatusActiveStyle
	disconnectedStyle := StatusInactiveStyle
	helpStyle := HelpDescStyle.MarginTop(1)

	boxWidth := modalBoxWidth(m.width, 80)
	contentWidth := modalContentWidth(boxWidth)

	var b strings.Builder
	b.WriteString(titleStyle.Render("Select Target Device"))
	// One newline, not two: the title style already renders a blank
	// line through its bottom margin.
	b.WriteString("\n")

	// Panorama option (index 0)
	{
		style := rowStyle
		if m.cursor == 0 {
			style = selectedStyle
		}

		indicator := "  "
		if m.targetSerial == "" {
			indicator = activeStyle.Render("► ")
		}

		// The explanation goes when the row would not otherwise fit; the
		// name of the Panorama itself has to stay.
		name := style.Render(fmt.Sprintf("[%s]", m.panoramaName))
		line := indicator + name + dimStyle.Render(" - Direct Panorama operations")
		if lipgloss.Width(line) > contentWidth {
			line = indicator + name
		}
		b.WriteString(line + "\n")
	}

	// Device options (index 1+)
	for i, device := range m.devices {
		style := rowStyle
		if m.cursor == i+1 {
			style = selectedStyle
		}

		indicator := "  "
		if device.Serial == m.targetSerial {
			indicator = activeStyle.Render("► ")
		}

		// Format: hostname (model) [ha-state] - IP - connected/disconnected
		hardware := fmt.Sprintf(" (%s)", device.Model)
		if device.HAState != "" {
			hardware += fmt.Sprintf(" [%s]", device.HAState)
		}

		connWord, connGlyph, connStyle := "connected", "●", connectedStyle
		if !device.Connected {
			connWord, connGlyph, connStyle = "disconnected", "○", disconnectedStyle
		}

		hostname := device.Hostname
		if hostname == "" {
			hostname = device.Serial
		}

		// The row was a fixed string, so on a narrow terminal it wrapped and
		// its second half landed on a line of its own without the marker
		// column. Give up the model and HA state first: they describe the
		// hardware, while the address and the connection state are what an
		// operator is choosing between. Only then cut the name.
		//
		// Measure the rendered line rather than the raw text: the row style
		// pads the name, and the status glyph is an ambiguous-width
		// character, so counting runes understates the row by several cells.
		render := func(name, hw string, withWord bool) string {
			status := connStyle.Render(connGlyph)
			if withWord {
				status += " " + connStyle.Render(connWord)
			}
			return indicator + style.Render(name) + dimStyle.Render(hw) +
				" - " + dimStyle.Render(device.IPAddress) + " - " + status
		}

		// Give up the hardware description first, then spell the connection
		// state with its glyph alone. The name and the address identify the
		// device, so they are the last things to go.
		line := render(hostname, hardware, true)
		if lipgloss.Width(line) > contentWidth {
			line = render(hostname, "", true)
		}
		if lipgloss.Width(line) > contentWidth {
			line = render(hostname, "", false)
		}
		if over := lipgloss.Width(line) - contentWidth; over > 0 {
			line = render(truncate(hostname, max(len([]rune(hostname))-over, 1)), "", false)
		}

		b.WriteString(line + "\n")
	}

	if len(m.devices) == 0 {
		b.WriteString(dimStyle.Render("  No managed devices found.") + "\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render(fitHints(contentWidth, "  ",
		"j/k: navigate", "enter: select", "esc: back", "r: refresh")))

	content := b.String()

	box := panelStyle.Width(boxWidth).Render(content)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}
