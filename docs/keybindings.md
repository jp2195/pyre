# Keybindings & Navigation

## Navigation model

Views are organized into three numbered groups. Press a number to switch
groups; press it again (or `Tab`) to cycle through views within the
current group.

| Key | Group   | Views                                         |
|-----|---------|-----------------------------------------------|
| `1` | Monitor | Overview · Network · Security · VPN           |
| `2` | Analyze | Policies · NAT · Sessions · Interfaces · Logs |
| `3` | Tools   | Config                                        |

The header shows the group tabs on top and the sub-tabs for the active
group underneath.

## Global

| Key         | Action                                          |
|-------------|-------------------------------------------------|
| `1` `2` `3` | Switch (or cycle within) navigation group       |
| `Tab`       | Next view in the current group                  |
| `Ctrl+P`    | Command palette — fuzzy jump anywhere           |
| `:`         | Connection picker (switch between firewalls)    |
| `D`         | Device picker (Panorama only)                   |
| `r`         | Refresh current view                            |
| `?`         | Toggle help overlay                             |
| `q`         | Quit                                            |
| `Ctrl+C`    | Quit                                            |

## List navigation

Works in any view that renders a scrollable list (Policies, Sessions,
Logs, Interfaces, …).

| Key                 | Action           |
|---------------------|------------------|
| `j` / `Down`        | Move down        |
| `k` / `Up`          | Move up          |
| `g` / `Home`        | Jump to top      |
| `G` / `End`         | Jump to bottom   |
| `Ctrl+d` / `PgDn`   | Page down        |
| `Ctrl+u` / `PgUp`   | Page up          |

## Filter

| Key     | Action                              |
|---------|-------------------------------------|
| `/`     | Enter filter mode                   |
| `Enter` | Apply filter                        |
| `Esc`   | Clear filter / exit filter mode     |

Filters use partial (substring) matching.

## Sort

| Key | Action                                |
|-----|---------------------------------------|
| `s` | Cycle through sort fields             |
| `S` | Toggle direction (most views) / cycle sort field (Objects) |

## Per-view

### Dashboard (group 1)

- `Tab` — cycle sub-views (Overview → Network → Security → VPN)

### Policies (group 2)

- `/` filter by name, tag, zone, or application
- `s` cycle sort: position, name, hits, last hit
- `Enter` toggle rule detail

### NAT (group 2)

- `/` filter by name or translation
- `s` cycle sort: position, name, hits
- `Enter` toggle rule detail

### Sessions (group 2)

- `/` filter by IP, application, zone, rule, or user
- `s` cycle sort: ID, bytes, age, application
- `d` / `Enter` toggle session detail

### Interfaces (group 2)

- `/` filter by name, zone, IP, or state
- `s` cycle sort: name, zone, state, IP
- `Enter` toggle interface detail

### Logs (group 2)

- `]` next log type (System → Traffic → Threat)
- `[` previous log type
- `/` filter
- `s` cycle sort
- `S` toggle direction
- `Enter` toggle log detail

## Modal views

### Command palette (`Ctrl+P`)

| Key               | Action                   |
|-------------------|--------------------------|
| type              | fuzzy-filter commands    |
| `j`/`k` / arrows  | navigate results         |
| `Enter`           | execute                  |
| `Esc`             | close                    |

### Connection picker (`:`)

| Key        | Action                       |
|------------|------------------------------|
| `j`/`k`    | navigate                     |
| `Enter`    | connect                      |
| `x`        | disconnect selected          |
| `Esc`, `:` | close                        |

### Device picker (`D`, Panorama only)

| Key        | Action                                         |
|------------|------------------------------------------------|
| `j`/`k`    | navigate                                       |
| `Enter`    | select device (or Panorama to target itself)   |
| `r`        | refresh managed-device list                    |
| `Esc`, `D` | close                                          |

On a standalone firewall connection, `D` falls through to the current
view's own handlers instead of opening a picker.

### Connection Hub (launch screen)

| Key   | Action                              |
|-------|-------------------------------------|
| `j`/`k` | navigate                          |
| `Enter` | connect                           |
| `n`   | add new connection                  |
| `e`   | edit selected                       |
| `d`   | delete selected                     |
| `Esc` | back / cancel                       |

### Connection form

| Key         | Action                |
|-------------|-----------------------|
| `Tab`       | next field            |
| `Shift+Tab` | previous field        |
| `Enter`     | submit                |
| `Esc`       | cancel and go back    |

### Login screen

| Key         | Action                                                      |
|-------------|-------------------------------------------------------------|
| `Tab`       | next field                                                  |
| `Enter`     | submit (all fields filled)                                  |
| `Esc`       | return to Connection Hub; form buffers are cleared          |
| `Ctrl+C`    | quit                                                        |
