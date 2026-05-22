# Getting Started

## Install

Download a binary from [Releases](https://github.com/jp2195/pyre/releases):

- `pyre_<version>_darwin_arm64.tar.gz` — macOS Apple Silicon
- `pyre_<version>_darwin_amd64.tar.gz` — macOS Intel
- `pyre_<version>_linux_amd64.tar.gz` — Linux x86-64
- `pyre_<version>_linux_arm64.tar.gz` — Linux ARM64
- `pyre_<version>_windows_amd64.zip` — Windows x86-64
- `pyre_<version>_windows_arm64.zip` — Windows ARM64

Each archive ships with an SPDX-JSON SBOM next to it, and `checksums.txt`
covers every archive.

```bash
# macOS / Linux
tar -xzf pyre_<version>_<os>_<arch>.tar.gz
chmod +x pyre
sudo mv pyre /usr/local/bin/pyre
```

Windows: extract the `.zip`, drop `pyre.exe` somewhere on your `PATH`.

### From source

Requires Go 1.26 or later.

```bash
go install github.com/jp2195/pyre/cmd/pyre@latest
```

## Connect

Pick whichever fits your workflow:

### 1. CLI flags

```bash
pyre --host firewall.example.com --api-key YOUR_API_KEY
# Self-signed cert:
pyre --host firewall.example.com --api-key YOUR_API_KEY --insecure
```

### 2. Environment variables

```bash
export PYRE_HOST=firewall.example.com
export PYRE_API_KEY=YOUR_API_KEY
pyre
```

Per-host env var also works: `PYRE_<HOST>_API_KEY` where `<HOST>` is the
host uppercased with `.` and `-` turned into `_`. Useful when you have
several firewalls and don't want a single shared `PYRE_API_KEY`.

### 3. Configuration file

Create `~/.pyre.yaml`:

```yaml
default: 10.0.0.1
connections:
  10.0.0.1:
    insecure: true
  10.0.0.2:
    insecure: true
```

Run `pyre` to get the Connection Hub (lets you pick), or `pyre -c 10.0.0.1`
to jump directly to one connection.

pyre does not persist credentials. The config file never contains API
keys or passwords; each invocation sources a key from `--api-key`,
`PYRE_API_KEY`, or `PYRE_<HOST>_API_KEY`. See
[Configuration](configuration.md) for the full reference.

### 4. Interactive login

If no key is found via any of the above, pyre shows a Quick Connect form:

1. hostname / IP
2. username
3. password

pyre runs keygen against the firewall and uses the returned API key for
the current session. The key is **not** saved anywhere — next launch
will prompt again unless you supply a key via env var or CLI flag.

## The interface

After connect you land on the dashboard:

- **Header** — pyre logo, connection indicator, navigation tabs, sub-tabs
- **Body** — the current view
- **Footer** — contextual keybindings

### The three groups

Press `1`, `2`, or `3` to switch groups. Press the same number again (or
`Tab`) to cycle through views within a group.

- **1 — Monitor**: Overview · Network · Security · VPN
- **2 — Analyze**: Policies · NAT · Objects · Sessions · Interfaces · Routes · IPSec · GP Users · Logs
- **3 — Tools**: Config

`Ctrl+P` opens the command palette — type to jump anywhere. `:` opens
the connection picker (switch between firewalls). `D` opens the device
picker (Panorama only).

## Next steps

- [Configuration](configuration.md) — all options in `~/.pyre.yaml`
- [Keybindings & Navigation](keybindings.md)
- [Panorama](panorama.md) — device targeting
- [View Reference](views/) — per-view docs
