package views

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
)

// ExportFormat is the file format a log export is written in.
type ExportFormat int

const (
	ExportCSV ExportFormat = iota
	ExportJSON
)

// Ext is the file extension for the format, without the dot.
func (f ExportFormat) Ext() string {
	if f == ExportJSON {
		return "json"
	}
	return "csv"
}

// ExportLogsCmd asks the parent to write the rows on screen to a file. The
// view builds it because only the view knows which rows are showing (the
// active tab, filtered and sorted); the parent does the file I/O off the
// event loop, the same split as FetchLogsCmd.
type ExportLogsCmd struct {
	Type    models.LogType
	Format  ExportFormat
	Columns []string
	Rows    [][]ExportCell
}

// ExportCell is one value in an export: text, or a number that JSON should
// keep numeric and CSV should never treat as a formula.
type ExportCell struct {
	Text  string
	Num   int64
	IsNum bool
}

func txt(s string) ExportCell { return ExportCell{Text: s} }
func num(n int64) ExportCell  { return ExportCell{Num: n, IsNum: true} }

// exportTime renders a timestamp with its offset, so the file still says
// when something happened once it leaves the device's time zone.
func exportTime(t time.Time) ExportCell {
	if t.IsZero() {
		return txt("")
	}
	return txt(t.Format(time.RFC3339))
}

// Encode renders the export in its format.
func (e ExportLogsCmd) Encode() ([]byte, error) {
	if e.Format == ExportJSON {
		return e.encodeJSON()
	}
	return e.encodeCSV()
}

func (e ExportLogsCmd) encodeCSV() ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(e.Columns); err != nil {
		return nil, err
	}
	record := make([]string, len(e.Columns))
	for _, row := range e.Rows {
		for i, c := range row {
			if c.IsNum {
				record[i] = strconv.FormatInt(c.Num, 10)
			} else {
				record[i] = csvSafe(c.Text)
			}
		}
		if err := w.Write(record); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

// csvSafe defuses spreadsheet formula injection. Log fields carry text an
// attacker controls (URLs, user names, threat names, file names), and a cell
// that starts with =, +, -, @, tab or carriage return runs as a formula when
// the file is opened in Excel or Sheets. Prefixing a single quote, the OWASP
// recommendation, makes the spreadsheet show it as text. Numbers never get
// here; they are written by strconv and cannot carry a formula.
func csvSafe(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	}
	return s
}

// encodeJSON writes an array of objects whose keys keep the column order,
// which a map would sort away.
func (e ExportLogsCmd) encodeJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("[")
	for r, row := range e.Rows {
		if r > 0 {
			buf.WriteString(",")
		}
		buf.WriteString("\n  {")
		for i, c := range row {
			if i > 0 {
				buf.WriteString(", ")
			}
			k, err := json.Marshal(e.Columns[i])
			if err != nil {
				return nil, err
			}
			var v []byte
			if c.IsNum {
				v = strconv.AppendInt(nil, c.Num, 10)
			} else if v, err = json.Marshal(c.Text); err != nil {
				return nil, err
			}
			buf.Write(k)
			buf.WriteString(": ")
			buf.Write(v)
		}
		buf.WriteString("}")
	}
	if len(e.Rows) > 0 {
		buf.WriteString("\n")
	}
	buf.WriteString("]\n")
	return buf.Bytes(), nil
}

var (
	trafficExportColumns = []string{
		"time", "receive_time", "subtype", "source_ip", "source_port", "dest_ip", "dest_port",
		"nat_source_ip", "nat_source_port", "nat_dest_ip", "nat_dest_port", "source_zone", "dest_zone",
		"rule", "application", "user", "session_id", "action", "session_end", "bytes", "bytes_sent",
		"bytes_recv", "packets", "duration_s", "protocol", "category", "vsys", "device_name", "serial",
	}
	threatExportColumns = []string{
		"time", "receive_time", "subtype", "severity", "threat_name", "threat_id", "threat_category",
		"direction", "source_ip", "source_port", "dest_ip", "dest_port", "source_zone", "dest_zone",
		"rule", "application", "user", "session_id", "action", "url", "filename", "file_hash",
		"content_type", "vsys", "device_name", "serial", "report_id",
	}
	systemExportColumns = []string{"time", "type", "severity", "description"}
)

// exportRequest builds the export for the active tab from the rows on
// screen, or false when there are none.
func (m LogsModel) exportRequest(f ExportFormat) (ExportLogsCmd, bool) {
	req := ExportLogsCmd{Type: m.activeLogType, Format: f}
	switch m.activeLogType {
	case models.LogTypeTraffic:
		req.Columns = trafficExportColumns
		for _, e := range m.filteredTraffic {
			req.Rows = append(req.Rows, []ExportCell{
				exportTime(e.Time), exportTime(e.ReceiveTime), txt(e.Subtype),
				txt(e.SourceIP), num(int64(e.SourcePort)), txt(e.DestIP), num(int64(e.DestPort)),
				txt(e.NATSourceIP), num(int64(e.NATSourcePort)), txt(e.NATDestIP), num(int64(e.NATDestPort)),
				txt(e.SourceZone), txt(e.DestZone), txt(e.Rule), txt(e.Application), txt(e.User),
				num(e.SessionID), txt(e.Action), txt(e.SessionEnd), num(e.Bytes), num(e.BytesSent),
				num(e.BytesRecv), num(e.Packets), num(e.Duration), txt(e.Protocol), txt(e.Category),
				txt(e.VirtualSystem), txt(e.DeviceName), txt(e.Serial),
			})
		}
	case models.LogTypeThreat:
		req.Columns = threatExportColumns
		for _, e := range m.filteredThreat {
			req.Rows = append(req.Rows, []ExportCell{
				exportTime(e.Time), exportTime(e.ReceiveTime), txt(e.Subtype), txt(e.Severity),
				txt(e.ThreatName), txt(e.ThreatID), txt(e.ThreatCategory), txt(e.Direction),
				txt(e.SourceIP), num(int64(e.SourcePort)), txt(e.DestIP), num(int64(e.DestPort)),
				txt(e.SourceZone), txt(e.DestZone), txt(e.Rule), txt(e.Application), txt(e.User),
				num(e.SessionID), txt(e.Action), txt(e.URL), txt(e.Filename), txt(e.FileHash),
				txt(e.ContentType), txt(e.VirtualSystem), txt(e.DeviceName), txt(e.Serial), num(e.ReportID),
			})
		}
	default:
		req.Columns = systemExportColumns
		for _, e := range m.filteredSystem {
			req.Rows = append(req.Rows, []ExportCell{
				exportTime(e.Time), txt(e.Type), txt(e.Severity), txt(e.Description),
			})
		}
	}
	return req, len(req.Rows) > 0
}

// exportPromptLine replaces the status line while the format prompt is up,
// so opening it does not shift the table.
func (m LogsModel) exportPromptLine() string {
	n := m.filteredCount()
	return ViewSubtitleStyle.Render(truncateEllipsis(
		"export "+strconv.Itoa(n)+" "+string(m.activeLogType)+" logs:  c CSV  ·  j JSON  ·  esc cancel", m.Width))
}

// IsExportPrompt reports whether the export format prompt is up. The parent
// routes every key to the view while it is, as it does for the filter.
func (m LogsModel) IsExportPrompt() bool { return m.exportPrompt }

// updateExportPrompt answers the format prompt: c or j sends the export to
// the parent, and anything else closes the prompt without exporting.
func (m LogsModel) updateExportPrompt(msg tea.Msg) (LogsModel, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	m.exportPrompt = false
	var f ExportFormat
	switch key.String() {
	case "c":
		f = ExportCSV
	case "j":
		f = ExportJSON
	default:
		return m, nil
	}
	req, ok := m.exportRequest(f)
	if !ok {
		return m, nil
	}
	return m, func() tea.Msg { return req }
}
