package views

import (
	"testing"
)

func TestNewNavbarModel(t *testing.T) {
	nav := NewNavbarModel()

	if nav.activeGroup != 0 {
		t.Errorf("expected activeGroup=0, got %d", nav.activeGroup)
	}
	if nav.activeItem != 0 {
		t.Errorf("expected activeItem=0, got %d", nav.activeItem)
	}
	if len(nav.groups) != 4 {
		t.Errorf("expected 4 groups, got %d", len(nav.groups))
	}

	// Verify group structure
	expectedGroups := []string{"monitor", "analyze", "tools", "connections"}
	for i, expected := range expectedGroups {
		if nav.groups[i].ID != expected {
			t.Errorf("expected group %d ID=%q, got %q", i, expected, nav.groups[i].ID)
		}
	}
}

func TestNavbarModel_SetSize(t *testing.T) {
	nav := NewNavbarModel()
	nav = nav.SetSize(100)

	if nav.width != 100 {
		t.Errorf("expected width=100, got %d", nav.width)
	}
}

func TestNavbarModel_SetActiveGroup(t *testing.T) {
	nav := NewNavbarModel()

	// Valid group
	nav = nav.SetActiveGroup(2)
	if nav.activeGroup != 2 {
		t.Errorf("expected activeGroup=2, got %d", nav.activeGroup)
	}
	if nav.activeItem != 0 {
		t.Error("expected activeItem to reset to 0")
	}

	// Set item first, then switch groups
	nav = nav.SetActiveItem(1)
	nav = nav.SetActiveGroup(1)
	if nav.activeItem != 0 {
		t.Error("expected activeItem to reset when switching groups")
	}

	// Invalid group (negative)
	nav = nav.SetActiveGroup(-1)
	if nav.activeGroup != 1 {
		t.Error("expected activeGroup to remain unchanged for invalid index")
	}

	// Invalid group (too high)
	nav = nav.SetActiveGroup(100)
	if nav.activeGroup != 1 {
		t.Error("expected activeGroup to remain unchanged for invalid index")
	}
}

func TestNavbarModel_SetActiveItem(t *testing.T) {
	nav := NewNavbarModel()
	nav = nav.SetActiveGroup(0) // Monitor group has 4 items

	// Valid item
	nav = nav.SetActiveItem(2)
	if nav.activeItem != 2 {
		t.Errorf("expected activeItem=2, got %d", nav.activeItem)
	}

	// Invalid item (negative)
	nav = nav.SetActiveItem(-1)
	if nav.activeItem != 2 {
		t.Error("expected activeItem to remain unchanged for invalid index")
	}

	// Invalid item (too high)
	nav = nav.SetActiveItem(100)
	if nav.activeItem != 2 {
		t.Error("expected activeItem to remain unchanged for invalid index")
	}
}

func TestNavbarModel_ActiveGroup(t *testing.T) {
	nav := NewNavbarModel()

	group := nav.ActiveGroup()
	if group == nil {
		t.Fatal("expected non-nil group")
	}
	if group.ID != "monitor" {
		t.Errorf("expected group ID 'monitor', got %q", group.ID)
	}

	// Switch to different group
	nav = nav.SetActiveGroup(1)
	group = nav.ActiveGroup()
	if group.ID != "analyze" {
		t.Errorf("expected group ID 'analyze', got %q", group.ID)
	}
}

func TestNavbarModel_ActiveItem(t *testing.T) {
	nav := NewNavbarModel()

	item := nav.ActiveItem()
	if item == nil {
		t.Fatal("expected non-nil item")
	}
	if item.ID != "overview" {
		t.Errorf("expected item ID 'overview', got %q", item.ID)
	}

	// Switch to different item
	nav = nav.SetActiveItem(1)
	item = nav.ActiveItem()
	if item.ID != "network" {
		t.Errorf("expected item ID 'network', got %q", item.ID)
	}
}

func TestNavbarModel_ActiveGroupIndex(t *testing.T) {
	nav := NewNavbarModel()

	if nav.ActiveGroupIndex() != 0 {
		t.Errorf("expected ActiveGroupIndex=0, got %d", nav.ActiveGroupIndex())
	}

	nav = nav.SetActiveGroup(2)
	if nav.ActiveGroupIndex() != 2 {
		t.Errorf("expected ActiveGroupIndex=2, got %d", nav.ActiveGroupIndex())
	}
}

func TestNavbarModel_ActiveItemIndex(t *testing.T) {
	nav := NewNavbarModel()

	if nav.ActiveItemIndex() != 0 {
		t.Errorf("expected ActiveItemIndex=0, got %d", nav.ActiveItemIndex())
	}

	nav = nav.SetActiveItem(3)
	if nav.ActiveItemIndex() != 3 {
		t.Errorf("expected ActiveItemIndex=3, got %d", nav.ActiveItemIndex())
	}
}

func TestNavbarModel_SetActiveByID(t *testing.T) {
	nav := NewNavbarModel()

	// Set to analyze/sessions
	nav = nav.SetActiveByID("analyze", "sessions")
	if nav.activeGroup != 1 {
		t.Errorf("expected activeGroup=1, got %d", nav.activeGroup)
	}
	if nav.activeItem != 2 {
		t.Errorf("expected activeItem=2, got %d", nav.activeItem)
	}

	// Set to tools/troubleshoot
	nav = nav.SetActiveByID("tools", "troubleshoot")
	if nav.activeGroup != 2 {
		t.Errorf("expected activeGroup=2, got %d", nav.activeGroup)
	}
	if nav.activeItem != 0 {
		t.Errorf("expected activeItem=0, got %d", nav.activeItem)
	}

	// Invalid group ID (should not change)
	nav = nav.SetActiveByID("invalid", "test")
	if nav.activeGroup != 2 {
		t.Error("expected activeGroup to remain unchanged for invalid group ID")
	}
}

func TestNavbarModel_GetItemID(t *testing.T) {
	nav := NewNavbarModel()

	id := nav.GetItemID()
	if id != "overview" {
		t.Errorf("expected 'overview', got %q", id)
	}

	nav = nav.SetActiveByID("analyze", "policies")
	id = nav.GetItemID()
	if id != "policies" {
		t.Errorf("expected 'policies', got %q", id)
	}
}

func TestNavbarModel_RenderTabs(t *testing.T) {
	nav := NewNavbarModel()

	rendered := nav.RenderTabs()
	if rendered == "" {
		t.Error("expected non-empty rendered tabs")
	}

	// Should contain group labels
	if len(rendered) < 10 {
		t.Error("rendered tabs seems too short")
	}
}

func TestNavbarModel_RenderSubTabs(t *testing.T) {
	nav := NewNavbarModel()

	rendered := nav.RenderSubTabs()
	if rendered == "" {
		t.Error("expected non-empty rendered sub-tabs")
	}

	// Should contain item labels
	if len(rendered) < 10 {
		t.Error("rendered sub-tabs seems too short")
	}
}

func TestNavbarModel_View(t *testing.T) {
	nav := NewNavbarModel()

	view := nav.View()
	if view == "" {
		t.Error("expected non-empty view")
	}

	// View should equal RenderTabs
	if view != nav.RenderTabs() {
		t.Error("expected View() to equal RenderTabs()")
	}
}

func TestNavbarModel_Height(t *testing.T) {
	nav := NewNavbarModel()

	if nav.Height() != 0 {
		t.Errorf("expected Height=0, got %d", nav.Height())
	}
}

func TestNavGroup_Fields(t *testing.T) {
	group := NavGroup{
		ID:    "test-id",
		Label: "Test Label",
		Key:   "1",
		Items: []NavItem{
			{ID: "item1", Label: "Item 1", Key: "1"},
		},
	}

	if group.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got %q", group.ID)
	}
	if group.Label != "Test Label" {
		t.Errorf("expected Label 'Test Label', got %q", group.Label)
	}
	if group.Key != "1" {
		t.Errorf("expected Key '1', got %q", group.Key)
	}
	if len(group.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(group.Items))
	}
}

func TestNavItem_Fields(t *testing.T) {
	item := NavItem{
		ID:    "test-item",
		Label: "Test Item",
		Key:   "5",
	}

	if item.ID != "test-item" {
		t.Errorf("expected ID 'test-item', got %q", item.ID)
	}
	if item.Label != "Test Item" {
		t.Errorf("expected Label 'Test Item', got %q", item.Label)
	}
	if item.Key != "5" {
		t.Errorf("expected Key '5', got %q", item.Key)
	}
}
