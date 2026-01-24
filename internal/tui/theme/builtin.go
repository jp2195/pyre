package theme

import "github.com/charmbracelet/lipgloss"

// dark is the default dark theme with purple accents.
var dark = Theme{
	Name: "dark",
	Colors: ColorPalette{
		// UI colors
		Primary: lipgloss.Color("#7C3AED"),
		Accent:  lipgloss.Color("#A78BFA"),
		Success: lipgloss.Color("#10B981"),
		Error:   lipgloss.Color("#EF4444"),
		Warning: lipgloss.Color("#F59E0B"),
		Info:    lipgloss.Color("#3B82F6"),

		// Severity colors
		Critical: lipgloss.Color("#EF4444"),
		High:     lipgloss.Color("#F97316"),
		Medium:   lipgloss.Color("#EAB308"),
		Low:      lipgloss.Color("#3B82F6"),

		// Text colors
		Text:      lipgloss.Color("#E5E7EB"),
		TextMuted: lipgloss.Color("#6B7280"),
		TextLight: lipgloss.Color("#F3F4F6"),
		TextLabel: lipgloss.Color("#9CA3AF"),

		// Background colors
		Background:    lipgloss.Color("#111827"),
		BackgroundAlt: lipgloss.Color("#1F2937"),
		Border:        lipgloss.Color("#374151"),
		Overlay:       lipgloss.Color("#000000"),

		// Special colors
		White: lipgloss.Color("#FFFFFF"),
		Black: lipgloss.Color("#000000"),
	},
}

// light is a light theme with purple accents.
var light = Theme{
	Name: "light",
	Colors: ColorPalette{
		// UI colors
		Primary: lipgloss.Color("#7C3AED"),
		Accent:  lipgloss.Color("#8B5CF6"),
		Success: lipgloss.Color("#059669"),
		Error:   lipgloss.Color("#DC2626"),
		Warning: lipgloss.Color("#D97706"),
		Info:    lipgloss.Color("#2563EB"),

		// Severity colors
		Critical: lipgloss.Color("#DC2626"),
		High:     lipgloss.Color("#EA580C"),
		Medium:   lipgloss.Color("#CA8A04"),
		Low:      lipgloss.Color("#2563EB"),

		// Text colors
		Text:      lipgloss.Color("#1F2937"),
		TextMuted: lipgloss.Color("#6B7280"),
		TextLight: lipgloss.Color("#111827"),
		TextLabel: lipgloss.Color("#4B5563"),

		// Background colors
		Background:    lipgloss.Color("#FFFFFF"),
		BackgroundAlt: lipgloss.Color("#F3F4F6"),
		Border:        lipgloss.Color("#D1D5DB"),
		Overlay:       lipgloss.Color("#000000"),

		// Special colors
		White: lipgloss.Color("#FFFFFF"),
		Black: lipgloss.Color("#000000"),
	},
}

// nord is a theme based on the Nord color palette.
var nord = Theme{
	Name: "nord",
	Colors: ColorPalette{
		// UI colors - Nord Frost and Aurora
		Primary: lipgloss.Color("#88C0D0"),
		Accent:  lipgloss.Color("#81A1C1"),
		Success: lipgloss.Color("#A3BE8C"),
		Error:   lipgloss.Color("#BF616A"),
		Warning: lipgloss.Color("#EBCB8B"),
		Info:    lipgloss.Color("#5E81AC"),

		// Severity colors
		Critical: lipgloss.Color("#BF616A"),
		High:     lipgloss.Color("#D08770"),
		Medium:   lipgloss.Color("#EBCB8B"),
		Low:      lipgloss.Color("#5E81AC"),

		// Text colors - Nord Snow Storm
		Text:      lipgloss.Color("#ECEFF4"),
		TextMuted: lipgloss.Color("#4C566A"),
		TextLight: lipgloss.Color("#E5E9F0"),
		TextLabel: lipgloss.Color("#D8DEE9"),

		// Background colors - Nord Polar Night
		Background:    lipgloss.Color("#2E3440"),
		BackgroundAlt: lipgloss.Color("#3B4252"),
		Border:        lipgloss.Color("#434C5E"),
		Overlay:       lipgloss.Color("#2E3440"),

		// Special colors
		White: lipgloss.Color("#ECEFF4"),
		Black: lipgloss.Color("#2E3440"),
	},
}

// dracula is a theme based on the Dracula color palette.
var dracula = Theme{
	Name: "dracula",
	Colors: ColorPalette{
		// UI colors
		Primary: lipgloss.Color("#BD93F9"),
		Accent:  lipgloss.Color("#FF79C6"),
		Success: lipgloss.Color("#50FA7B"),
		Error:   lipgloss.Color("#FF5555"),
		Warning: lipgloss.Color("#FFB86C"),
		Info:    lipgloss.Color("#8BE9FD"),

		// Severity colors
		Critical: lipgloss.Color("#FF5555"),
		High:     lipgloss.Color("#FFB86C"),
		Medium:   lipgloss.Color("#F1FA8C"),
		Low:      lipgloss.Color("#8BE9FD"),

		// Text colors
		Text:      lipgloss.Color("#F8F8F2"),
		TextMuted: lipgloss.Color("#6272A4"),
		TextLight: lipgloss.Color("#F8F8F2"),
		TextLabel: lipgloss.Color("#6272A4"),

		// Background colors
		Background:    lipgloss.Color("#282A36"),
		BackgroundAlt: lipgloss.Color("#44475A"),
		Border:        lipgloss.Color("#44475A"),
		Overlay:       lipgloss.Color("#282A36"),

		// Special colors
		White: lipgloss.Color("#F8F8F2"),
		Black: lipgloss.Color("#282A36"),
	},
}

// solarized is a theme based on the Solarized Dark color palette.
var solarized = Theme{
	Name: "solarized",
	Colors: ColorPalette{
		// UI colors
		Primary: lipgloss.Color("#268BD2"), // blue
		Accent:  lipgloss.Color("#2AA198"), // cyan
		Success: lipgloss.Color("#859900"), // green
		Error:   lipgloss.Color("#DC322F"), // red
		Warning: lipgloss.Color("#B58900"), // yellow
		Info:    lipgloss.Color("#6C71C4"), // violet

		// Severity colors
		Critical: lipgloss.Color("#DC322F"), // red
		High:     lipgloss.Color("#CB4B16"), // orange
		Medium:   lipgloss.Color("#B58900"), // yellow
		Low:      lipgloss.Color("#268BD2"), // blue

		// Text colors
		Text:      lipgloss.Color("#839496"), // base0
		TextMuted: lipgloss.Color("#586E75"), // base01
		TextLight: lipgloss.Color("#93A1A1"), // base1
		TextLabel: lipgloss.Color("#657B83"), // base00

		// Background colors
		Background:    lipgloss.Color("#002B36"), // base03
		BackgroundAlt: lipgloss.Color("#073642"), // base02
		Border:        lipgloss.Color("#586E75"), // base01
		Overlay:       lipgloss.Color("#002B36"),

		// Special colors
		White: lipgloss.Color("#FDF6E3"), // base3
		Black: lipgloss.Color("#002B36"), // base03
	},
}

// gruvbox is a theme based on the Gruvbox Dark color palette.
var gruvbox = Theme{
	Name: "gruvbox",
	Colors: ColorPalette{
		// UI colors
		Primary: lipgloss.Color("#D79921"), // yellow
		Accent:  lipgloss.Color("#D65D0E"), // orange
		Success: lipgloss.Color("#98971A"), // green
		Error:   lipgloss.Color("#CC241D"), // red
		Warning: lipgloss.Color("#D79921"), // yellow
		Info:    lipgloss.Color("#458588"), // blue

		// Severity colors
		Critical: lipgloss.Color("#CC241D"), // red
		High:     lipgloss.Color("#D65D0E"), // orange
		Medium:   lipgloss.Color("#D79921"), // yellow
		Low:      lipgloss.Color("#458588"), // blue

		// Text colors
		Text:      lipgloss.Color("#EBDBB2"), // fg
		TextMuted: lipgloss.Color("#928374"), // gray
		TextLight: lipgloss.Color("#FBF1C7"), // fg0
		TextLabel: lipgloss.Color("#A89984"), // gray light

		// Background colors
		Background:    lipgloss.Color("#282828"), // bg
		BackgroundAlt: lipgloss.Color("#3C3836"), // bg1
		Border:        lipgloss.Color("#504945"), // bg2
		Overlay:       lipgloss.Color("#1D2021"), // bg0_h

		// Special colors
		White: lipgloss.Color("#FBF1C7"), // fg0
		Black: lipgloss.Color("#1D2021"), // bg0_h
	},
}

// tokyonight is a theme based on the Tokyo Night color palette.
var tokyonight = Theme{
	Name: "tokyonight",
	Colors: ColorPalette{
		// UI colors
		Primary: lipgloss.Color("#7AA2F7"), // blue
		Accent:  lipgloss.Color("#BB9AF7"), // purple
		Success: lipgloss.Color("#9ECE6A"), // green
		Error:   lipgloss.Color("#F7768E"), // red
		Warning: lipgloss.Color("#E0AF68"), // yellow
		Info:    lipgloss.Color("#7DCFFF"), // cyan

		// Severity colors
		Critical: lipgloss.Color("#F7768E"), // red
		High:     lipgloss.Color("#FF9E64"), // orange
		Medium:   lipgloss.Color("#E0AF68"), // yellow
		Low:      lipgloss.Color("#7DCFFF"), // cyan

		// Text colors
		Text:      lipgloss.Color("#C0CAF5"), // fg
		TextMuted: lipgloss.Color("#565F89"), // comment
		TextLight: lipgloss.Color("#A9B1D6"), // fg_dark
		TextLabel: lipgloss.Color("#9AA5CE"), // fg_gutter

		// Background colors
		Background:    lipgloss.Color("#1A1B26"), // bg
		BackgroundAlt: lipgloss.Color("#24283B"), // bg_highlight
		Border:        lipgloss.Color("#414868"), // border
		Overlay:       lipgloss.Color("#16161E"), // bg_dark

		// Special colors
		White: lipgloss.Color("#C0CAF5"),
		Black: lipgloss.Color("#16161E"),
	},
}

// catppuccin is a theme based on the Catppuccin Mocha color palette.
var catppuccin = Theme{
	Name: "catppuccin",
	Colors: ColorPalette{
		// UI colors
		Primary: lipgloss.Color("#CBA6F7"), // mauve
		Accent:  lipgloss.Color("#F5C2E7"), // pink
		Success: lipgloss.Color("#A6E3A1"), // green
		Error:   lipgloss.Color("#F38BA8"), // red
		Warning: lipgloss.Color("#F9E2AF"), // yellow
		Info:    lipgloss.Color("#89DCEB"), // sky

		// Severity colors
		Critical: lipgloss.Color("#F38BA8"), // red
		High:     lipgloss.Color("#FAB387"), // peach
		Medium:   lipgloss.Color("#F9E2AF"), // yellow
		Low:      lipgloss.Color("#89B4FA"), // blue

		// Text colors
		Text:      lipgloss.Color("#CDD6F4"), // text
		TextMuted: lipgloss.Color("#6C7086"), // overlay0
		TextLight: lipgloss.Color("#BAC2DE"), // subtext1
		TextLabel: lipgloss.Color("#A6ADC8"), // subtext0

		// Background colors
		Background:    lipgloss.Color("#1E1E2E"), // base
		BackgroundAlt: lipgloss.Color("#313244"), // surface0
		Border:        lipgloss.Color("#45475A"), // surface1
		Overlay:       lipgloss.Color("#11111B"), // crust

		// Special colors
		White: lipgloss.Color("#CDD6F4"), // text
		Black: lipgloss.Color("#11111B"), // crust
	},
}

// onedark is a theme based on the Atom One Dark color palette.
var onedark = Theme{
	Name: "onedark",
	Colors: ColorPalette{
		// UI colors
		Primary: lipgloss.Color("#61AFEF"), // blue
		Accent:  lipgloss.Color("#C678DD"), // purple
		Success: lipgloss.Color("#98C379"), // green
		Error:   lipgloss.Color("#E06C75"), // red
		Warning: lipgloss.Color("#E5C07B"), // yellow
		Info:    lipgloss.Color("#56B6C2"), // cyan

		// Severity colors
		Critical: lipgloss.Color("#E06C75"), // red
		High:     lipgloss.Color("#D19A66"), // orange
		Medium:   lipgloss.Color("#E5C07B"), // yellow
		Low:      lipgloss.Color("#56B6C2"), // cyan

		// Text colors
		Text:      lipgloss.Color("#ABB2BF"), // fg
		TextMuted: lipgloss.Color("#5C6370"), // comment
		TextLight: lipgloss.Color("#DCDFE4"), // fg light
		TextLabel: lipgloss.Color("#828997"), // gutter

		// Background colors
		Background:    lipgloss.Color("#282C34"), // bg
		BackgroundAlt: lipgloss.Color("#2C323C"), // bg highlight
		Border:        lipgloss.Color("#3E4451"), // border
		Overlay:       lipgloss.Color("#21252B"), // bg dark

		// Special colors
		White: lipgloss.Color("#DCDFE4"),
		Black: lipgloss.Color("#21252B"),
	},
}

// monokai is a theme based on the Monokai color palette.
var monokai = Theme{
	Name: "monokai",
	Colors: ColorPalette{
		// UI colors
		Primary: lipgloss.Color("#66D9EF"), // blue
		Accent:  lipgloss.Color("#AE81FF"), // purple
		Success: lipgloss.Color("#A6E22E"), // green
		Error:   lipgloss.Color("#F92672"), // red/pink
		Warning: lipgloss.Color("#FD971F"), // orange
		Info:    lipgloss.Color("#66D9EF"), // blue

		// Severity colors
		Critical: lipgloss.Color("#F92672"), // red/pink
		High:     lipgloss.Color("#FD971F"), // orange
		Medium:   lipgloss.Color("#E6DB74"), // yellow
		Low:      lipgloss.Color("#66D9EF"), // blue

		// Text colors
		Text:      lipgloss.Color("#F8F8F2"), // fg
		TextMuted: lipgloss.Color("#75715E"), // comment
		TextLight: lipgloss.Color("#F8F8F0"), // fg light
		TextLabel: lipgloss.Color("#CFCFC2"), // gutter

		// Background colors
		Background:    lipgloss.Color("#272822"), // bg
		BackgroundAlt: lipgloss.Color("#3E3D32"), // bg highlight
		Border:        lipgloss.Color("#49483E"), // border
		Overlay:       lipgloss.Color("#1E1F1C"), // bg dark

		// Special colors
		White: lipgloss.Color("#F8F8F2"),
		Black: lipgloss.Color("#1E1F1C"),
	},
}
