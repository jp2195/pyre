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
| `t` | Cycle time range forward: 15m → 1h → 24h → 7d → all |
| `T` | Cycle time range backward |
| `f` | Open the device query bar (raw PAN-OS expression) |
| `m` | Load the next page (appends; cursor stays where it was) |
| `s` | Cycle sort field (resets cursor) |
| `S` | Toggle sort direction |
| `/` | Open filter input |
| `enter` | Toggle detail panel |
| `esc` | Clear active filter (no detail-collapse behavior — `esc` only clears filter in Logs) |
| `r` | Refresh (app-level) |

Switching tabs resets the cursor and collapses any open detail panel.
Only the tab on screen is fetched — the other two tabs are left alone
until you switch to them, and each tab remembers the range and query it
was last fetched under, so switching range or query marks the other two
stale without refetching them right away.

## Time range (`t` / `T`)

`t` cycles forward and `T` cycles backward through five presets: `15m`,
`1h`, `24h`, `7d`, `all`. `all` is the default and sends no time bound
at all — the same behavior the view had before server-side queries
existed. Changing the range refetches the active tab from its first
page; the other two tabs refetch the next time they are shown.

## Device query (`f`)

`f` opens a bar that sends a raw PAN-OS log expression to the device,
combined with the active time range's bound. `Enter` commits the text
and refetches page one; `Esc` discards the edit and leaves the
previously accepted query in place. This is a different filter from
`/`: `/` only ever narrows rows already loaded into pyre, while `f`
changes what the firewall itself matches and sends back.

Examples:

- `(addr.src in 203.0.113.5)`
- `(app eq ssl) and (action eq deny)`

Two device behaviors are worth knowing before writing one of these by
hand (verified on a PA-440 running PAN-OS 11.2.10-h8):

- **Bad queries are rejected synchronously with a useful message.**
  `(bogus_field eq x)` comes back as something like `Invalid operator eq
  for field bogus_field`, and the rows already on screen are left alone
  rather than cleared.
- **A malformed time bound is accepted and silently matches nothing.**
  pyre already adds its own `receive_time` bound from the time-range
  preset, so a query typed into the bar does not normally need one. An
  operator who writes an explicit `receive_time` clause anyway should
  know the device does not validate it: `(receive_time geq '2025/13/45
  99:99:99')` returned zero rows with no error at all. This is exactly
  why a zero-row result always prints the assembled expression that was
  sent — otherwise a malformed bound looks identical to "no logs match."

## Paging (`m`)

`m` loads the next page and appends it to the tab, leaving the cursor
where it was. A page is capped at 5000 rows — PAN-OS rejects a larger
request — though pyre's default page size is smaller than that cap.

PAN-OS returns no total match count for a log query anywhere in its
response, so pyre cannot say "500 of 12,000" even if it wanted to. The
status line instead reports "500 shown, more available" when the last
page came back exactly full (the only signal that more rows might
exist), and plain "500 shown" once a page comes back short.

## Filter behavior

While the filter input is focused, `enter` commits the filter and
resets the cursor; `esc` exits the input without clearing the typed
text (text is preserved but not committed — the filter does not update
until `enter`). This differs from the standard chrome: Logs re-applies
the filter only on `enter`, not on `esc`.

`/` and `f` compose: `f` narrows what the device sends back, `/` then
narrows what is shown from the rows already loaded. An empty table
names which one is responsible — the filter and the count of loaded
rows it searched, if `/` matched nothing; the assembled expression sent
to the device, if `f` did.
