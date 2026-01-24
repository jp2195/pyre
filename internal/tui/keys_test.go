package tui

import (
	"testing"
)

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()

	// Test some key bindings exist
	if len(km.Quit.Keys()) == 0 {
		t.Error("expected Quit keys to be set")
	}
	if len(km.Help.Keys()) == 0 {
		t.Error("expected Help keys to be set")
	}
	if len(km.Refresh.Keys()) == 0 {
		t.Error("expected Refresh keys to be set")
	}

	// Test navigation group keys
	if len(km.NavGroup1.Keys()) == 0 {
		t.Error("expected NavGroup1 keys to be set")
	}
	if len(km.NavGroup2.Keys()) == 0 {
		t.Error("expected NavGroup2 keys to be set")
	}
	if len(km.NavGroup3.Keys()) == 0 {
		t.Error("expected NavGroup3 keys to be set")
	}
	if len(km.NavGroup4.Keys()) == 0 {
		t.Error("expected NavGroup4 keys to be set")
	}

	// Test navigation keys
	if len(km.Up.Keys()) == 0 {
		t.Error("expected Up keys to be set")
	}
	if len(km.Down.Keys()) == 0 {
		t.Error("expected Down keys to be set")
	}
}

func TestKeyMap_ShortHelp(t *testing.T) {
	km := DefaultKeyMap()
	help := km.ShortHelp()

	if len(help) == 0 {
		t.Error("expected ShortHelp to return bindings")
	}
}

func TestKeyMap_FullHelp(t *testing.T) {
	km := DefaultKeyMap()
	help := km.FullHelp()

	if len(help) == 0 {
		t.Error("expected FullHelp to return binding groups")
	}

	// Should have 4 groups
	if len(help) != 4 {
		t.Errorf("expected 4 help groups, got %d", len(help))
	}
}

func TestDefaultLoginKeyMap(t *testing.T) {
	km := DefaultLoginKeyMap()

	if len(km.Submit.Keys()) == 0 {
		t.Error("expected Submit keys to be set")
	}
	if len(km.Tab.Keys()) == 0 {
		t.Error("expected Tab keys to be set")
	}
	if len(km.Quit.Keys()) == 0 {
		t.Error("expected Quit keys to be set")
	}
}

func TestDefaultPickerKeyMap(t *testing.T) {
	km := DefaultPickerKeyMap()

	if len(km.Select.Keys()) == 0 {
		t.Error("expected Select keys to be set")
	}
	if len(km.Add.Keys()) == 0 {
		t.Error("expected Add keys to be set")
	}
	if len(km.Delete.Keys()) == 0 {
		t.Error("expected Delete keys to be set")
	}
	if len(km.Back.Keys()) == 0 {
		t.Error("expected Back keys to be set")
	}
	if len(km.Up.Keys()) == 0 {
		t.Error("expected Up keys to be set")
	}
	if len(km.Down.Keys()) == 0 {
		t.Error("expected Down keys to be set")
	}
}

func TestDefaultDevicePickerKeyMap(t *testing.T) {
	km := DefaultDevicePickerKeyMap()

	if len(km.Select.Keys()) == 0 {
		t.Error("expected Select keys to be set")
	}
	if len(km.Refresh.Keys()) == 0 {
		t.Error("expected Refresh keys to be set")
	}
	if len(km.Back.Keys()) == 0 {
		t.Error("expected Back keys to be set")
	}
	if len(km.Up.Keys()) == 0 {
		t.Error("expected Up keys to be set")
	}
	if len(km.Down.Keys()) == 0 {
		t.Error("expected Down keys to be set")
	}
}
