# Dashboard

At-a-glance firewall monitoring. Four sub-views reachable via the Monitor
group (`1`). Press `1` again or `Tab` to cycle; `Ctrl+P` jumps directly.
The Config dashboard is under the Tools group (`3`).

## Overview

System health + resource utilization. Laid out in two columns (left:
Device Health; right: Operations & Security).

**Left column (always present)**

- **System** — hostname, model, serial, PAN-OS version, uptime.
- **Resources** — CPU (data plane + management plane), memory.
- **Sessions** — active/max with utilization bar, CPS (connections per
  second), throughput; TCP/UDP/ICMP protocol breakdown when available.
- **Disk Usage** — per-filesystem utilization bars (shown when data
  available).
- **Hardware Status** — environmental sensor readings (shown when data
  available).

**Right column (conditional panels, shown when data available)**

- **HA Status** — state (active/passive/suspended/initial), peer state,
  sync state. Shown only when HA is enabled; does not include a "last
  state change" field or "standalone" mode.
- **NAT Pool Utilization** — shown when NAT pools exist.
- **Content Versions** — always shown.
- **Licenses** — shown when license data is available.
- **Threat Summary** — shown when total threats > 0.
- **Admins Online** — logged-in admin sessions (shown when present).
- **GlobalProtect** — active users and gateway count (shown when GP is
  configured and has users or gateways).
- **Recent Jobs** — shown when job data is available.
- **Certificates** — expiring/expired certs (shown when present).

## Network

Network-layer view of interfaces, ARP, routing, and neighbors. Laid out in
two columns (left: interfaces; right: ARP, routing, neighbors).

- **Top Interfaces by Traffic** — up to 8 interfaces sorted by total
  bytes (in + out); shows state, bytes in, bytes out.
- **Interface Errors & Drops** — interfaces with non-zero error or drop
  counters; shows in/out errors and drops.
- **ARP Table** — count summary only: total entries and how many are
  complete (e.g. "12 entries (10 complete)"). Individual IP→MAC/interface
  entries are not listed.
- **Routing Table** — total route count plus protocol breakdown using
  single-letter abbreviations: C (connected), L (local), S (static),
  B (BGP), O (OSPF). No default-route or static-vs-dynamic breakdown.
- **Routing Neighbors** — BGP neighbor count (established/total) and
  OSPF neighbor count (full/total). Shown only when BGP or OSPF neighbors
  exist.

## Security

Threat visibility and policy analysis. Laid out in two columns (left:
threat data; right: policy analysis).

- **Threat Summary** — total threats; blocked vs alerted counts with
  progress bars.
- **Threats by Severity** — critical / high / medium / low counts with
  progress bars.
- **Zero-Hit Rules** — enabled security rules with no hits; count, percentage
  of active rules, and a list of up to 6 rule names with actions.
- **Most-Hit Rules** — top 8 security rules by hit count; name, action,
  and hit count.

Note: there is no "top applications" panel and no disabled-rule-count
metric in this dashboard.

## VPN

IPSec and GlobalProtect status. Laid out in two columns (left: IPSec;
right: GlobalProtect).

- **IPSec VPN Status** — tunnel up/down/init counts with a utilization
  bar.
- **IPSec Tunnels** — per-tunnel list (up to 10): state icon, name,
  gateway, and traffic bytes when available.
- **GlobalProtect Status** — active user count; breakdown by gateway when
  more than one gateway exists.
- **GlobalProtect Users** — per-user list (up to 12): username, virtual
  IP, and duration or login-time-ago.

Note: there is no "recent connects" field in the GlobalProtect panels.

## Config (Tools group, key `3`)

Policy statistics and pending configuration changes. Laid out in two
columns (left: statistics and pending changes; right: rule analysis).

- **Policy Statistics** — total rules, enabled count, allow/deny-drop
  breakdown, zero-hit count, and total hit count across all rules.
- **Pending Changes** — uncommitted change count; breakdown by user when
  multiple users have changes; list of up to 4 recent changes with type
  (add/edit/delete) and description.
- **Zero-Hit Rules** — enabled security rules with no hits; count,
  percentage of active rules, and a list of up to 8 rule names with
  actions.
- **Most-Hit Rules** — top 10 security rules by hit count; name, action,
  and hit count.

## Keys

Standard navigation applies — see [keybindings.md](../keybindings.md).
View-specific: none beyond cycling sub-views with `1` / `Tab`.
