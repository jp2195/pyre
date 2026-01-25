# Interfaces

The Interfaces view displays network interface status and statistics. Access it through the Analyze group (press `2`, then cycle to Interfaces).

## Display

Interfaces are displayed as cards showing:

| Field | Description |
|-------|-------------|
| Name | Interface name (e.g., ethernet1/1) |
| State | Up or Down |
| Zone | Assigned security zone |
| IP Address | Configured IP address |
| MAC Address | Hardware address |
| Virtual Router | Assigned virtual router |
| Speed/Duplex | Link speed and duplex mode |
| Bytes In/Out | Traffic counters |
| Packets In/Out | Packet counters |
| Errors | Error counts (if any) |

### State Indicators

- **Green/Up** - Interface is operational
- **Red/Down** - Interface is down
- **Yellow** - Interface has errors or warnings

### Interface Types

The view shows various interface types:
- **Physical** (ethernet1/1, etc.)
- **Aggregate** (ae1, etc.)
- **VLAN** (ethernet1/1.100, etc.)
- **Loopback** (loopback.1, etc.)
- **Tunnel** (tunnel.1, etc.)

## Filtering

Press `/` to enter filter mode. The filter matches against:

- Interface name
- Zone name
- IP address
- State (up/down)

Filter examples:
- `ethernet1` - Physical interfaces on slot 1
- `trust` - Interfaces in the trust zone
- `10.0` - Interfaces with IPs starting with 10.0
- `down` - Interfaces that are down

Press `Esc` to clear the filter.

## Sorting

Press `s` to cycle through sort fields:

1. **Name** (default) - Alphabetical by interface name
2. **Zone** - Grouped by zone
3. **State** - Up interfaces first or last
4. **IP** - By IP address

Press `S` (shift+s) to toggle sort direction.

## Interface Details

Press `Enter` on an interface to expand its details. The expanded view shows:

### Configuration
- Full interface configuration
- IP address and netmask
- Zone and virtual router assignment
- Link state settings

### Counters
- Detailed byte and packet counters
- Error and drop counters by type
- Multicast/broadcast statistics

### Physical Layer
- Speed and duplex negotiation
- Media type
- Hardware details

Press `Enter` again or `Esc` to collapse.

## Keybindings

| Key | Action |
|-----|--------|
| `j` / `Down` | Move cursor down |
| `k` / `Up` | Move cursor up |
| `g` / `Home` | Jump to first interface |
| `G` / `End` | Jump to last interface |
| `Ctrl+d` / `PgDn` | Page down |
| `Ctrl+u` / `PgUp` | Page up |
| `/` | Enter filter mode |
| `Esc` | Clear filter |
| `s` | Cycle sort field |
| `S` | Toggle sort direction |
| `Enter` | Toggle interface details |
| `r` | Refresh interfaces |

## Tips

### Identifying Problem Interfaces

1. Filter by `down` to see non-operational interfaces
2. Sort by State to group down interfaces
3. Check error counters in the details view

### Monitoring Traffic Distribution

1. Look at Bytes In/Out across interfaces
2. High traffic interfaces may need capacity attention
3. Compare against expected traffic patterns

### Troubleshooting Connectivity

1. Verify the interface is Up
2. Check the IP address configuration
3. Verify zone assignment matches your policy
4. Look for errors or drops in the details

### Understanding Error Counters

Common error types:
- **Input errors** - Problems receiving packets (CRC, framing)
- **Output errors** - Problems transmitting packets
- **Drops** - Packets intentionally dropped (queue full, etc.)
- **Collisions** - Half-duplex collision issues
