# Contributing

## Prerequisites

- Go 1.26 or later
- macOS 13+, modern Linux, or Windows 10+

## Setup

```bash
git clone https://github.com/jp2195/pyre.git
cd pyre
make build       # builds ./pyre
make test-race   # runs the race-enabled test suite
```

The `Makefile` wraps the common Go commands: `build`, `test`, `test-race`,
`vet`, `lint` (golangci-lint), `fix` (go fix + tidy), `tidy`, `smoke`, `clean`.

## Layout

- `cmd/pyre/` — entry point
- `internal/api/` — PAN-OS XML API client. All responses go through
  `decodeXML` (`xmlsafe.go`) which rejects DOCTYPE / entity declarations.
- `internal/auth/` — session, connection, credential resolution, keygen
- `internal/config/` — `~/.pyre.yaml`, CLI flags, env-var resolution
- `internal/tui/` — Bubble Tea v2 application
  - `app.go` — top-level `Model` and Update dispatcher
  - `handlers.go` — key handlers
  - `navigation.go` — table-driven nav (`navTargets` + `viewToNavbar`)
  - `commands.go` — `tea.Cmd` factories
  - `render.go` — header/footer
  - `views/` — individual view models
  - `theme/` — color palette
- `internal/troubleshoot/` — pattern-matching engine + embedded runbooks
- `internal/models/` — data structures

## Key patterns

- **Value receivers** on Bubble Tea models (immutable Update). **Pointer
  receivers** for mutation + `tea.Cmd` return.
- Only the top-level `Model.View()` returns `tea.View`. Sub-view models
  return `string`; the top composes them and sets program options
  (`AltScreen`, `MouseMode`) on the returned `tea.View`.
- Key handlers type-switch on `tea.KeyPressMsg` (v2), not `tea.KeyMsg`.
- `saveConfig()` / `saveState()` return `tea.Cmd` to avoid goroutine races.
- Navigation lookups use `navTargets` (key → view) and `viewToNavbar`
  (view → key). Keep both sides in sync; `navigation_test.go` asserts the
  bijection.

## Adding a new view

1. Create `internal/tui/views/<name>.go` with a `Model` that has
   `SetSize`, `Update(msg tea.Msg) (Model, tea.Cmd)`, and `View() string`.
2. Wire it into `internal/tui/app.go`: add a field, a `ViewState`
   constant, init in `NewModel`, dispatch cases in Update and the render
   composition in the top-level `View()`.
3. Add an entry to `navTargets` and `viewToNavbar` in
   `internal/tui/navigation.go`. The bijection test will flag omissions.
4. If the view needs a dedicated keybinding, add it to
   `internal/tui/keys.go`.
5. Document in `docs/views/<name>.md`.

## Testing

```bash
make test-race           # race-enabled test suite
make lint                # golangci-lint (v2, 5-minute timeout)
go test -cover ./...     # package-level coverage
```

Tests use `httptest.NewTLSServer` (see `internal/api/client_internal_test.go`)
for anything touching the API client. Credential tests use `t.Setenv`
to drive the resolution order — see `internal/auth/credentials_test.go`.

## Style

- Conventional Commits (`feat:`, `fix:`, `refactor:`, `chore:`, `docs:`,
  `test:`, `perf:`, `ci:`, `build:`). Scope optional.
- Tabs for Go indentation.
- Prefer Go 1.26 idioms: `for range N`, `min`/`max` builtins,
  `wg.Go(func(){...})`, `slices.SortFunc`.
- Handle errors explicitly. Use `context.Context` for cancellation.

## Pull requests

1. Branch from `main`.
2. Keep PRs focused on one logical change.
3. Run `make test-race lint vet` before opening.
4. Describe what the change does and why.

## Reporting issues

Include:

- `pyre --version`
- `go version`
- OS and version
- Steps to reproduce
- Expected vs actual behavior
- Relevant output with `PYRE_DEBUG=1` if you can reproduce a failure
  (scrub credentials and hostnames before sharing)
