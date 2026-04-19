# NAT Policies

NAT rulebase browser. Analyze group (`2`).

## Columns

| Column            | Description                                      |
|-------------------|--------------------------------------------------|
| `#`               | Position                                         |
| Name              | Rule name                                        |
| Type              | `ipv4` / `nat64` / `nptv6`                       |
| From / To         | Source / destination zones                       |
| Source / Dest     | Original addresses                               |
| Service           | Original service / port                          |
| Src Translation   | SNAT target (or "none")                          |
| Dst Translation   | DNAT target (or "none")                          |
| Hits              | Hit count                                        |

- Disabled rules render with strikethrough.

### Translation types

- **SNAT**: dynamic IP and port (many-to-one), dynamic IP
  (many-to-many), static IP (one-to-one).
- **DNAT**: static to internal IP, port forwarding.

## Filter (`/`)

Rule name, source / destination addresses, translation addresses,
zones.

## Sort (`s` cycle, `S` reverse)

Position → Name → Hits.

## Detail (`Enter`)

Complete address lists, bi-directional translation settings,
interface-based source translation, fallback, description, tags.

## Standard keys

See [keybindings.md](../keybindings.md).
