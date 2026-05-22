# GP Users

GlobalProtect connected users. Analyze group (`2`).

## Columns

| Column      | Description                              |
|-------------|------------------------------------------|
| Username    | Logged-in user                           |
| Computer    | Client hostname                          |
| Client IP   | Public source IP                         |
| Virtual IP  | Tunnel address assigned by the gateway   |
| Gateway     | GP gateway the user connected through    |
| Login Time  | Tunnel establishment timestamp           |
| Duration    | Time since login                         |
| Region      | Source region from User-ID / GeoIP       |

## Filter (`/`)

Substring match across user, domain, computer, gateway, client IP,
virtual IP, and source region.

## Sort (`s` cycle)

Username → Gateway → Login Time → Duration.

## Detail (`Enter`)

Full session details for the selected user: HIP report summary
(when enabled), gateway, tunnel encapsulation, virtual + public IPs,
client OS / app version, login time and duration.

## Standard keys

See [keybindings.md](../keybindings.md).

## Tips

- Sort by Duration descending to see the longest-running sessions —
  often a sign of forgotten clients you can ask to reconnect.
- Filter by gateway name to see who is on each pool — useful when
  redistributing load.
- HIP-only fields populate when host-information policies are active
  on the gateway.
