package views

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// RuleListConfig defines the type-specific behavior for a RuleListModel.
type RuleListConfig[T any] struct {
	Title             string
	LoadingMsg        string
	EmptyMsg          string
	FilterPlaceholder string
	SortLabels        []string                                 // Labels for each sort field
	DefaultSortAsc    func(sortIdx int) bool                   // Whether a sort field defaults to ascending
	MatchFilter       func(item T, query string) bool          // Returns true if item matches filter query
	CompareItems      func(a, b T, sortIdx int) bool           // Returns true if a < b for the given sort field
	FormatHeaderRow   func(width int) string                   // Renders the table header for available width
	FormatRow         func(item T, width int) string           // Renders a single row
	RenderDetail      func(item T, width int) string           // Renders the detail panel
	IsDisabled        func(item T) bool                        // Returns true if item should render as disabled
}

// RuleListModel provides a generic, filterable, sortable list with detail expansion.
// It is used by PoliciesModel and NATPoliciesModel to eliminate structural duplication.
type RuleListModel[T any] struct {
	TableBase
	config   RuleListConfig[T]
	items    []T
	filtered []T
	sortBy   int
}

// NewRuleListModel creates a new rule list with the given config.
func NewRuleListModel[T any](config RuleListConfig[T]) RuleListModel[T] {
	return RuleListModel[T]{
		TableBase: NewTableBase(config.FilterPlaceholder),
		config:    config,
	}
}

func (m RuleListModel[T]) SetSize(width, height int) RuleListModel[T] {
	m.TableBase = m.TableBase.SetSize(width, height)

	// Clamp cursor to valid range after resize
	count := len(m.filtered)
	if m.Cursor >= count && count > 0 {
		m.Cursor = count - 1
	}

	// Adjust offset to keep cursor visible
	visibleRows := m.visibleRows()
	if visibleRows > 0 && m.Cursor >= m.Offset+visibleRows {
		m.Offset = m.Cursor - visibleRows + 1
	}

	return m
}

func (m RuleListModel[T]) SetLoading(loading bool) RuleListModel[T] {
	m.TableBase = m.TableBase.SetLoading(loading)
	return m
}

// HasData returns true if items have been loaded.
func (m RuleListModel[T]) HasData() bool {
	return m.items != nil
}

// SetItems replaces the item list, resets cursor, and re-applies filter/sort.
func (m RuleListModel[T]) SetItems(items []T, err error) RuleListModel[T] {
	m.items = items
	m.Err = err
	m.Loading = false
	m.Cursor = 0
	m.Offset = 0
	m.applyFilter()
	return m
}

// Items returns the full (unfiltered) items slice.
func (m RuleListModel[T]) Items() []T {
	return m.items
}

// Filtered returns the filtered/sorted items slice.
func (m RuleListModel[T]) Filtered() []T {
	return m.filtered
}

func (m *RuleListModel[T]) applyFilter() {
	if m.FilterValue() == "" {
		m.filtered = make([]T, len(m.items))
		copy(m.filtered, m.items)
	} else {
		query := strings.ToLower(m.FilterValue())
		m.filtered = nil
		for _, item := range m.items {
			if m.config.MatchFilter(item, query) {
				m.filtered = append(m.filtered, item)
			}
		}
	}
	m.applySort()
}

func (m *RuleListModel[T]) applySort() {
	sortBy := m.sortBy
	asc := m.SortAsc
	sort.Slice(m.filtered, func(i, j int) bool {
		less := m.config.CompareItems(m.filtered[i], m.filtered[j], sortBy)
		if asc {
			return less
		}
		return !less
	})
}

func (m *RuleListModel[T]) cycleSort() {
	numFields := len(m.config.SortLabels)
	if numFields == 0 {
		return
	}
	m.sortBy = (m.sortBy + 1) % numFields
	m.SortAsc = m.config.DefaultSortAsc(m.sortBy)
	m.applySort()
}

func (m RuleListModel[T]) sortLabel() string {
	if m.sortBy >= len(m.config.SortLabels) {
		return ""
	}
	dir := "↓"
	if m.SortAsc {
		dir = "↑"
	}
	return fmt.Sprintf("%s %s", m.config.SortLabels[m.sortBy], dir)
}

func (m RuleListModel[T]) visibleRows() int {
	rows := m.Height - 8
	if m.Expanded {
		rows -= 14
	}
	if rows < 1 {
		rows = 1
	}
	return rows
}

// Update processes messages for the rule list.
func (m RuleListModel[T]) Update(msg tea.Msg) (RuleListModel[T], tea.Cmd) {
	if m.FilterMode {
		return m.updateFilter(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.HandleCollapseIfExpanded() {
				return m, nil
			}
			if m.HandleClearFilter() {
				m.applyFilter()
			}
			return m, nil
		case "s":
			m.cycleSort()
			m.Cursor = 0
			m.Offset = 0
			return m, nil
		}

		// Delegate to TableBase for common navigation
		visible := m.visibleRows()
		base, handled, cmd := m.HandleNavigation(msg, len(m.filtered), visible)
		if handled {
			m.TableBase = base
			return m, cmd
		}
	}

	return m, nil
}

func (m RuleListModel[T]) updateFilter(msg tea.Msg) (RuleListModel[T], tea.Cmd) {
	base, exited, cmd := m.HandleFilterMode(msg)
	m.TableBase = base
	if exited {
		m.applyFilter()
	}
	return m, cmd
}

// View renders the full rule list view.
func (m RuleListModel[T]) View() string {
	if m.Width == 0 {
		return RenderLoadingInline(m.SpinnerFrame, "Loading...")
	}

	titleStyle := ViewTitleStyle.MarginBottom(1)
	panelStyle := ViewPanelStyle.Width(m.Width - 4)

	var b strings.Builder
	title := m.config.Title
	sortInfo := BannerInfoStyle.Render(fmt.Sprintf(" [%d rules | Sort: %s | s: change | /: filter | enter: details]", len(m.filtered), m.sortLabel()))
	b.WriteString(titleStyle.Render(title) + sortInfo)
	b.WriteString("\n")

	if m.FilterMode {
		b.WriteString(FilterBorderStyle.Render(m.Filter.View()))
		b.WriteString("\n\n")
	} else if m.IsFiltered() {
		filterInfo := FilterActiveStyle.Render(fmt.Sprintf("Filtered: \"%s\"", m.FilterValue()))
		clearHint := FilterClearHintStyle.Render(" (esc to clear)")
		b.WriteString(filterInfo + clearHint)
		b.WriteString("\n\n")
	}

	if m.Err != nil {
		b.WriteString(ErrorMsgStyle.Render("Error: " + m.Err.Error()))
		return panelStyle.Render(b.String())
	}

	if m.Loading || m.items == nil {
		b.WriteString(RenderLoadingInline(m.SpinnerFrame, m.config.LoadingMsg))
		return panelStyle.Render(b.String())
	}

	if len(m.filtered) == 0 {
		b.WriteString(EmptyMsgStyle.Render(m.config.EmptyMsg))
		return panelStyle.Render(b.String())
	}

	b.WriteString(m.renderTable())

	if m.Expanded && m.Cursor < len(m.filtered) {
		b.WriteString("\n")
		b.WriteString(m.config.RenderDetail(m.filtered[m.Cursor], m.Width))
	}

	return panelStyle.Render(b.String())
}

func (m RuleListModel[T]) renderTable() string {
	headerStyle := DetailLabelStyle.Bold(true)
	selectedStyle := TableRowSelectedStyle.Bold(true)
	normalStyle := DetailValueStyle
	disabledStyle := TableRowDisabledStyle
	dimStyle := DetailDimStyle

	availableWidth := m.Width - 12

	var b strings.Builder

	// Header
	header := m.config.FormatHeaderRow(availableWidth)
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", minInt(availableWidth, len(header)+10))))
	b.WriteString("\n")

	visibleRows := m.visibleRows()
	end := min(m.Offset+visibleRows, len(m.filtered))

	for i := m.Offset; i < end; i++ {
		item := m.filtered[i]
		isSelected := i == m.Cursor

		row := m.config.FormatRow(item, availableWidth)

		if isSelected {
			b.WriteString(selectedStyle.Render(row))
		} else if m.config.IsDisabled(item) {
			b.WriteString(disabledStyle.Render(row))
		} else {
			b.WriteString(normalStyle.Render(row))
		}
		b.WriteString("\n")
	}

	// Show scroll indicator if needed
	if len(m.filtered) > visibleRows {
		scrollInfo := fmt.Sprintf("  Showing %d-%d of %d", m.Offset+1, end, len(m.filtered))
		b.WriteString(dimStyle.Render(scrollInfo))
	}

	return b.String()
}
