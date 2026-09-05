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
the lifetime of the process and are never written anywhere.

Note what that does *not* claim. Go strings cannot be reliably scrubbed
from memory, and the API client keeps its own copy of the key, so pyre
does not promise that a credential is unrecoverable from a core dump
while it is running. The guarantee is about persistence, not memory
hygiene.

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
- `RemoveConnection` clears credential fields before dropping a
  connection, which shortens their in-memory lifetime when a connection
  is explicitly removed.
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

`decodeXML` also sanitizes every string it decodes, so the guarantee
does not depend on each fetcher remembering to ask for it.

## Terminal-injection and display spoofing

Rule names, object names, descriptions, and log messages are
attacker-influenceable: a compromised firewall, a Panorama pushing rules
from elsewhere, or a MITM on an `--insecure` session all control them.
Rendering them raw in a terminal is a code-execution-adjacent risk and,
worse for this tool, a way to make the display disagree with the
configuration. `api.SanitizeForDisplay` removes:

- ESC-introduced CSI / OSC / DCS / PM / APC / SOS sequences, and any
  other two-byte ESC sequence.
- The C1 control block (U+0080–U+009F). This matters more than the ESC
  forms: Go's XML decoder rejects a raw ESC outright, as a byte or as a
  character reference, so an ESC cannot reach the screen through a
  parsed response at all — but the C1 block passes through untouched,
  and U+009B is a single-character CSI that many terminals honor exactly
  like `ESC [`. The C1 introducers are consumed as full sequences.
- C0 controls other than tab and newline, plus DEL.
- Unicode bidi controls (U+202A–U+202E, U+2066–U+2069, U+200E, U+200F)
  and zero-width characters (U+200B–U+200D, U+FEFF). These are how a
  rule name is made to render as something other than what it says.

Text that is merely non-ASCII is left alone; the goal is that displayed
text cannot misrepresent itself or drive the terminal, not that it be
ASCII. Stripping rather than escaping means two names differing only in
these characters collapse to the same display string, which is the
lesser evil against a name that reads as its own opposite.

## Request/response logging

pyre has two independent debug mechanisms:

- **`--debug` / `DEBUG` env var** — routes the standard Go logger to
  `~/.pyre/logs/debug.log`. Without this, all `log.Printf` output is
  discarded so it never reaches the terminal or any file.

- **`PYRE_DEBUG=1` (or `PYRE_DEBUG=true`)** — enables per-request API
  trace logging: xpath, target serial, op-command bodies, response
  status/timing, and response-body previews. Traces are written via
  `log.Printf`, so they only reach the log file when `--debug` / `DEBUG`
  is **also** set. Neither mechanism is on by default.

Error-path `log.Printf` calls always fire regardless of `PYRE_DEBUG`,
so unexpected failures are never silently swallowed. Server-supplied
error strings are sanitized before display; see the section above for
what that covers.

## Dependencies

Direct:
- `charm.land/bubbletea/v2` — TUI framework
- `charm.land/bubbles/v2` — TUI components
- `charm.land/lipgloss/v2` — styling
- `go.yaml.in/yaml/v4` — YAML parsing (pinned to a release candidate pending a stable v4)

CI runs `govulncheck ./...` on every push and weekly. Dependency pins
are managed by Renovate.

## Network notes

- pyre talks to firewalls over HTTPS only, typically port 443.
- Each client caps itself at 4 concurrent API calls
  (`api.MaxConcurrentRequests`). The management plane is shared with the
  web UI and logs every call, so an unbounded fan-out is both slow and
  noisy in the audit trail. The cap is per connection, so it bounds load
  per firewall.
- Every request sends `User-Agent: pyre`, so calls are attributable in
  the firewall's own API log.
- `HTTPS_PROXY` / `HTTP_PROXY` / `NO_PROXY` are honored, so traffic can
  be routed through a corporate egress proxy or an inspection proxy.
- The same permissions a user needs in PAN-OS also apply here — pyre
  doesn't elevate.
- Firewall API calls are logged by PAN-OS; review those logs for audit.
  pyre itself does not keep a local audit log.
