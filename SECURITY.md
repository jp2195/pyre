# Security Policy

## Supported versions

Only the most recent minor release is supported with security fixes.
See [Releases](https://github.com/jp2195/pyre/releases) for the current
version.

## Reporting a vulnerability

1. **Do not** open a public issue.
2. Email the repository owner via GitHub private message with steps to
   reproduce and potential impact.
3. You'll get a response within 72 hours.

## How pyre handles credentials

**pyre does not persist credentials.** No keychain, no token cache, no
on-disk storage of any kind. API keys and passwords live in memory for
the duration of a session and are zeroed on disconnect.

Credential management is the user's responsibility. Supply keys through
whichever mechanism fits your environment — shell env vars, direnv, a
secrets manager that exports to env, a CI/CD secret store, etc.

### Resolution order

When pyre needs an API key for a host, it checks in this order:

1. `--api-key` CLI flag
2. `PYRE_API_KEY` environment variable
3. `PYRE_<HOST>_API_KEY` — host-specific, where `<HOST>` is the connection
   host uppercased with `.` and `-` replaced by `_`
4. Interactive login — pyre prompts for username + password and runs
   keygen. The returned key is used for the session only; on the next
   launch pyre will prompt again unless you supplied a key upfront.

### What guards against accidental persistence

- Credential fields on `config.ConnectionConfig` (`APIKey`, `Password`)
  carry `yaml:"-"` tags, so `config.Save` cannot write them to disk
  even if they are set in memory. A regression test
  (`TestConfig_DoesNotPersistCredentials`) guards this invariant.
- On `RemoveConnection`, credential fields are zeroed before the
  connection struct is discarded to shorten in-memory lifetime.
- `~/.pyre.yaml` with world- or group-readable permissions triggers a
  startup warning. The file is expected to be `0600`.

## TLS

- Every HTTP client sets `MinVersion: tls.VersionTLS12`.
- Each client owns its own `*http.Transport` (no sharing of
  `http.DefaultTransport`).
- For firewalls using a private CA, set `ca_cert_path:
  /path/to/ca.pem` in the connection config. CA load failures surface
  as startup errors rather than silently falling back to system roots
  — this prevents "I thought my private CA was trusted but actually
  every request is validating against system roots" bugs.
- `insecure: true` should be reserved for lab environments. In
  production, add the firewall CA to the connection config instead.

## XML parsing

All PAN-OS responses go through a hardened `decodeXML` helper that
rejects `xml.Directive` tokens (DOCTYPE, entity declarations). This
prevents billion-laughs-style entity expansion attacks from a
compromised firewall or a man-in-the-middle (especially relevant when
`--insecure` is in use).

## Request/response logging

Per-request trace logging is **off by default**. Enable it with
`PYRE_DEBUG=1` (or `PYRE_DEBUG=true`). Error-path logs always fire.
When debug is on, traces include xpath, target serial, op-command
bodies, and response-body previews — useful for troubleshooting, noisy
in production. Server-supplied error strings are sanitized
(`api.SanitizeForDisplay`) before display, stripping ANSI CSI / OSC /
DCS sequences and C0 / DEL control chars.

## Dependencies

Direct:
- `charm.land/bubbletea/v2` — TUI framework
- `charm.land/bubbles/v2` — TUI components
- `charm.land/lipgloss/v2` — styling
- `go.yaml.in/yaml/v4` — YAML parsing (pinned to rc.4 pending stable v4)

CI runs `govulncheck ./...` on every push and weekly. Dependency pins
are managed by Renovate.

## Network notes

- pyre talks to firewalls over HTTPS only, typically port 443.
- The same permissions a user needs in PAN-OS also apply here — pyre
  doesn't elevate.
- Firewall API calls are logged by PAN-OS; review those logs for audit.
  pyre itself does not keep a local audit log.
