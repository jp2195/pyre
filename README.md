# pyre

[![Go Version](https://img.shields.io/github/go-mod/go-version/jp2195/pyre)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/jp2195/pyre)](https://github.com/jp2195/pyre/releases)

A terminal user interface for Palo Alto firewalls (PAN-OS).

## Why pyre?

The PAN-OS web interface turns every question into a sequence of clicks — system info, then policies, then sessions, then logs. Each view is a separate page load.

pyre combines those calls into unified terminal views: system health, HA, resource usage, and sessions in one dashboard; instant filter and sort on policies and sessions; no context switches. Built for network engineers who want answers fast.

## Features

- **Dashboard** — live system, resources, HA, network, security, VPN
- **Security & NAT policies** — browse, filter, sort, hit-count analysis
- **Sessions** — active sessions with filtering and detail view
- **Logs** — system / traffic / threat logs
- **Interfaces** — status, counters, errors
- **Panorama** — manage multiple firewalls via device targeting
- **Multi-firewall** — connection hub + quick picker
- **Command palette** — `Ctrl+P` jumps to any view or action
- **Theming** — 10 built-in themes (nord, dracula, catppuccin, …)

## Installation

### Download

Grab a binary for your platform from [Releases](https://github.com/jp2195/pyre/releases). Archives ship with an SPDX SBOM and a shared `checksums.txt`.

```bash
# macOS / Linux
chmod +x pyre-<os>-<arch>
sudo mv pyre-<os>-<arch> /usr/local/bin/pyre
```

Windows: rename to `pyre.exe` and place somewhere on your `PATH`.

### Build from source

Requires Go 1.26 or later.

```bash
go install github.com/jp2195/pyre/cmd/pyre@latest
```

Or clone and use the `Makefile`: `make build` → `./pyre`.

## Quick start

```bash
# one-off
pyre --host firewall.example.com --api-key YOUR_API_KEY

# env var
export PYRE_API_KEY=...
pyre --host firewall.example.com

# saved connection
cat > ~/.pyre.yaml <<'YAML'
default: 10.0.0.1
connections:
  10.0.0.1:
    insecure: true
YAML
pyre -c 10.0.0.1
```

**pyre does not persist credentials.** Supply an API key at each
invocation via `--api-key`, `PYRE_API_KEY`, or per-host
`PYRE_<HOST>_API_KEY`. If none is provided, pyre runs an interactive
login (username/password → keygen) and uses the resulting key for the
current session only. `~/.pyre.yaml` never contains credentials.

For private-CA firewalls, use `ca_cert_path: /path/to/ca.pem` in the
connection config instead of `--insecure`.

See [Getting Started](docs/getting-started.md) for more detail.

## Navigation

Three numbered groups plus the command palette:

| Key      | Group   | Views                                            |
|----------|---------|--------------------------------------------------|
| `1`      | Monitor | Overview, Network, Security, VPN                 |
| `2`      | Analyze | Policies, NAT, Sessions, Interfaces, Logs        |
| `3`      | Tools   | Config                                           |
| `Tab`    |         | next view in the current group                   |
| `Ctrl+P` |         | command palette (jump anywhere)                  |
| `:`      |         | connection picker                                |
| `D`      |         | device picker (Panorama only)                    |

Common keys within a list view: `j`/`k` to move, `/` to filter, `s` to cycle
sort, `Enter` to expand detail, `r` to refresh, `?` for help, `q` to quit.

Full reference: [docs/keybindings.md](docs/keybindings.md).

## Documentation

- [Getting Started](docs/getting-started.md) — install and first connection
- [Configuration](docs/configuration.md) — `~/.pyre.yaml`, env vars, CLI flags
- [Keybindings & Navigation](docs/keybindings.md) — every key in every view
- [Panorama](docs/panorama.md) — managing devices through Panorama
- View reference: [Dashboard](docs/views/dashboard.md) · [Policies](docs/views/policies.md) · [NAT](docs/views/nat.md) · [Sessions](docs/views/sessions.md) · [Interfaces](docs/views/interfaces.md) · [Logs](docs/views/logs.md)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Apache 2.0. See [LICENSE](LICENSE).
