# pyre

[![Go Version](https://img.shields.io/github/go-mod/go-version/jp2195/pyre)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/jp2195/pyre)](https://github.com/jp2195/pyre/releases)

A terminal user interface (TUI) for managing and monitoring Palo Alto firewalls.

## Why pyre?

The Palo Alto web interface requires clicking through multiple menus to gather basic information—checking system health, then navigating to policies, then sessions, then logs. Each view is a separate page load and context switch.

pyre solves this by combining multiple API calls into unified terminal views. Get system info, HA status, resource usage, and session counts in a single dashboard. Filter policies and sessions instantly with keyboard shortcuts. No more clicking through menus to get the information you need.

**Built for network engineers who want answers fast.**

## Features

- **Dashboard** - Real-time system info, resource usage, HA status, network, security, and VPN monitoring
- **Security Policies** - Browse, filter, and sort security rules with hit count analysis
- **NAT Policies** - View NAT translation rules and hit counts
- **Active Sessions** - View and filter live sessions with detailed traffic information
- **Logs** - Browse system, traffic, and threat logs with filtering
- **Interfaces** - Monitor interface status, traffic counters, and errors
- **Panorama Support** - Manage multiple firewalls through Panorama device targeting
- **Multi-Firewall** - Switch between multiple firewall connections
- **Command Palette** - Quick access to any view or action with `Ctrl+P`
- **Theming** - 10 built-in color themes including nord, dracula, catppuccin, and more

## Installation

### Download Binary

Download the latest release for your platform from the [Releases](https://github.com/jp2195/pyre/releases) page.

**macOS/Linux:**
```bash
chmod +x pyre-darwin-arm64
sudo mv pyre-darwin-arm64 /usr/local/bin/pyre
```

**Windows:**
1. Download `pyre-windows-amd64.exe`
2. Rename to `pyre.exe` and move to a directory in your PATH, or run directly:
```powershell
.\pyre-windows-amd64.exe --host firewall.example.com --api-key YOUR_API_KEY
```

### Build from Source

Requires Go 1.25 or later.

```bash
go install github.com/jp2195/pyre/cmd/pyre@latest
```

## Quick Start

### CLI Flags

```bash
pyre --host firewall.example.com --api-key YOUR_API_KEY
```

### Environment Variables

```bash
export PYRE_HOST=firewall.example.com
export PYRE_API_KEY=YOUR_API_KEY
pyre
```

### Configuration File

Create `~/.pyre.yaml`:

```yaml
default: 10.0.0.1

connections:
  10.0.0.1:
    insecure: true

settings:
  theme: dark
```

Then set your API key and run:

```bash
export PYRE_API_KEY=YOUR_API_KEY
pyre
```

Or use the `-c` flag to connect to a saved connection:

```bash
pyre -c 10.0.0.1
```

See [Getting Started](docs/getting-started.md) for more options.

## Navigation

pyre uses a group-based navigation system:

| Key | Group | Views |
|-----|-------|-------|
| `1` | Monitor | Overview, Network, Security, VPN |
| `2` | Analyze | Policies, NAT, Sessions, Interfaces, Logs |
| `3` | Tools | Config |
| `4` | Connections | Switch Device |

- Press the same number to cycle through views in that group
- Press `Tab` to move to the next view in the current group
- Press `Ctrl+P` to open the command palette

See [Navigation](docs/navigation.md) for details.

## Keybindings

### Global

| Key | Action |
|-----|--------|
| `1-4` | Switch navigation groups |
| `Tab` | Next view in group |
| `Ctrl+P` | Command palette |
| `:` | Connection picker |
| `D` | Device picker (Panorama) |
| `r` | Refresh |
| `?` | Help |
| `q` | Quit |

### View Navigation

| Key | Action |
|-----|--------|
| `j` / `k` | Move up/down |
| `/` | Filter |
| `s` | Cycle sort |
| `Enter` | Expand details |

See [Keybindings](docs/keybindings.md) for the complete reference.

## Documentation

- [Getting Started](docs/getting-started.md) - Installation and first connection
- [Navigation](docs/navigation.md) - How to navigate pyre
- [Keybindings](docs/keybindings.md) - Complete keyboard shortcut reference
- [Configuration](docs/configuration.md) - Configuration file reference
- [Panorama](docs/panorama.md) - Panorama-specific features

### View Reference

- [Dashboard](docs/views/dashboard.md) - Monitor sub-views
- [Policies](docs/views/policies.md) - Security policies
- [NAT](docs/views/nat.md) - NAT policies
- [Sessions](docs/views/sessions.md) - Active sessions
- [Interfaces](docs/views/interfaces.md) - Interface status
- [Logs](docs/views/logs.md) - Log viewer

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.
