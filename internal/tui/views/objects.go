package views

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
)

// ObjectsTab selects which sub-tab the Objects view is currently showing.
type ObjectsTab int

const (
	ObjectsTabAddress ObjectsTab = iota
	ObjectsTabService
)

// AddressSortField cycles through address sort modes.
type AddressSortField int

const (
	AddressSortName AddressSortField = iota
	AddressSortType
	AddressSortValue
)

// ServiceSortField cycles through service sort modes.
type ServiceSortField int

const (
	ServiceSortName ServiceSortField = iota
	ServiceSortProtocol
	ServiceSortDestPort
)

// objectsAddressTab holds the per-tab state for the Address sub-tab.
type objectsAddressTab struct {
	TableBase
	addresses []models.AddressObject
	filtered  []models.AddressObject
	sortBy    AddressSortField
}

// objectsServiceTab holds the per-tab state for the Service sub-tab.
type objectsServiceTab struct {
	TableBase
	services []models.ServiceObject
	filtered []models.ServiceObject
	sortBy   ServiceSortField
}

// ObjectsModel renders address + service objects in two tabs.
type ObjectsModel struct {
	tab          ObjectsTab
	addressTab   objectsAddressTab
	serviceTab   objectsServiceTab
	width        int
	height       int
	spinnerFrame string
}

// NewObjectsModel returns an ObjectsModel with the Address tab selected.
func NewObjectsModel() ObjectsModel {
	addrBase := NewTableBase("Filter addresses...")
	addrBase.SortAsc = true
	svcBase := NewTableBase("Filter services...")
	svcBase.SortAsc = true
	return ObjectsModel{
		tab:        ObjectsTabAddress,
		addressTab: objectsAddressTab{TableBase: addrBase},
		serviceTab: objectsServiceTab{TableBase: svcBase},
	}
}

// ActiveTab returns the currently selected sub-tab.
func (m ObjectsModel) ActiveTab() ObjectsTab { return m.tab }

// Addresses returns the loaded address objects (unfiltered).
func (m ObjectsModel) Addresses() []models.AddressObject { return m.addressTab.addresses }

// Services returns the loaded service objects (unfiltered).
func (m ObjectsModel) Services() []models.ServiceObject { return m.serviceTab.services }

// HasData reports whether at least one tab has data loaded (used by nav to
// suppress duplicate fetches on view entry).
func (m ObjectsModel) HasData() bool {
	return m.addressTab.addresses != nil || m.serviceTab.services != nil
}

// SetSize propagates dimensions to both sub-tabs.
func (m ObjectsModel) SetSize(width, height int) ObjectsModel {
	m.width, m.height = width, height
	m.addressTab.TableBase = m.addressTab.SetSize(width, height)
	m.serviceTab.TableBase = m.serviceTab.SetSize(width, height)
	return m
}

// SetLoading sets the loading state on both sub-tabs.
func (m ObjectsModel) SetLoading(loading bool) ObjectsModel {
	m.addressTab.TableBase = m.addressTab.SetLoading(loading)
	m.serviceTab.TableBase = m.serviceTab.SetLoading(loading)
	return m
}

// SetSpinnerFrame propagates spinner frame to both sub-tabs.
func (m ObjectsModel) SetSpinnerFrame(frame string) ObjectsModel {
	m.spinnerFrame = frame
	m.addressTab.TableBase = m.addressTab.SetSpinnerFrame(frame)
	m.serviceTab.TableBase = m.serviceTab.SetSpinnerFrame(frame)
	return m
}

// SetAddresses replaces the address tab's data and refreshes its filter/sort.
func (m ObjectsModel) SetAddresses(addresses []models.AddressObject, err error) ObjectsModel {
	m.addressTab.addresses = addresses
	m.addressTab.Err = err
	m.addressTab.Loading = false
	m.addressTab.Cursor = 0
	m.addressTab.Offset = 0
	m.addressTab.applyFilter()
	return m
}

// SetServices replaces the service tab's data and refreshes its filter/sort.
func (m ObjectsModel) SetServices(services []models.ServiceObject, err error) ObjectsModel {
	m.serviceTab.services = services
	m.serviceTab.Err = err
	m.serviceTab.Loading = false
	m.serviceTab.Cursor = 0
	m.serviceTab.Offset = 0
	m.serviceTab.applyFilter()
	return m
}

// LookupAddress is the Phase 2 hook: returns the matching address object by name.
func (m ObjectsModel) LookupAddress(name string) (models.AddressObject, bool) {
	for _, a := range m.addressTab.addresses {
		if a.Name == name {
			return a, true
		}
	}
	return models.AddressObject{}, false
}

// LookupService is the Phase 2 hook: returns the matching service object by name.
func (m ObjectsModel) LookupService(name string) (models.ServiceObject, bool) {
	for _, s := range m.serviceTab.services {
		if s.Name == name {
			return s, true
		}
	}
	return models.ServiceObject{}, false
}

func (t *objectsAddressTab) applyFilter() {
	if t.FilterValue() == "" {
		t.filtered = make([]models.AddressObject, len(t.addresses))
		copy(t.filtered, t.addresses)
	} else {
		query := strings.ToLower(t.FilterValue())
		t.filtered = nil
		for _, a := range t.addresses {
			if matchesAddress(a, query) {
				t.filtered = append(t.filtered, a)
			}
		}
	}
	t.applySort()
}

func (t *objectsAddressTab) applySort() {
	slices.SortFunc(t.filtered, func(a, b models.AddressObject) int {
		var c int
		switch t.sortBy {
		case AddressSortType:
			c = cmp.Compare(a.Type, b.Type)
		case AddressSortValue:
			c = cmp.Compare(a.Value, b.Value)
		default:
			c = cmp.Compare(a.Name, b.Name)
		}
		if !t.SortAsc {
			c = -c
		}
		return c
	})
}

func (t *objectsServiceTab) applyFilter() {
	if t.FilterValue() == "" {
		t.filtered = make([]models.ServiceObject, len(t.services))
		copy(t.filtered, t.services)
	} else {
		query := strings.ToLower(t.FilterValue())
		t.filtered = nil
		for _, s := range t.services {
			if matchesService(s, query) {
				t.filtered = append(t.filtered, s)
			}
		}
	}
	t.applySort()
}

func (t *objectsServiceTab) applySort() {
	slices.SortFunc(t.filtered, func(a, b models.ServiceObject) int {
		var c int
		switch t.sortBy {
		case ServiceSortProtocol:
			c = cmp.Compare(a.Protocol, b.Protocol)
		case ServiceSortDestPort:
			c = cmp.Compare(a.DestPort, b.DestPort)
		default:
			c = cmp.Compare(a.Name, b.Name)
		}
		if !t.SortAsc {
			c = -c
		}
		return c
	})
}

func matchesAddress(a models.AddressObject, query string) bool {
	if strings.Contains(strings.ToLower(a.Name), query) ||
		strings.Contains(strings.ToLower(a.Type), query) ||
		strings.Contains(strings.ToLower(a.Value), query) ||
		strings.Contains(strings.ToLower(a.Description), query) {
		return true
	}
	for _, tag := range a.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	return false
}

func matchesService(s models.ServiceObject, query string) bool {
	if strings.Contains(strings.ToLower(s.Name), query) ||
		strings.Contains(strings.ToLower(s.Protocol), query) ||
		strings.Contains(strings.ToLower(s.DestPort), query) ||
		strings.Contains(strings.ToLower(s.SrcPort), query) ||
		strings.Contains(strings.ToLower(s.Description), query) {
		return true
	}
	for _, tag := range s.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	return false
}

// Update handles a single bubbletea message for the active sub-tab.
func (m ObjectsModel) Update(msg tea.Msg) (ObjectsModel, tea.Cmd) {
	// Filter mode (per-tab) consumes most keys.
	switch m.tab {
	case ObjectsTabAddress:
		if m.addressTab.FilterMode {
			return m.updateAddressFilter(msg)
		}
	case ObjectsTabService:
		if m.serviceTab.FilterMode {
			return m.updateServiceFilter(msg)
		}
	}

	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	// Tab-switch keys take precedence (active in either tab).
	switch key.String() {
	case "tab":
		if m.tab == ObjectsTabAddress {
			m.tab = ObjectsTabService
		} else {
			m.tab = ObjectsTabAddress
		}
		return m, nil
	case "a":
		m.tab = ObjectsTabAddress
		return m, nil
	case "s":
		// 's' is also used by other views for sort. In Objects view the
		// tab-switch wins because we have two tabs to choose between; sort
		// cycling uses capital 'S' instead (see Task 9).
		m.tab = ObjectsTabService
		return m, nil
	case "S":
		switch m.tab {
		case ObjectsTabAddress:
			m.addressTab.sortBy = (m.addressTab.sortBy + 1) % 3
			m.addressTab.SortAsc = true
			m.addressTab.applySort()
			m.addressTab.Cursor = 0
			m.addressTab.Offset = 0
		case ObjectsTabService:
			m.serviceTab.sortBy = (m.serviceTab.sortBy + 1) % 3
			m.serviceTab.SortAsc = true
			m.serviceTab.applySort()
			m.serviceTab.Cursor = 0
			m.serviceTab.Offset = 0
		}
		return m, nil
	case "esc":
		switch m.tab {
		case ObjectsTabAddress:
			if m.addressTab.HandleCollapseIfExpanded() {
				return m, nil
			}
			if m.addressTab.HandleClearFilter() {
				m.addressTab.applyFilter()
			}
		case ObjectsTabService:
			if m.serviceTab.HandleCollapseIfExpanded() {
				return m, nil
			}
			if m.serviceTab.HandleClearFilter() {
				m.serviceTab.applyFilter()
			}
		}
		return m, nil
	}

	// Delegate navigation to the active tab.
	switch m.tab {
	case ObjectsTabAddress:
		visible := m.addressTab.VisibleRows(8, 14)
		base, handled, cmd := m.addressTab.HandleNavigation(key, len(m.addressTab.filtered), visible)
		if handled {
			m.addressTab.TableBase = base
			return m, cmd
		}
	case ObjectsTabService:
		visible := m.serviceTab.VisibleRows(8, 14)
		base, handled, cmd := m.serviceTab.HandleNavigation(key, len(m.serviceTab.filtered), visible)
		if handled {
			m.serviceTab.TableBase = base
			return m, cmd
		}
	}

	return m, nil
}

func (m ObjectsModel) updateAddressFilter(msg tea.Msg) (ObjectsModel, tea.Cmd) {
	base, exited, cmd := m.addressTab.HandleFilterMode(msg)
	m.addressTab.TableBase = base
	if exited {
		m.addressTab.applyFilter()
	}
	return m, cmd
}

func (m ObjectsModel) updateServiceFilter(msg tea.Msg) (ObjectsModel, tea.Cmd) {
	base, exited, cmd := m.serviceTab.HandleFilterMode(msg)
	m.serviceTab.TableBase = base
	if exited {
		m.serviceTab.applyFilter()
	}
	return m, cmd
}

// View renders the objects screen for the active sub-tab.
func (m ObjectsModel) View() string {
	if m.width == 0 {
		return RenderLoadingInline(m.spinnerFrame, "Loading...")
	}

	titleStyle := ViewTitleStyle.MarginBottom(1)
	panelStyle := ViewPanelStyle.Width(m.width - 4)

	var b strings.Builder
	b.WriteString(titleStyle.Render("Objects"))
	b.WriteString("  ")
	b.WriteString(m.renderTabIndicator())
	b.WriteString("\n")

	switch m.tab {
	case ObjectsTabAddress:
		b.WriteString(m.renderAddressTab())
	case ObjectsTabService:
		b.WriteString(m.renderServiceTab())
	}

	return panelStyle.Render(b.String())
}

func (m ObjectsModel) renderTabIndicator() string {
	addr := "Address"
	svc := "Service"
	if m.tab == ObjectsTabAddress {
		addr = StatusActiveStyle.Render("[Address]")
		svc = StatusMutedStyle.Render(svc)
	} else {
		addr = StatusMutedStyle.Render(addr)
		svc = StatusActiveStyle.Render("[Service]")
	}
	hint := BannerInfoStyle.Render("  (a/s/Tab to switch)")
	return addr + "  " + svc + hint
}

func (m ObjectsModel) renderAddressTab() string {
	t := m.addressTab
	var b strings.Builder

	if t.FilterMode {
		b.WriteString(FilterBorderStyle.Render(t.Filter.View()))
		b.WriteString("\n\n")
	} else if t.IsFiltered() {
		filterInfo := FilterActiveStyle.Render(fmt.Sprintf("Filtered: %q", t.FilterValue()))
		clearHint := FilterClearHintStyle.Render(" (esc to clear)")
		b.WriteString(filterInfo + clearHint)
		b.WriteString("\n\n")
	}

	if t.Err != nil {
		b.WriteString(ErrorMsgStyle.Render("Error: " + t.Err.Error()))
		return b.String()
	}
	if t.Loading || t.addresses == nil {
		b.WriteString(RenderLoadingInline(t.SpinnerFrame, "Loading addresses..."))
		return b.String()
	}
	if len(t.filtered) == 0 {
		b.WriteString(EmptyMsgStyle.Render("No address objects defined"))
		return b.String()
	}

	b.WriteString(m.renderAddressTable())
	if t.Expanded && t.Cursor < len(t.filtered) {
		b.WriteString("\n")
		b.WriteString(m.renderAddressDetail(t.filtered[t.Cursor]))
	}
	return b.String()
}

func (m ObjectsModel) renderServiceTab() string {
	t := m.serviceTab
	var b strings.Builder

	if t.FilterMode {
		b.WriteString(FilterBorderStyle.Render(t.Filter.View()))
		b.WriteString("\n\n")
	} else if t.IsFiltered() {
		filterInfo := FilterActiveStyle.Render(fmt.Sprintf("Filtered: %q", t.FilterValue()))
		clearHint := FilterClearHintStyle.Render(" (esc to clear)")
		b.WriteString(filterInfo + clearHint)
		b.WriteString("\n\n")
	}

	if t.Err != nil {
		b.WriteString(ErrorMsgStyle.Render("Error: " + t.Err.Error()))
		return b.String()
	}
	if t.Loading || t.services == nil {
		b.WriteString(RenderLoadingInline(t.SpinnerFrame, "Loading services..."))
		return b.String()
	}
	if len(t.filtered) == 0 {
		b.WriteString(EmptyMsgStyle.Render("No service objects defined"))
		return b.String()
	}

	b.WriteString(m.renderServiceTable())
	if t.Expanded && t.Cursor < len(t.filtered) {
		b.WriteString("\n")
		b.WriteString(m.renderServiceDetail(t.filtered[t.Cursor]))
	}
	return b.String()
}

func (m ObjectsModel) renderAddressTable() string {
	t := m.addressTab
	headerStyle := DetailLabelStyle.Bold(true)
	selectedStyle := TableRowSelectedStyle.Bold(true)
	dimStyle := DetailDimStyle
	availableWidth := m.width - 12

	var b strings.Builder
	header := fmt.Sprintf("%-24s %-12s %-26s %s", "NAME", "TYPE", "VALUE", "TAGS")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("-", min(availableWidth, len(header)+10))))
	b.WriteString("\n")

	visibleRows := t.VisibleRows(8, 14)
	end := min(t.Offset+visibleRows, len(t.filtered))
	for i := t.Offset; i < end; i++ {
		a := t.filtered[i]
		row := fmt.Sprintf("%-24s %-12s %-26s %s",
			truncateEllipsis(a.Name, 24),
			truncateEllipsis(strings.TrimPrefix(a.Type, "ip-"), 12),
			truncateEllipsis(a.Value, 26),
			strings.Join(a.Tags, " "),
		)
		if i == t.Cursor {
			b.WriteString(selectedStyle.Render(row))
		} else {
			b.WriteString(DetailValueStyle.Render(row))
		}
		b.WriteString("\n")
	}
	if len(t.filtered) > visibleRows {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  Showing %d-%d of %d", t.Offset+1, end, len(t.filtered))))
	}
	return b.String()
}

func (m ObjectsModel) renderAddressDetail(a models.AddressObject) string {
	dr := NewDetailRenderer(m.width, 18)
	dr.Raw(ViewTitleStyle.Render(a.Name) + "\n")
	dr.Newline()
	dr.Section("Object")
	dr.Field("Type:", a.Type)
	dr.Field("Value:", a.Value)
	dr.FieldIf("Description:", a.Description)
	if len(a.Tags) > 0 {
		dr.Field("Tags:", strings.Join(a.Tags, ", "))
	}
	return dr.Render()
}

func (m ObjectsModel) renderServiceDetail(s models.ServiceObject) string {
	dr := NewDetailRenderer(m.width, 18)
	dr.Raw(ViewTitleStyle.Render(s.Name) + "\n")
	dr.Newline()
	dr.Section("Service")
	dr.Field("Protocol:", s.Protocol)
	dr.Field("Dest Port:", s.DestPort)
	dr.FieldIf("Src Port:", s.SrcPort)
	dr.FieldIf("Description:", s.Description)
	if len(s.Tags) > 0 {
		dr.Field("Tags:", strings.Join(s.Tags, ", "))
	}
	return dr.Render()
}

func (m ObjectsModel) renderServiceTable() string {
	t := m.serviceTab
	headerStyle := DetailLabelStyle.Bold(true)
	selectedStyle := TableRowSelectedStyle.Bold(true)
	dimStyle := DetailDimStyle
	availableWidth := m.width - 12

	var b strings.Builder
	header := fmt.Sprintf("%-24s %-8s %-16s %-16s %s", "NAME", "PROTO", "DEST PORT", "SRC PORT", "TAGS")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("-", min(availableWidth, len(header)+10))))
	b.WriteString("\n")

	visibleRows := t.VisibleRows(8, 14)
	end := min(t.Offset+visibleRows, len(t.filtered))
	for i := t.Offset; i < end; i++ {
		s := t.filtered[i]
		row := fmt.Sprintf("%-24s %-8s %-16s %-16s %s",
			truncateEllipsis(s.Name, 24),
			truncateEllipsis(s.Protocol, 8),
			truncateEllipsis(s.DestPort, 16),
			truncateEllipsis(s.SrcPort, 16),
			strings.Join(s.Tags, " "),
		)
		if i == t.Cursor {
			b.WriteString(selectedStyle.Render(row))
		} else {
			b.WriteString(DetailValueStyle.Render(row))
		}
		b.WriteString("\n")
	}
	if len(t.filtered) > visibleRows {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  Showing %d-%d of %d", t.Offset+1, end, len(t.filtered))))
	}
	return b.String()
}
