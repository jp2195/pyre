# NAT Policies View

NAT rulebase browser. Uses the [standard view chrome](README.md#standard-view-chrome).

## Columns

All breakpoints include `#` (position) and `Base` (pre/post rulebase
abbreviation). A `*` suffix on the name indicates the rule has tags.

| Breakpoint | Columns |
|------------|---------|
| ≥ 150 | `#`, `Base`, `Name`, `Src Zone`, `Dst Zone`, `Src NAT`, `Dst NAT`, `Hits`, `Last Hit` |
| ≥ 120 | `#`, `Base`, `Name`, `Zones`, `Service`, `Src NAT`, `Dst NAT`, `Hits` |
| ≥ 100 | `#`, `Base`, `Name`, `Zones`, `Src NAT`, `Dst NAT`, `Hits` |
| < 100 | `#`, `Name`, `Zones`, `Src NAT`, `Hits` |

`Src NAT` and `Dst NAT` show abbreviated translation types:
- `DIPP:<pool>` — Dynamic IP and Port
- `DIP:<pool>` — Dynamic IP
- `Static:<addr>` — Static IP
- `None` — no source translation

`Dst NAT` shows `<translated-ip>:<translated-port>` or `None`.

## Sort fields

Cycled with `s`; direction toggled with `S`.

| Index | Label | Default direction |
|-------|-------|-------------------|
| 0 | Position | ascending |
| 1 | Name | ascending |
| 2 | Hits | descending |
| 3 | Last Hit | descending |

## Filter scope

Matches (case-insensitive substring) against: name, description,
rulebase, tags, source zones, destination zones, source addresses,
destination addresses, translated source, translated destination.

## Detail panel (`enter`)

- **Title / subtitle** — rule name (with `(disabled)` if applicable),
  position, and rulebase label.
- **Tags** — tag list (if any).
- **Description** — rule description (if set).
- **Original Packet (Match)** — Source Zones, Source Addresses, Dest
  Zones, Dest Addresses, Service, Dest Interface (if not "any").
- **Translated Packet** — Source Translation type and translated-to
  address; Dest Translation address and port (or "None" for each).
- **Usage Statistics** — Hit Count, Last Hit, First Hit (if non-zero).

Note: there is no "fallback" field in the detail panel.
