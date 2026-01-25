# Troubleshooting

The Troubleshoot view provides interactive troubleshooting runbooks that execute commands via SSH. Access it through the Tools group (press `3`).

## Requirements

Troubleshooting runbooks require SSH access to the firewall. See [SSH Setup](../ssh-setup.md) for configuration instructions.

## Using Runbooks

### Runbook Categories

Runbooks are organized by category:

- **Network** - Connectivity, interfaces, routing
- **Security** - Policy analysis, session investigation
- **System** - Resources, HA, licenses
- **VPN** - IPSec and GlobalProtect diagnostics

### Running a Runbook

1. Press `3` to open the Tools group (Troubleshoot view)
2. Use `Tab` to select a runbook category
3. Navigate with `j`/`k` to select a runbook
4. Press `Enter` to start

### Following Prompts

Some runbooks require input. When prompted:
1. Enter the requested value (IP address, interface name, etc.)
2. Press `Enter` to continue

### Viewing Results

After a runbook completes:
- Each step shows the command executed
- Output is displayed below each command
- Recommendations appear based on findings
- Press `Esc` to return to the runbook list

## SSH Connection Status

The troubleshoot view shows SSH status:

- **SSH Available** - Ready to run runbooks
- **SSH Connecting** - Establishing connection
- **SSH Error** - Connection failed (see error message)
- **SSH Not Configured** - No SSH credentials in config

### Retrying SSH Connection

If SSH fails:
1. Press `R` (uppercase) to retry the connection
2. Check your SSH configuration if it continues to fail

## Available Runbooks

### Network Category

**Connectivity Test**
- Ping test to specified host
- Traceroute analysis
- DNS resolution check

**Interface Diagnostics**
- Interface state and counters
- Error analysis
- Traffic statistics

**ARP Analysis**
- ARP table inspection
- MAC address lookups
- ARP timeout issues

**Routing Verification**
- Routing table check
- Route lookup for destination
- BGP/OSPF status (if configured)

### Security Category

**Policy Hit Analysis**
- Rule hit counts
- Zero-hit rule identification
- Shadow rule detection

**Session Investigation**
- Session lookup by IP
- NAT verification
- Application identification

**Threat Log Review**
- Recent threat activity
- Top threats by severity
- Blocked application analysis

### System Category

**Resource Check**
- CPU and memory usage
- Disk space
- Process analysis

**HA Status Verification**
- HA state and sync
- Peer connectivity
- Failover readiness

**License Status**
- License inventory
- Expiration warnings
- Feature availability

### VPN Category

**IPSec Tunnel Diagnostics**
- Tunnel status
- Phase 1/Phase 2 analysis
- Traffic flow verification

**GlobalProtect Status**
- Gateway status
- Connected users
- Authentication issues

## Keybindings

| Key | Action |
|-----|--------|
| `Tab` | Switch runbook category |
| `j` / `k` | Navigate runbooks |
| `Enter` | Run selected runbook |
| `R` | Retry SSH connection |
| `Esc` | Return to runbook list |
| `r` | Refresh runbook list |

## Tips

### Preparing for Troubleshooting

1. Ensure SSH is configured in your `~/.pyre.yaml`
2. Test SSH connectivity before needing it urgently
3. Have relevant information ready (IPs, interface names)

### Interpreting Results

- Green checkmarks indicate passed checks
- Yellow warnings suggest attention needed
- Red errors indicate problems found
- Follow recommendations for remediation

### When SSH Fails

Common causes:
- Incorrect SSH credentials
- Firewall not allowing SSH
- Network connectivity issues
- SSH key not authorized on firewall

See [SSH Setup](../ssh-setup.md) for troubleshooting SSH configuration.
