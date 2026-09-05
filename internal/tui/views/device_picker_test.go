package views

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
)

func devicePickerFixture() []models.ManagedDevice {
	return []models.ManagedDevice{
		{Serial: "001901000001", Hostname: "fw-edge-a", IPAddress: "10.0.104.50",
			Model: "PA-3260", HAState: "active", Connected: true, DeviceGroup: "Edge"},
		{Serial: "001901000002", Hostname: "fw-edge-b", IPAddress: "10.0.104.51",
			Model: "PA-3260", HAState: "passive", Connected: true, DeviceGroup: "Edge"},
		{Serial: "001901000003", Hostname: "", IPAddress: "10.0.104.52",
			Model: "PA-440", Connected: false},
	}
}

func key(s string) tea.KeyPressMsg {
	switch s {
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "home":
		return tea.KeyPressMsg{Code: tea.KeyHome}
	case "end":
		return tea.KeyPressMsg{Code: tea.KeyEnd}
	default:
		return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
	}
}

func pressAll(m DevicePickerModel, keys ...string) DevicePickerModel {
	for _, k := range keys {
		m, _ = m.Update(key(k))
	}
	return m
}

// TestDevicePicker_CursorMapsToDevices covers the offset that the whole
// screen turns on: index zero is Panorama itself, so device i sits at cursor
// i+1. Getting that wrong targets the wrong firewall.
func TestDevicePicker_CursorMapsToDevices(t *testing.T) {
	devices := devicePickerFixture()
	m := NewDevicePickerModel().SetDevices(devices, "", "panorama-1")

	if got := m.SelectedDevice(); got != nil {
		t.Errorf("cursor at rest selects %q, want Panorama (nil)", got.Serial)
	}

	for i, want := range devices {
		moved := m
		for range i + 1 {
			moved, _ = moved.Update(key("j"))
		}
		got := moved.SelectedDevice()
		if got == nil {
			t.Fatalf("after %d moves down, no device selected, want %s", i+1, want.Serial)
		}
		if got.Serial != want.Serial {
			t.Errorf("after %d moves down, selected %s, want %s", i+1, got.Serial, want.Serial)
		}
	}
}

// TestDevicePicker_NavigationStopsAtTheEnds checks the cursor cannot leave
// the list. Running off the end would index past the device slice.
func TestDevicePicker_NavigationStopsAtTheEnds(t *testing.T) {
	devices := devicePickerFixture()
	m := NewDevicePickerModel().SetDevices(devices, "", "panorama-1")

	past := pressAll(m, "j", "j", "j", "j", "j", "j")
	if got := past.SelectedDevice(); got == nil || got.Serial != devices[len(devices)-1].Serial {
		t.Errorf("running off the bottom lands on %v, want the last device", got)
	}

	back := pressAll(past, "k", "k", "k", "k", "k", "k")
	if got := back.SelectedDevice(); got != nil {
		t.Errorf("running off the top selects %q, want Panorama (nil)", got.Serial)
	}
}

// TestDevicePicker_JumpKeys covers g/G and home/end.
func TestDevicePicker_JumpKeys(t *testing.T) {
	devices := devicePickerFixture()
	m := NewDevicePickerModel().SetDevices(devices, "", "panorama-1")

	for _, k := range []string{"G", "end"} {
		got := pressAll(m, k).SelectedDevice()
		if got == nil || got.Serial != devices[len(devices)-1].Serial {
			t.Errorf("%q did not jump to the last device, got %v", k, got)
		}
	}
	for _, k := range []string{"g", "home"} {
		bottom := pressAll(m, "G")
		if got := pressAll(bottom, k).SelectedDevice(); got != nil {
			t.Errorf("%q did not jump back to Panorama, got %q", k, got.Serial)
		}
	}
}

// TestDevicePicker_JumpToEndWithNoDevices guards the empty case: with no
// managed devices the only row is Panorama, and end must stay on it.
func TestDevicePicker_JumpToEndWithNoDevices(t *testing.T) {
	m := NewDevicePickerModel().SetDevices(nil, "", "panorama-1")
	if got := pressAll(m, "G", "j").SelectedDevice(); got != nil {
		t.Errorf("empty list selects %q, want Panorama (nil)", got.Serial)
	}
}

// TestDevicePicker_OpensOnTheCurrentTarget checks the screen opens with the
// cursor on the device already being targeted, rather than making the
// operator find it again.
func TestDevicePicker_OpensOnTheCurrentTarget(t *testing.T) {
	devices := devicePickerFixture()
	m := NewDevicePickerModel().SetDevices(devices, "001901000002", "panorama-1")

	got := m.SelectedDevice()
	if got == nil || got.Serial != "001901000002" {
		t.Errorf("opened on %v, want the targeted device 001901000002", got)
	}
}

// TestDevicePicker_UnknownTargetOpensOnPanorama covers a target serial that
// is no longer in the device list, which happens when a device is removed
// from Panorama while pyre holds the old target.
func TestDevicePicker_UnknownTargetOpensOnPanorama(t *testing.T) {
	m := NewDevicePickerModel().SetDevices(devicePickerFixture(), "not-a-serial", "panorama-1")
	if got := m.SelectedDevice(); got != nil {
		t.Errorf("unknown target selects %q, want Panorama (nil)", got.Serial)
	}
}

// TestDevicePicker_ViewListsEveryDevice checks the screen actually shows what
// it is asked to show, including the serial standing in for a missing
// hostname and the connection state.
func TestDevicePicker_ViewListsEveryDevice(t *testing.T) {
	devices := devicePickerFixture()
	view := NewDevicePickerModel().SetDevices(devices, "001901000001", "panorama-1").
		SetSize(120, 40).View()

	for _, want := range []string{
		"panorama-1", "fw-edge-a", "fw-edge-b",
		"001901000003", // no hostname, so the serial stands in
		"10.0.104.50", "PA-3260", "active", "passive",
		"connected", "disconnected",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("device picker does not show %q", want)
		}
	}
}

// TestDevicePicker_ViewMarksTheActiveTarget checks the marker sits on the
// device currently being targeted, not on the cursor.
func TestDevicePicker_ViewMarksTheActiveTarget(t *testing.T) {
	devices := devicePickerFixture()
	m := NewDevicePickerModel().SetDevices(devices, "001901000002", "panorama-1").SetSize(120, 40)
	m = pressAll(m, "g") // move the cursor away from the target

	for _, line := range strings.Split(m.View(), "\n") {
		if strings.Contains(line, "fw-edge-b") && !strings.Contains(line, "►") {
			t.Errorf("active target is not marked:\n  %q", line)
		}
		if strings.Contains(line, "fw-edge-a") && strings.Contains(line, "►") {
			t.Errorf("marker is on a device that is not the target:\n  %q", line)
		}
	}
}

// TestDevicePicker_ViewWithNoDevices checks the empty state says so rather
// than rendering a bare Panorama row with no explanation.
func TestDevicePicker_ViewWithNoDevices(t *testing.T) {
	view := NewDevicePickerModel().SetDevices(nil, "", "panorama-1").SetSize(120, 40).View()
	if !strings.Contains(view, "No managed devices found") {
		t.Errorf("empty device picker does not explain itself:\n%s", view)
	}
}

// TestDevicePicker_ViewBeforeSize covers the pre-layout render.
func TestDevicePicker_ViewBeforeSize(t *testing.T) {
	if got := NewDevicePickerModel().View(); got != "Loading..." {
		t.Errorf("view before a size is %q, want a loading placeholder", got)
	}
}
