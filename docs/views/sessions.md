# Sessions

Active sessions on the firewall. Analyze group (`2`).

## Columns

| Column       | Description                          |
|--------------|--------------------------------------|
| ID           | Session ID                           |
| Application  | Identified application               |
| Source       | Source IP:Port                       |
| Destination  | Destination IP:Port                  |
| Protocol     | TCP / UDP / ICMP / …                 |
| State        | `ACTIVE` / `INIT` / `OPENING` / …    |
| From Zone    | Source zone                          |
| To Zone      | Destination zone                     |
| Age          | Session duration                     |
| Bytes        | Total bytes                          |

### States

- `ACTIVE` — established, passing traffic
- `INIT` — initial handshake
- `OPENING` — connection being established
- `CLOSING` — terminating
- `CLOSED` — briefly visible
- `DISCARD` — being dropped

## Filter (`/`)

Source or destination IP, application, zones, matched rule, username
(when User-ID is active). Case-insensitive substring match.

## Sort (`s` cycle, `S` reverse)

ID → Bytes → Age → Application.

## Detail (`d` or `Enter`)

Full 5-tuple, NAT translations, matched rule, byte / packet counters
per direction, session flags, application subcategory, URL category,
User-ID (if active).

## Standard keys

See [keybindings.md](../keybindings.md).

## Tips

- Sort by Bytes descending to find heavy talkers.
- Expand a session to see original vs translated addresses for NAT
  troubleshooting.
- Sessions are a point-in-time snapshot — press `r` to refresh.
