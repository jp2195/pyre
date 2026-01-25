# Navigation

pyre uses a group-based navigation system that organizes views into logical categories. This makes it easy to find what you need while keeping the interface clean.

## Navigation Groups

Press number keys `1-4` to switch between navigation groups:

| Key | Group | Views |
|-----|-------|-------|
| `1` | Monitor | Overview, Network, Security, VPN |
| `2` | Analyze | Policies, NAT, Sessions, Interfaces, Logs |
| `3` | Tools | Troubleshoot, Config |
| `4` | Connections | Switch Device |

## Navigating Within Groups

Once you're in a group, there are two ways to switch between views:

### Press the same number key
Pressing the same group number cycles through items in that group. For example, pressing `1` repeatedly cycles: Overview -> Network -> Security -> VPN -> Overview.

### Use Tab
Press `Tab` to move to the next view in the current group.

## The Navbar

The navbar appears at the top of the screen and shows:

1. **Group tabs** - Shows all four groups with the active one highlighted
2. **Sub-tabs** - Shows items in the current group with the active item highlighted

```
1:Monitor  2:Analyze  3:Tools  4:Conn                Overview
     Overview  Network(2)  Security(3)  VPN(4)
```

The numbers in parentheses show keyboard hints for inactive items within the group.

## Command Palette

Press `Ctrl+P` to open the command palette. This provides a searchable list of all available commands and views. Start typing to filter, then press `Enter` to select.

The command palette is organized by category:
- **Monitor** - Dashboard views (Overview, Network, Security, VPN)
- **Analyze** - Data views (Policies, NAT, Sessions, Interfaces, Logs)
- **Tools** - Utilities (Troubleshoot, Config)
- **Connections** - Device management
- **Actions** - Common actions (Refresh)
- **System** - Help, Quit

## Connection Picker

Press `:` to open the connection picker (or use the command palette filtered to Connections). This shows all configured firewalls and lets you switch between them.

In the picker:
- `j`/`k` or arrow keys to navigate
- `Enter` to connect
- `a` to add a new connection
- `x` to disconnect
- `Esc` to cancel

## Device Picker (Panorama)

When connected to Panorama, press `D` to open the device picker. This shows all managed firewalls and lets you select which device to target.

The device picker shows:
- Hostname
- Serial number
- Model
- HA state
- Connection status

Select "Panorama" to run commands directly on Panorama rather than a managed device.

## Quick Reference

| Key | Action |
|-----|--------|
| `1` | Monitor group (or cycle within) |
| `2` | Analyze group (or cycle within) |
| `3` | Tools group (or cycle within) |
| `4` | Connections group |
| `Tab` | Next view in current group |
| `Ctrl+P` | Command palette |
| `:` | Connection picker |
| `D` | Device picker (Panorama) |
| `?` | Help overlay |
| `q` | Quit |
