# Configuration

pyre uses a YAML configuration file for managing connections and settings.

## Configuration File Location

pyre looks for configuration at `~/.pyre.yaml`.

You can specify a custom path with the `--config` flag:

```bash
pyre --config /path/to/config.yaml
```

## Full Configuration Example

```yaml
# Default connection (host/IP)
default: 10.0.0.1

# Connections keyed by host/IP
connections:
  10.0.0.1:
    username: admin              # Username for login
    type: firewall               # "firewall" or "panorama"
    insecure: true               # Skip TLS verification

  panorama.example.com:
    type: panorama
    insecure: true

# Global settings
settings:
  session_page_size: 50         # Sessions per page
  theme: dark                   # Color theme
  default_view: dashboard       # Initial view on connect
```

## Connection Options

Connections are keyed by host/IP address. Each connection supports:

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `username` | string | | Username for API/login authentication |
| `type` | string | firewall | Connection type: `firewall` or `panorama` |
| `insecure` | bool | false | Skip TLS certificate verification |

### TLS Certificate Verification

Set `insecure: true` to skip TLS certificate verification. This is common for firewalls with self-signed certificates:

```yaml
connections:
  10.0.0.1:
    insecure: true
```

## Settings Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `session_page_size` | int | 50 | Number of sessions per page |
| `theme` | string | dark | Color theme (see Theme Options) |
| `default_view` | string | dashboard | Initial view on connect |

### Default View Options

Valid values for `default_view`:
- `dashboard` - Overview dashboard (default)
- `policies` - Security policies
- `sessions` - Active sessions
- `logs` - Log viewer
- `interfaces` - Interface status

### Theme Options

pyre includes 10 built-in color themes:

| Theme | Description |
|-------|-------------|
| `dark` | Default dark theme with purple accents |
| `light` | Light theme with purple accents |
| `nord` | Arctic, north-bluish color palette |
| `dracula` | Dark theme with vibrant colors |
| `solarized` | Solarized Dark color scheme |
| `gruvbox` | Retro groove with warm colors |
| `tokyonight` | Clean dark theme with blue tones |
| `catppuccin` | Soothing pastel theme (Mocha variant) |
| `onedark` | Atom One Dark inspired |
| `monokai` | Classic Monokai editor colors |

Example:

```yaml
settings:
  theme: catppuccin
```

## Configuration Precedence

Values are resolved in this order (highest priority first):

1. **CLI flags** (`--host`, `--api-key`, `--insecure`)
2. **Environment variables** (`PYRE_HOST`, `PYRE_API_KEY`)
3. **Configuration file** (~/.pyre.yaml)
4. **Default values**

This means CLI flags override everything, and environment variables override config file values.

## Environment Variables

| Variable | Description |
|----------|-------------|
| `PYRE_HOST` | Firewall hostname or IP |
| `PYRE_API_KEY` | API key for authentication |
| `PYRE_INSECURE` | Skip TLS verification (true/false) |

## CLI Flags

| Flag | Description |
|------|-------------|
| `--host` | Firewall hostname or IP |
| `--user` | Username for login |
| `--api-key` | API key for authentication |
| `--insecure` | Skip TLS verification |
| `--config` | Path to config file |
| `-c` | Connect to a saved connection by host |

## Examples

### Minimal Configuration

Connect to a single firewall:

```yaml
connections:
  192.168.1.1:
    insecure: true
```

Then connect with:

```bash
export PYRE_API_KEY=YOUR_API_KEY
pyre -c 192.168.1.1
```

### Multiple Connections

Manage several firewalls:

```yaml
default: 10.1.0.1

connections:
  10.1.0.1:
    username: admin
    insecure: true

  10.2.0.1:
    username: admin
    insecure: true

  192.168.100.1:
    username: labadmin
    insecure: true
```

### Panorama

Connect to Panorama and target managed firewalls:

```yaml
default: panorama.example.com

connections:
  panorama.example.com:
    type: panorama
    insecure: true
```

Use `D` in pyre to select a managed firewall to target.

## Connection Hub

When you run `pyre` without flags and have connections configured, you'll see the Connection Hub. This lets you:

- View all saved connections
- See last connected time and user
- Connect to any saved connection
- Add new connections
- Delete connections

Press `n` to add a new connection or `Enter` to connect to the selected one.

## State File

pyre stores connection state (last connected time, user) in `~/.pyre/state.json`. This file is managed automatically and doesn't need to be edited.
