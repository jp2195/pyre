# Dashboard

At-a-glance firewall monitoring. Four sub-views reachable via the Monitor
group (`1`). Press `1` again or `Tab` to cycle; `Ctrl+P` jumps directly.

## Overview

System health + resource utilization.

- **System** — hostname, model, serial, PAN-OS version, uptime.
- **Resources** — CPU (data plane + management plane), memory, session
  count (current / max), disk.
- **HA** — mode (active-passive / active-active / standalone), state,
  peer info, sync status, last state change.
- **Sessions summary** — active, sessions/sec, TCP/UDP/ICMP breakdown.
- **Extras** — logged-in admins, active licenses + expiration, recent
  jobs, environmental sensors, cert-expiration warnings, NAT pool use.

## Network

- **Top interfaces** — by traffic volume, bytes/packets in/out, state.
- **Interface errors** — interfaces with errors or drops.
- **ARP table** — recent entries, IP→MAC, interface, status.
- **Routing summary** — counts by type, default route, static vs
  dynamic.

## Security

- **Threat summary** — blocked in last 24h, by severity (critical /
  high / medium / low), by type (virus / spyware / vulnerability).
- **Rule analysis** — zero-hit rules (cleanup candidates), most-hit
  rules, disabled rule count.
- **Top applications** — most-matched, session counts per app.

## VPN

- **IPSec tunnels** — status, peer gateway, crypto, counters.
- **GlobalProtect** — active users, users by gateway, recent connects.

## Keys

Standard navigation applies — see [keybindings.md](../keybindings.md).
View-specific: none beyond cycling sub-views with `1` / `Tab`.
