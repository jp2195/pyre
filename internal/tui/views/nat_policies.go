package views

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
)

type NATPoliciesModel struct {
	list RuleListModel[models.NATRule]
}

func NewNATPoliciesModel() NATPoliciesModel {
	config := RuleListConfig[models.NATRule]{
		Title:             "NAT Policies",
		ItemNoun:          "rules",
		LoadingMsg:        "Loading NAT rules...",
		EmptyMsg:          "No NAT rules found",
		FilterPlaceholder: "Filter NAT rules...",
		SortLabels:        []string{"Position", "Name", "Hits", "Last Hit"},
		DefaultSortAsc:    func(idx int) bool { return idx == 0 || idx == 1 },
		MatchFilter:       matchNATRule,
		CompareItems:      compareNATRule,
		FormatHeaderRow:   formatNATHeader,
		FormatRow:         formatNATRow,
		RenderDetail:      renderNATDetail,
		IsDisabled:        func(r models.NATRule) bool { return r.Disabled },
	}
	return NATPoliciesModel{list: NewRuleListModel(config)}
}

func (m NATPoliciesModel) SetSize(width, height int) NATPoliciesModel {
	m.list = m.list.SetSize(width, height)
	return m
}

func (m NATPoliciesModel) SetLoading(loading bool) NATPoliciesModel {
	m.list = m.list.SetLoading(loading)
	return m
}

func (m NATPoliciesModel) HasData() bool {
	return m.list.HasData()
}

func (m NATPoliciesModel) SetRules(rules []models.NATRule, err error) NATPoliciesModel {
	m.list = m.list.SetItems(rules, err)
	return m
}

func (m NATPoliciesModel) Update(msg tea.Msg) (NATPoliciesModel, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m NATPoliciesModel) View() string {
	return m.list.View()
}

func (m NATPoliciesModel) SetSpinnerFrame(frame string) NATPoliciesModel {
	m.list.SpinnerFrame = frame
	return m
}

// --- Type-specific functions ---

func matchNATRule(r models.NATRule, query string) bool {
	return strings.Contains(strings.ToLower(r.Name), query) ||
		strings.Contains(strings.ToLower(r.Description), query) ||
		strings.Contains(strings.ToLower(string(r.RuleBase)), query) ||
		containsAny(r.Tags, query) ||
		containsAny(r.SourceZones, query) ||
		containsAny(r.DestZones, query) ||
		containsAny(r.Sources, query) ||
		containsAny(r.Destinations, query) ||
		strings.Contains(strings.ToLower(r.TranslatedSource), query) ||
		strings.Contains(strings.ToLower(r.TranslatedDest), query)
}

func compareNATRule(a, b models.NATRule, sortIdx int) bool {
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

func formatNATHeader(width int) string {
	if width >= 150 {
		return fmt.Sprintf("%-4s %-5s %-20s %-16s %-16s %-18s %-18s %-10s %-10s",
			"#", "Base", "Name", "Src Zone", "Dst Zone", "Src NAT", "Dst NAT", "Hits", "Last Hit")
	} else if width >= 120 {
		return fmt.Sprintf("%-4s %-5s %-18s %-14s %-14s %-16s %-16s %-10s",
			"#", "Base", "Name", "Zones", "Service", "Src NAT", "Dst NAT", "Hits")
	} else if width >= 100 {
		return fmt.Sprintf("%-4s %-5s %-14s %-12s %-14s %-14s %-8s",
			"#", "Base", "Name", "Zones", "Src NAT", "Dst NAT", "Hits")
	}
	return fmt.Sprintf("%-4s %-14s %-12s %-14s %-8s",
		"#", "Name", "Zones", "Src NAT", "Hits")
}

func formatNATRow(r models.NATRule, width int) string {
	base := formatRuleBase(r.RuleBase)
	srcZone := formatZoneCompact(r.SourceZones)
	dstZone := formatZoneCompact(r.DestZones)
	zones := srcZone + "->" + dstZone
	srcNAT := formatSourceNAT(r)
	dstNAT := formatDestNAT(r)
	hits := formatHitCount(r.HitCount)
	lastHit := formatTimeAgo(r.LastHit)

	service := "any"
	if len(r.Services) > 0 && r.Services[0] != "any" {
		service = formatListCompact(r.Services, 14)
	}

	name := r.Name
	if len(r.Tags) > 0 {
		name = name + " *"
	}

	if width >= 150 {
		return fmt.Sprintf("%-4d %-5s %-20s %-16s %-16s %-18s %-18s %-10s %-10s",
			r.Position, base, truncateEllipsis(name, 20),
			truncateEllipsis(srcZone, 16), truncateEllipsis(dstZone, 16),
			truncateEllipsis(srcNAT, 18), truncateEllipsis(dstNAT, 18),
			hits, lastHit)
	} else if width >= 120 {
		return fmt.Sprintf("%-4d %-5s %-18s %-14s %-14s %-16s %-16s %-10s",
			r.Position, base, truncateEllipsis(name, 18),
			truncateEllipsis(zones, 14), truncateEllipsis(service, 14),
			truncateEllipsis(srcNAT, 16), truncateEllipsis(dstNAT, 16), hits)
	} else if width >= 100 {
		return fmt.Sprintf("%-4d %-5s %-14s %-12s %-14s %-14s %-8s",
			r.Position, base, truncateEllipsis(name, 14),
			truncateEllipsis(zones, 12),
			truncateEllipsis(srcNAT, 14), truncateEllipsis(dstNAT, 14), hits)
	}
	return fmt.Sprintf("%-4d %-14s %-12s %-14s %-8s",
		r.Position, truncateEllipsis(name, 14),
		truncateEllipsis(zones, 12),
		truncateEllipsis(srcNAT, 14), hits)
}

func formatSourceNAT(r models.NATRule) string {
	switch r.SourceTransType {
	case models.SourceTransDynamicIPPort:
		if r.SourceInterfaceIP {
			return "DIPP:" + r.TranslatedSource
		}
		return "DIPP:" + truncateEllipsis(r.TranslatedSource, 12)
	case models.SourceTransDynamicIP:
		return "DIP:" + truncateEllipsis(r.TranslatedSource, 13)
	case models.SourceTransStaticIP:
		return "Static:" + truncateEllipsis(r.TranslatedSource, 10)
	default:
		return "None"
	}
}

func formatDestNAT(r models.NATRule) string {
	if r.TranslatedDest == "" {
		return "None"
	}
	result := truncateEllipsis(r.TranslatedDest, 14)
	if r.TranslatedDestPort != "" {
		result += ":" + r.TranslatedDestPort
	}
	return result
}

func renderNATDetail(r models.NATRule, width int) string {
	dr := NewDetailRenderer(width, 18)

	title := r.Name
	if r.Disabled {
		title += DetailDimStyle.Render(" (disabled)")
	}
	dr.Title(title)
	dr.Subtitle(fmt.Sprintf("Position: %d | %s", r.Position, formatRuleBaseFull(r.RuleBase)))
	dr.Tags(r.Tags)
	dr.Description(r.Description)

	dr.Section("Original Packet (Match)")
	dr.Field("Source Zones:", formatListFull(r.SourceZones))
	dr.Field("Source Addresses:", formatListFull(r.Sources))
	dr.Field("Dest Zones:", formatListFull(r.DestZones))
	dr.Field("Dest Addresses:", formatListFull(r.Destinations))
	dr.Field("Service:", formatListFull(r.Services))
	if r.DestInterface != "" && r.DestInterface != "any" {
		dr.Field("Dest Interface:", r.DestInterface)
	}

	dr.Section("Translated Packet")

	switch r.SourceTransType {
	case models.SourceTransDynamicIPPort:
		transType := "Dynamic IP and Port"
		if r.SourceInterfaceIP {
			transType += " (Interface)"
		}
		dr.Field("Source Translation:", transType)
		dr.Field("  Translated To:", r.TranslatedSource)
	case models.SourceTransDynamicIP:
		dr.Field("Source Translation:", "Dynamic IP")
		dr.Field("  Translated To:", r.TranslatedSource)
	case models.SourceTransStaticIP:
		dr.Field("Source Translation:", "Static IP")
		dr.Field("  Translated To:", r.TranslatedSource)
	default:
		dr.FieldDim("Source Translation:", "None")
	}

	if r.TranslatedDest != "" {
		dr.Field("Dest Translation:", r.TranslatedDest)
		dr.FieldIf("  Translated Port:", r.TranslatedDestPort)
	} else {
		dr.FieldDim("Dest Translation:", "None")
	}

	dr.Section("Usage Statistics")
	dr.Field("Hit Count:", formatNumberWithCommas(r.HitCount))
	dr.Field("Last Hit:", formatTimestamp(r.LastHit))
	if !r.FirstHit.IsZero() {
		dr.Field("First Hit:", formatTimestamp(r.FirstHit))
	}

	return dr.Render()
}
