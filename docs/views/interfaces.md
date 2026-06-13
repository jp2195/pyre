# Interfaces View

Interface status and counters. Uses the
[standard view chrome](README.md#standard-view-chrome).

## Columns

The `St` prefix column is a colored state bullet: `●` green = up,
`○` red = down.

| Breakpoint | Columns |
|------------|---------|
| ≥ 120 | `St`, `Name`, `Type`, `Zone`, `IP`, `MAC`, `VR` |
| ≥ 90 | `St`, `Name`, `Type`, `Zone`, `IP`, `VR` |
| < 90 | `St`, `Name`, `Zone`, `IP` |

`MAC` and `VR` are dropped at narrower widths. `Type` is dropped at the
narrowest breakpoint. Speed, duplex, counters, and ARP entries are in
the detail panel — they are not columns.

## Sort fields

Cycled with `s`; direction toggled with `S`. All fields default to
ascending.

| Index | Label | Notes |
|-------|-------|-------|
| 0 | Name | alphabetical |
| 1 | Zone | alphabetical |
| 2 | State | up interfaces sort first; ties broken by name |
| 3 | IP | lexicographic |

## Filter scope

Matches (case-insensitive substring) against: name, zone, IP, state,
type, virtual router.

## Detail panel (`enter`)

Two-column layout at ≥ 100 wide; single column otherwise. Sections:

- **Basic Information** — State (● UP / ○ DOWN), Type, Zone, Mode,
  Vsys (if set).
- **Network** — IP Address, MAC Address, Virtual Router, MTU (if > 0),
  VLAN Tag (if > 0).
- **Physical** — Speed, Duplex.
- **Traffic Statistics** (if any counters > 0) — Bytes In/Out, Packets
  In/Out, Errors in/out (if non-zero), Drops in/out (if non-zero).
- **ARP Entries** (if any for this interface) — up to 5 entries showing
  status bullet, IP, and MAC; count of additional entries shown if more.
