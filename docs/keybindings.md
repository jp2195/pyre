# Keybindings Reference

Complete reference of keyboard shortcuts in pyre.

## Global Keybindings

These work from any view (except when in filter mode or text input).

### Navigation

| Key | Action |
|-----|--------|
| `1` | Monitor group (Overview, Network, Security, VPN) |
| `2` | Analyze group (Policies, NAT, Sessions, Interfaces, Logs) |
| `3` | Tools group (Config) |
| `4` | Connections group (Switch Device) |
| `Tab` | Next view in current group |
| `Ctrl+P` | Open command palette |
| `:` | Open connection picker |
| `D` | Open device picker (Panorama only) |

### Actions

| Key | Action |
|-----|--------|
| `r` | Refresh current view |
| `?` | Toggle help overlay |
| `q` | Quit application |
| `Ctrl+C` | Quit application |

## List Navigation

These work in views with scrollable lists (Policies, Sessions, Logs, etc.).

| Key | Action |
|-----|--------|
| `j` / `Down` | Move cursor down |
| `k` / `Up` | Move cursor up |
| `g` / `Home` | Jump to top |
| `G` / `End` | Jump to bottom |
| `Ctrl+d` / `PgDn` | Page down |
| `Ctrl+u` / `PgUp` | Page up |

## Filter Mode

| Key | Action |
|-----|--------|
| `/` | Enter filter mode |
| `Enter` | Apply filter |
| `Esc` | Clear filter / Cancel |

When in filter mode, type your search query. Filters support partial matching.

## Sorting

| Key | Action |
|-----|--------|
| `s` | Cycle through sort fields |
| `S` | Toggle sort direction (asc/desc) |

## View-Specific Keybindings

### Dashboard Views

| Key | Action |
|-----|--------|
| `Tab` | Cycle dashboard sub-views (within Monitor group) |

### Policies View

| Key | Action |
|-----|--------|
| `/` | Filter by name, tag, zone, or application |
| `s` | Cycle sort (position, name, hits, last hit) |
| `Enter` | Toggle rule detail view |

### NAT Policies View

| Key | Action |
|-----|--------|
| `/` | Filter by name or translation |
| `s` | Cycle sort (position, name, hits) |
| `Enter` | Toggle rule detail view |

### Sessions View

| Key | Action |
|-----|--------|
| `/` | Filter by IP, application, zone, rule, or user |
| `s` | Cycle sort (ID, bytes, age, application) |
| `Enter` | Toggle session detail view |

### Interfaces View

| Key | Action |
|-----|--------|
| `/` | Filter by name, zone, IP, or state |
| `s` | Cycle sort (name, zone, state, IP) |
| `Enter` | Toggle interface detail view |

### Logs View

| Key | Action |
|-----|--------|
| `]` | Next log type (System -> Traffic -> Threat) |
| `[` | Previous log type (System -> Threat -> Traffic) |
| `/` | Filter log entries |
| `s` | Cycle sort field |
| `S` | Toggle sort direction |
| `Enter` | Toggle log detail view |

### Connection Hub

| Key | Action |
|-----|--------|
| `j`/`k` | Navigate up/down |
| `Enter` | Connect to selected connection |
| `n` | Add new connection |
| `e` | Edit selected connection |
| `d` | Delete selected connection |
| `Esc` | Back / Cancel |

### Connection Form

| Key | Action |
|-----|--------|
| `Tab` | Move to next field |
| `Shift+Tab` | Move to previous field |
| `Enter` | Submit form |
| `Esc` | Cancel and go back |

### Connection Picker

| Key | Action |
|-----|--------|
| `j`/`k` | Navigate up/down |
| `Enter` | Connect to selected firewall |
| `x` | Disconnect selected |
| `Esc` / `:` | Close picker |

### Device Picker (Panorama)

| Key | Action |
|-----|--------|
| `j`/`k` | Navigate up/down |
| `Enter` | Select device to target |
| `r` | Refresh device list |
| `Esc` / `D` | Close picker |

### Command Palette

| Key | Action |
|-----|--------|
| Type | Filter commands |
| `j`/`k` / `Up`/`Down` | Navigate results |
| `Enter` | Execute selected command |
| `Esc` | Close palette |

## Login Screen

| Key | Action |
|-----|--------|
| `Tab` | Move to next field |
| `Enter` | Submit (when all fields filled) |
| `Ctrl+C` | Quit |
