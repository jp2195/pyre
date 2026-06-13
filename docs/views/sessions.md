# Sessions View

Active sessions on the firewall. Uses the
[standard view chrome](README.md#standard-view-chrome).

## Columns

The column layout is fixed (no width breakpoints):

| Column | Description |
|--------|-------------|
| ID | Session ID |
| Source | Source IP (truncated to 15 chars) |
| Destination | Destination IP (truncated to 15 chars) |
| Port | Destination port |
| Pro | Protocol (tcp/udp/…, truncated to 4 chars) |
| App | Identified application (truncated to 10 chars) |
| State | Session state (`ACTIVE`, `INIT`, `CLOSED`, etc.) |
| Zones | `SrcZone→DstZone` combined (truncated to 15 chars) |
| Age | Duration since session start |
| Bytes | Total bytes in + out |

In non-selected rows the `State` cell is color-coded: green = `ACTIVE`,
yellow = `DISCARD`/`DROP`, muted = `CLOSED`/`INIT`.

## Sort fields

Cycled with `s`; direction toggled with `S`.

| Index | Label | Default direction |
|-------|-------|-------------------|
| 0 | ID | ascending |
| 1 | Bytes | descending |
| 2 | Age | descending |
| 3 | App | ascending |

## Filter scope

Matches (case-insensitive) against: application, source IP, destination
IP, source zone, destination zone, rule name, username.

## Detail — two steps

### Step 1: basic detail (`enter`)

Toggles the basic detail panel. Shows:
- Application, Protocol, State
- Source: `IP:port (zone)`
- Destination: `IP:port (zone)`
- NAT Source (if translated): `IP:port`
- User (if User-ID is active)
- Rule name
- Bytes In, Bytes Out
- Start Time, Duration
- Hint: `[d: fetch extended details]`

### Step 2: extended detail (`d`, only while panel is open)

Fetches additional session data from the API and adds:
- **NAT Details** (if applicable) — NAT Dest IP:port, NAT Rule
- **Security** (if applicable) — URL Category, URL Rule, Decrypt Rule
- **Traffic Statistics** — Pkts to Client, Pkts to Server, Bytes to
  Client, Bytes to Server
- **Timing** (if available) — Timeout, TTL, Idle time
- **Flags** — offloaded, decrypt-mirror (if set)

Note: there is no "application subcategory" field in the detail panel.
