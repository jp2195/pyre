# Objects

Address & service objects from the firewall's effective config.
Analyze group (`2`).

Phase 1 ships forward lookup only (browse + filter + detail). Inline
expansion from rule views and reverse "where used" lookup will come in
later phases.

## Tabs

Two sub-tabs, each with independent filter / sort / cursor / detail
state:

| Key   | Action                              |
|-------|-------------------------------------|
| `Tab` | Cycle Address ↔ Service             |
| `a`   | Jump to Address tab                 |
| `s`   | Jump to Service tab                 |

The header row shows `[Address]  Service` (or vice-versa) and the
keybinding hint.

## Address tab

| Column | Description                                          |
|--------|------------------------------------------------------|
| NAME   | Object name                                          |
| TYPE   | `netmask` / `range` / `fqdn` / `wildcard`            |
| VALUE  | The CIDR / range / FQDN / wildcard the object holds  |
| TAGS   | Space-separated tag list                             |

Address objects are read from both vsys1 and shared scopes and
concatenated. On a Panorama-targeted device this returns the device's
effective merged config.

## Service tab

| Column     | Description                                |
|------------|--------------------------------------------|
| NAME       | Object name                                |
| PROTO      | `tcp` or `udp`                             |
| DEST PORT  | `443`, `1024-65535`, `1433,1434`, …        |
| SRC PORT   | Optional source-port restriction           |
| TAGS       | Space-separated tag list                   |

Ports are preserved as strings so ranges and comma-lists survive
without lossy parsing. Built-in PAN-OS services (`service-http`,
`service-https`) are not included — they don't appear under `/service`.

## Filter (`/`)

Substring match across name, type/protocol, value/ports, description,
and tags. Filter state is preserved per tab — switching tabs does not
clear it.

## Sort (`S` to cycle)

`s` is already taken for the Service-tab shortcut, so Objects uses
capital `S` to cycle sort modes:

- Address: Name → Type → Value
- Service: Name → Protocol → Dest Port

Direction defaults to ascending; cycling the field also resets to
ascending.

## Detail (`Enter` / `Esc`)

Opens the detail panel for the highlighted row. Shows the object's
type, value, description, and tags. `Esc` closes the panel (or, if the
panel is already closed, clears the active filter).

## Refresh (`r`)

Refetches both tabs in parallel. Loading state shows the spinner per
tab.

## Standard keys

See [keybindings.md](../keybindings.md).

## Tips

- Filter for a CIDR fragment (e.g. `10.0.`) to find every address
  object overlapping a subnet.
- Sort Address by Type to group all FQDNs together — useful for
  hunting wildcard or legacy entries.
- The detail panel is the source of truth for what the firewall sees.
  If a rule references `web-servers`, open Objects → Address →
  filter `web-servers` → Enter to confirm the resolution.
