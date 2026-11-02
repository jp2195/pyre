package tui

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
	"github.com/jp2195/pyre/internal/tui/views"
)

func TestWriteNewFile_NeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	first, err := writeNewFile(dir, "pyre-x", "csv", []byte("one"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := writeNewFile(dir, "pyre-x", "csv", []byte("two"))
	if err != nil {
		t.Fatal(err)
	}
	if first == second || filepath.Base(second) != "pyre-x-1.csv" {
		t.Errorf("second export went to %s, want pyre-x-1.csv beside %s", second, first)
	}
	if got, _ := os.ReadFile(first); string(got) != "one" {
		t.Errorf("first file overwritten: %q", got)
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(first)
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("mode = %o, want 600: exports hold device data", perm)
		}
	}
}

func TestExportLogs_EndToEnd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	m := newTestModel(t, ViewLogs)
	req := views.ExportLogsCmd{
		Type: models.LogTypeSystem, Format: views.ExportCSV,
		Columns: []string{"time", "type", "severity", "description"},
		Rows:    [][]views.ExportCell{{{Text: ""}, {Text: "SYSTEM"}, {Text: "high"}, {Text: "HA link down"}}},
	}
	next, cmd := m.Update(req)
	if cmd == nil {
		t.Fatal("export issued no command")
	}
	done, ok := cmd().(LogsExportedMsg)
	if !ok || done.Err != nil {
		t.Fatalf("export result %#v", done)
	}
	if filepath.Dir(done.Path) != home || !strings.HasSuffix(done.Path, ".csv") {
		t.Errorf("wrote %s, want a .csv in the home directory", done.Path)
	}
	data, _ := os.ReadFile(done.Path)
	if !strings.Contains(string(data), "HA link down") {
		t.Errorf("file = %q", data)
	}

	after, _ := next.Update(done)
	notice := after.(Model).activeNotice(time.Now())
	if len(notice) == 0 || !strings.HasPrefix(notice[0], "exported 1 system logs to ~"+string(filepath.Separator)) {
		t.Errorf("notice = %q", notice)
	}
}

func TestExportLogs_FailureIsAnError(t *testing.T) {
	m := newTestModel(t, ViewLogs)
	next, _ := m.Update(LogsExportedMsg{Err: os.ErrPermission})
	if err := next.(Model).err; err == nil || !strings.Contains(err.Error(), "export failed") {
		t.Errorf("err = %v, want an export failure", err)
	}
}

// TestExportPrompt_OwnsTheKeys keeps the prompt from leaking keys: q must not
// quit and y must not copy while it waits for c or j.
func TestExportPrompt_OwnsTheKeys(t *testing.T) {
	m := newTestModel(t, ViewLogs)
	m.logs = logsWithRow(t)
	next, _ := m.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})
	m = next.(Model)
	if !m.logs.IsExportPrompt() {
		t.Fatal("e did not open the prompt")
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd != nil {
		if _, quit := cmd().(tea.QuitMsg); quit {
			t.Error("q quit pyre while the export prompt was open")
		}
	}
}

// logsWithRow returns a logs view holding one system row, loaded through the
// real request-and-page path so the view accepts it.
func logsWithRow(t *testing.T) views.LogsModel {
	t.Helper()
	l := views.NewLogsModel().SetSize(120, 40)
	l, cmd := l.RefreshActiveTab()
	req, ok := cmd().(views.FetchLogsCmd)
	if !ok {
		t.Fatal("refresh did not mint a request")
	}
	l, _ = l.SetSystemLogs([]models.SystemLogEntry{{Severity: "high", Description: "HA link down"}}, views.LogPageMeta{Req: req}, nil)
	return l
}
