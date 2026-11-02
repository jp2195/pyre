package views

import (
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
)

func TestCSVSafe(t *testing.T) {
	for in, want := range map[string]string{
		"":                           "",
		"allow-web":                  "allow-web",
		`=HYPERLINK("http://x","y")`: `'=HYPERLINK("http://x","y")`,
		"+1-555":                     "'+1-555",
		"-":                          "'-",
		"@SUM(A1)":                   "'@SUM(A1)",
		"\tcmd":                      "'\tcmd",
	} {
		if got := csvSafe(in); got != want {
			t.Errorf("csvSafe(%q) = %q, want %q", in, got, want)
		}
	}
}

func threatLogs() LogsModel {
	m := NewLogsModel()
	m.activeLogType = models.LogTypeThreat
	m.filteredThreat = []models.ThreatLogEntry{{
		Time:       time.Date(2026, 11, 1, 9, 30, 0, 0, time.UTC),
		ThreatName: "Evil Thing", Severity: "high", SourceIP: "10.0.0.5", SourcePort: 51000,
		URL: `=HYPERLINK("http://attacker.example","click")`, SessionID: 4417, ReportID: -1,
	}}
	return m
}

// TestExportCSV_DefusesFormulas is the reason csvSafe exists: a URL from the
// wire must not become a live formula when the export is opened.
func TestExportCSV_DefusesFormulas(t *testing.T) {
	req, ok := threatLogs().exportRequest(ExportCSV)
	if !ok {
		t.Fatal("no export for a tab with rows")
	}
	data, err := req.Encode()
	if err != nil {
		t.Fatal(err)
	}
	records, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	if err != nil {
		t.Fatalf("export is not valid CSV: %v", err)
	}
	if len(records) != 2 || len(records[0]) != len(threatExportColumns) {
		t.Fatalf("got %d records of %d columns", len(records), len(records[0]))
	}
	row := map[string]string{}
	for i, col := range records[0] {
		row[col] = records[1][i]
	}
	if got := row["url"]; !strings.HasPrefix(got, "'=") {
		t.Errorf("url = %q, want the formula defused with a leading quote", got)
	}
	if row["time"] != "2026-11-01T09:30:00Z" || row["session_id"] != "4417" {
		t.Errorf("time %q session_id %q", row["time"], row["session_id"])
	}
	// Numbers are never quoted, even negative ones.
	if row["report_id"] != "-1" {
		t.Errorf("report_id = %q, want -1 unprefixed", row["report_id"])
	}
}

func TestExportJSON_KeepsTypesAndOrder(t *testing.T) {
	req, _ := threatLogs().exportRequest(ExportJSON)
	data, err := req.Encode()
	if err != nil {
		t.Fatal(err)
	}
	var rows []map[string]any
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatalf("export is not valid JSON: %v\n%s", err, data)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows", len(rows))
	}
	if id, ok := rows[0]["session_id"].(float64); !ok || id != 4417 {
		t.Errorf("session_id = %#v, want the number 4417", rows[0]["session_id"])
	}
	// JSON needs no formula defusing; the value stays as the device sent it.
	if url := rows[0]["url"]; url != `=HYPERLINK("http://attacker.example","click")` {
		t.Errorf("url = %q", url)
	}
	if strings.Index(string(data), `"time"`) > strings.Index(string(data), `"severity"`) {
		t.Error("JSON keys lost their column order")
	}
}

func TestExportJSON_Empty(t *testing.T) {
	data, err := ExportLogsCmd{Format: ExportJSON, Columns: systemExportColumns}.Encode()
	if err != nil || strings.TrimSpace(string(data)) != "[]" {
		t.Errorf("empty export = %q, %v; want []", data, err)
	}
}

func press(m LogsModel, s string) (LogsModel, tea.Cmd) {
	var msg tea.KeyPressMsg
	if s == "esc" {
		msg = tea.KeyPressMsg{Code: tea.KeyEscape}
	} else {
		msg = tea.KeyPressMsg{Code: []rune(s)[0], Text: s}
	}
	return m.Update(msg)
}

func TestExportPrompt(t *testing.T) {
	InitStyles()
	m := threatLogs()
	m = m.SetSize(120, 40)

	m, _ = press(m, "e")
	if !m.IsExportPrompt() {
		t.Fatal("e did not open the export prompt")
	}
	if out := m.View(); !strings.Contains(out, "export 1 threat logs") {
		t.Errorf("prompt not shown:\n%s", out)
	}

	m, cmd := press(m, "j")
	if m.IsExportPrompt() {
		t.Error("prompt still open after choosing a format")
	}
	req, ok := cmd().(ExportLogsCmd)
	if !ok || req.Format != ExportJSON || len(req.Rows) != 1 {
		t.Fatalf("j produced %#v", req)
	}

	m, _ = press(m, "e")
	m, cmd = press(m, "esc")
	if m.IsExportPrompt() || cmd != nil {
		t.Error("esc did not cancel the prompt cleanly")
	}
}

func TestExportPrompt_NothingToExport(t *testing.T) {
	m := NewLogsModel()
	m, _ = press(m, "e")
	if m.IsExportPrompt() {
		t.Error("prompt opened on an empty tab")
	}
}
