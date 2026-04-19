package views

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/models"
)

// filterThreatLogs returns threat logs matching the query.
func filterThreatLogs(logs []models.ThreatLogEntry, query string) []models.ThreatLogEntry {
	if query == "" {
		result := make([]models.ThreatLogEntry, len(logs))
		copy(result, logs)
		return result
	}

	var result []models.ThreatLogEntry
	for _, log := range logs {
		if strings.Contains(strings.ToLower(log.SourceIP), query) ||
			strings.Contains(strings.ToLower(log.DestIP), query) ||
			strings.Contains(strings.ToLower(log.ThreatName), query) ||
			strings.Contains(strings.ToLower(log.Severity), query) ||
			strings.Contains(strings.ToLower(log.Action), query) ||
			strings.Contains(strings.ToLower(log.ThreatCategory), query) {
			result = append(result, log)
		}
	}
	return result
}

// sortThreatLogs sorts the slice in place by the given field.
func sortThreatLogs(logs []models.ThreatLogEntry, sortBy LogSortField, asc bool) {
	sort.Slice(logs, func(i, j int) bool {
		var less bool
		switch sortBy {
		case LogSortSeverity:
			less = severityRank(logs[i].Severity) < severityRank(logs[j].Severity)
		case LogSortSource:
			less = logs[i].SourceIP < logs[j].SourceIP
		case LogSortAction:
			less = logs[i].Action < logs[j].Action
		default: // LogSortTime
			less = logs[i].Time.Before(logs[j].Time)
		}
		if !asc {
			return !less
		}
		return less
	})
}

func (m LogsModel) renderThreatTable() string {
	if m.Loading && len(m.threatLogs) == 0 {
		return LoadingMsgStyle.Padding(1, 0).Render("Loading threat logs...")
	}
	if len(m.filteredThreat) == 0 {
		return EmptyMsgStyle.Padding(1, 0).Render("No threat logs found")
	}

	var b strings.Builder

	// Header
	header := fmt.Sprintf("%-19s %-9s %-20s %-15s %-7s %-15s",
		"Time", "Severity", "Threat", "Source", "Action", "Category")
	b.WriteString(TableHeaderStyle.Render(header) + "\n")

	visibleRows := m.visibleRows()
	end := min(m.Offset+visibleRows, len(m.filteredThreat))

	for i := m.Offset; i < end; i++ {
		log := m.filteredThreat[i]
		isSelected := i == m.Cursor

		timeStr := log.Time.Format("2006-01-02 15:04:05")

		row := fmt.Sprintf("%-19s %-9s %-20s %-15s %-7s %-15s",
			timeStr,
			truncate(log.Severity, 9),
			truncate(log.ThreatName, 20),
			truncate(log.SourceIP, 15),
			truncate(log.Action, 7),
			truncate(log.ThreatCategory, 15))

		if isSelected {
			b.WriteString(TableRowSelectedStyle.Render(row) + "\n")
		} else {
			// Color code by severity
			row = colorBySeverity(row, log.Severity)
			b.WriteString(row + "\n")
		}
	}

	return b.String()
}

func (m LogsModel) renderThreatDetail(log models.ThreatLogEntry) string {
	panelStyle := DetailPanelStyle.Width(m.Width - 2)
	labelStyle := DetailLabelStyle.Width(14)

	var lines []string
	lines = append(lines, ViewTitleStyle.Render("Threat Log Details"))

	lines = append(lines, DetailSectionStyle.Render("Threat"))
	lines = append(lines, labelStyle.Render("Time")+DetailValueStyle.Render(log.Time.Format("2006-01-02 15:04:05")))
	lines = append(lines, labelStyle.Render("Severity")+colorBySeverity(log.Severity, log.Severity))
	lines = append(lines, labelStyle.Render("Threat Name")+DetailValueStyle.Render(log.ThreatName))
	lines = append(lines, labelStyle.Render("Threat ID")+DetailValueStyle.Render(fmt.Sprintf("%d", log.ThreatID)))
	lines = append(lines, labelStyle.Render("Category")+DetailValueStyle.Render(log.ThreatCategory))
	lines = append(lines, labelStyle.Render("Subtype")+DetailValueStyle.Render(log.Subtype))
	lines = append(lines, labelStyle.Render("Action")+colorByAction(log.Action, log.Action))
	lines = append(lines, labelStyle.Render("Direction")+DetailValueStyle.Render(log.Direction))

	lines = append(lines, DetailSectionStyle.Render("Source / Destination"))
	lines = append(lines, labelStyle.Render("Source")+DetailValueStyle.Render(fmt.Sprintf("%s:%d (%s)", log.SourceIP, log.SourcePort, log.SourceZone)))
	lines = append(lines, labelStyle.Render("Destination")+DetailValueStyle.Render(fmt.Sprintf("%s:%d (%s)", log.DestIP, log.DestPort, log.DestZone)))

	lines = append(lines, DetailSectionStyle.Render("Context"))
	lines = append(lines, labelStyle.Render("Application")+DetailValueStyle.Render(log.Application))
	lines = append(lines, labelStyle.Render("Rule")+DetailValueStyle.Render(log.Rule))
	if log.User != "" {
		lines = append(lines, labelStyle.Render("User")+DetailValueStyle.Render(log.User))
	}
	if log.URL != "" {
		lines = append(lines, labelStyle.Render("URL")+DetailValueStyle.Render(truncate(log.URL, m.Width-20)))
	}
	if log.Filename != "" {
		lines = append(lines, labelStyle.Render("Filename")+DetailValueStyle.Render(log.Filename))
	}

	return panelStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}
