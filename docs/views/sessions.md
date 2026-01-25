# Sessions

The Sessions view displays active sessions on the firewall. Access it through the Analyze group (press `2`, then cycle to Sessions).

## Display

Each session shows:

| Column | Description |
|--------|-------------|
| ID | Session ID (unique identifier) |
| Application | Identified application |
| Source | Source IP:Port |
| Destination | Destination IP:Port |
| Protocol | TCP, UDP, ICMP, etc. |
| State | Session state (ACTIVE, INIT, etc.) |
| From Zone | Source zone |
| To Zone | Destination zone |
| Age | How long the session has been active |
| Bytes | Total bytes transferred |

### Session States

- **ACTIVE** - Established and passing traffic
- **INIT** - Initial handshake in progress
- **OPENING** - Connection being established
- **CLOSING** - Connection terminating
- **CLOSED** - Session ended (briefly visible)
- **DISCARD** - Session being dropped

## Filtering

Press `/` to enter filter mode. The filter matches against:

- Source or destination IP address
- Application name
- Zone names (source and destination)
- Rule name
- Username (if User-ID is enabled)

Filter examples:
- `10.0.1.50` - Sessions involving this IP
- `ssl` - SSL/TLS sessions
- `trust` - Sessions from the trust zone
- `facebook` - Facebook application sessions

Filters are case-insensitive and support partial matching.

Press `Esc` to clear the filter.

## Sorting

Press `s` to cycle through sort fields:

1. **ID** (default) - Session ID order
2. **Bytes** - By data transferred
3. **Age** - By session duration
4. **Application** - Alphabetical by app

Press `S` (shift+s) to toggle sort direction.

## Session Details

Press `Enter` on a session to expand its details. The expanded view shows:

### Connection Info
- Full 5-tuple (src IP, dst IP, src port, dst port, protocol)
- NAT translations (if applicable)
- Assigned security rule

### Traffic Stats
- Bytes and packets (client to server, server to client)
- Session flags

### Identification
- Application and subcategory
- URL category (if applicable)
- User (if User-ID is active)

Press `Enter` again or `Esc` to collapse.

## Keybindings

| Key | Action |
|-----|--------|
| `j` / `Down` | Move cursor down |
| `k` / `Up` | Move cursor up |
| `g` / `Home` | Jump to first session |
| `G` / `End` | Jump to last session |
| `Ctrl+d` / `PgDn` | Page down |
| `Ctrl+u` / `PgUp` | Page up |
| `/` | Enter filter mode |
| `Esc` | Clear filter |
| `s` | Cycle sort field |
| `S` | Toggle sort direction |
| `Enter` | Toggle session details |
| `r` | Refresh sessions |

## Tips

### Investigating Traffic Issues

1. Filter by the IP address having issues
2. Check if sessions are being established (ACTIVE state)
3. Look at the matched rule to verify policy
4. Check NAT translation in details

### Finding Heavy Users

1. Sort by Bytes (descending)
2. Top sessions are transferring the most data
3. Expand to see the application and user

### Monitoring Real-Time Traffic

The session table updates on refresh (`r`). Use this to monitor live traffic patterns. Sessions are a point-in-time snapshot; inactive sessions age out.

### Understanding NAT

When you expand a session, you'll see both the original and translated addresses if NAT is applied. This helps troubleshoot NAT translation issues by showing exactly how addresses are being modified.
