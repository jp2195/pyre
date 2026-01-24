# pyre

A terminal user interface (TUI) for managing and monitoring Palo Alto firewalls.

![pyre demo](docs/demo.gif)

## Features

- **Dashboard** - Real-time system info, resource usage, HA status, network, security, and VPN monitoring
- **Security Policies** - Browse, filter, and sort security rules with hit count analysis
- **NAT Policies** - View NAT translation rules and hit counts
- **Active Sessions** - View and filter live sessions with detailed traffic information
- **Logs** - Browse system, traffic, and threat logs with filtering
- **Interfaces** - Monitor interface status, traffic counters, and errors
- **Troubleshooting** - Run interactive troubleshooting runbooks via SSH
- **Panorama Support** - Manage multiple firewalls through Panorama device targeting
- **Multi-Firewall** - Switch between multiple firewall connections
- **Command Palette** - Quick access to any view or action with `Ctrl+P`
- **Theming** - 10 built-in color themes including nord, dracula, catppuccin, and more

## Installation

### Download Binary

Download the latest release from the [Releases](https://github.com/jp2195/pyre/releases) page.

```bash
# macOS/Linux: Make executable and move to PATH
chmod +x pyre-darwin-arm64
sudo mv pyre-darwin-arm64 /usr/local/bin/pyre
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
default_firewall: prod-fw01

firewalls:
  prod-fw01:
    host: 10.0.0.1
    api_key_env: PROD_FW01_API_KEY
    insecure: true

settings:
  theme: "dark"
```

Then run `pyre`.

See [Getting Started](docs/getting-started.md) for more options.

## Navigation

pyre uses a group-based navigation system:

| Key | Group | Views |
|-----|-------|-------|
| `1` | Monitor | Overview, Network, Security, VPN |
| `2` | Analyze | Policies, NAT, Sessions, Interfaces, Logs |
| `3` | Tools | Troubleshoot, Config |
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
- [SSH Setup](docs/ssh-setup.md) - SSH configuration for troubleshooting

### View Reference

- [Dashboard](docs/views/dashboard.md) - Monitor sub-views
- [Policies](docs/views/policies.md) - Security policies
- [NAT](docs/views/nat.md) - NAT policies
- [Sessions](docs/views/sessions.md) - Active sessions
- [Interfaces](docs/views/interfaces.md) - Interface status
- [Logs](docs/views/logs.md) - Log viewer
- [Troubleshoot](docs/views/troubleshoot.md) - Diagnostic runbooks

## License

MIT License - see [LICENSE](LICENSE) for details.
