# Security Policies

The Policies view displays security rules from the firewall's rulebase. Access it through the Analyze group (press `2`).

## Display

Each rule shows:

| Column | Description |
|--------|-------------|
| # | Rule position in the rulebase |
| Name | Rule name |
| Action | Allow, Deny, Drop, or Reset |
| From | Source zone(s) |
| To | Destination zone(s) |
| Source | Source addresses/groups |
| Destination | Destination addresses/groups |
| Application | Matched applications |
| Service | Port/protocol |
| Hits | Number of times rule matched |
| Last Hit | When the rule last matched traffic |

### Visual Indicators

- **Disabled rules** are shown with strikethrough text
- **Zero-hit rules** are highlighted for easy identification
- **High-hit rules** show larger hit counts

## Filtering

Press `/` to enter filter mode. The filter matches against:

- Rule name
- Zone names (source and destination)
- Application names
- Tag names

Filter examples:
- `web` - Rules with "web" in the name or application
- `trust` - Rules involving the trust zone
- `deny` - Rules with deny action

Filters are case-insensitive and support partial matching.

Press `Esc` to clear the filter.

## Sorting

Press `s` to cycle through sort fields:

1. **Position** (default) - Rulebase order
2. **Name** - Alphabetical
3. **Hits** - By hit count
4. **Last Hit** - By most recent match

Press `S` (shift+s) to toggle sort direction (ascending/descending).

## Rule Details

Press `Enter` on a rule to expand its details. The expanded view shows:

- Complete source/destination address lists
- All applications
- Security profiles applied
- Log settings
- Description and tags
- Rule UUID

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
| `r` | Refresh policies |

## Tips

### Finding Zero-Hit Rules

1. Press `s` to sort by Hits
2. Rules with zero hits appear at one end
3. These are candidates for cleanup or review

### Analyzing Rule Usage

1. Sort by Hits (descending) to see most-used rules
2. Sort by Last Hit to find stale rules that haven't matched recently

### Quick Navigation

- Use `g` to jump to the top of the rulebase
- Use `G` to jump to the bottom (often where catch-all rules are)
- Use `/` and filter by zone to see rules for specific traffic flows
