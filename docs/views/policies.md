# Security Policies

Security rulebase browser. Monitor group (`2`).

## Columns

| Column        | Description                                   |
|---------------|-----------------------------------------------|
| `#`           | Position in the rulebase                      |
| Name          | Rule name                                     |
| Action        | Allow / Deny / Drop / Reset                   |
| From          | Source zone(s)                                |
| To            | Destination zone(s)                           |
| Source        | Source addresses / groups                     |
| Destination   | Destination addresses / groups                |
| Application   | Matched applications                          |
| Service       | Port / protocol                               |
| Hits          | Hit count                                     |
| Last Hit      | When the rule last matched                    |

- Disabled rules render with strikethrough.
- Zero-hit rules are highlighted.

## Filter (`/`)

Matches against rule name, zone names, application names, tag names.
Case-insensitive substring match. Examples: `web`, `trust`, `deny`.

## Sort (`s` to cycle, `S` to reverse)

Position → Name → Hits → Last Hit.

## Detail (`Enter`)

Expands to show: full source / destination address lists, all
applications, security profiles, log settings, description, tags,
rule UUID.

## Standard keys

Navigation, filter, sort, refresh — see
[keybindings.md](../keybindings.md).

## Tips

- Sort by Hits to find zero-hit rules (cleanup candidates).
- Sort by Last Hit to find rules that haven't matched recently.
- `G` jumps to the bottom — often where catch-all rules live.
