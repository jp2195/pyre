package views

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/models"
)

// filterTrafficLogs returns traffic logs matching the query.
func filterTrafficLogs(logs []models.TrafficLogEntry, query string) []models.TrafficLogEntry {
	if query == "" {
		result := make([]models.TrafficLogEntry, len(logs))
		copy(result, logs)
		return result
	}

	var result []models.TrafficLogEntry
	for _, log := range logs {
		if strings.Contains(strings.ToLower(log.SourceIP), query) ||
			strings.Contains(strings.ToLower(log.DestIP), query) ||
			strings.Contains(strings.ToLower(log.Application), query) ||
			strings.Contains(strings.ToLower(log.Rule), query) ||
			strings.Contains(strings.ToLower(log.Action), query) ||
			strings.Contains(strings.ToLower(log.User), query) {
			result = append(result, log)
		}
	}
	return result
}

// sortTrafficLogs sorts the slice in place by the given field.
func sortTrafficLogs(logs []models.TrafficLogEntry, sortBy LogSortField, asc bool) {
	sort.Slice(logs, func(i, j int) bool {
		var less bool
		switch sortBy {
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

func (m LogsModel) renderTrafficTable() string {
	if m.Loading && len(m.trafficLogs) == 0 {
		return LoadingMsgStyle.Padding(1, 0).Render("Loading traffic logs...")
	}
	if len(m.filteredTraffic) == 0 {
		return EmptyMsgStyle.Padding(1, 0).Render("No traffic logs found")
	}

	var b strings.Builder

	// Header
	header := fmt.Sprintf("%-19s %-7s %-15s %-15s %-12s %-15s %-10s",
		"Time", "Action", "Source", "Dest", "App", "Rule", "Bytes")
	b.WriteString(TableHeaderStyle.Render(header) + "\n")

	visibleRows := m.visibleRows()
	end := min(m.Offset+visibleRows, len(m.filteredTraffic))

	for i := m.Offset; i < end; i++ {
		log := m.filteredTraffic[i]
		isSelected := i == m.Cursor

		timeStr := log.Time.Format("2006-01-02 15:04:05")

		row := fmt.Sprintf("%-19s %-7s %-15s %-15s %-12s %-15s %-10s",
			timeStr,
			truncate(log.Action, 7),
			truncate(log.SourceIP, 15),
			truncate(log.DestIP, 15),
			truncate(log.Application, 12),
			truncate(log.Rule, 15),
			formatBytes(log.Bytes))

		if isSelected {
			b.WriteString(TableRowSelectedStyle.Render(row) + "\n")
		} else {
			// Color code by action
			row = colorByAction(row, log.Action)
			b.WriteString(row + "\n")
		}
	}

	return b.String()
}

func (m LogsModel) renderTrafficDetail(log models.TrafficLogEntry) string {
	panelStyle := DetailPanelStyle.Width(m.Width - 2)
	labelStyle := DetailLabelStyle.Width(14)

	var lines []string
	lines = append(lines, ViewTitleStyle.Render("Traffic Log Details"))

	lines = append(lines, DetailSectionStyle.Render("Session"))
	lines = append(lines, labelStyle.Render("Time")+DetailValueStyle.Render(log.Time.Format("2006-01-02 15:04:05")))
	lines = append(lines, labelStyle.Render("Action")+colorByAction(log.Action, log.Action))
	lines = append(lines, labelStyle.Render("Session ID")+DetailValueStyle.Render(fmt.Sprintf("%d", log.SessionID)))
	lines = append(lines, labelStyle.Render("Duration")+DetailValueStyle.Render(fmt.Sprintf("%ds", log.Duration)))

	lines = append(lines, DetailSectionStyle.Render("Source / Destination"))
	lines = append(lines, labelStyle.Render("Source")+DetailValueStyle.Render(fmt.Sprintf("%s:%d (%s)", log.SourceIP, log.SourcePort, log.SourceZone)))
	lines = append(lines, labelStyle.Render("Destination")+DetailValueStyle.Render(fmt.Sprintf("%s:%d (%s)", log.DestIP, log.DestPort, log.DestZone)))
	if log.NATSourceIP != "" {
		lines = append(lines, labelStyle.Render("NAT Source")+DetailValueStyle.Render(fmt.Sprintf("%s:%d", log.NATSourceIP, log.NATSourcePort)))
	}
	if log.NATDestIP != "" {
		lines = append(lines, labelStyle.Render("NAT Dest")+DetailValueStyle.Render(fmt.Sprintf("%s:%d", log.NATDestIP, log.NATDestPort)))
	}

	lines = append(lines, DetailSectionStyle.Render("Application"))
	lines = append(lines, labelStyle.Render("Application")+DetailValueStyle.Render(log.Application))
	lines = append(lines, labelStyle.Render("Protocol")+DetailValueStyle.Render(log.Protocol))
	lines = append(lines, labelStyle.Render("Rule")+DetailValueStyle.Render(log.Rule))
	if log.User != "" {
		lines = append(lines, labelStyle.Render("User")+DetailValueStyle.Render(log.User))
	}

	lines = append(lines, DetailSectionStyle.Render("Traffic"))
	lines = append(lines, labelStyle.Render("Bytes")+DetailValueStyle.Render(fmt.Sprintf("%s (sent: %s, recv: %s)", formatBytes(log.Bytes), formatBytes(log.BytesSent), formatBytes(log.BytesRecv))))
	lines = append(lines, labelStyle.Render("Packets")+DetailValueStyle.Render(fmt.Sprintf("%d (sent: %d, recv: %d)", log.Packets, log.PacketsSent, log.PacketsRecv)))

	return panelStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}
