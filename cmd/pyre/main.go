package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

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
		user       = flag.String("user", "", "Username for authentication (prompts for password)")
		apiKey     = flag.String("api-key", "", "API key for authentication")
		insecure   = flag.Bool("insecure", false, "Skip TLS certificate verification (for self-signed certs)")
		configPath = flag.String("config", "", "Path to config file (default: ~/.pyre.yaml)")
		connection = flag.String("c", "", "Connect to a named connection from config")
		debug      = flag.Bool("debug", false, "Enable debug logging to ~/.pyre/logs/debug.log")
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
		fmt.Fprintf(os.Stderr, "  DEBUG          Enable debug logging (same as --debug)\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  pyre                                    # Connection hub (if config exists)\n")
		fmt.Fprintf(os.Stderr, "  pyre -c myfw                            # Connect to 'myfw' from config\n")
		fmt.Fprintf(os.Stderr, "  pyre --host 10.0.0.1\n")
		fmt.Fprintf(os.Stderr, "  pyre --host fw.example.com --user admin --insecure  # Prompts for password\n")
		fmt.Fprintf(os.Stderr, "  pyre --host fw.example.com --api-key LUFRPT...\n")
		fmt.Fprintf(os.Stderr, "  PYRE_HOST=10.0.0.1 PYRE_API_KEY=LUFRPT... pyre\n")
		fmt.Fprintf(os.Stderr, "  pyre --debug                            # Enable debug logging\n")
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

	// Always redirect the default logger away from stderr so log.Printf
	// calls (e.g. in internal/api/) don't bleed into the TUI.
	if *debug || os.Getenv("DEBUG") != "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			logDir := filepath.Join(homeDir, ".pyre", "logs")
			if err := os.MkdirAll(logDir, 0700); err == nil {
				logPath := filepath.Join(logDir, "debug.log")
				f, err := tea.LogToFile(logPath, "pyre")
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not enable debug logging: %v\n", err)
					log.SetOutput(io.Discard)
				} else {
					defer f.Close() //nolint:errcheck // best-effort log file close
					fmt.Fprintf(os.Stderr, "Debug logging: %s\n", logPath)
				}
			}
		}
	} else {
		// No debug flag: silence all log output to prevent TUI corruption
		log.SetOutput(io.Discard)
	}

	flags := config.CLIFlags{
		Host:       *host,
		Username:   *user,
		APIKey:     *apiKey,
		Insecure:   *insecure,
		Config:     *configPath,
		Connection: *connection,
	}

	cfg, err := config.LoadWithFlags(flags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Load state
	state, err := config.LoadState()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load state: %v\n", err)
		state = &config.State{Connections: make(map[string]config.ConnectionState)}
	}

	// Initialize theme system
	theme.Init(cfg.Settings.Theme)
	tui.InitStyles()
	views.InitStyles()

	creds := auth.ResolveCredentials(cfg, flags)

	// Determine starting view based on flags and config
	startView := determineStartView(cfg, flags, creds)

	// Handle -c flag: validate connection exists (flags.Connection is the host)
	if flags.Connection != "" {
		conn, ok := cfg.GetConnection(flags.Connection)
		if !ok {
			fmt.Fprintf(os.Stderr, "Error: connection %q not found in config\n", flags.Connection)
			os.Exit(1)
		}
		// Set up credentials for the specified connection
		creds.Host = flags.Connection // Host is now the key
		creds.Insecure = conn.Insecure
		creds.PromptForPassword = true
	}

	model := tui.NewModel(cfg, state, creds, startView)

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

// determineStartView decides which view to show first based on CLI flags and config
func determineStartView(cfg *config.Config, flags config.CLIFlags, creds *auth.Credentials) tui.ViewState {
	// If --host flag or PYRE_HOST is set, go to login
	if flags.Host != "" || os.Getenv("PYRE_HOST") != "" {
		return tui.ViewLogin
	}

	// If -c flag is set, go to login for that connection
	if flags.Connection != "" {
		return tui.ViewLogin
	}

	// If we have full credentials (API key from env), go straight to dashboard
	if creds.HasAPIKey() && creds.HasHost() {
		return tui.ViewDashboard
	}

	// If config has connections, show the connection hub
	if cfg.HasConnections() {
		return tui.ViewConnectionHub
	}

	// No config, show quick connect form
	return tui.ViewConnectionForm
}
