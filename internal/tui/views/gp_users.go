package views

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
)

type GPUsersModel struct {
	list RuleListModel[models.GlobalProtectUser]
	// Width, Height, SpinnerFrame, and Loading mirror the corresponding
	// TableBase fields so that the tui package's fanout tests can read them
	// without reaching into the unexported list field.
	Width        int
	Height       int
	SpinnerFrame string
	Loading      bool
}

func NewGPUsersModel() GPUsersModel {
	config := RuleListConfig[models.GlobalProtectUser]{
		Title:             "GlobalProtect Users",
		ItemNoun:          "users",
		LoadingMsg:        "Loading GlobalProtect users...",
		EmptyMsg:          "No GlobalProtect users found",
		FilterPlaceholder: "Filter users...",
		SortLabels:        []string{"Username", "Gateway", "Login Time", "Duration"},
		DefaultSortAsc:    func(idx int) bool { return idx == 0 || idx == 1 },
		MatchFilter:       matchGPUser,
		CompareItems:      compareGPUser,
		FormatHeaderRow:   formatGPUserHeader,
		FormatRow:         formatGPUserRow,
		RenderDetail:      renderGPUserDetail,
	}
	return GPUsersModel{list: NewRuleListModel(config)}
}

func (m GPUsersModel) SetSize(width, height int) GPUsersModel {
	m.list = m.list.SetSize(width, height)
	m.Width = m.list.Width
	m.Height = m.list.Height
	return m
}

func (m GPUsersModel) SetLoading(loading bool) GPUsersModel {
	m.list = m.list.SetLoading(loading)
	m.Loading = loading
	return m
}

// SetSpinnerFrame updates the current spinner animation frame.
func (m GPUsersModel) SetSpinnerFrame(frame string) GPUsersModel {
	m.list.SpinnerFrame = frame
	m.SpinnerFrame = frame
	return m
}

// HasData returns true if user data has been loaded.
func (m GPUsersModel) HasData() bool {
	return m.list.HasData()
}

func (m GPUsersModel) SetUsers(users []models.GlobalProtectUser, err error) GPUsersModel {
	m.list = m.list.SetItems(users, err)
	return m
}

func (m GPUsersModel) Update(msg tea.Msg) (GPUsersModel, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m GPUsersModel) View() string {
	return m.list.View()
}

// --- Type-specific functions ---

func matchGPUser(u models.GlobalProtectUser, query string) bool {
	return strings.Contains(strings.ToLower(u.Username), query) ||
		strings.Contains(strings.ToLower(u.Domain), query) ||
		strings.Contains(strings.ToLower(u.Computer), query) ||
		strings.Contains(strings.ToLower(u.Gateway), query) ||
		strings.Contains(strings.ToLower(u.ClientIP), query) ||
		strings.Contains(strings.ToLower(u.VirtualIP), query) ||
		strings.Contains(strings.ToLower(u.SourceRegion), query)
}

func compareGPUser(a, b models.GlobalProtectUser, sortIdx int) bool {
	switch sortIdx {
	case 1: // Gateway
		return a.Gateway < b.Gateway
	case 2: // Login Time
		return a.LoginTime.Before(b.LoginTime)
	case 3: // Duration
		return a.Duration < b.Duration
	default: // Username
		return a.Username < b.Username
	}
}

func formatGPUserHeader(width int) string {
	if width >= 140 {
		return fmt.Sprintf("%-18s %-15s %-15s %-15s %-15s %-10s %-12s %-10s",
			"Username", "Domain", "Gateway", "Virtual IP", "Client IP", "Duration", "Region", "Traffic")
	} else if width >= 100 {
		return fmt.Sprintf("%-18s %-15s %-15s %-15s %-10s %-10s",
			"Username", "Gateway", "Virtual IP", "Client IP", "Duration", "Traffic")
	}
	return fmt.Sprintf("%-16s %-15s %-15s %-10s",
		"Username", "Gateway", "Virtual IP", "Duration")
}

func formatGPUserRow(u models.GlobalProtectUser, width int) string {
	duration := u.Duration
	if duration == "" && !u.LoginTime.IsZero() {
		duration = formatTimeAgo(u.LoginTime)
	}
	traffic := formatBytes(u.BytesIn + u.BytesOut)

	if width >= 140 {
		return fmt.Sprintf("%-18s %-15s %-15s %-15s %-15s %-10s %-12s %-10s",
			truncateEllipsis(u.Username, 18),
			truncateEllipsis(u.Domain, 15),
			truncateEllipsis(u.Gateway, 15),
			truncateEllipsis(u.VirtualIP, 15),
			truncateEllipsis(u.ClientIP, 15),
			truncateEllipsis(duration, 10),
			truncateEllipsis(u.SourceRegion, 12),
			traffic)
	} else if width >= 100 {
		return fmt.Sprintf("%-18s %-15s %-15s %-15s %-10s %-10s",
			truncateEllipsis(u.Username, 18),
			truncateEllipsis(u.Gateway, 15),
			truncateEllipsis(u.VirtualIP, 15),
			truncateEllipsis(u.ClientIP, 15),
			truncateEllipsis(duration, 10),
			traffic)
	}
	return fmt.Sprintf("%-16s %-15s %-15s %-10s",
		truncateEllipsis(u.Username, 16),
		truncateEllipsis(u.Gateway, 15),
		truncateEllipsis(u.VirtualIP, 15),
		truncateEllipsis(duration, 10))
}

func renderGPUserDetail(u models.GlobalProtectUser, width int) string {
	dr := NewDetailRenderer(width, 18)
	dr.Title(u.Username)
	dr.Newline()

	dr.Section("User Information")
	dr.FieldIf("Domain:", u.Domain)
	dr.FieldIf("Computer:", u.Computer)
	dr.FieldIf("Client Version:", u.Client)

	dr.Section("Connection")
	dr.FieldIf("Gateway:", u.Gateway)
	dr.FieldIf("Virtual IP:", u.VirtualIP)
	dr.FieldIf("Public IP:", u.ClientIP)
	dr.FieldIf("Source Region:", u.SourceRegion)

	dr.Section("Session")
	if !u.LoginTime.IsZero() {
		dr.Field("Login Time:", u.LoginTime.Format("2006-01-02 15:04:05"))
	}
	dr.FieldIf("Duration:", u.Duration)

	if u.BytesIn > 0 || u.BytesOut > 0 {
		dr.Section("Traffic")
		dr.Field("Bytes In:", formatBytes(u.BytesIn))
		dr.Field("Bytes Out:", formatBytes(u.BytesOut))
	}

	return dr.Render()
}
