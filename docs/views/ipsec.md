# IPSec Tunnels

Site-to-site IPSec tunnel status. Analyze group (`2`).

## Columns

| Column     | Description                              |
|------------|------------------------------------------|
| Name       | Tunnel name                              |
| Gateway    | Peer IP / hostname                       |
| State      | `up`, `init`, `down`                     |
| Protocol   | Negotiated protocol (`ipsec`, `ike`)     |
| Encryption | Negotiated cipher                        |
| In / Out   | Byte counters per direction              |
| Uptime     | Time in current state                    |

Rows are colored by state: green = up, yellow = init, red = down.

## Filter (`/`)

Substring match across name, gateway, state, protocol, and
encryption.

## Sort (`s` cycle)

Name → Gateway → State → Traffic (sum of in + out).

## Detail (`Enter`)

Full IPSec / IKE state for the selected tunnel: local / remote IPs,
authentication, encryption + integrity proposals, SA lifetime,
in / out byte and packet counters, SPI values, last error.

## Standard keys

See [keybindings.md](../keybindings.md).

## Tips

- Sort by State to surface flapping / down tunnels first.
- Filter `init` to catch tunnels stuck in negotiation.
- The detail panel exposes the SA SPI values — useful when you need
  to correlate with the peer's logs.
