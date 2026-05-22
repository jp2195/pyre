# Configuration

pyre reads `~/.pyre.yaml` for connection definitions and UI settings.
**pyre does not store credentials.** API keys and passwords are never
written to `~/.pyre.yaml` or anywhere else on disk — you supply them at
each invocation via env var, CLI flag, or the interactive login flow
(session-only). See [Credentials](#credentials) below.

## File location

Default: `~/.pyre.yaml`. Override with `--config /path/to/file.yaml`.

## Example

```yaml
# Host/IP of the default connection
default: 10.0.0.1

# Keyed by host/IP
connections:
  10.0.0.1:
    username: admin          # for the interactive-login flow
    type: firewall           # firewall (default) or panorama
    insecure: true           # skip TLS cert verification (lab only)

  firewall-with-private-ca.example.com:
    type: firewall
    ca_cert_path: /etc/pyre/corp-ca.pem   # verify against this CA

  panorama.example.com:
    type: panorama
    insecure: true

# Global UI settings
settings:
  session_page_size: 50
  theme: catppuccin
```

## Connection options

| Option         | Type   | Default    | Description                                               |
|----------------|--------|------------|-----------------------------------------------------------|
| `username`     | string | —          | Username for interactive login / keygen                   |
| `type`         | string | `firewall` | `firewall` or `panorama`                                  |
| `insecure`     | bool   | `false`    | Skip TLS certificate verification                         |
| `ca_cert_path` | string | —          | Path to a PEM CA bundle; used instead of system roots     |

`insecure: true` and `ca_cert_path` are mutually exclusive — if both are
set, `insecure` wins. Prefer `ca_cert_path` in production; use
`insecure` only for lab gear with self-signed certs. If `ca_cert_path`
is set but the file can't be read or parsed, pyre exits with an error
rather than silently falling back to system roots.

## Global settings

| Option              | Type   | Default     | Description                          |
|---------------------|--------|-------------|--------------------------------------|
| `session_page_size` | int    | 50          | Sessions per page                    |
| `theme`             | string | `dark`      | Color theme (see below)              |

Themes: `dark`, `light`, `nord`, `dracula`, `solarized`, `gruvbox`,
`tokyonight`, `catppuccin`, `onedark`, `monokai`. Unrecognized names
fall back to `dark`.

## Credentials

pyre resolves an API key for a host in this order (first hit wins):

1. `--api-key` CLI flag
2. `PYRE_API_KEY` env var (global)
3. `PYRE_<HOST>_API_KEY` env var (host-scoped; `<HOST>` is the host
   uppercased with `.` and `-` replaced by `_`)
4. Interactive login — pyre prompts for username + password, runs
   keygen against the firewall, and uses the returned key for the
   current session. The key is **not** saved.

pyre never writes credentials to disk, no keychain, no token cache.
If you want credentials to survive reboots, use env vars (in your
shell profile, direnv, a password manager, etc.) — that's the user's
responsibility, not pyre's. Credentials are zeroed in memory on
disconnect. The fields that could hold them (`APIKey`, `Password`) are
marked `yaml:"-"` so they cannot leak into `~/.pyre.yaml` via `Save`.

`~/.pyre.yaml` is expected to be `0600`. Permissive modes trigger a
startup warning.

## Environment variables

| Variable                | Purpose                                           |
|-------------------------|---------------------------------------------------|
| `PYRE_HOST`             | Default host when no `--host` / `-c` is given     |
| `PYRE_API_KEY`          | API key (applies to any host)                     |
| `PYRE_<HOST>_API_KEY`   | Host-scoped API key                               |
| `PYRE_INSECURE`         | `true` to skip TLS verification                   |
| `PYRE_DEBUG`            | `1` or `true` to enable per-request API logging   |

`PYRE_DEBUG` is off by default because traces include xpath, op-command
bodies, and response previews that are useful for debugging but noisy
in normal use.

## CLI flags

| Flag         | Purpose                                         |
|--------------|-------------------------------------------------|
| `--host`     | Firewall hostname or IP                         |
| `--user`     | Username for interactive login                  |
| `--api-key`  | API key                                         |
| `--insecure` | Skip TLS verification                           |
| `--config`   | Path to config file (default `~/.pyre.yaml`)    |
| `-c`         | Connect to a saved connection by host/IP        |

## Precedence

Highest to lowest:

1. CLI flags
2. Environment variables (including host-scoped `PYRE_<HOST>_*`)
3. `~/.pyre.yaml`
4. Built-in defaults

## Connection Hub

Run `pyre` with a config but no specific `-c`/`--host` and you get the
Connection Hub: saved connections, last-connected time, last user.
Keys: see [keybindings.md](keybindings.md#connection-hub-launch-screen).

## State file

pyre writes `~/.pyre/state.json` with last-connected time and user.
Managed automatically; editing it isn't supported.
