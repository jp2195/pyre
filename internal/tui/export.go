package tui

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/tui/views"
)

// LogsExportedMsg reports a finished log export.
type LogsExportedMsg struct {
	Path  string
	Count int
	Kind  string
	Err   error
}

// exportLogs writes req to a new file in the home directory. The directory is
// resolved here on the event loop; encoding and writing happen in the
// command, off it, because an export can be tens of thousands of rows.
func (m Model) exportLogs(req views.ExportLogsCmd) (Model, tea.Cmd) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return m.setError(fmt.Errorf("export failed: no home directory: %w", err))
	}
	name := fmt.Sprintf("pyre-%s-logs-%s", req.Type, time.Now().Format("20060102-150405"))
	kind := string(req.Type)
	return m, func() tea.Msg {
		data, err := req.Encode()
		if err != nil {
			return LogsExportedMsg{Err: err}
		}
		path, err := writeNewFile(dir, name, req.Format.Ext(), data)
		return LogsExportedMsg{Path: path, Count: len(req.Rows), Kind: kind, Err: err}
	}
}

// writeNewFile writes data to dir/name.ext, or name-1.ext and so on if that
// exists. It never overwrites: O_EXCL makes the existence check and the
// create one step. The file is 0600 because a log export is device data
// (addresses, user names, URLs), the same reason the config file is.
func writeNewFile(dir, name, ext string, data []byte) (string, error) {
	for i := range 100 {
		base := name
		if i > 0 {
			base += "-" + strconv.Itoa(i)
		}
		path := filepath.Join(dir, base+"."+ext)
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) // #nosec G304 -- name is built by pyre, not taken from input
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		// A partial export is removed rather than left looking complete;
		// a failure to clean up is reported alongside the write failure.
		if _, err := f.Write(data); err != nil {
			return "", errors.Join(err, f.Close(), os.Remove(path))
		}
		if err := f.Close(); err != nil {
			return "", errors.Join(err, os.Remove(path))
		}
		return path, nil
	}
	return "", fmt.Errorf("%s: 100 files with that name already exist", filepath.Join(dir, name))
}

// handleLogsExported reports the export in the footer, or as an error.
func (m Model) handleLogsExported(msg LogsExportedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		return m.setError(fmt.Errorf("export failed: %w", msg.Err))
	}
	shown := msg.Path
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(shown, home+string(filepath.Separator)) {
		shown = "~" + shown[len(home):]
	}
	what := fmt.Sprintf("exported %d %s logs", msg.Count, msg.Kind)
	return m.setNotice(time.Now(), what+" to "+shown, "exported to "+shown, what), nil
}
