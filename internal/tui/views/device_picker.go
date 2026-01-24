package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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

// SelectedSerial returns the serial of the selected device, or empty string for Panorama.
func (m DevicePickerModel) SelectedSerial() string {
	if m.cursor == 0 {
		return ""
	}
	idx := m.cursor - 1
	if idx >= 0 && idx < len(m.devices) {
		return m.devices[idx].Serial
	}
	return ""
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
	case tea.KeyMsg:
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

	var b strings.Builder
	b.WriteString(titleStyle.Render("Select Target Device"))
	b.WriteString("\n\n")

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

		line := indicator + style.Render(fmt.Sprintf("[%s]", m.panoramaName)) +
			dimStyle.Render(" - Direct Panorama operations")
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
		haState := ""
		if device.HAState != "" {
			haState = fmt.Sprintf(" [%s]", device.HAState)
		}

		connStatus := connectedStyle.Render("connected")
		connIcon := connectedStyle.Render("●")
		if !device.Connected {
			connStatus = disconnectedStyle.Render("disconnected")
			connIcon = disconnectedStyle.Render("○")
		}

		hostname := device.Hostname
		if hostname == "" {
			hostname = device.Serial
		}

		line := indicator + style.Render(hostname) +
			dimStyle.Render(fmt.Sprintf(" (%s)%s", device.Model, haState)) +
			" - " + dimStyle.Render(device.IPAddress) +
			" - " + connIcon + " " + connStatus

		b.WriteString(line + "\n")
	}

	if len(m.devices) == 0 {
		b.WriteString(dimStyle.Render("  No managed devices found.") + "\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k: navigate  enter: select  esc: back  r: refresh"))

	content := b.String()

	boxWidth := 80
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
