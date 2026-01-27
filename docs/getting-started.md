# Getting Started

This guide walks you through installing pyre and connecting to your first firewall.

## Installation

### Download Binary

Download the latest release for your platform from the [Releases](https://github.com/jp2195/pyre/releases) page:

- `pyre-darwin-arm64` - macOS Apple Silicon
- `pyre-darwin-amd64` - macOS Intel
- `pyre-windows-amd64.exe` - Windows

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

Or clone and build:

```bash
git clone https://github.com/jp2195/pyre.git
cd pyre
go build -o pyre ./cmd/pyre
```

## First Connection

There are several ways to connect to a firewall. Choose the method that fits your workflow.

NOTE: If the .exe was not renamed, the command will be pyre-windows-amd64.exe

### Option 1: CLI Flags

Best for quick one-off connections:

```bash
pyre --host firewall.example.com --api-key YOUR_API_KEY
```

Add `--insecure` if the firewall uses a self-signed certificate:

```bash
pyre --host firewall.example.com --api-key YOUR_API_KEY --insecure
```

### Option 2: Environment Variables

Good for CI/CD or when you don't want credentials in command history:

```bash
export PYRE_HOST=firewall.example.com
export PYRE_API_KEY=YOUR_API_KEY
pyre
```

### Option 3: Configuration File

Best for managing multiple firewalls. Create `~/.pyre.yaml`:

```yaml
default: 10.0.0.1

connections:
  10.0.0.1:
    insecure: true

  10.0.0.2:
    insecure: true
```

Then set your API key and run:

```bash
export PYRE_API_KEY=YOUR_API_KEY
pyre
```

With a config file, pyre shows the Connection Hub where you can select which firewall to connect to.

Use `-c` to connect directly to a specific connection:

```bash
pyre -c 10.0.0.1
```

See [Configuration](configuration.md) for all options.

### Option 4: Interactive Login

If no credentials are configured, pyre shows a Quick Connect form:

1. Enter the firewall hostname or IP
2. Enter your username
3. Enter your password

pyre generates an API key via the keygen endpoint and uses it for the session.

## Understanding the Interface

Once connected, you'll see the main dashboard.

### Header

The top of the screen shows:
- **pyre** logo and connection status (green dot = connected)
- **Navigation tabs** - The four main groups (Monitor, Analyze, Tools, Conn)
- **Current view name**

Below the main header is a sub-tab row showing views in the current group.

### Content Area

The middle section displays the current view's content.

### Footer

The bottom shows available keybindings for quick reference.

## Quick Tour

### Monitor Group (Press 1)

Get an at-a-glance view of firewall health:

- **Overview** - System info, resources, HA status, sessions
- **Network** - Interfaces, ARP table, routing
- **Security** - Threats, blocked apps, rule analysis
- **VPN** - IPSec tunnels, GlobalProtect users

### Analyze Group (Press 2)

Dig into detailed data:

- **Policies** - Security rules with filtering and hit counts
- **NAT** - NAT translation rules
- **Sessions** - Active connections
- **Interfaces** - Interface status and counters
- **Logs** - System, traffic, and threat logs

### Tools Group (Press 3)

View configuration status:

- **Config** - Pending changes, rule statistics

### Connections Group (Press 4)

Manage firewall connections:

- **Switch Device** - Open the connection picker

## Next Steps

- [Navigation](navigation.md) - Learn the group-based navigation system
- [Keybindings](keybindings.md) - Full keyboard shortcut reference
- [Configuration](configuration.md) - Configure multiple firewalls
- [View Reference](views/) - Detailed documentation for each view
