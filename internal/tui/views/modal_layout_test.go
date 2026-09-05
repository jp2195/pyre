package views

import (
	"regexp"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/auth"
	"github.com/jp2195/pyre/internal/config"
)

// modalFixtures builds a session and config carrying one connection, which
// is what the picker and the connection hub each render from.
func modalFixtures() (*auth.Session, *config.Config, *config.State) {
	cfg := config.DefaultConfig()
	session := auth.NewSession(cfg)
	_, _ = session.AddConnection("10.0.104.50", &config.ConnectionConfig{Username: "testuser"}, "key")
	state := &config.State{Connections: map[string]config.ConnectionState{}}
	return session, cfg, state
}

// ansiPattern matches the escape sequences lipgloss emits, so a rendered
// line can be measured as the text a reader sees.
var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// modalBody returns the lines of a centered modal with the surrounding
// whitespace and the panel frame stripped, so the assertions below read
// against the content the screen actually shows.
func modalBody(t *testing.T, view string) []string {
	t.Helper()
	var body []string
	inBox := false
	for _, line := range strings.Split(view, "\n") {
		switch {
		case strings.Contains(line, "╭"):
			inBox = true
		case strings.Contains(line, "╰"):
			inBox = false
		case inBox:
			// A blank row inside the box still carries the escape sequences
			// that colored its border, and they sit after the trailing
			// spaces, so the styling has to come off before trimming.
			plain := ansiPattern.ReplaceAllString(line, "")
			body = append(body, strings.TrimSpace(strings.ReplaceAll(plain, "│", "")))
		}
	}
	return body
}

// TestModals_TitleIsFollowedByOneBlankLine checks the spacing under the title
// of each centered modal.
//
// The title style carries a bottom margin, which already renders a blank
// line, and each screen then wrote two more newlines. The result was two
// blank lines under every title, which on a short terminal is two rows of
// the list that do not fit.
func TestModals_TitleIsFollowedByOneBlankLine(t *testing.T) {
	session, cfg, state := modalFixtures()

	modals := []struct {
		name  string
		title string
		view  string
	}{
		{
			"device picker", "Select Target Device",
			NewDevicePickerModel().SetDevices(devicePickerFixture(), "", "panorama-1").
				SetSize(120, 40).View(),
		},
		{
			"picker", "Firewall Connections",
			NewPickerModel(session).SetSize(120, 40).View(),
		},
		{
			"connection hub", "PYRE - Connections",
			NewConnectionHubModel().SetConnections(cfg, state).SetSize(120, 40).View(),
		},
	}

	for _, modal := range modals {
		body := modalBody(t, modal.view)
		idx := -1
		for i, line := range body {
			if strings.Contains(line, modal.title) {
				idx = i
				break
			}
		}
		if idx < 0 {
			t.Errorf("%s: title %q not found in:\n%s", modal.name, modal.title, strings.Join(body, "\n"))
			continue
		}
		blanks := 0
		for _, line := range body[idx+1:] {
			if line != "" {
				break
			}
			blanks++
		}
		if blanks != 1 {
			t.Errorf("%s: %d blank lines under the title, want 1", modal.name, blanks)
		}
	}
}

// TestDevicePicker_RowsDoNotWrap checks that each managed device occupies one
// line at the narrowest supported terminal.
//
// The row is a fixed string: hostname, model, HA state, address and
// connection status. At sixty columns the modal is fifty wide, so the row
// wrapped and the second half landed on its own line without the marker
// column, which makes the list unreadable at exactly the width where an
// operator is most likely reading it in a split pane.
func TestDevicePicker_RowsDoNotWrap(t *testing.T) {
	devices := devicePickerFixture()
	for _, width := range []int{60, 80, 100, 120} {
		view := NewDevicePickerModel().SetDevices(devices, "", "panorama-1").
			SetSize(width, 40).View()
		body := modalBody(t, view)

		for _, device := range devices {
			name := device.Hostname
			if name == "" {
				name = device.Serial
			}
			// The address is the last field the row is allowed to drop, so a
			// row that wrapped carries the name without it.
			found := false
			for _, line := range body {
				if !strings.Contains(line, name) {
					continue
				}
				found = true
				if !strings.Contains(line, device.IPAddress) {
					t.Errorf("width %d: row for %s wrapped away from its address:\n%s",
						width, name, strings.Join(body, "\n"))
				}
				break
			}
			if !found {
				t.Errorf("width %d: device %s is not listed", width, name)
			}
		}
	}
}

// TestModals_BoxFitsTheTerminal checks the modal never renders wider than the
// terminal it is placed in. Two of the three shrank the box with the terminal
// but set no lower bound, and the third clamped to a floor wider than a small
// terminal.
func TestModals_BoxFitsTheTerminal(t *testing.T) {
	session, cfg, state := modalFixtures()

	for width := 60; width >= 20; width -= 5 {
		views := map[string]string{
			"device picker": NewDevicePickerModel().SetDevices(devicePickerFixture(), "", "panorama-1").
				SetSize(width, 30).View(),
			"picker":         NewPickerModel(session).SetSize(width, 30).View(),
			"connection hub": NewConnectionHubModel().SetConnections(cfg, state).SetSize(width, 30).View(),
		}
		for name, view := range views {
			for _, line := range strings.Split(view, "\n") {
				if got := lipgloss.Width(line); got > width {
					t.Errorf("%s at width %d: line is %d cells, %d over", name, width, got, got-width)
					break
				}
			}
		}
	}
}

// TestModals_HelpLineStaysOnOneLine checks the key hints at the foot of each
// centered modal.
//
// The hints were a fixed string, and a modal box is only as wide as the
// terminal allows, so on a narrow terminal the line wrapped and the last few
// hints sat on a line of their own. The help line is the last thing in every
// one of these screens, so a wrap is easy to see and easy to test: the final
// non-blank line stops carrying the first hint.
func TestModals_HelpLineStaysOnOneLine(t *testing.T) {
	session, cfg, state := modalFixtures()

	modals := []struct {
		name      string
		firstHint string
		view      func(int) string
	}{
		{"device picker", "j/k: navigate", func(w int) string {
			return NewDevicePickerModel().SetDevices(devicePickerFixture(), "", "panorama-1").
				SetSize(w, 30).View()
		}},
		{"picker", "j/k: navigate", func(w int) string {
			return NewPickerModel(session).SetSize(w, 30).View()
		}},
		{"connection hub", "[n]", func(w int) string {
			return NewConnectionHubModel().SetConnections(cfg, state).SetSize(w, 30).View()
		}},
	}

	for _, modal := range modals {
		for width := 60; width <= 140; width += 5 {
			body := modalBody(t, modal.view(width))
			last := ""
			for _, line := range body {
				if line != "" {
					last = line
				}
			}
			if !strings.Contains(last, modal.firstHint) {
				t.Errorf("%s at width %d: help line wrapped, last line is:\n  %q",
					modal.name, width, last)
			}
		}
	}
}

// TestDevicePicker_RowCountIsStable checks the screen renders one line per
// row at every width: a title, the Panorama entry, one line per device, and
// the help line. Anything that wraps shows up as an extra line.
func TestDevicePicker_RowCountIsStable(t *testing.T) {
	devices := devicePickerFixture()
	want := 1 + 1 + len(devices) + 1

	for width := 60; width <= 140; width += 5 {
		body := modalBody(t, NewDevicePickerModel().
			SetDevices(devices, "", "panorama-1").SetSize(width, 40).View())
		got := 0
		for _, line := range body {
			if line != "" {
				got++
			}
		}
		if got != want {
			t.Errorf("width %d: %d content lines, want %d:\n%s",
				width, got, want, strings.Join(body, "\n"))
		}
	}
}
