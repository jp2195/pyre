# Getting Started

## Install

Grab a binary from [Releases](https://github.com/jp2195/pyre/releases)
or `go install github.com/jp2195/pyre/cmd/pyre@latest` (Go 1.26+).
The [README install section](../README.md#install) has the
copy-paste shell snippet. Each release archive ships with an SPDX
SBOM and `checksums.txt`.

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

## Your first 60 seconds

You've connected. You're staring at the Dashboard. Here's a quick tour
to get you productive without reading the full key reference.

**1. The screen has three parts.** Top header shows your connection,
the active navigation group, and sub-tabs. Body shows the current view.
Footer shows the keys that apply right now — when in doubt, look down.

**2. Three numbered groups, hit a number to jump.**

- `1` Monitor — dashboards (system health, network, security, VPN)
- `2` Analyze — list views (policies, NAT, objects, sessions, interfaces,
  routes, IPSec tunnels, GP users, logs)
- `3` Tools — config dashboard

Press the same number again, or `Tab`, to cycle through sub-views in
that group. Try `2`, `2`, `2` to walk through Policies → NAT → Objects.

**3. Inside a list view, four keys do almost everything:**

- `/` filter (substring match, case-insensitive)
- `s` cycle sort field
- `Enter` open the detail panel for the highlighted row
- `r` refresh

Try it: press `2` to land on Policies, `/web` to filter for "web", `s`
to flip sort fields, `Enter` to see the full rule, `Esc` to close.

**4. `Ctrl+P` jumps anywhere.** Don't memorize keybindings. Type the
view name (or any partial — `obj`, `sess`, `logs`) and Enter. Same
muscle memory as VS Code's command palette.

**5. Two modal pickers.**

- `:` opens the connection picker — switch between saved firewalls.
- `d` opens the device picker — switch between managed devices when
  you're connected to a Panorama. (On a standalone firewall, `d` falls
  through to the view's own handlers.)

**6. `?` toggles help.** `q` or `Ctrl+C` quits.

That's the whole navigation model. Everything else is a refinement.

## Next steps

- [Keybindings & Navigation](keybindings.md) — the full key reference
  for every view and every modal
- [View Reference](views/README.md) — what each view shows, what its
  filter / sort fields are, what the detail panel reveals
- [Configuration](configuration.md) — all options in `~/.pyre.yaml`
- [Panorama](panorama.md) — connecting to Panorama and targeting
  managed devices
