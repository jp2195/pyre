package views

import (
	"fmt"
	"sort"
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
	sort.Slice(logs, func(i, j int) bool {
		var less bool
		switch sortBy {
		case LogSortSeverity:
			less = severityRank(logs[i].Severity) < severityRank(logs[j].Severity)
		default: // LogSortTime
			less = logs[i].Time.Before(logs[j].Time)
		}
		if !asc {
			return !less
		}
		return less
	})
}

func (m LogsModel) renderSystemTable() string {
	if m.Loading && len(m.systemLogs) == 0 {
		return LoadingMsgStyle.Padding(1, 0).Render("Loading system logs...")
	}
	if len(m.filteredSystem) == 0 {
		return EmptyMsgStyle.Padding(1, 0).Render("No system logs found")
	}

	var b strings.Builder

	// Header - compact severity, more space for description
	header := fmt.Sprintf("%-19s %-4s %-18s %s",
		"Time", "Sev", "Type", "Description")
	b.WriteString(TableHeaderStyle.Render(header) + "\n")

	visibleRows := m.visibleRows()
	end := min(m.Offset+visibleRows, len(m.filteredSystem))

	for i := m.Offset; i < end; i++ {
		log := m.filteredSystem[i]
		isSelected := i == m.Cursor

		timeStr := log.Time.Format("2006-01-02 15:04:05")
		sevAbbrev := abbreviateSeverity(log.Severity)
		desc := truncate(log.Description, m.Width-46)

		if isSelected {
			row := fmt.Sprintf("%-19s %-4s %-18s %s",
				timeStr,
				sevAbbrev,
				truncate(log.Type, 18),
				desc)
			b.WriteString(TableRowSelectedStyle.Render(row) + "\n")
		} else {
			// Build row with colored severity indicator
			sevStyle := SeverityStyle(log.Severity)

			row := DetailLabelStyle.Render(fmt.Sprintf("%-19s", timeStr)) + " " +
				sevStyle.Render(fmt.Sprintf("%-4s", sevAbbrev)) + " " +
				StatusMutedStyle.Render(fmt.Sprintf("%-18s", truncate(log.Type, 18))) + " " +
				DetailValueStyle.Render(desc)
			b.WriteString(row + "\n")
		}
	}

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
