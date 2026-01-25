# NAT Policies

The NAT Policies view displays NAT (Network Address Translation) rules from the firewall. Access it through the Analyze group (press `2`, then cycle with `Tab` or `2`).

## Display

Each NAT rule shows:

| Column | Description |
|--------|-------------|
| # | Rule position |
| Name | Rule name |
| Type | NAT type (ipv4, nat64, nptv6) |
| From | Source zone(s) |
| To | Destination zone(s) |
| Source | Original source addresses |
| Destination | Original destination addresses |
| Service | Original service/port |
| Src Translation | Source address translation (SNAT) |
| Dst Translation | Destination address translation (DNAT) |
| Hits | Number of times rule matched |

### Translation Types

**Source NAT (SNAT)**
- Dynamic IP and Port (many-to-one)
- Dynamic IP (many-to-many)
- Static IP (one-to-one)

**Destination NAT (DNAT)**
- Static translation to internal IP
- Port forwarding

### Visual Indicators

- **Disabled rules** are shown with strikethrough text
- Translation columns show the translated address or "none" if not translating

## Filtering

Press `/` to enter filter mode. The filter matches against:

- Rule name
- Source and destination addresses
- Translation addresses
- Zone names

Filter examples:
- `10.0.0` - Rules involving addresses starting with 10.0.0
- `web-server` - Rules named "web-server" or translating to that address
- `outside` - Rules involving the outside zone

Press `Esc` to clear the filter.

## Sorting

Press `s` to cycle through sort fields:

1. **Position** (default) - Rulebase order
2. **Name** - Alphabetical
3. **Hits** - By hit count

Press `S` (shift+s) to toggle sort direction.

## Rule Details

Press `Enter` on a rule to expand its details. The expanded view shows:

- Complete address lists
- Bi-directional translation settings
- Interface-based source translation
- Fallback settings
- Description and tags

Press `Enter` again or `Esc` to collapse.

## Keybindings

| Key | Action |
|-----|--------|
| `j` / `Down` | Move cursor down |
| `k` / `Up` | Move cursor up |
| `g` / `Home` | Jump to first rule |
| `G` / `End` | Jump to last rule |
| `Ctrl+d` / `PgDn` | Page down |
| `Ctrl+u` / `PgUp` | Page up |
| `/` | Enter filter mode |
| `Esc` | Clear filter |
| `s` | Cycle sort field |
| `S` | Toggle sort direction |
| `Enter` | Toggle rule details |
| `r` | Refresh NAT policies |

## Tips

### Understanding NAT Hit Counts

NAT hit counts show how many sessions have matched the rule. High hit counts on source NAT rules are normal for outbound traffic. Low hit counts on destination NAT rules might indicate the service isn't being accessed.

### Finding Translation Issues

1. Filter by the internal or external IP you're troubleshooting
2. Check that the correct NAT rule matches
3. Verify the translation addresses are correct

### NAT Policy Order

Like security policies, NAT rules are processed in order. If traffic matches the wrong NAT rule, check if there's a more specific rule that should be higher in the rulebase.
