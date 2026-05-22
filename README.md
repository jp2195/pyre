<h1 align="center">pyre</h1>

<p align="center">
  <!-- TODO: drop a logo PNG here once one exists (suggested width=350) -->
  <!-- <img src="docs/assets/logo.png" width="350" alt="pyre"><br> -->
  <a href="https://go.dev/"><img src="https://img.shields.io/github/go-mod/go-version/jp2195/pyre" alt="Go Version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="License"></a>
  <a href="https://github.com/jp2195/pyre/releases"><img src="https://img.shields.io/github/v/release/jp2195/pyre" alt="Release"></a>
</p>

<p align="center">
  A terminal UI for Palo Alto firewalls. Answers the questions you'd
  otherwise click through thirty tabs of the PAN-OS web UI to find.
</p>

<p align="center">
  <!-- TODO: drop a demo GIF / hero screenshot here -->
  <!-- <img src="docs/assets/demo.gif" width="100%" alt="pyre in action"> -->
</p>

## Why pyre?

The PAN-OS web interface turns every question into a sequence of clicks
— system info, then policies, then sessions, then logs. Each view is a
separate page load.

pyre combines those calls into unified terminal views: system health,
HA, resource usage, and sessions in one dashboard; instant filter and
sort on policies, sessions, and objects; no context switches. Built
for network engineers who want answers fast.

## Features

- **Dashboards** — system, network, security, VPN at-a-glance
- **Policies, NAT, objects** — browse, filter, sort, hit-count analysis,
  inline detail
- **Sessions, routes, interfaces** — live state with substring filter and
  per-view sort
- **VPN** — IPSec tunnel status + GlobalProtect connected users
- **Logs** — system / traffic / threat with cycle-on-key
- **Panorama** — connect to Panorama and target managed firewalls; the
  same views, scoped per device
- **Multi-firewall** — connection hub + quick picker (`:`)
- **Command palette** — `Ctrl+P` fuzzy-jumps to any view
- **10 themes** — nord, dracula, catppuccin, gruvbox, …

## Install

Download a binary from [Releases](https://github.com/jp2195/pyre/releases).
Archives ship with an SPDX SBOM and a shared `checksums.txt`.

```bash
# macOS / Linux
tar -xzf pyre_<version>_<os>_<arch>.tar.gz
chmod +x pyre
sudo mv pyre /usr/local/bin/pyre
```

Windows: extract the `.zip`, drop `pyre.exe` on your `PATH`.

Or build from source (Go 1.26+):

```bash
go install github.com/jp2195/pyre/cmd/pyre@latest
```

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

> [!IMPORTANT]
> **pyre does not persist credentials.** Supply an API key at each
> invocation via `--api-key`, `PYRE_API_KEY`, or per-host
> `PYRE_<HOST>_API_KEY`. If none is provided, pyre prompts for
> username + password, runs keygen, and uses the resulting key for
> the current session only. `~/.pyre.yaml` never contains credentials.

For private-CA firewalls, use `ca_cert_path: /path/to/ca.pem` in the
connection config instead of `--insecure`.

## What it looks like

<!-- TODO: add per-view screenshots. Suggested set:
     - dashboard.png   (Monitor → Overview, ideally with HA + resources visible)
     - policies.png    (Analyze → Policies with hit counts + a filter active)
     - sessions.png    (Analyze → Sessions with a session detail panel open)
     - objects.png     (Analyze → Objects, ideally on the Service tab)
     - logs.png        (Analyze → Logs, threat tab, with detail expanded)
     - palette.png     (Ctrl+P open with a filter typed)
-->

<!--
<p align="center">
  <img src="docs/assets/dashboard.png" width="48%" alt="Dashboard">
  <img src="docs/assets/policies.png" width="48%" alt="Policies">
</p>
<p align="center">
  <img src="docs/assets/sessions.png" width="48%" alt="Sessions">
  <img src="docs/assets/logs.png"     width="48%" alt="Logs">
</p>
-->

## Navigation

Three numbered groups: `1` Monitor (dashboards), `2` Analyze (list
views), `3` Tools (config). Same number again — or `Tab` — cycles
sub-views in the group. `Ctrl+P` opens a fuzzy command palette that
jumps anywhere.

Inside a list view: `/` filter, `s` cycle sort, `Enter` open detail,
`r` refresh, `?` help, `q` quit.

New to pyre? The **"first 60 seconds"** section of
[Getting Started](docs/getting-started.md) walks the model in one
read. Full key reference: [docs/keybindings.md](docs/keybindings.md).

## Documentation

- [Getting Started](docs/getting-started.md) — install, first
  connection, and a 60-second navigation walkthrough
- [Configuration](docs/configuration.md) — `~/.pyre.yaml`, env vars,
  CLI flags, credential resolution
- [Keybindings & Navigation](docs/keybindings.md) — every key in
  every view
- [Panorama](docs/panorama.md) — managing devices through Panorama
- [View reference](docs/views/README.md) — what each view shows and how
  its filter / sort / detail panel work

## Contributing

Bug reports, feature ideas, and docs PRs all welcome. See
[CONTRIBUTING.md](CONTRIBUTING.md) for the setup and PR process.

## License

Apache 2.0. See [LICENSE](LICENSE).
