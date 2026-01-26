package views

import (
	"strings"
)

// NavGroup represents a top-level navigation group
type NavGroup struct {
	ID    string
	Label string
	Key   string // Display key (e.g., "1", "2")
	Items []NavItem
}

// NavItem represents a navigation item within a group
type NavItem struct {
	ID    string
	Label string
	Key   string // Display key (e.g., "1", "2")
}

// NavbarModel manages the tabbed navigation bar
type NavbarModel struct {
	groups      []NavGroup
	activeGroup int
	activeItem  int
	width       int
}

// NewNavbarModel creates a new navigation bar model
func NewNavbarModel() NavbarModel {
	return NavbarModel{
		groups: []NavGroup{
			{
				ID:    "monitor",
				Label: "Monitor",
				Key:   "1",
				Items: []NavItem{
					{ID: "overview", Label: "Overview", Key: "1"},
					{ID: "network", Label: "Network", Key: "2"},
					{ID: "security", Label: "Security", Key: "3"},
					{ID: "vpn", Label: "VPN", Key: "4"},
				},
			},
			{
				ID:    "analyze",
				Label: "Analyze",
				Key:   "2",
				Items: []NavItem{
					{ID: "policies", Label: "Policies", Key: "1"},
					{ID: "nat", Label: "NAT", Key: "2"},
					{ID: "sessions", Label: "Sessions", Key: "3"},
					{ID: "interfaces", Label: "Interfaces", Key: "4"},
					{ID: "logs", Label: "Logs", Key: "5"},
				},
			},
			{
				ID:    "tools",
				Label: "Tools",
				Key:   "3",
				Items: []NavItem{
					{ID: "config", Label: "Config", Key: "1"},
				},
			},
			{
				ID:    "connections",
				Label: "Conn",
				Key:   "4",
				Items: []NavItem{
					{ID: "picker", Label: "Switch Device", Key: "1"},
				},
			},
		},
		activeGroup: 0,
		activeItem:  0,
	}
}

// SetSize sets the width for rendering
func (m NavbarModel) SetSize(width int) NavbarModel {
	m.width = width
	return m
}

// SetActiveGroup sets the active navigation group by index
func (m NavbarModel) SetActiveGroup(idx int) NavbarModel {
	if idx >= 0 && idx < len(m.groups) {
		m.activeGroup = idx
		m.activeItem = 0 // Reset item selection when switching groups
	}
	return m
}

// SetActiveItem sets the active item within the current group
func (m NavbarModel) SetActiveItem(idx int) NavbarModel {
	if m.activeGroup >= 0 && m.activeGroup < len(m.groups) {
		items := m.groups[m.activeGroup].Items
		if idx >= 0 && idx < len(items) {
			m.activeItem = idx
		}
	}
	return m
}

// ActiveGroup returns the currently active group
func (m NavbarModel) ActiveGroup() *NavGroup {
	if m.activeGroup >= 0 && m.activeGroup < len(m.groups) {
		return &m.groups[m.activeGroup]
	}
	return nil
}

// ActiveItem returns the currently active item
func (m NavbarModel) ActiveItem() *NavItem {
	if group := m.ActiveGroup(); group != nil {
		if m.activeItem >= 0 && m.activeItem < len(group.Items) {
			return &group.Items[m.activeItem]
		}
	}
	return nil
}

// ActiveGroupIndex returns the active group index
func (m NavbarModel) ActiveGroupIndex() int {
	return m.activeGroup
}

// ActiveItemIndex returns the active item index
func (m NavbarModel) ActiveItemIndex() int {
	return m.activeItem
}

// SetActiveByID sets the active group and item by their IDs
func (m NavbarModel) SetActiveByID(groupID, itemID string) NavbarModel {
	for gi, group := range m.groups {
		if group.ID == groupID {
			m.activeGroup = gi
			for ii, item := range group.Items {
				if item.ID == itemID {
					m.activeItem = ii
					break
				}
			}
			break
		}
	}
	return m
}

// GetItemID returns the current item ID for view switching
func (m NavbarModel) GetItemID() string {
	if item := m.ActiveItem(); item != nil {
		return item.ID
	}
	return ""
}

// RenderTabs renders the navigation tabs for the header
// Returns: "1:Monitor  2:Analyze  3:Tools  4:Conn" with active group highlighted
func (m NavbarModel) RenderTabs() string {
	// Styles
	tabInactive := StatusMutedStyle.Padding(0, 1)
	tabActive := NavTabActiveStyle

	var tabs []string
	for i, group := range m.groups {
		tab := group.Key + ":" + group.Label
		if i == m.activeGroup {
			tabs = append(tabs, tabActive.Render(tab))
		} else {
			tabs = append(tabs, tabInactive.Render(tab))
		}
	}

	return strings.Join(tabs, "  ")
}

// RenderSubTabs renders the sub-navigation items for the active group
// Returns: "[Overview]  Network  Security  VPN" with active item highlighted
func (m NavbarModel) RenderSubTabs() string {
	group := m.ActiveGroup()
	if group == nil || len(group.Items) == 0 {
		return ""
	}

	// Styles
	itemInactive := StatusMutedStyle.Padding(0, 1)
	itemActive := StatusActiveStyle.Padding(0, 1)

	var items []string
	for i, item := range group.Items {
		label := item.Label
		if i == m.activeItem {
			items = append(items, itemActive.Render(label))
		} else {
			items = append(items, itemInactive.Render(label))
		}
	}

	return strings.Join(items, "  ")
}

// View renders the navigation bar (now just returns tabs)
func (m NavbarModel) View() string {
	return m.RenderTabs()
}

// Height returns the height of the navbar when rendered
func (m NavbarModel) Height() int {
	return 0 // navbar is now integrated into header row
}
