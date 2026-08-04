# Objects View

Address and service objects. Analyze group (`2`).

## Tabs

Two sub-tabs with independent filter, sort, cursor, and detail state:

| Key | Action |
|-----|--------|
| `Tab` | Cycle Address ↔ Service |
| `a` | Jump to Address tab |
| `s` | Jump to Service tab |

The header shows `[Address]  Service` or `Address  [Service]` with a
`(a/s/Tab to switch)` hint.

## Address tab

### Columns (fixed layout)

| Column | Description |
|--------|-------------|
| NAME | Object name (truncated to 24 chars) |
| TYPE | `ip-netmask`, `ip-range`, `fqdn`, `ip-wildcard` (prefix `ip-` stripped in display; truncated to 12 chars) |
| VALUE | CIDR / range / FQDN / wildcard (truncated to 26 chars) |
| TAGS | Space-separated tag list |

### Sort (`S` to cycle, resets to ascending each time)

Name → Type → Value

### Filter scope

Matches against: name, type, value, description, tags.

### Detail panel (`enter`)

Object name, Type, Value, Description (if set), Tags (if any).

## Service tab

### Columns (fixed layout)

| Column | Description |
|--------|-------------|
| NAME | Object name (truncated to 24 chars) |
| PROTO | `tcp` or `udp` (truncated to 8 chars) |
| DEST PORT | Port, range, or comma-list (truncated to 16 chars) |
| SRC PORT | Source-port restriction, if any (truncated to 16 chars) |
| TAGS | Space-separated tag list |

### Sort (`S` to cycle, resets to ascending each time)

Name → Protocol → Dest Port

### Filter scope

Matches against: name, protocol, dest port, src port, description, tags.

### Detail panel (`enter`)

Object name, Protocol, Dest Port, Src Port (if set), Description (if
set), Tags (if any).

## Keys

`s` switches to the Service tab (not a sort key here — sort is `S`).
`esc` collapses an open detail panel on the first press, then clears
the active filter on a second press.

## Refresh (`r`)

App-level refresh re-fetches both tabs.
