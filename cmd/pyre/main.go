package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jp2195/pyre/internal/auth"
	"github.com/jp2195/pyre/internal/config"
	"github.com/jp2195/pyre/internal/tui"
	"github.com/jp2195/pyre/internal/tui/theme"
	"github.com/jp2195/pyre/internal/tui/views"
)

var (
	version = "dev"
)

func main() {
	var (
		host       = flag.String("host", "", "Firewall hostname or IP address")
		apiKey     = flag.String("api-key", "", "API key for authentication")
		insecure   = flag.Bool("insecure", false, "Skip TLS certificate verification (for self-signed certs)")
		configPath = flag.String("config", "", "Path to config file (default: ~/.pyre.yaml)")
		showHelp   = flag.Bool("help", false, "Show help message")
		showVer    = flag.Bool("version", false, "Show version")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "pyre - Palo Alto Firewall TUI\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  pyre [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  PYRE_HOST      Firewall hostname or IP\n")
		fmt.Fprintf(os.Stderr, "  PYRE_API_KEY   API key for authentication\n")
		fmt.Fprintf(os.Stderr, "  PYRE_INSECURE  Skip TLS verification (true/false)\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  pyre --host 10.0.0.1\n")
		fmt.Fprintf(os.Stderr, "  pyre --host fw.example.com --api-key LUFRPT...\n")
		fmt.Fprintf(os.Stderr, "  PYRE_HOST=10.0.0.1 PYRE_API_KEY=LUFRPT... pyre\n")
	}

	flag.Parse()

	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if *showVer {
		fmt.Printf("pyre version %s\n", version)
		os.Exit(0)
	}

	flags := config.CLIFlags{
		Host:     *host,
		APIKey:   *apiKey,
		Insecure: *insecure,
		Config:   *configPath,
	}

	cfg, err := config.LoadWithFlags(flags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Initialize theme system
	theme.Init(cfg.Settings.Theme)
	tui.InitStyles()
	views.InitStyles()

	creds := auth.ResolveCredentials(cfg, flags)

	model := tui.NewModel(cfg, creds)

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running pyre: %v\n", err)
		os.Exit(1)
	}
}
