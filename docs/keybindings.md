# Keybindings & Navigation

## Navigation model

Views are organized into three numbered groups. Press a number to switch
groups; press it again (or `Tab`) to cycle through views within the
current group.

| Key | Group   | Views                                                                               |
|-----|---------|-------------------------------------------------------------------------------------|
| `1` | Monitor | Overview · Network · Security · VPN                                                 |
| `2` | Analyze | Policies · NAT · Objects · Sessions · Interfaces · Routes · IPSec · GP Users · Logs |
| `3` | Tools   | Config                                                                              |

The header shows the group tabs on top and the sub-tabs for the active
group underneath.

## Filter-mode guard (M8)

When any view's `/` filter input is focused, **all keys except `Ctrl+C`
are routed to the filter input**. Global keys (`q`, `r`, `1`–`3`, `?`,
`:`, `Tab`) do not fire while the user is typing a filter query. Press
`Enter` or `Esc` to leave filter mode before using global keys.

## Global

Active only when a main view is displayed and no filter input is focused.

| Key              | Action                                                    |
|------------------|-----------------------------------------------------------|
| `1` / `2` / `3`  | Switch (or cycle within) navigation group                 |
| `Tab`            | Next view in the current group                            |
| `Shift+Tab`      | Previous view in the current group                        |
| `Ctrl+P`         | Command palette — fuzzy jump anywhere                     |
| `:`              | Connection picker (switch between firewalls)              |
| `d`              | Device picker (Panorama only; falls through to view on standalone firewall) |
| `r`              | Refresh current view                                      |
| `?`              | Toggle help overlay                                       |
| `q` / `Ctrl+C`   | Quit                                                      |

## Table navigation

Works in any view that renders a scrollable list (Policies, Sessions,
Logs, Interfaces, …).

| Key                 | Action           |
|---------------------|------------------|
| `j` / `Down`        | Move down        |
| `k` / `Up`          | Move up          |
| `g` / `Home`        | Jump to top      |
| `G` / `End`         | Jump to bottom   |
| `Ctrl+D` / `PgDn`   | Page down        |
| `Ctrl+U` / `PgUp`   | Page up          |
| `Enter`             | Toggle detail panel |

## Filter

| Key     | Action                                                    |
|---------|-----------------------------------------------------------|
| `/`     | Enter filter mode                                         |
| `Enter` | Commit filter and reset cursor to top                     |
| `Esc`   | Exit filter mode (typed text is preserved in the input)   |

Filters use partial (substring) matching. Text typed while in filter mode
stays in the input after closing with `Esc`; press `Esc` again outside
filter mode to clear it.

## Sort

Applies to rule-list views (Policies, NAT, Sessions, Interfaces, IPSec,
GP Users) and Logs.

| Key | Action                                                                  |
|-----|-------------------------------------------------------------------------|
| `s` | Cycle to the next sort field; direction resets to that field's default  |
| `S` | Toggle sort direction (ascending ↔ descending)                          |

## Per-view extras

### Policies and NAT (group 2)

| Key     | Action                                                     |
|---------|------------------------------------------------------------|
| `s`     | Cycle sort field (resets direction to field default)       |
| `S`     | Toggle sort direction                                      |
| `Enter` | Toggle rule detail panel                                   |
| `Esc`   | Collapse expanded detail first; then clear filter on next press |

### Objects (group 2)

| Key     | Action                                                              |
|---------|---------------------------------------------------------------------|
| `Tab`   | Cycle Address ↔ Service tab                                         |
| `a`     | Jump to Address tab                                                 |
| `s`     | Jump to Service tab                                                 |
| `S`     | Cycle sort field for the active tab (always resets to ascending)    |
| `/`     | Enter filter mode for the active tab                                |
| `Enter` | Toggle detail panel for the selected object                         |
| `Esc`   | Collapse expanded detail first; then clear filter on next press     |

### Sessions (group 2)

| Key     | Action                                                              |
|---------|---------------------------------------------------------------------|
| `s`     | Cycle sort field                                                    |
| `S`     | Toggle sort direction                                               |
| `Enter` | Toggle basic detail panel                                           |
| `d`     | Fetch extended detail (only active while detail panel is expanded)  |
| `Esc`   | Collapse expanded detail first; then clear filter on next press     |

### Routes (group 2)

| Key     | Action                                                   |
|---------|----------------------------------------------------------|
| `[`     | Toggle between Routes and Neighbors tabs                 |
| `]`     | Toggle between Routes and Neighbors tabs                 |
| `a`     | Show all protocols (Routes tab only)                     |
| `c`     | Filter: Connected routes (Routes tab only)               |
| `s`     | Filter: Static routes (Routes tab only)                  |
| `b`     | Filter: BGP routes (Routes tab only)                     |
| `o`     | Filter: OSPF routes (Routes tab only)                    |
| `/`     | Enter text filter (Routes tab only)                      |

There is no sort-cycle key on Routes. The `s` key sets the protocol
filter to Static; it does not cycle sort fields. `Enter` toggles an
expansion state in the underlying model but has no visible effect because
Routes has no detail panel.

### Logs (group 2)

| Key     | Action                                              |
|---------|-----------------------------------------------------|
| `[`     | Cycle to previous log type (System → Threat → Traffic) |
| `]`     | Cycle to next log type (System → Traffic → Threat)  |
| `s`     | Cycle sort field                                    |
| `S`     | Toggle sort direction                               |
| `Enter` | Toggle log detail panel                             |
| `Esc`   | Clear filter (does not collapse the detail panel)   |

### Interfaces and IPSec and GP Users (group 2)

| Key     | Action                           |
|---------|----------------------------------|
| `s`     | Cycle sort field                 |
| `S`     | Toggle sort direction            |
| `Enter` | Toggle detail panel              |
| `Esc`   | Collapse detail, then clear filter |

## Modal views

### Command palette (`Ctrl+P`)

| Key               | Action                |
|-------------------|-----------------------|
| type              | Fuzzy-filter commands |
| `j`/`k` / arrows  | Navigate results      |
| `Enter`           | Execute selected      |
| `Esc`             | Close palette         |

### Connection picker (`:`)

| Key         | Action                          |
|-------------|---------------------------------|
| `j` / `k`   | Navigate                        |
| `Enter`     | Connect to selected             |
| `a`         | Add new connection (opens login)|
| `x`         | Disconnect selected             |
| `Esc` / `:` | Close                           |

### Device picker (`d`, Panorama only)

| Key         | Action                                       |
|-------------|----------------------------------------------|
| `j` / `k`   | Navigate                                     |
| `Enter`     | Select device                                |
| `r`         | Refresh managed-device list                  |
| `Esc` / `d` | Close                                        |

On a standalone firewall connection, `d` falls through to the current
view's own handlers instead of opening a picker.

### Connection Hub (launch screen)

`q` on this screen is **Quick Connect**, not Quit — use `Ctrl+C` to quit.

| Key              | Action                              |
|------------------|-------------------------------------|
| `j` / `k`        | Navigate list                       |
| `g` / `Home`     | Jump to top                         |
| `G` / `End`      | Jump to bottom                      |
| `Enter`          | Connect to selected                 |
| `n`              | Add new connection                  |
| `e`              | Edit selected                       |
| `d`              | Delete selected (prompts y/n)       |
| `q`              | Quick connect (open quick-connect form) |
| `Ctrl+C`         | Quit                                |

While the delete confirmation is shown:

| Key              | Action          |
|------------------|-----------------|
| `y` / `Y`        | Confirm delete  |
| `n` / `N` / `Esc`| Cancel delete   |

### Connection form

| Key           | Action                  |
|---------------|-------------------------|
| `Tab`         | Next field              |
| `Shift+Tab`   | Previous field          |
| `Space`       | Toggle checkbox field   |
| `Enter`       | Submit (when filled)    |
| `Esc`         | Cancel and go back      |
| `Ctrl+C`      | Quit                    |

### Login screen

| Key           | Action                                             |
|---------------|----------------------------------------------------|
| `Tab`         | Next field                                         |
| `Shift+Tab`   | Previous field                                     |
| `Space`       | Toggle insecure-skip-verify checkbox               |
| `Enter`       | Submit (when all required fields are filled)       |
| `Esc`         | Return to Connection Hub; form buffers are cleared |
| `Ctrl+C`      | Quit                                               |
