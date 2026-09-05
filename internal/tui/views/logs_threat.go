package views

import (
	"cmp"
	"fmt"
	"slices"
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
	slices.SortFunc(logs, func(a, b models.ThreatLogEntry) int {
		var c int
		switch sortBy {
		case LogSortSeverity:
			c = cmp.Compare(severityRank(a.Severity), severityRank(b.Severity))
		case LogSortSource:
			c = cmp.Compare(a.SourceIP, b.SourceIP)
		case LogSortAction:
			c = cmp.Compare(a.Action, b.Action)
		default: // LogSortTime
			c = a.Time.Compare(b.Time)
		}
		if !asc {
			c = -c
		}
		return c
	})
}

// Threat table columns. The wide set adds the destination, which matters for
// triage and had no column at all, and spells severity out in full.
const (
	thTime     = 19
	thSeverity = 13 // "informational"
	thSevShort = 4  // "INFO"
	thSource   = 15
	thDest     = 15
	thAction   = 12 // "reset-client"
	thCategory = 15

	// Compact variant: the date is dropped from the timestamp. What is left
	// is the threat, where it came from, and what the firewall did.
	thTimeCompact = 8 // "15:04:05"
)

func (m LogsModel) threatLayout() logColumnLayout {
	// +1 per gap between columns.
	fixedWide := thTime + thSeverity + thSource + thDest + thAction + thCategory + 6
	fixedNarrow := thTime + thSevShort + thSource + thAction + 4
	fixedCompact := thTimeCompact + thSevShort + thSource + thAction + 4
	return logLayout(m.Width, [3]int{fixedWide, fixedNarrow, fixedCompact}, 150)
}

func (m LogsModel) formatThreatHeader(l logColumnLayout) string {
	if l.compact() {
		return formatCompactRow(l, thTimeCompact, "Time",
			[]int{thSevShort, thSource, thAction}, []string{"Sev", "Source", "Action"}, "Threat")
	}
	if l.wide() {
		return fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s %-*s",
			thTime, "Time", thSeverity, "Severity", l.flex, "Threat",
			thSource, "Source", thDest, "Destination", thAction, "Action", thCategory, "Category")
	}
	return fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s",
		thTime, "Time", thSevShort, "Sev", l.flex, "Threat", thSource, "Source", thAction, "Action")
}

func (m LogsModel) formatThreatRow(log models.ThreatLogEntry, l logColumnLayout) string {
	timeStr := log.Time.Format("2006-01-02 15:04:05")
	if l.compact() {
		return formatCompactRow(l, thTimeCompact, log.Time.Format(logTimeOnlyLayout),
			[]int{thSevShort, thSource, thAction},
			[]string{abbreviateSeverity(log.Severity), log.SourceIP, log.Action}, log.ThreatName)
	}
	if l.wide() {
		return fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s %-*s",
			thTime, timeStr,
			thSeverity, truncate(log.Severity, thSeverity),
			l.flex, truncate(log.ThreatName, l.flex),
			thSource, truncate(log.SourceIP, thSource),
			thDest, truncate(log.DestIP, thDest),
			thAction, truncate(log.Action, thAction),
			thCategory, truncate(log.ThreatCategory, thCategory))
	}
	return fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s",
		thTime, timeStr,
		thSevShort, abbreviateSeverity(log.Severity),
		l.flex, truncate(log.ThreatName, l.flex),
		thSource, truncate(log.SourceIP, thSource),
		thAction, truncate(log.Action, thAction))
}

func (m LogsModel) renderThreatTable() string {
	if m.Loading && len(m.threatLogs) == 0 {
		return LoadingMsgStyle.Padding(1, 0).Render("Loading threat logs...")
	}
	if len(m.filteredThreat) == 0 {
		return EmptyMsgStyle.Padding(1, 0).Render("No threat logs found")
	}

	layout := m.threatLayout()

	var b strings.Builder
	b.WriteString(TableHeaderStyle.Render(m.formatThreatHeader(layout)) + "\n")
	b.WriteString(renderLogRows(m.Offset, m.Cursor, m.visibleRows(), m.filteredThreat, func(log models.ThreatLogEntry, selected bool) string {
		row := m.formatThreatRow(log, layout)
		if selected {
			return TableSelectedRowStyle().Render(row)
		}
		return colorBySeverity(row, log.Severity)
	}))

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
	lines = append(lines, labelStyle.Render("Threat ID")+DetailValueStyle.Render(log.ThreatID))
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
