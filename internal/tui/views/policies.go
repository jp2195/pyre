package views

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
)

type PoliciesModel struct {
	list RuleListModel[models.SecurityRule]
}

func NewPoliciesModel() PoliciesModel {
	config := RuleListConfig[models.SecurityRule]{
		Title:             "Security Policies",
		LoadingMsg:        "Loading policies...",
		EmptyMsg:          "No policies found",
		FilterPlaceholder: "Filter rules...",
		SortLabels:        []string{"Position", "Name", "Hits", "Last Hit"},
		DefaultSortAsc:    func(idx int) bool { return idx == 0 || idx == 1 }, // Position and Name ascending
		MatchFilter:       matchSecurityRule,
		CompareItems:      compareSecurityRule,
		FormatHeaderRow:   formatSecurityHeader,
		FormatRow:         formatSecurityRow,
		RenderDetail:      renderSecurityDetail,
		IsDisabled:        func(r models.SecurityRule) bool { return r.Disabled },
	}
	return PoliciesModel{list: NewRuleListModel(config)}
}

func (m PoliciesModel) SetSize(width, height int) PoliciesModel {
	m.list = m.list.SetSize(width, height)
	return m
}

func (m PoliciesModel) SetLoading(loading bool) PoliciesModel {
	m.list = m.list.SetLoading(loading)
	return m
}

func (m PoliciesModel) HasData() bool {
	return m.list.HasData()
}

func (m PoliciesModel) SetPolicies(policies []models.SecurityRule, err error) PoliciesModel {
	m.list = m.list.SetItems(policies, err)
	return m
}

func (m PoliciesModel) Update(msg tea.Msg) (PoliciesModel, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m PoliciesModel) View() string {
	return m.list.View()
}

// Expose TableBase fields needed by app.go
func (m PoliciesModel) SetSpinnerFrame(frame string) PoliciesModel {
	m.list.SpinnerFrame = frame
	return m
}

// --- Type-specific functions ---

func matchSecurityRule(p models.SecurityRule, query string) bool {
	return strings.Contains(strings.ToLower(p.Name), query) ||
		strings.Contains(strings.ToLower(p.Description), query) ||
		strings.Contains(strings.ToLower(string(p.RuleBase)), query) ||
		containsAny(p.Tags, query) ||
		containsAny(p.SourceZones, query) ||
		containsAny(p.DestZones, query) ||
		containsAny(p.Sources, query) ||
		containsAny(p.Destinations, query) ||
		containsAny(p.Applications, query) ||
		containsAny(p.Services, query)
}

func compareSecurityRule(a, b models.SecurityRule, sortIdx int) bool {
	switch sortIdx {
	case 1: // Name
		return a.Name < b.Name
	case 2: // Hits
		return a.HitCount < b.HitCount
	case 3: // Last Hit
		return a.LastHit.Before(b.LastHit)
	default: // Position
		return a.Position < b.Position
	}
}

func formatSecurityHeader(width int) string {
	if width >= 150 {
		return fmt.Sprintf("%-4s %-5s %-24s %-8s %-20s %-18s %-16s %-10s %-10s",
			"#", "Base", "Name", "Action", "Source → Dest Zone", "Application", "Service", "Hits", "Last Hit")
	} else if width >= 120 {
		return fmt.Sprintf("%-4s %-5s %-20s %-8s %-18s %-14s %-10s %-10s",
			"#", "Base", "Name", "Action", "Zones", "Application", "Hits", "Last Hit")
	} else if width >= 100 {
		return fmt.Sprintf("%-4s %-5s %-16s %-7s %-14s %-10s %-8s",
			"#", "Base", "Name", "Action", "Zones", "App", "Hits")
	}
	return fmt.Sprintf("%-4s %-16s %-7s %-14s %-8s",
		"#", "Name", "Action", "Zones", "Hits")
}

func formatSecurityRow(p models.SecurityRule, width int) string {
	action := strings.ToUpper(p.Action)
	if len(action) > 8 {
		action = action[:7] + "…"
	}

	base := formatRuleBase(p.RuleBase)
	srcZone := formatZoneCompact(p.SourceZones)
	dstZone := formatZoneCompact(p.DestZones)
	zones := srcZone + "→" + dstZone
	apps := formatListCompact(p.Applications, 14)
	services := formatListCompact(p.Services, 14)
	hits := formatHitCount(p.HitCount)
	lastHit := formatLastHit(p.LastHit)

	name := p.Name
	if len(p.Tags) > 0 {
		name = name + " •"
	}

	if width >= 150 {
		return fmt.Sprintf("%-4d %-5s %-24s %-8s %-20s %-18s %-16s %-10s %-10s",
			p.Position, base, truncateStr(name, 24), action,
			truncateStr(zones, 20), truncateStr(apps, 18),
			truncateStr(services, 16), hits, lastHit)
	} else if width >= 120 {
		return fmt.Sprintf("%-4d %-5s %-20s %-8s %-18s %-14s %-10s %-10s",
			p.Position, base, truncateStr(name, 20), action,
			truncateStr(zones, 18), truncateStr(apps, 14), hits, lastHit)
	} else if width >= 100 {
		return fmt.Sprintf("%-4d %-5s %-16s %-7s %-14s %-10s %-8s",
			p.Position, base, truncateStr(name, 16), truncateStr(action, 7),
			truncateStr(zones, 14), truncateStr(apps, 10), hits)
	}
	return fmt.Sprintf("%-4d %-16s %-7s %-14s %-8s",
		p.Position, truncateStr(name, 16), truncateStr(action, 7),
		truncateStr(zones, 14), hits)
}

func renderSecurityDetail(p models.SecurityRule, width int) string {
	dr := NewDetailRenderer(width, 16)

	title := p.Name
	if p.Disabled {
		title += DetailDimStyle.Render(" (disabled)")
	}
	dr.Title(title)
	dr.Subtitle(fmt.Sprintf("Position: %d | %s", p.Position, formatRuleBaseFull(p.RuleBase)))
	dr.Tags(p.Tags)
	dr.Description(p.Description)

	// Source/Destination Section
	dr.Section("Traffic Match")
	dr.Field("Source Zones:", formatListFull(p.SourceZones))
	dr.FieldStyled("Source Addr:", formatAddresses(p.Sources, p.NegateSource, DetailValueStyle, DetailDimStyle))
	if len(p.SourceUsers) > 0 && (len(p.SourceUsers) != 1 || p.SourceUsers[0] != "any") {
		dr.Field("Source Users:", formatListFull(p.SourceUsers))
	}
	dr.Field("Dest Zones:", formatListFull(p.DestZones))
	dr.FieldStyled("Dest Addr:", formatAddresses(p.Destinations, p.NegateDest, DetailValueStyle, DetailDimStyle))

	// Application/Service Section
	dr.Section("Application/Service")
	dr.Field("Applications:", formatListFull(p.Applications))
	dr.Field("Services:", formatListFull(p.Services))
	if len(p.URLCategories) > 0 && (len(p.URLCategories) != 1 || p.URLCategories[0] != "any") {
		dr.Field("URL Categories:", formatListFull(p.URLCategories))
	}

	// Action & Profiles Section
	dr.Section("Action & Profiles")
	dr.FieldStyled("Action:", ActionStyle(p.Action).Render(strings.ToUpper(p.Action)))

	if p.Profile != "" {
		dr.Field("Profile Group:", p.Profile)
	} else if hasProfiles(p) {
		var profiles []string
		if p.AntivirusProfile != "" {
			profiles = append(profiles, "AV:"+p.AntivirusProfile)
		}
		if p.VulnerabilityProfile != "" {
			profiles = append(profiles, "Vuln:"+p.VulnerabilityProfile)
		}
		if p.SpywareProfile != "" {
			profiles = append(profiles, "Spy:"+p.SpywareProfile)
		}
		if p.URLFilteringProfile != "" {
			profiles = append(profiles, "URL:"+p.URLFilteringProfile)
		}
		if p.WildFireProfile != "" {
			profiles = append(profiles, "WF:"+p.WildFireProfile)
		}
		dr.Field("Profiles:", strings.Join(profiles, ", "))
	}

	// Logging
	var logSettings []string
	if p.LogStart {
		logSettings = append(logSettings, "start")
	}
	if p.LogEnd {
		logSettings = append(logSettings, "end")
	}
	if len(logSettings) > 0 {
		logStr := strings.Join(logSettings, ", ")
		if p.LogForwarding != "" {
			logStr += " → " + p.LogForwarding
		}
		dr.Field("Logging:", logStr)
	}

	// Usage Stats Section
	dr.Section("Usage Statistics")
	dr.Field("Hit Count:", formatHitCountFull(p.HitCount))
	dr.Field("Last Hit:", formatTimestamp(p.LastHit))
	if !p.FirstHit.IsZero() {
		dr.Field("First Hit:", formatTimestamp(p.FirstHit))
	}

	return dr.Render()
}
