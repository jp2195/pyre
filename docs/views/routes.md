# Routes View

Routing table and BGP/OSPF neighbor state. Analyze group (`2`).

## Tabs

Two tabs, switched with `[` or `]`:

| Tab | Content |
|-----|---------|
| **Routes** | Filtered routing table |
| **Neighbors** | BGP peers and OSPF neighbors combined |

There is no sort-cycling key and no per-row detail panel on either tab.
`enter` toggles an `Expanded` flag that renders nothing visible.

## Routes tab

### Columns

| Breakpoint | Columns |
|------------|---------|
| ≥ 100 | `Proto`, `Destination`, `Next Hop`, `Interface`, `Metric`, `VR` |
| ≥ 70 | `Proto`, `Destination`, `Next Hop`, `Interface`, `Metric` |
| < 70 | `Proto`, `Destination`, `Next Hop` |

`Proto` is a single-letter code: `C` = connected, `S` = static, `L` =
local, `B` = BGP, `O` = OSPF. Full protocol names are abbreviated in
display only; they are stored and filtered by their full name. There is
no Age column.

### Filter

Text filter (`/`) matches against destination, next hop, interface,
protocol, and virtual router.

### Protocol filter shortcuts (Routes tab only)

| Key | Filter |
|-----|--------|
| `a` | All protocols (clear filter) |
| `c` | connected |
| `s` | static |
| `b` | bgp |
| `o` | ospf |

The active protocol filter is shown in the summary line and in the help
text at the bottom.

### Help line

```
[/] switch  /filter  a/c/s/b/o protocol [<active>]  r refresh
```

## Neighbors tab

A single combined list of BGP peers and OSPF neighbors, each with a
`Type` column identifying which protocol it belongs to.

### Columns

| Breakpoint | Columns |
|------------|---------|
| ≥ 100 | `Type`, `Peer/Neighbor`, `State`, `AS/Area`, `Prefixes`, `Uptime`, `VR` |
| ≥ 70 | `Type`, `Peer/Neighbor`, `State`, `AS/Area`, `Prefixes`, `Uptime` |
| < 70 | `Type`, `Peer/Neighbor`, `State`, `AS/Area` |

- BGP rows: `Type` = `BGP`, `AS/Area` = `AS<number>`, `Prefixes` =
  received prefix count, `Uptime` = session uptime.
- OSPF rows: `Type` = `OSPF`, `AS/Area` = area ID, `Prefixes` = `-`,
  `Uptime` = dead-time remaining.

### Help line

```
[/] switch  r refresh
```

## Navigation

`j`/`k`/`g`/`G`/`pgdown`/`pgup` work on both tabs. The routes tab also
supports the text filter (`/`). There is no detail panel (`enter` has no
visible effect).
