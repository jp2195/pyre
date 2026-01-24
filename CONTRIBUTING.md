# Contributing to pyre

Thank you for your interest in contributing to pyre! This document provides guidelines for development and contribution.

## Development Setup

### Prerequisites

- Go 1.25 or later
- macOS 12 Monterey+ (if on macOS)

### Clone and Build

```bash
git clone https://github.com/jp2195/pyre.git
cd pyre
go mod download
go build -o pyre ./cmd/pyre
```

### Run Tests

```bash
go test -v ./...
```

## Project Architecture

```
pyre/
├── cmd/
│   ├── pyre/              # Main application entry point
│   └── pyre-demo/         # Mock server for development
├── internal/
│   ├── api/               # PAN-OS XML API client
│   │   ├── client.go      # Base HTTP client
│   │   ├── firewall.go    # Firewall API methods
│   │   └── panorama.go    # Panorama API methods
│   ├── auth/              # Authentication and session management
│   │   ├── auth.go        # Session state, connection handling
│   │   └── keygen.go      # API key generation
│   ├── config/            # Configuration loading
│   │   └── config.go      # YAML config, defaults, CLI overrides
│   ├── models/            # Data structures
│   │   ├── device.go      # SystemInfo, Resources, HAStatus
│   │   ├── policy.go      # SecurityRule
│   │   ├── nat.go         # NATRule
│   │   ├── session.go     # Session, SessionInfo
│   │   └── logs.go        # LogEntry types
│   ├── ssh/               # SSH client for troubleshooting
│   │   └── client.go      # SSH connection and command execution
│   ├── troubleshoot/      # Troubleshooting runbooks
│   │   ├── engine.go      # Runbook execution engine
│   │   └── patterns.go    # Output parsing patterns
│   └── tui/               # Bubbletea TUI
│       ├── app.go         # Main model, view switching
│       ├── keys.go        # Keybindings
│       ├── styles.go      # Lipgloss styles
│       └── views/         # Individual view components
│           ├── dashboard.go
│           ├── dashboard_config.go
│           ├── dashboard_network.go
│           ├── dashboard_security.go
│           ├── dashboard_vpn.go
│           ├── device_picker.go
│           ├── interfaces.go
│           ├── login.go
│           ├── logs.go
│           ├── nat_policies.go
│           ├── navbar.go
│           ├── picker.go
│           ├── policies.go
│           ├── sessions.go
│           └── troubleshoot.go
├── docs/                  # Documentation
│   ├── getting-started.md
│   ├── navigation.md
│   ├── keybindings.md
│   ├── configuration.md
│   ├── panorama.md
│   ├── ssh-setup.md
│   └── views/             # Per-view documentation
```

## Running with pyre-demo

The `pyre-demo` command starts mock API and SSH servers for development without a real firewall:

```bash
# Build and run the demo server
go build -o pyre-demo ./cmd/pyre-demo
./pyre-demo

# In another terminal, connect pyre to the mock server
./pyre --host localhost:8443 --api-key demo-key --insecure
```

The demo server provides:
- Mock PAN-OS XML API responses
- Simulated system info, policies, sessions, logs
- Mock SSH server for troubleshooting

## Adding New Views

1. Create a new file in `internal/tui/views/` (e.g., `myview.go`)
2. Define the model struct with required fields:

```go
type MyViewModel struct {
    data    []MyDataType
    err     error
    cursor  int
    width   int
    height  int
}

func NewMyViewModel() MyViewModel {
    return MyViewModel{}
}

func (m MyViewModel) SetSize(width, height int) MyViewModel {
    m.width = width
    m.height = height
    return m
}

func (m MyViewModel) Update(msg tea.Msg) (MyViewModel, tea.Cmd) {
    // Handle key events
    return m, nil
}

func (m MyViewModel) View() string {
    // Render the view
    return ""
}
```

3. Add the view to `internal/tui/app.go`:
   - Add field to `Model` struct
   - Add `ViewState` constant
   - Initialize in `NewModel`
   - Add case in `Update` for view switching
   - Add case in `View` for rendering

4. Add to navigation in `internal/tui/views/navbar.go`:
   - Add `NavItem` to appropriate group

5. Add keybinding in `internal/tui/keys.go` if needed

6. Document the view in `docs/views/`

## Adding New Runbooks

Troubleshooting runbooks are defined in `internal/troubleshoot/runbooks/`. Each runbook is a YAML file:

```yaml
name: My Runbook
description: Description of what this runbook does
category: network  # network, security, system, vpn

steps:
  - name: Step Name
    description: What this step does
    command: show system info
    parse: text  # text, table, or custom parser name

  - name: Conditional Step
    description: Only runs if condition met
    command: show interface {{ .interface }}
    condition: "{{ .previous.status }} == 'down'"
    inputs:
      - name: interface
        prompt: "Enter interface name"
        default: ethernet1/1
```

## Code Style

- Follow standard Go conventions
- Use `go fmt` before committing
- Keep functions focused and small
- Add comments for exported types and functions
- Handle errors explicitly, don't panic
- Use context propagation for cancellation

## Pull Request Process

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Make your changes
4. Run tests (`go test ./...`)
5. Run linting (`go vet ./...`)
6. Commit with descriptive messages
7. Push to your fork
8. Open a Pull Request

### PR Guidelines

- Keep PRs focused on a single change
- Include tests for new functionality
- Update documentation if needed
- Describe what the PR does and why

## Reporting Issues

When reporting bugs, please include:
- pyre version (`pyre --version`)
- Go version (`go version`)
- OS and version
- Steps to reproduce
- Expected vs actual behavior
- Relevant log output (sanitized of sensitive data)

## Questions?

Open an issue with the `question` label or start a discussion.
