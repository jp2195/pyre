# IPSec Tunnels View

Site-to-site IPSec tunnel status. Uses the
[standard view chrome](README.md#standard-view-chrome).

## Columns

A state indicator icon precedes each row: `●` = up, `~` = init,
`○` = down. Non-selected rows are color-coded: green = up, yellow =
init, red = down.

| Breakpoint | Columns |
|------------|---------|
| ≥ 120 | icon, `Name`, `Gateway`, `State`, `Proto`, `Encrypt`, `In`, `Out`, `Uptime` |
| ≥ 90 | icon, `Name`, `Gateway`, `State`, `In`, `Out`, `Uptime` |
| < 90 | icon, `Name`, `Gateway`, `State`, `Traffic` (combined) |

## Sort fields

Cycled with `s`; direction toggled with `S`.

| Index | Label | Default direction |
|-------|-------|-------------------|
| 0 | Name | ascending |
| 1 | Gateway | ascending |
| 2 | State | descending |
| 3 | Traffic | descending (sum of bytes in + out) |

## Filter scope

Matches (case-insensitive substring) against: name, gateway, state,
protocol, encryption.

## Detail panel (`enter`)

- **Title** — tunnel name and colored state label.
- **Connection** — Gateway, Local IP (if set), Remote IP (if set).
- **Security** — Protocol, Encryption, Authentication, Local SPI (if
  set), Remote SPI (if set).
- **Traffic Statistics** — Bytes In, Bytes Out, Packets In, Packets Out,
  Uptime (if set), Errors (if > 0, shown in red).

Note: there are no SA lifetime, integrity proposals, or last-error fields
in the detail panel.
