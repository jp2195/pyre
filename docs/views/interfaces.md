# Interfaces

Interface status + counters. Analyze group (`2`).

## Display

Cards showing:

| Field          | Description                            |
|----------------|----------------------------------------|
| Name           | `ethernet1/1`, `ae1`, `tunnel.1`, …    |
| State          | Up / Down                              |
| Zone           | Assigned security zone                 |
| IP Address     | Configured IP                          |
| MAC            | Hardware address                       |
| Virtual Router | Assigned VR                            |
| Speed / Duplex | Link speed and duplex mode             |
| Bytes In/Out   | Traffic counters                       |
| Packets In/Out | Packet counters                        |
| Errors         | Error counts, when non-zero            |

Colors: green = up, red = down, yellow = warnings / errors.

Covers physical, aggregate (`ae*`), VLAN (`ethernet1/1.100`),
loopback, tunnel interfaces.

## Filter (`/`)

Name, zone, IP, state. Examples: `ethernet1`, `trust`, `10.0`, `down`.

## Sort (`s` cycle, `S` reverse)

Name → Zone → State → IP.

## Detail (`Enter`)

- Configuration: full settings, netmask, zone + VR, link-state.
- Counters: bytes/packets, error / drop breakdown, multicast /
  broadcast stats.
- Physical: speed/duplex negotiation, media type, hardware details.

## Standard keys

See [keybindings.md](../keybindings.md).

## Tips

- Filter `down` to see non-operational interfaces only.
- Sort by State to group down interfaces.
- Error types in the detail view:
  - **Input errors** — receive problems (CRC, framing)
  - **Output errors** — transmit problems
  - **Drops** — intentional (queue full, etc.)
  - **Collisions** — half-duplex contention
