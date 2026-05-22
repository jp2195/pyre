# Routes

Routing table + BGP / OSPF neighbor state. Analyze group (`2`).

## Display

| Column         | Description                              |
|----------------|------------------------------------------|
| Destination    | CIDR or `0.0.0.0/0` for default route    |
| Next Hop       | Gateway address or interface             |
| Interface      | Egress interface                         |
| Protocol       | `static`, `bgp`, `ospf`, `connect`, …    |
| Virtual Router | Owning VR                                |
| Age            | When the route was installed             |
| Metric         | Protocol-specific cost                   |

Each row is colored by protocol family so static / dynamic routes are
distinguishable at a glance.

## Tabs

The view is organised as Routes (default), BGP, OSPF — flipping
between them keeps the cursor in place per tab.

## Filter (`/`)

Substring match across destination, next hop, interface, protocol,
and virtual router. Filter is per-tab and survives tab switches.

There are also protocol filter shortcuts (single-key fast filters) on
the Routes tab that limit the table to a single protocol.

## Sort (`s` cycle)

Destination → Protocol → Next Hop → Interface.

## Detail (`Enter`)

For static / dynamic routes: full route attributes (preference,
flags, BFD, source).

For BGP neighbors: peer AS, hold/keepalive timers, prefixes
sent/received, last error.

For OSPF neighbors: area, state, DR / BDR, dead time, retransmit
queues.

## Standard keys

See [keybindings.md](../keybindings.md).

## Tips

- Sort by Protocol to group the routing table by source.
- The neighbor tabs are the fastest way to confirm a BGP peering is
  Established without opening the GUI.
- Combine `/` with a destination CIDR fragment to verify a specific
  prefix is in the FIB.
