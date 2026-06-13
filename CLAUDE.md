# Pyre - Claude Code Instructions

## Project Overview

Pyre is a TUI for Palo Alto firewalls (PAN-OS), written in Go with Bubbletea.

## Build & Test

```bash
go build ./cmd/pyre          # Build the binary
go test ./...                 # Run all tests
go test -race -v ./...        # Run tests with race detector (CI mode)
go vet ./...                  # Run static analysis
go fix ./...                  # Apply modernizers (safe, behavior-preserving)
```

## Architecture

- `cmd/pyre/` - Entry point
- `internal/api/` - PAN-OS XML API client (XPath config, op commands, log polling)
- `internal/tui/` - Bubbletea app: `app.go` (main Update dispatcher), `handlers.go` (key handlers), `navigation.go` (table-driven nav), `commands.go` (tea.Cmd factories), `render.go` (header/footer)
- `internal/tui/views/` - Individual view models (policies, logs, sessions, dashboards, etc.)
- `internal/tui/theme/` - Theme/color system
- `internal/models/` - Data models
- `internal/config/` - Config/state persistence (~/.pyre.yaml)
- `internal/auth/` - Session/connection management
- SSH access is intentionally removed; Pyre currently uses only the PAN-OS XML API. SSH may return in a future redesign.

## Key Patterns

- **Value receivers** on Bubbletea Models (immutable Update pattern), **pointer receivers** for mutation + Cmd return
- Generic `fetchCmd[T any]()` helper in `commands.go` reduces fetch boilerplate
- Generic `RuleListModel[T]`/`RuleListConfig[T]` in `views/rule_list.go` powers the policies, NAT, interfaces, IPSec tunnels, GP users, and sessions views; per-view wrappers hold only config + format/render functions. Logs intentionally keeps a custom shell (tab bar + cross-type filtering).
- Generic `fetchRulesFromPaths[T any]()` in `api/policies.go` for XPath rule fetching
- `saveConfig()` / `saveState()` return `tea.Cmd` (avoid goroutine race conditions)
- `setError()` is a value receiver that returns an updated Model with `m.err` set plus an auto-dismiss tick Cmd
- Navigation has a single source of truth: the ordered `navDefs` table in `tui/navigation.go` derives the navbar groups (`navbarGroups()`), `navTargets`, and `viewToNavbar`. Adding a nav item = one `navDefs` entry; `views.NewNavbarModel(groups)` takes the groups as a parameter.
- Format helpers shared in `views/format_helpers.go`
- **Bubble Tea v2 View composition**: only the top-level `tui.Model.View()` returns `tea.View`; every sub-view model returns `string`. The top-level composes sub-view strings and sets program options (alt-screen, mouse mode, window title, cursor) on the returned `tea.View` rather than on `tea.NewProgram`.
- Use `tea.KeyPressMsg` in key handler type switches (not `tea.KeyMsg`, which in v2 is the union interface of press and release). Construct test messages as `tea.KeyPressMsg{Code: tea.KeyDown}` or `tea.KeyPressMsg{Code: 'j', Text: "j"}` — `Runes`/`Type` from v1 no longer exist.
- Theme palette fields are `image/color.Color`, not a string alias. Construct concrete values via `lipgloss.Color("#RRGGBB")`.

## Code Style

- Go standard: tabs for indentation
- Use modern Go idioms (see Go 1.26 features below)
- Prefer `for range N` over `for i := 0; i < N; i++` when index is unused
- Prefer `for i := range N` over `for i := 0; i < N; i++` when index is used
- Use `max()`/`min()` builtins instead of manual if/else clamping
- Use `wg.Go(func() { ... })` instead of `wg.Add(1); go func() { defer wg.Done(); ... }()`

## Credential Resolution

pyre does not persist credentials — no keychain, no token cache. API keys
are resolved per invocation in this order (see `auth.ResolveCredentials`):

1. CLI flag `--api-key` (or `flags.APIKey`).
2. Environment variable `PYRE_API_KEY`.
3. Host-specific environment variable `PYRE_<HOST>_API_KEY`, where `<HOST>`
   is the connection host uppercased with `.` and `-` replaced by `_`.
4. Fall through to `Credentials.PromptForPassword = true` so the TUI
   prompts for username/password and runs keygen. The returned API key
   lives in session memory only; it is not saved anywhere.

Credential fields (`APIKey`, `Password`) on `config.ConnectionConfig` carry
`yaml:"-"` tags, so they cannot round-trip to `~/.pyre.yaml`. Connections
are zeroed in `Session.RemoveConnection`. `TestConfig_DoesNotPersistCredentials`
regression-guards the yaml tag.

## Debugging

Two independent debug knobs:

- `--debug` flag (or `DEBUG` env var) routes Go's standard logger to
  `~/.pyre/logs/debug.log` via `tea.LogToFile` (`cmd/pyre/main.go`). Without
  it the logger is set to `io.Discard` so stray `log.Printf` output can't
  corrupt the TUI.
- `PYRE_DEBUG=1` (or `PYRE_DEBUG=true`) turns on per-request API trace logging
  in `internal/api/client.go` (`debugf`): request type, action, xpath, target
  serial, op command bodies, response status/timing, and a response preview.
  Off by default — these fields may contain PAN-OS config paths and command
  bodies (sensitive).

The two compose: `PYRE_DEBUG` traces are written through the standard logger,
so they only land in a file when `--debug`/`DEBUG` is **also** set (otherwise
the logger is `io.Discard` and the traces go nowhere). Error-path `log.Printf`
calls always fire regardless of `PYRE_DEBUG`, so unexpected failures are never
swallowed — though they too are discarded unless `--debug`/`DEBUG` points the
logger at a file.

## Dependencies

- TUI: Bubble Tea v2 (`charm.land/bubbletea/v2`), lipgloss v2 (`charm.land/lipgloss/v2`), bubbles v2 (`charm.land/bubbles/v2`). Migrated from `github.com/charmbracelet/{bubbletea,lipgloss,bubbles}` on 2026-04-18.
- YAML: `go.yaml.in/yaml/v4` (not gopkg.in/yaml.v3). Pinned to `v4.0.0-rc.5` pending a stable `v4.Y.Z` release upstream; only release-candidate tags exist today. Revisit quarterly: `go list -m -versions go.yaml.in/yaml/v4`.
- `maxResponseSize = 50MB` const in `client.go`, used with `io.LimitReader`
- Log polling: `logPollMaxAttempts=30`, `logPollInterval=500ms` in `api/logs.go`

## Go 1.26 (Current Version)

`go.mod` is pinned to `go 1.26.4`; CI pins `go-version: '1.26.4'` (the six
`go-version` lines across `.github/workflows/` plus `go.mod` move together).
The 1.26.x series has shipped three stdlib CVE patches:

- **1.26.2** — `crypto/tls` / `crypto/x509` issues from 1.26.0–1.26.1.
- **1.26.3** — GO-2026-4971 (`net.Dial` / `LookupPort` NUL-byte panic on
  Windows), GO-2026-4918 (HTTP/2 infinite loop in `golang.org/x/net`).
- **1.26.4** — GO-2026-5039 (`net/textproto` error escaping) and GO-2026-5037
  (`crypto/x509` hostname parsing); both reached this codebase's keygen and
  TLS paths and were caught by `govulncheck`.

When a new patch lands, bump `go.mod` + the CI pins together and re-run
`govulncheck ./...`.

**1.26 idioms this project uses:** the ones in Code Style above (`for range N`,
`for i := range N`, `max()`/`min()`, `wg.Go`), plus `reflect` `Value.Fields()`
iteration (`internal/api/sanitize.go`) and `go fix ./...` modernizers
(`go fix -diff ./...` to preview).
