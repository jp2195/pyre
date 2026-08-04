# View Reference

One page per view. Use [keybindings.md](../keybindings.md) for the
canonical key reference; the per-view pages here cover what each view
shows, what its filter / sort fields are, and what the detail panel
reveals.

## Navigation index

| Group | Key | View |
|-------|-----|------|
| Monitor | `1` | Overview dashboard |
| Monitor | `1` (again) | Network dashboard |
| Monitor | `1` (again) | Security dashboard |
| Monitor | `1` (again) | VPN dashboard |
| Analyze | `2` | Policies |
| Analyze | `2` (again) | NAT |
| Analyze | `2` (again) | Objects |
| Analyze | `2` (again) | Sessions |
| Analyze | `2` (again) | Interfaces |
| Analyze | `2` (again) | Routes |
| Analyze | `2` (again) | IPSec |
| Analyze | `2` (again) | GP Users |
| Analyze | `2` (again) | Logs |
| Tools | `3` | Config dashboard |

Pressing a group key when already in that group cycles to the next item
within the group.

## Standard view chrome

Six views — Policies, NAT, Sessions, Interfaces, IPSec, and GP Users —
share the same `RuleListModel` shell. The chrome is described once here;
the per-view pages document only the differences.

### Banner

```
ViewTitle  [N <noun> | Sort: <field> <dir> | s: change | S: dir | /: filter | enter: details]
```

`N` is the filtered count. `<noun>` is view-specific (rules, sessions,
interfaces, tunnels, users). `<dir>` is `↑` ascending or `↓` descending.

### Sort cycling

- `s` — advance to the next sort field; direction resets to that field's
  default (ascending for name/position fields, descending for
  count/traffic fields as defined by each view's `DefaultSortAsc`).
- `S` — toggle the current sort direction without changing the field.

### Filter

- `/` — open the filter input; the list narrows live as you type.
- `enter` (in filter input) — commit; resets the cursor to the top.
- `esc` (in filter input) — close the input; typed text is preserved and
  the filter stays active.

### esc outside the filter input

1. If a detail panel is expanded, collapse it (first press).
2. Otherwise, clear the active filter (second press or when not expanded).

### Detail panel

- `enter` — toggle the detail panel open/closed.
- A `─` divider separates the header row from the data rows.
- When the list overflows the screen, `Showing x–y of z` appears below
  the rows.

### Refresh

`r` is handled at the app level; it re-fetches data and resets the cursor
to the top. This key is consumed by the filter input while the filter is
focused (M8 filter guard — global keys `q`, `r`, `1`–`3`, `Tab` are
intercepted by the focused filter instead of triggering navigation;
only `ctrl+c` quits).

## View pages

### Monitor (group `1`)

- [Dashboard](dashboard.md) — Overview / Network / Security / VPN at-a-glance panels, plus the Config dashboard under Tools (`3`)

### Analyze (group `2`)

- [Policies](policies.md) — security rules with hit counts
- [NAT](nat.md) — NAT rules with translation details and hit counts
- [Objects](objects.md) — address and service objects
- [Sessions](sessions.md) — active sessions with extended detail
- [Interfaces](interfaces.md) — interface status, counters, and ARP
- [Routes](routes.md) — routing table and BGP/OSPF neighbors
- [IPSec](ipsec.md) — site-to-site IPSec tunnel status
- [GP Users](gpusers.md) — GlobalProtect connected users
- [Logs](logs.md) — system, traffic, and threat logs

### Tools (group `3`)

- Config dashboard — policy statistics and pending changes. See [Dashboard](dashboard.md).

## See also

- [Getting Started](../getting-started.md)
- [Configuration](../configuration.md)
- [Keybindings](../keybindings.md)
- [Panorama](../panorama.md)
