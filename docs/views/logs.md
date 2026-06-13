# Logs View

System, traffic, and threat logs. Analyze group (`2`).

## Tab bar

Three tabs cycled with `]` (forward) and `[` (backward):

```
System (N)   Traffic (N)   Threat (N)          Sort: <field> <dir>  |  Updated Xs ago
```

Each tab label shows the live filtered count for that log type. The
right side of the tab bar shows the current sort field/direction and
how long ago the data was last fetched. The tab bar updates in place
as the filter changes.

## System logs

Config changes, HA state, auth events, license events, etc.

### Columns (fixed layout)

| Column | Description |
|--------|-------------|
| Time | `2006-01-02 15:04:05` |
| Sev | Abbreviated severity: `CRIT`, `HIGH`, `MED`, `LOW`, `INFO` |
| Type | Event type / category (truncated to 18 chars) |
| Description | Event description (width-adjusted) |

Non-selected rows: Time in label style, Sev colored by severity, Type
muted, Description in value style.

### Filter scope

Matches against: description, type, severity.

### Sort fields

| Label | Notes |
|-------|-------|
| Time | Default; newest first (descending) |
| Severity | By severity rank |

(Source and Action sort labels exist in the sort cycle but system logs
fall through to Time for those fields.)

### Detail panel (`enter`)

Time, Severity (colored), Type, and the full description word-wrapped
to the panel width.

## Traffic logs

Session traffic events.

### Columns (fixed layout)

| Column | Description |
|--------|-------------|
| Time | `2006-01-02 15:04:05` |
| Action | `allow`, `deny`, `drop`, etc. (truncated to 7 chars) |
| Source | Source IP (truncated to 15 chars) |
| Dest | Destination IP (truncated to 15 chars) |
| App | Application (truncated to 12 chars) |
| Rule | Matched rule (truncated to 15 chars) |
| Bytes | Total bytes formatted |

Non-selected rows are colored by action (allow = green, deny/drop = red).

### Filter scope

Matches against: source IP, destination IP, application, rule, action,
user.

### Sort fields

| Label | Notes |
|-------|-------|
| Time | Default; newest first (descending) |
| Source | Source IP alphabetical |
| Action | Action alphabetical |

### Detail panel (`enter`)

**Session**: Time, Action (colored), Session ID, Duration.
**Source / Destination**: Source `IP:port (zone)`, Destination
`IP:port (zone)`, NAT Source (if translated), NAT Dest (if translated).
**Application**: Application, Protocol, Rule, User (if set).
**Traffic**: Bytes (total + sent/recv), Packets (total + sent/recv).

## Threat logs

Security events from threat profiles.

### Columns (fixed layout)

| Column | Description |
|--------|-------------|
| Time | `2006-01-02 15:04:05` |
| Severity | Full severity string (truncated to 9 chars) |
| Threat | Threat name / signature (truncated to 20 chars) |
| Source | Source IP (truncated to 15 chars) |
| Action | Action (truncated to 7 chars) |
| Category | Threat category (truncated to 15 chars) |

There is no Destination column. Non-selected rows are colored by
severity.

### Filter scope

Matches against: source IP, destination IP, threat name, severity,
action, threat category.

### Sort fields

| Label | Notes |
|-------|-------|
| Time | Default; newest first (descending) |
| Severity | By severity rank (critical > high > medium > low > informational) |
| Source | Source IP alphabetical |
| Action | Action alphabetical |

### Detail panel (`enter`)

**Threat**: Time, Severity (colored), Threat Name, Threat ID, Category,
Subtype, Action (colored), Direction.
**Source / Destination**: Source `IP:port (zone)`, Destination
`IP:port (zone)`.
**Context**: Application, Rule, User (if set), URL (if set), Filename
(if set).

## Keys

| Key | Action |
|-----|--------|
| `]` | Next log type (System → Traffic → Threat → System) |
| `[` | Previous log type (System → Threat → Traffic → System) |
| `s` | Cycle sort field (resets cursor) |
| `S` | Toggle sort direction |
| `/` | Open filter input |
| `enter` | Toggle detail panel |
| `esc` | Clear active filter (no detail-collapse behavior — `esc` only clears filter in Logs) |
| `r` | Refresh (app-level) |

Switching tabs resets the cursor and collapses any open detail panel.

## Filter behavior

While the filter input is focused, `enter` commits the filter and
resets the cursor; `esc` exits the input without clearing the typed
text (text is preserved but not committed — the filter does not update
until `enter`). This differs from the standard chrome: Logs re-applies
the filter only on `enter`, not on `esc`.
