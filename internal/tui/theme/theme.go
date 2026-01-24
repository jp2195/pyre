package theme

// Theme represents a complete color theme for the application.
type Theme struct {
	Name   string
	Colors ColorPalette
}

// current holds the active theme. Default is dark.
var current = &dark

// Init initializes the theme system with the specified theme name.
// Valid names are: "dark", "light", "nord", "dracula", "solarized",
// "gruvbox", "tokyonight", "catppuccin", "onedark", "monokai".
// If an invalid name is provided, "dark" is used as the default.
func Init(name string) {
	switch name {
	case "light":
		current = &light
	case "nord":
		current = &nord
	case "dracula":
		current = &dracula
	case "solarized":
		current = &solarized
	case "gruvbox":
		current = &gruvbox
	case "tokyonight":
		current = &tokyonight
	case "catppuccin":
		current = &catppuccin
	case "onedark":
		current = &onedark
	case "monokai":
		current = &monokai
	default:
		current = &dark
	}
}

// Current returns the currently active theme.
func Current() *Theme {
	return current
}

// Colors returns the color palette for the current theme.
// This is a convenience function for accessing theme colors.
func Colors() *ColorPalette {
	return &current.Colors
}

// Name returns the name of the current theme.
func Name() string {
	return current.Name
}
