# Dashboard

The Dashboard provides at-a-glance monitoring of your firewall. It's divided into four sub-views that you access through the Monitor group (press `1`).

## Overview (Default)

The main dashboard shows system health and resource utilization.

### System Information
- Hostname and model
- Serial number
- PAN-OS version
- Uptime

### Resources
- CPU utilization (data plane and management plane)
- Memory usage
- Session count (current/max)
- Disk usage

### HA Status
- HA mode (active-passive, active-active, standalone)
- State (active, passive, initial, etc.)
- Peer information and sync status
- Last state change

### Session Summary
- Active sessions
- Sessions per second
- TCP/UDP/ICMP breakdown

### Additional Information
- Logged-in administrators
- Active licenses and expiration
- Recent jobs
- Environmental sensors (temperature, fans)
- Certificate expiration warnings
- NAT pool utilization

## Network

The Network sub-view focuses on network connectivity.

### Top Interfaces
- Interfaces sorted by traffic volume
- Shows bytes in/out and packets
- State indicators (up/down)

### Interface Errors
- Interfaces with errors or drops
- Error counts and types

### ARP Table
- Recent ARP entries
- IP to MAC mappings
- Interface and status

### Routing Summary
- Route count by type
- Default route information
- Static vs dynamic routes

## Security

The Security sub-view shows threat activity and policy effectiveness.

### Threat Summary
- Threats blocked (last 24h)
- Threats by severity (critical, high, medium, low)
- Threats by type (virus, spyware, vulnerability)

### Rule Analysis
- Zero-hit rules (candidates for cleanup)
- Most-hit rules (highest traffic)
- Disabled rules count

### Top Applications
- Most frequently matched applications
- Session counts per application

## VPN

The VPN sub-view monitors VPN connectivity.

### IPSec Tunnels
- Tunnel status (up/down)
- Peer gateway
- Encryption/authentication
- Traffic counters

### GlobalProtect
- Active user count
- Users by gateway
- Recent connections

## Keybindings

| Key | Action |
|-----|--------|
| `1` | Cycle to next Monitor sub-view |
| `Tab` | Cycle to next Monitor sub-view |
| `r` | Refresh dashboard data |
| `?` | Help overlay |

## Navigation

The Dashboard sub-views are part of the Monitor group:

- Press `1` once to go to Monitor group (Overview by default)
- Press `1` again to cycle: Overview -> Network -> Security -> VPN -> Overview
- Or use `Tab` to cycle through sub-views
- Or use the command palette (`Ctrl+P`) to jump directly to any sub-view
