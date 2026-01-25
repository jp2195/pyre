# Configuration

pyre uses a YAML configuration file for managing firewall connections and settings.

## Configuration File Locations

pyre looks for configuration in this order:

1. `$PYRE_CONFIG` environment variable (if set)
2. `~/.pyre.yaml`
3. `~/.config/pyre/config.yaml`

## Full Configuration Example

```yaml
# Default firewall to connect to on startup
default_firewall: prod-fw01

# Firewall definitions
firewalls:
  prod-fw01:
    host: fw1.example.com        # Hostname or IP
    port: 443                    # API port (default: 443)
    api_key: LUFRPT...           # Direct API key
    insecure: false              # Verify TLS certificate
    ssh_user: admin              # SSH username for troubleshooting
    ssh_port: 22                 # SSH port (default: 22)
    ssh_key_file: ~/.ssh/fw_key  # SSH private key path
    ssh_password_env: FW1_SSH_PASS  # Env var for SSH password

  prod-fw02:
    host: fw2.example.com
    api_key_env: FW2_API_KEY     # Read API key from environment
    insecure: true               # Skip TLS verification

  panorama:
    host: panorama.example.com
    api_key_env: PANORAMA_API_KEY
    insecure: true

# Global settings
settings:
  refresh_interval: 5s          # Auto-refresh interval
  session_page_size: 50         # Sessions per page
  log_page_size: 100            # Log entries per page
  theme: dark                   # Color theme (see Theme Options)
  default_view: dashboard       # Initial view on connect
  ssh_known_hosts: ~/.ssh/known_hosts  # Known hosts file
```

## Firewall Entry Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `host` | string | required | Hostname or IP address |
| `port` | int | 443 | API port number |
| `api_key` | string | | API key (direct value) |
| `api_key_env` | string | | Environment variable containing API key |
| `insecure` | bool | false | Skip TLS certificate verification |
| `ssh_user` | string | | SSH username for troubleshooting |
| `ssh_port` | int | 22 | SSH port number |
| `ssh_key_file` | string | | Path to SSH private key |
| `ssh_password_env` | string | | Env var containing SSH password |

### API Key Options

You can specify the API key in two ways:

**Direct value:**
```yaml
firewalls:
  myfw:
    host: firewall.example.com
    api_key: LUFRPT14ZlpoYTNwL2...
```

**Environment variable reference:**
```yaml
firewalls:
  myfw:
    host: firewall.example.com
    api_key_env: MY_FW_API_KEY
```

Using `api_key_env` is recommended for security, as the key isn't stored in the config file.

### TLS Certificate Verification

Set `insecure: true` to skip TLS certificate verification. This is common for firewalls with self-signed certificates:

```yaml
firewalls:
  myfw:
    host: firewall.example.com
    api_key_env: MY_FW_API_KEY
    insecure: true
```

## Settings Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `refresh_interval` | duration | 5s | How often to auto-refresh data |
| `session_page_size` | int | 50 | Number of sessions per page |
| `log_page_size` | int | 100 | Number of log entries per page |
| `theme` | string | dark | Color theme (see Theme Options) |
| `default_view` | string | dashboard | Initial view on connect |
| `ssh_known_hosts` | string | ~/.ssh/known_hosts | SSH known hosts file |

### Duration Format

Duration values use Go's duration format:
- `5s` - 5 seconds
- `1m` - 1 minute
- `30s` - 30 seconds
- `1m30s` - 1 minute 30 seconds

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
  theme: "catppuccin"
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
| `PYRE_CONFIG` | Path to configuration file |
| `PYRE_HOST` | Firewall hostname or IP |
| `PYRE_API_KEY` | API key |

## Examples

### Minimal Configuration

Connect to a single firewall:

```yaml
firewalls:
  main:
    host: 192.168.1.1
    api_key_env: FW_API_KEY
    insecure: true
```

### Multiple Firewalls

Manage several firewalls with different credentials:

```yaml
default_firewall: prod-dc1

firewalls:
  prod-dc1:
    host: 10.1.0.1
    api_key_env: DC1_API_KEY
    ssh_user: admin
    ssh_key_file: ~/.ssh/palo_key

  prod-dc2:
    host: 10.2.0.1
    api_key_env: DC2_API_KEY
    ssh_user: admin
    ssh_key_file: ~/.ssh/palo_key

  lab:
    host: 192.168.100.1
    api_key_env: LAB_API_KEY
    insecure: true
```

### Panorama with Managed Firewalls

Connect through Panorama:

```yaml
default_firewall: panorama

firewalls:
  panorama:
    host: panorama.example.com
    api_key_env: PANORAMA_API_KEY
    insecure: true
```

Use `D` in pyre to select a managed firewall to target.

### SSH for Troubleshooting

Enable SSH for troubleshooting runbooks:

```yaml
firewalls:
  prod-fw01:
    host: firewall.example.com
    api_key_env: FW_API_KEY
    ssh_user: admin
    ssh_key_file: ~/.ssh/palo_alto_key
    ssh_port: 22
```

See [SSH Setup](ssh-setup.md) for more details.
