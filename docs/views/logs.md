# Logs

The Logs view displays system, traffic, and threat logs from the firewall. Access it through the Analyze group (press `2`, then cycle to Logs).

## Log Types

Use `Tab` to switch between log types:

### System Logs

System events including:
- Configuration changes
- HA state changes
- System errors and warnings
- Authentication events
- License events

### Traffic Logs

Session traffic logging showing:
- Source and destination
- Application identified
- Action (allow, deny, drop)
- Rule matched
- Bytes transferred
- Session duration

### Threat Logs

Security events including:
- Threat name and type
- Severity level
- Action taken
- Source of threat
- Affected destination
- URL/file information

## Display

### System Log Columns

| Column | Description |
|--------|-------------|
| Time | When the event occurred |
| Severity | Info, Warning, Error, Critical |
| Event Type | Category of event |
| Description | Event details |

### Traffic Log Columns

| Column | Description |
|--------|-------------|
| Time | Session end time |
| Source | Source IP address |
| Destination | Destination IP address |
| App | Identified application |
| Action | Allow, Deny, Drop, Reset |
| Rule | Matched security rule |
| Bytes | Data transferred |

### Threat Log Columns

| Column | Description |
|--------|-------------|
| Time | When threat was detected |
| Threat | Threat name/signature |
| Severity | Critical, High, Medium, Low, Info |
| Source | Threat source IP |
| Destination | Target IP |
| Action | Alert, Block, Reset, etc. |

## Filtering

Press `/` to enter filter mode. The filter matches against all displayed columns.

Filter examples:
- `10.0.1.50` - Events involving this IP
- `deny` - Denied traffic
- `critical` - Critical severity events
- `config` - Configuration changes

Press `Esc` to clear the filter.

## Sorting

Press `s` to cycle through sort fields (varies by log type).

Press `S` (shift+s) to toggle sort direction.

By default, logs are sorted by time (newest first).

## Log Details

Press `Enter` on a log entry to expand its details. The expanded view shows all available fields for that log type.

Press `Enter` again or `Esc` to collapse.

## Keybindings

| Key | Action |
|-----|--------|
| `Tab` | Cycle log type (System/Traffic/Threat) |
| `j` / `Down` | Move cursor down |
| `k` / `Up` | Move cursor up |
| `g` / `Home` | Jump to first entry |
| `G` / `End` | Jump to last entry |
| `Ctrl+d` / `PgDn` | Page down |
| `Ctrl+u` / `PgUp` | Page up |
| `/` | Enter filter mode |
| `Esc` | Clear filter |
| `s` | Cycle sort field |
| `S` | Toggle sort direction |
| `Enter` | Toggle log details |
| `r` | Refresh logs |

## Tips

### Investigating Security Events

1. Switch to Threat logs with `Tab`
2. Sort by Severity to see critical threats first
3. Expand entries to see full threat details
4. Note the source IP and rule that triggered

### Tracking Configuration Changes

1. Use System logs
2. Filter by `config` or `commit`
3. See who made changes and when

### Analyzing Denied Traffic

1. Switch to Traffic logs
2. Filter by `deny` or `drop`
3. Check which rule is blocking
4. Verify it's expected behavior

### Understanding Threat Actions

- **Alert** - Threat logged but allowed
- **Block** - Connection blocked
- **Reset client** - RST sent to client
- **Reset server** - RST sent to server
- **Reset both** - RST sent to both sides
- **Drop** - Silently dropped
