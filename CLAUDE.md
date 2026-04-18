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
- `internal/troubleshoot/` - Troubleshooting patterns
- SSH access is intentionally removed; Pyre currently uses only the PAN-OS XML API. SSH may return in a future redesign.

## Key Patterns

- **Value receivers** on Bubbletea Models (immutable Update pattern), **pointer receivers** for mutation + Cmd return
- Generic `fetchCmd[T any]()` helper in `commands.go` reduces fetch boilerplate
- Generic `fetchRulesFromPaths[T any]()` in `api/policies.go` for XPath rule fetching
- `saveConfig()` / `saveState()` return `tea.Cmd` (avoid goroutine race conditions)
- `setError()` is a pointer receiver that sets `m.err` and returns auto-dismiss tick Cmd
- Navigation uses table-driven `navTargets` map and `viewToNavbar` for reverse lookup
- Format helpers shared in `views/format_helpers.go`

## Code Style

- Go standard: tabs for indentation
- Use modern Go idioms (see Go 1.26 features below)
- Prefer `for range N` over `for i := 0; i < N; i++` when index is unused
- Prefer `for i := range N` over `for i := 0; i < N; i++` when index is used
- Use `max()`/`min()` builtins instead of manual if/else clamping
- Use `wg.Go(func() { ... })` instead of `wg.Add(1); go func() { defer wg.Done(); ... }()`

## Dependencies

- YAML: `go.yaml.in/yaml/v4` (not gopkg.in/yaml.v3)
- `maxResponseSize = 50MB` const in `client.go`, used with `io.LimitReader`
- Log polling: `logPollMaxAttempts=30`, `logPollInterval=500ms` in `api/logs.go`

## Go 1.26 (Current Version)

Released February 10, 2026. Key features relevant to this project:

### Language Changes
- **Enhanced `new()` builtin**: `new` now accepts an expression as initial value - `new(expr)` allocates and initializes in one step. Useful for pointer fields: `Age: new(yearsSince(born))`
- **Self-referential generic types**: Generic types may now refer to themselves in their own type parameter list

### Runtime & Performance
- **Green Tea GC (default)**: 10-40% lower GC overhead. Disable with `GOEXPERIMENT=nogreenteagc`
- **~30% faster cgo calls**
- **Better small object allocation**: Up to 30% cost reduction; more slice backing stores allocated on stack
- **Randomized heap base address** on 64-bit platforms (security hardening)

### Toolchain
- **`go fix` revamped**: Now a modernizer framework built on analysis framework (same as `go vet`). Run `go fix ./...` to apply safe modernizations. Run `go fix -diff ./...` to preview changes
- **`go mod init`** with Go 1.26 toolchain creates `go.mod` with `go 1.25.0`
- **`cmd/doc` removed**: Use `go doc` instead

### Standard Library Highlights
- **`errors.AsType[T]()`**: Generic, type-safe replacement for `errors.As`
- **`io.ReadAll`**: ~2x faster, ~50% less memory
- **`bytes.Buffer.Peek(n)`**: Returns next n bytes without advancing
- **`log/slog.NewMultiHandler`**: Invoke multiple slog handlers
- **`net.Dialer` typed methods**: `DialIP`, `DialTCP`, `DialUDP`, `DialUnix` with context
- **`net/netip.Prefix.Compare`**: Compare two prefixes
- **`reflect` iterators**: `Type.Fields()`, `Type.Methods()`, `Value.Fields()`, `Value.Methods()`
- **`testing.T.ArtifactDir()`**: Persistent artifact directory for tests
- **`B.Loop()` allows inlining**: Benchmark functions can now be inlined
- **`crypto/hpke`**: Hybrid Public Key Encryption (RFC 9180)
- **`crypto/tls`**: Post-quantum hybrid key exchanges enabled by default

### Experimental Features
- **Goroutine leak profiling**: `GOEXPERIMENT=goroutineleakprofile` enables `goroutineleak` profile type
- **`simd/archsimd`**: `GOEXPERIMENT=simd` enables architecture-specific SIMD operations (amd64)
- **`runtime/secret`**: `GOEXPERIMENT=runtimesecret` enables secure erasure of crypto temporaries

### Deprecations / Breaking Changes
- `crypto/ecdsa` `PublicKey`/`PrivateKey` `big.Int` fields deprecated
- `crypto/rsa` PKCS #1 v1.5 encryption deprecated (`EncryptPKCS1v15`, `DecryptPKCS1v15`)
- `net/http/httputil.ReverseProxy.Director` deprecated (use `Rewrite`)
- `net/url.Parse` now rejects malformed URLs with colons in host
- `image/jpeg` encoder/decoder rewritten (may produce different bit-for-bit output)
- Go 1.26 is the **last release supporting macOS 12 Monterey** (Go 1.27 requires macOS 13+)
- `windows/arm` (32-bit) port removed

### GODEBUG settings being removed in Go 1.27
- `tlsunsafeekm`, `tlsrsakex`, `tls10server`, `tls3des`, `x509keypairleaf`
- `gotypesalias`, `asynctimerchan`
