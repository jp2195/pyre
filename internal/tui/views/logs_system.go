package views

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/models"
)

// filterSystemLogs returns system logs matching the query.
func filterSystemLogs(logs []models.SystemLogEntry, query string) []models.SystemLogEntry {
	if query == "" {
		result := make([]models.SystemLogEntry, len(logs))
		copy(result, logs)
		return result
	}

	var result []models.SystemLogEntry
	for _, log := range logs {
		if strings.Contains(strings.ToLower(log.Description), query) ||
			strings.Contains(strings.ToLower(log.Type), query) ||
			strings.Contains(strings.ToLower(log.Severity), query) {
			result = append(result, log)
		}
	}
	return result
}

// sortSystemLogs sorts the slice in place by the given field.
func sortSystemLogs(logs []models.SystemLogEntry, sortBy LogSortField, asc bool) {
	slices.SortFunc(logs, func(a, b models.SystemLogEntry) int {
		var c int
		switch sortBy {
		case LogSortSeverity:
			c = cmp.Compare(severityRank(a.Severity), severityRank(b.Severity))
		default: // LogSortTime
			c = a.Time.Compare(b.Time)
		}
		if !asc {
			c = -c
		}
		return c
	})
}

// System table columns. The description is the flexible one: it is the whole
// reason to read the table, and the only field with no natural length.
const (
	sysTime        = 19
	sysTimeCompact = 8 // "15:04:05"
	sysSeverity    = 4
	sysType        = 18
)

func (m LogsModel) systemLayout() logColumnLayout {
	// +1 per gap between columns, including the one before the description.
	fixedWide := sysTime + sysSeverity + sysType + 3
	fixedNarrow := sysTime + sysSeverity + 2
	fixedCompact := sysTimeCompact + sysSeverity + 2
	// The full set is worth showing as soon as it fits: unlike the traffic
	// and threat tables it adds one column rather than a different shape.
	return logLayout(m.Width, [3]int{fixedWide, fixedNarrow, fixedCompact}, fixedWide+minFlexColumn)
}

// systemCells returns the fixed cells of a system log row, already padded, in
// the order the selected layout renders them. Header and data rows share it
// so their columns cannot drift apart.
func systemCells(l logColumnLayout, timeText, severity, logType string) []string {
	timeWidth := sysTime
	if l.compact() {
		timeWidth = sysTimeCompact
	}
	cells := []string{
		fmt.Sprintf("%-*s", timeWidth, truncate(timeText, timeWidth)),
		fmt.Sprintf("%-*s", sysSeverity, truncate(severity, sysSeverity)),
	}
	if l.wide() {
		cells = append(cells, fmt.Sprintf("%-*s", sysType, truncate(logType, sysType)))
	}
	return cells
}

func (m LogsModel) renderSystemTable() string {
	if m.Loading && len(m.systemLogs) == 0 {
		return LoadingMsgStyle.Padding(1, 0).Render("Loading system logs...")
	}
	if len(m.filteredSystem) == 0 {
		return EmptyMsgStyle.Padding(1, 0).Render("No system logs found")
	}

	layout := m.systemLayout()

	var b strings.Builder

	header := strings.Join(systemCells(layout, "Time", "Sev", "Type"), " ")
	if layout.flex > 0 {
		header += " " + fmt.Sprintf("%-*s", layout.flex, "Description")
	}
	b.WriteString(TableHeaderStyle.Render(header) + "\n")

	b.WriteString(renderLogRows(m.Offset, m.Cursor, m.visibleRows(), m.filteredSystem, func(log models.SystemLogEntry, selected bool) string {
		timeStr := log.Time.Format("2006-01-02 15:04:05")
		if layout.compact() {
			timeStr = log.Time.Format(logTimeOnlyLayout)
		}
		cells := systemCells(layout, timeStr, abbreviateSeverity(log.Severity), log.Type)
		desc := ""
		if layout.flex > 0 {
			desc = fmt.Sprintf("%-*s", layout.flex, truncate(log.Description, layout.flex))
		}

		if selected {
			row := strings.Join(cells, " ")
			if desc != "" {
				row += " " + desc
			}
			return TableSelectedRowStyle().Render(row)
		}

		// The severity cell is color-coded, so the row is styled cell by cell.
		styles := []lipgloss.Style{DetailLabelStyle, SeverityStyle(log.Severity), StatusMutedStyle}
		parts := make([]string, len(cells))
		for i, cell := range cells {
			parts[i] = styles[i].Render(cell)
		}
		row := strings.Join(parts, " ")
		if desc != "" {
			row += " " + DetailValueStyle.Render(desc)
		}
		return row
	}))

	return b.String()
}

func (m LogsModel) renderSystemDetail(log models.SystemLogEntry) string {
	panelStyle := DetailPanelStyle.Width(m.Width - 2)
	labelStyle := DetailLabelStyle.Width(12)

	// Word wrap the description for better readability
	descWidth := min(m.Width-10, 100)
	wrapped := wrapText(log.Description, descWidth)

	lines := make([]string, 0, 7+len(wrapped))
	lines = append(lines, ViewTitleStyle.Render("System Log Details"))
	lines = append(lines, "")
	lines = append(lines, labelStyle.Render("Time")+DetailValueStyle.Render(log.Time.Format("2006-01-02 15:04:05")))
	lines = append(lines, labelStyle.Render("Severity")+SeverityStyle(log.Severity).Render(log.Severity))
	lines = append(lines, labelStyle.Render("Type")+DetailValueStyle.Render(log.Type))
	lines = append(lines, "")
	lines = append(lines, ViewTitleStyle.Render("Message"))

	for _, line := range wrapped {
		lines = append(lines, DetailValueStyle.Render(line))
	}

	return panelStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}
