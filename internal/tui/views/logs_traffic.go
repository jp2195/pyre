package views

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
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
	slices.SortFunc(logs, func(a, b models.TrafficLogEntry) int {
		var c int
		switch sortBy {
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

// Traffic table columns. The rule name is the flexible one: it is the field
// most often long enough to be cut, and the one an operator is usually
// reading the table to find.
const (
	tfTime   = 19
	tfAction = 12 // "reset-client"
	tfSource = 15
	tfDest   = 15
	tfApp    = 16
	tfBytes  = 10

	// Narrow variants.
	tfActionNarrow = 10
	tfAppNarrow    = 12

	// Compact variant: the date is dropped from the timestamp and the
	// application column goes with it. On a terminal this narrow the flow
	// itself is all that fits, so keep who talked to whom, what the firewall
	// did about it, and the rule that decided.
	tfTimeCompact = 8 // "15:04:05"
)

func (m LogsModel) trafficLayout() logColumnLayout {
	fixedWide := tfTime + tfAction + tfSource + tfDest + tfApp + tfBytes + 6
	// The byte count is the first thing to go on a narrow terminal. What
	// identifies a flow is who talked to whom, over what application, under
	// which rule; the volume is context rather than identity.
	fixedNarrow := tfTime + tfActionNarrow + tfSource + tfDest + tfAppNarrow + 5
	fixedCompact := tfTimeCompact + tfActionNarrow + tfSource + tfDest + 4
	return logLayout(m.Width, [3]int{fixedWide, fixedNarrow, fixedCompact}, 150)
}

func (m LogsModel) formatTrafficHeader(l logColumnLayout) string {
	if l.compact() {
		return formatCompactRow(l, tfTimeCompact, "Time",
			[]int{tfActionNarrow, tfSource, tfDest}, []string{"Action", "Source", "Dest"}, "Rule")
	}
	if l.wide() {
		return fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s %-*s",
			tfTime, "Time", tfAction, "Action", tfSource, "Source", tfDest, "Dest",
			tfApp, "App", l.flex, "Rule", tfBytes, "Bytes")
	}
	return fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s",
		tfTime, "Time", tfActionNarrow, "Action", tfSource, "Source", tfDest, "Dest",
		tfAppNarrow, "App", l.flex, "Rule")
}

func (m LogsModel) formatTrafficRow(log models.TrafficLogEntry, l logColumnLayout) string {
	timeStr := log.Time.Format("2006-01-02 15:04:05")
	if l.compact() {
		return formatCompactRow(l, tfTimeCompact, log.Time.Format(logTimeOnlyLayout),
			[]int{tfActionNarrow, tfSource, tfDest},
			[]string{log.Action, log.SourceIP, log.DestIP}, log.Rule)
	}
	if l.wide() {
		return fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s %-*s",
			tfTime, timeStr,
			tfAction, truncate(log.Action, tfAction),
			tfSource, truncate(log.SourceIP, tfSource),
			tfDest, truncate(log.DestIP, tfDest),
			tfApp, truncate(log.Application, tfApp),
			l.flex, truncate(log.Rule, l.flex),
			tfBytes, formatBytes(log.Bytes))
	}
	return fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s",
		tfTime, timeStr,
		tfActionNarrow, truncate(log.Action, tfActionNarrow),
		tfSource, truncate(log.SourceIP, tfSource),
		tfDest, truncate(log.DestIP, tfDest),
		tfAppNarrow, truncate(log.Application, tfAppNarrow),
		l.flex, truncate(log.Rule, l.flex))
}

func (m LogsModel) renderTrafficTable() string {
	if m.Loading && len(m.trafficLogs) == 0 {
		return LoadingMsgStyle.Padding(1, 0).Render("Loading traffic logs...")
	}
	if len(m.filteredTraffic) == 0 {
		return EmptyMsgStyle.Padding(1, 0).Render("No traffic logs found")
	}

	layout := m.trafficLayout()

	var b strings.Builder
	b.WriteString(TableHeaderStyle.Render(m.formatTrafficHeader(layout)) + "\n")
	b.WriteString(renderLogRows(m.Offset, m.Cursor, m.visibleRows(), m.filteredTraffic, func(log models.TrafficLogEntry, selected bool) string {
		row := m.formatTrafficRow(log, layout)
		if selected {
			return TableSelectedRowStyle().Render(row)
		}
		return colorByAction(row, log.Action)
	}))

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
	lines = append(lines, labelStyle.Render("Session ID")+DetailValueStyle.Render(strconv.FormatInt(log.SessionID, 10)))
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
