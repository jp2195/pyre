# Policies View

Security rules fetched from all configured rulebases. Uses the
[standard view chrome](README.md#standard-view-chrome).

## Columns

All breakpoints include `#` (position) and `Base` (pre/post rulebase
abbreviation). A `•` suffix on the name indicates the rule has tags.

| Breakpoint | Columns |
|------------|---------|
| ≥ 150 | `#`, `Base`, `Name`, `Action`, `Source → Dest Zone`, `Application`, `Service`, `Hits`, `Last Hit` |
| ≥ 120 | `#`, `Base`, `Name`, `Action`, `Zones`, `Application`, `Hits`, `Last Hit` |
| ≥ 100 | `#`, `Base`, `Name`, `Action`, `Zones`, `App`, `Hits` |
| < 100 | `#`, `Name`, `Action`, `Zones`, `Hits` |

At ≥ 150, zones are split into separate `Source → Dest Zone`. At narrower
widths they are merged into a single `Zones` column. `Base` and `Service`
are dropped at the two narrowest breakpoints.

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
destination addresses, applications, services.

## Detail panel (`enter`)

- **Title / subtitle** — rule name (with `(disabled)` if applicable),
  position, and rulebase label.
- **Tags** — tag list (if any).
- **Description** — rule description (if set).
- **Traffic Match** — Source Zones, Source Addr (with negate flag),
  Source Users (if not "any"), Dest Zones, Dest Addr (with negate flag).
- **Application/Service** — Applications, Services, URL Categories
  (if not "any").
- **Action & Profiles** — Action (styled by allow/deny/drop), Profile
  Group or individual AV/Vuln/Spyware/URL/WildFire profile names,
  Logging (start/end + forwarding profile name).
- **Usage Statistics** — Hit Count, Last Hit, First Hit (if non-zero).

Note: there is no Rule UUID field in the detail panel.
