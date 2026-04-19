# Logs

System, traffic, and threat logs. Analyze group (`2`). Cycle log type
with `]` (forward) or `[` (backward).

## System logs

Config changes, HA state changes, errors / warnings, auth events,
license events.

| Column      | Description                                     |
|-------------|-------------------------------------------------|
| Time        | Event time                                      |
| Severity    | Info / Warning / Error / Critical               |
| Event Type  | Category                                        |
| Description | Details                                         |

## Traffic logs

Session traffic.

| Column       | Description                                  |
|--------------|----------------------------------------------|
| Time         | Session end time                             |
| Source       | Source IP                                    |
| Destination  | Destination IP                               |
| App          | Identified application                       |
| Action       | Allow / Deny / Drop / Reset                  |
| Rule         | Matched security rule                        |
| Bytes        | Transferred                                  |

## Threat logs

Security events.

| Column       | Description                                    |
|--------------|------------------------------------------------|
| Time         | Detection time                                 |
| Threat       | Signature / threat name                        |
| Severity     | Critical / High / Medium / Low / Info          |
| Source       | Threat source IP                               |
| Destination  | Target IP                                      |
| Action       | Alert / Block / Reset / …                      |

## View-specific keys

| Key   | Action                                               |
|-------|------------------------------------------------------|
| `]`   | Next log type (System → Traffic → Threat)            |
| `[`   | Previous log type                                    |

## Standard keys

Filter (`/`), sort (`s` cycle, `S` reverse), detail (`Enter`), refresh
(`r`). Defaults to time-descending (newest first). See
[keybindings.md](../keybindings.md).

## Threat action glossary

- **Alert** — logged, traffic allowed
- **Block** — connection blocked
- **Reset client / server / both** — RST sent to the named side(s)
- **Drop** — silently dropped

## Tips

- Threat investigation: `]` twice → sort by Severity → expand.
- Config audit: System logs → filter `config` or `commit`.
- Denied traffic: Traffic logs → filter `deny` / `drop` → check the
  Rule column.
