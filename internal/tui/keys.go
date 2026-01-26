package tui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	// Global
	Quit         key.Binding
	Help         key.Binding
	DevicePicker key.Binding
	Refresh      key.Binding
	OpenPalette  key.Binding

	// Navigation groups (1-4 for top-level groups)
	NavGroup1 key.Binding
	NavGroup2 key.Binding
	NavGroup3 key.Binding
	NavGroup4 key.Binding

	// Navigation
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding

	// Actions
	Enter    key.Binding
	Filter   key.Binding
	Escape   key.Binding
	Tab      key.Binding
	ShiftTab key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		DevicePicker: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "devices"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		OpenPalette: key.NewBinding(
			key.WithKeys("ctrl+p"),
			key.WithHelp("ctrl+p", "search"),
		),

		// Navigation groups
		NavGroup1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "Monitor"),
		),
		NavGroup2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "Analyze"),
		),
		NavGroup3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "Tools"),
		),
		NavGroup4: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "Connections"),
		),

		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("pgdn", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("home/g", "top"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("end/G", "bottom"),
		),

		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back/cancel"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next field"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev field"),
		),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.NavGroup1, k.NavGroup2, k.NavGroup3, k.NavGroup4, k.Refresh, k.Help, k.Quit,
	}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.NavGroup1, k.NavGroup2, k.NavGroup3, k.NavGroup4},
		{k.Refresh, k.OpenPalette, k.Help, k.Quit},
		{k.Up, k.Down, k.PageUp, k.PageDown},
		{k.Filter, k.Enter, k.Escape},
	}
}

type LoginKeyMap struct {
	Submit key.Binding
	Tab    key.Binding
	Quit   key.Binding
}

func DefaultLoginKeyMap() LoginKeyMap {
	return LoginKeyMap{
		Submit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "submit"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next field"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
	}
}

type PickerKeyMap struct {
	Select key.Binding
	Add    key.Binding
	Delete key.Binding
	Back   key.Binding
	Up     key.Binding
	Down   key.Binding
}

func DefaultPickerKeyMap() PickerKeyMap {
	return PickerKeyMap{
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add new"),
		),
		Delete: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "disconnect"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", ":"),
			key.WithHelp("esc", "back"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
	}
}

type DevicePickerKeyMap struct {
	Select  key.Binding
	Refresh key.Binding
	Back    key.Binding
	Up      key.Binding
	Down    key.Binding
}

func DefaultDevicePickerKeyMap() DevicePickerKeyMap {
	return DevicePickerKeyMap{
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "d"),
			key.WithHelp("esc", "back"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
	}
}
