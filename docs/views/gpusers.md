# GP Users View

GlobalProtect connected users. Uses the
[standard view chrome](README.md#standard-view-chrome).

## Columns

| Breakpoint | Columns |
|------------|---------|
| ≥ 140 | `Username`, `Domain`, `Gateway`, `Virtual IP`, `Client IP`, `Duration`, `Region`, `Traffic` |
| ≥ 100 | `Username`, `Gateway`, `Virtual IP`, `Client IP`, `Duration`, `Traffic` |
| < 100 | `Username`, `Gateway`, `Virtual IP`, `Duration` |

`Domain` and `Region` appear only at ≥ 140. `Traffic` (bytes in + out)
appears at ≥ 100. `Computer` is only in the detail panel — it is not a
column at any width.

## Sort fields

Cycled with `s`; direction toggled with `S`.

| Index | Label | Default direction |
|-------|-------|-------------------|
| 0 | Username | ascending |
| 1 | Gateway | ascending |
| 2 | Login Time | descending |
| 3 | Duration | descending |

## Filter scope

Matches (case-insensitive substring) against: username, domain,
computer, gateway, client IP, virtual IP, source region.

## Detail panel (`enter`)

- **User Information** — Domain (if set), Computer (if set), Client
  Version (if set).
- **Connection** — Gateway, Virtual IP, Public IP (client IP), Source
  Region (if set).
- **Session** — Login Time (if available), Duration (if set).
- **Traffic** (if any bytes > 0) — Bytes In, Bytes Out.

Note: there are no HIP report or tunnel-encapsulation fields in the
detail panel.
