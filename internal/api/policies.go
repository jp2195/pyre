package api

import (
	"context"
	"encoding/xml"
	"strconv"
	"strings"
	"time"

	"github.com/jp2195/pyre/internal/models"
)

// securityRuleEntry defines the XML structure for security rule parsing
type securityRuleEntry struct {
	Name        string `xml:"name,attr"`
	Disabled    string `xml:"disabled"`
	Description string `xml:"description"`
	RuleType    string `xml:"rule-type"`
	Action      string `xml:"action"`
	Tag         struct {
		Member []string `xml:"member"`
	} `xml:"tag"`
	From struct {
		Member []string `xml:"member"`
	} `xml:"from"`
	To struct {
		Member []string `xml:"member"`
	} `xml:"to"`
	Source struct {
		Member []string `xml:"member"`
	} `xml:"source"`
	SourceUser struct {
		Member []string `xml:"member"`
	} `xml:"source-user"`
	NegateSource string `xml:"negate-source"`
	Destination  struct {
		Member []string `xml:"member"`
	} `xml:"destination"`
	NegateDest  string `xml:"negate-destination"`
	Application struct {
		Member []string `xml:"member"`
	} `xml:"application"`
	Service struct {
		Member []string `xml:"member"`
	} `xml:"service"`
	Category struct {
		Member []string `xml:"member"`
	} `xml:"category"`
	LogStart       string `xml:"log-start"`
	LogEnd         string `xml:"log-end"`
	LogSetting     string `xml:"log-setting"`
	ProfileSetting struct {
		Group struct {
			Member []string `xml:"member"`
		} `xml:"group"`
		Profiles struct {
			Virus struct {
				Member []string `xml:"member"`
			} `xml:"virus"`
			Spyware struct {
				Member []string `xml:"member"`
			} `xml:"spyware"`
			Vulnerability struct {
				Member []string `xml:"member"`
			} `xml:"vulnerability"`
			URLFiltering struct {
				Member []string `xml:"member"`
			} `xml:"url-filtering"`
			FileBlocking struct {
				Member []string `xml:"member"`
			} `xml:"file-blocking"`
			WildFireAnalysis struct {
				Member []string `xml:"member"`
			} `xml:"wildfire-analysis"`
		} `xml:"profiles"`
	} `xml:"profile-setting"`
}

// parseSecurityRuleEntries parses XML response into security rule entries
func parseSecurityRuleEntries(inner []byte) []securityRuleEntry {
	var entries []securityRuleEntry

	// Try parsing with <rules> wrapper (xpath ends at /rules, so response includes the element)
	var withWrapper struct {
		Entry []securityRuleEntry `xml:"rules>entry"`
	}
	if unmarshalErr := xml.Unmarshal(WrapInner(inner), &withWrapper); unmarshalErr == nil && len(withWrapper.Entry) > 0 {
		entries = withWrapper.Entry
	} else {
		// Try parsing without wrapper (entries directly in result)
		var withoutWrapper struct {
			Entry []securityRuleEntry `xml:"entry"`
		}
		if xml.Unmarshal(WrapInner(inner), &withoutWrapper) == nil {
			entries = withoutWrapper.Entry
		}
	}

	return entries
}

// convertSecurityRuleEntry converts a parsed XML entry to a SecurityRule model
func convertSecurityRuleEntry(e securityRuleEntry, position int) models.SecurityRule {
	// Determine rule type
	ruleType := models.RuleTypeUniversal
	switch e.RuleType {
	case "intrazone":
		ruleType = models.RuleTypeIntrazone
	case "interzone":
		ruleType = models.RuleTypeInterzone
	}

	// Get profile group or individual profiles
	var profileGroup string
	var profileType string
	if len(e.ProfileSetting.Group.Member) > 0 {
		profileGroup = e.ProfileSetting.Group.Member[0]
		profileType = "group"
	} else {
		profileType = "profiles"
	}

	// Extract individual profiles
	var avProfile, vulnProfile, spyProfile, urlProfile, fbProfile, wfProfile string
	if len(e.ProfileSetting.Profiles.Virus.Member) > 0 {
		avProfile = e.ProfileSetting.Profiles.Virus.Member[0]
	}
	if len(e.ProfileSetting.Profiles.Vulnerability.Member) > 0 {
		vulnProfile = e.ProfileSetting.Profiles.Vulnerability.Member[0]
	}
	if len(e.ProfileSetting.Profiles.Spyware.Member) > 0 {
		spyProfile = e.ProfileSetting.Profiles.Spyware.Member[0]
	}
	if len(e.ProfileSetting.Profiles.URLFiltering.Member) > 0 {
		urlProfile = e.ProfileSetting.Profiles.URLFiltering.Member[0]
	}
	if len(e.ProfileSetting.Profiles.FileBlocking.Member) > 0 {
		fbProfile = e.ProfileSetting.Profiles.FileBlocking.Member[0]
	}
	if len(e.ProfileSetting.Profiles.WildFireAnalysis.Member) > 0 {
		wfProfile = e.ProfileSetting.Profiles.WildFireAnalysis.Member[0]
	}

	return models.SecurityRule{
		Name:                 e.Name,
		Position:             position,
		Disabled:             e.Disabled == "yes",
		Description:          e.Description,
		Tags:                 e.Tag.Member,
		RuleType:             ruleType,
		Action:               e.Action,
		SourceZones:          e.From.Member,
		Sources:              e.Source.Member,
		SourceUsers:          e.SourceUser.Member,
		NegateSource:         e.NegateSource == "yes",
		DestZones:            e.To.Member,
		Destinations:         e.Destination.Member,
		NegateDest:           e.NegateDest == "yes",
		Applications:         e.Application.Member,
		Services:             e.Service.Member,
		URLCategories:        e.Category.Member,
		Profile:              profileGroup,
		ProfileType:          profileType,
		AntivirusProfile:     avProfile,
		VulnerabilityProfile: vulnProfile,
		SpywareProfile:       spyProfile,
		URLFilteringProfile:  urlProfile,
		FileBlockingProfile:  fbProfile,
		WildFireProfile:      wfProfile,
		LogStart:             e.LogStart == "yes",
		LogEnd:               e.LogEnd == "yes",
		LogForwarding:        e.LogSetting,
	}
}

// fetchSecurityRulesFromPaths tries to fetch security rules from multiple XPaths
func (c *Client) fetchSecurityRulesFromPaths(ctx context.Context, xpaths []string) []securityRuleEntry {
	for _, xpath := range xpaths {
		resp, err := c.Show(ctx, xpath)
		if err == nil && resp.IsSuccess() && len(resp.Result.Inner) > 0 {
			if entries := parseSecurityRuleEntries(resp.Result.Inner); len(entries) > 0 {
				return entries
			}
		}
		// Try Get if Show didn't work
		resp, err = c.Get(ctx, xpath)
		if err == nil && resp.IsSuccess() && len(resp.Result.Inner) > 0 {
			if entries := parseSecurityRuleEntries(resp.Result.Inner); len(entries) > 0 {
				return entries
			}
		}
	}
	return nil
}

func (c *Client) GetSecurityPolicies(ctx context.Context) ([]models.SecurityRule, error) {
	// Pre-rulebase paths (Panorama-pushed rules evaluated first)
	preRulebasePaths := []string{
		"/config/devices/entry[@name='localhost.localdomain']/vsys/entry[@name='vsys1']/pre-rulebase/security/rules",
		"/config/devices/entry/vsys/entry[@name='vsys1']/pre-rulebase/security/rules",
	}

	// Local rulebase paths (locally defined rules)
	localRulebasePaths := []string{
		"/config/devices/entry[@name='localhost.localdomain']/vsys/entry[@name='vsys1']/rulebase/security/rules",
		"/config/devices/entry/vsys/entry[@name='vsys1']/rulebase/security/rules",
		"/config/devices/entry/vsys/entry/rulebase/security/rules",
	}

	// Post-rulebase paths (Panorama-pushed rules evaluated last)
	postRulebasePaths := []string{
		"/config/devices/entry[@name='localhost.localdomain']/vsys/entry[@name='vsys1']/post-rulebase/security/rules",
		"/config/devices/entry/vsys/entry[@name='vsys1']/post-rulebase/security/rules",
	}

	// Fetch from all three rulebase locations
	preEntries := c.fetchSecurityRulesFromPaths(ctx, preRulebasePaths)
	localEntries := c.fetchSecurityRulesFromPaths(ctx, localRulebasePaths)
	postEntries := c.fetchSecurityRulesFromPaths(ctx, postRulebasePaths)

	// Combine in evaluation order: pre -> local -> post
	totalEntries := len(preEntries) + len(localEntries) + len(postEntries)
	rules := make([]models.SecurityRule, 0, totalEntries)
	position := 1

	for _, e := range preEntries {
		rules = append(rules, convertSecurityRuleEntry(e, position))
		position++
	}
	for _, e := range localEntries {
		rules = append(rules, convertSecurityRuleEntry(e, position))
		position++
	}
	for _, e := range postEntries {
		rules = append(rules, convertSecurityRuleEntry(e, position))
		position++
	}

	if len(rules) == 0 {
		return []models.SecurityRule{}, nil
	}

	// Fetch rule hit counts with extended stats
	hitCountResp, err := c.Op(ctx, "<show><rule-hit-count><vsys><vsys-name><entry name='vsys1'><rule-base><entry name='security'><rules><all/></rules></entry></rule-base></entry></vsys-name></vsys></rule-hit-count></show>")
	if err == nil && hitCountResp.IsSuccess() {
		var hitResult struct {
			Entry []struct {
				Name      string `xml:"name,attr"`
				HitCount  int64  `xml:"hit-count"`
				LastHit   string `xml:"last-hit-timestamp"`
				FirstHit  string `xml:"first-hit-timestamp"`
				LastReset string `xml:"last-reset-timestamp"`
			} `xml:"rule-hit-count>vsys>entry>rule-base>entry>rules>entry"`
		}
		if xml.Unmarshal(hitCountResp.Result.Inner, &hitResult) == nil {
			type hitStats struct {
				count     int64
				lastHit   time.Time
				firstHit  time.Time
				lastReset time.Time
			}
			hitMap := make(map[string]hitStats)
			for _, h := range hitResult.Entry {
				stats := hitStats{count: h.HitCount}
				if h.LastHit != "" && h.LastHit != "0" {
					if ts, _ := strconv.ParseInt(h.LastHit, 10, 64); ts > 0 { //nolint:errcheck // intentional - default to zero time on parse error
						stats.lastHit = time.Unix(ts, 0)
					}
				}
				if h.FirstHit != "" && h.FirstHit != "0" {
					if ts, _ := strconv.ParseInt(h.FirstHit, 10, 64); ts > 0 { //nolint:errcheck // intentional - default to zero time on parse error
						stats.firstHit = time.Unix(ts, 0)
					}
				}
				if h.LastReset != "" && h.LastReset != "0" {
					if ts, _ := strconv.ParseInt(h.LastReset, 10, 64); ts > 0 { //nolint:errcheck // intentional - default to zero time on parse error
						stats.lastReset = time.Unix(ts, 0)
					}
				}
				hitMap[h.Name] = stats
			}
			for i := range rules {
				if hit, ok := hitMap[rules[i].Name]; ok {
					rules[i].HitCount = hit.count
					rules[i].LastHit = hit.lastHit
					rules[i].FirstHit = hit.firstHit
					rules[i].LastReset = hit.lastReset
				}
			}
		}
	}

	return rules, nil
}

// natRuleEntry defines the XML structure for NAT rule parsing
type natRuleEntry struct {
	Name        string `xml:"name,attr"`
	Disabled    string `xml:"disabled"`
	Description string `xml:"description"`
	NATType     string `xml:"nat-type"`
	Tag         struct {
		Member []string `xml:"member"`
	} `xml:"tag"`
	From struct {
		Member []string `xml:"member"`
	} `xml:"from"`
	To struct {
		Member []string `xml:"member"`
	} `xml:"to"`
	Source struct {
		Member []string `xml:"member"`
	} `xml:"source"`
	Destination struct {
		Member []string `xml:"member"`
	} `xml:"destination"`
	Service           string `xml:"service"`
	ToInterface       string `xml:"to-interface"`
	SourceTranslation struct {
		DynamicIPAndPort struct {
			InterfaceAddress struct {
				Interface string `xml:"interface"`
				IP        string `xml:"ip"`
			} `xml:"interface-address"`
			TranslatedAddress struct {
				Member []string `xml:"member"`
			} `xml:"translated-address"`
		} `xml:"dynamic-ip-and-port"`
		DynamicIP struct {
			TranslatedAddress struct {
				Member []string `xml:"member"`
			} `xml:"translated-address"`
			Fallback struct {
				Interface struct {
					Interface string `xml:"interface"`
					IP        string `xml:"ip"`
				} `xml:"interface-address"`
			} `xml:"fallback"`
		} `xml:"dynamic-ip"`
		StaticIP struct {
			TranslatedAddress string `xml:"translated-address"`
			BiDirectional     string `xml:"bi-directional"`
		} `xml:"static-ip"`
	} `xml:"source-translation"`
	DestinationTranslation struct {
		TranslatedAddress string `xml:"translated-address"`
		TranslatedPort    string `xml:"translated-port"`
	} `xml:"destination-translation"`
	ActiveActiveDeviceBinding string `xml:"active-active-device-binding"`
}

// parseNATRuleEntries parses XML response into NAT rule entries
func parseNATRuleEntries(inner []byte) []natRuleEntry {
	var entries []natRuleEntry

	// Try parsing with <rules> wrapper
	var withWrapper struct {
		Entry []natRuleEntry `xml:"rules>entry"`
	}
	if unmarshalErr := xml.Unmarshal(WrapInner(inner), &withWrapper); unmarshalErr == nil && len(withWrapper.Entry) > 0 {
		entries = withWrapper.Entry
	} else {
		// Try parsing without wrapper
		var withoutWrapper struct {
			Entry []natRuleEntry `xml:"entry"`
		}
		if xml.Unmarshal(WrapInner(inner), &withoutWrapper) == nil {
			entries = withoutWrapper.Entry
		}
	}

	return entries
}

// convertNATRuleEntry converts a parsed XML entry to a NATRule model
func convertNATRuleEntry(e natRuleEntry, position int) models.NATRule {
	rule := models.NATRule{
		Name:          e.Name,
		Position:      position,
		Disabled:      e.Disabled == "yes",
		Description:   e.Description,
		Tags:          e.Tag.Member,
		SourceZones:   e.From.Member,
		DestZones:     e.To.Member,
		Sources:       e.Source.Member,
		Destinations:  e.Destination.Member,
		DestInterface: e.ToInterface,
		NATType:       e.NATType,
		ActiveActive:  e.ActiveActiveDeviceBinding != "",
	}

	// Handle service - can be a single value
	if e.Service != "" {
		rule.Services = []string{e.Service}
	}

	// Determine source translation type
	if e.SourceTranslation.DynamicIPAndPort.InterfaceAddress.Interface != "" {
		rule.SourceTransType = models.SourceTransDynamicIPPort
		rule.TranslatedSource = e.SourceTranslation.DynamicIPAndPort.InterfaceAddress.Interface
		rule.SourceInterfaceIP = true
	} else if len(e.SourceTranslation.DynamicIPAndPort.TranslatedAddress.Member) > 0 {
		rule.SourceTransType = models.SourceTransDynamicIPPort
		rule.TranslatedSource = strings.Join(e.SourceTranslation.DynamicIPAndPort.TranslatedAddress.Member, ", ")
	} else if len(e.SourceTranslation.DynamicIP.TranslatedAddress.Member) > 0 {
		rule.SourceTransType = models.SourceTransDynamicIP
		rule.TranslatedSource = strings.Join(e.SourceTranslation.DynamicIP.TranslatedAddress.Member, ", ")
	} else if e.SourceTranslation.StaticIP.TranslatedAddress != "" {
		rule.SourceTransType = models.SourceTransStaticIP
		rule.TranslatedSource = e.SourceTranslation.StaticIP.TranslatedAddress
	} else {
		rule.SourceTransType = models.SourceTransNone
	}

	// Destination translation
	rule.TranslatedDest = e.DestinationTranslation.TranslatedAddress
	rule.TranslatedDestPort = e.DestinationTranslation.TranslatedPort

	return rule
}

// fetchNATRulesFromPaths tries to fetch NAT rules from multiple XPaths
func (c *Client) fetchNATRulesFromPaths(ctx context.Context, xpaths []string) []natRuleEntry {
	for _, xpath := range xpaths {
		resp, err := c.Show(ctx, xpath)
		if err == nil && resp.IsSuccess() && len(resp.Result.Inner) > 0 {
			if entries := parseNATRuleEntries(resp.Result.Inner); len(entries) > 0 {
				return entries
			}
		}
		// Try Get if Show didn't work
		resp, err = c.Get(ctx, xpath)
		if err == nil && resp.IsSuccess() && len(resp.Result.Inner) > 0 {
			if entries := parseNATRuleEntries(resp.Result.Inner); len(entries) > 0 {
				return entries
			}
		}
	}
	return nil
}

// GetNATRules retrieves NAT policy rules from the firewall
func (c *Client) GetNATRules(ctx context.Context) ([]models.NATRule, error) {
	// Pre-rulebase paths (Panorama-pushed rules evaluated first)
	preRulebasePaths := []string{
		"/config/devices/entry[@name='localhost.localdomain']/vsys/entry[@name='vsys1']/pre-rulebase/nat/rules",
		"/config/devices/entry/vsys/entry[@name='vsys1']/pre-rulebase/nat/rules",
	}

	// Local rulebase paths (locally defined rules)
	localRulebasePaths := []string{
		"/config/devices/entry[@name='localhost.localdomain']/vsys/entry[@name='vsys1']/rulebase/nat/rules",
		"/config/devices/entry/vsys/entry[@name='vsys1']/rulebase/nat/rules",
		"/config/devices/entry/vsys/entry/rulebase/nat/rules",
	}

	// Post-rulebase paths (Panorama-pushed rules evaluated last)
	postRulebasePaths := []string{
		"/config/devices/entry[@name='localhost.localdomain']/vsys/entry[@name='vsys1']/post-rulebase/nat/rules",
		"/config/devices/entry/vsys/entry[@name='vsys1']/post-rulebase/nat/rules",
	}

	// Fetch from all three rulebase locations
	preEntries := c.fetchNATRulesFromPaths(ctx, preRulebasePaths)
	localEntries := c.fetchNATRulesFromPaths(ctx, localRulebasePaths)
	postEntries := c.fetchNATRulesFromPaths(ctx, postRulebasePaths)

	// Combine in evaluation order: pre -> local -> post
	totalEntries := len(preEntries) + len(localEntries) + len(postEntries)
	rules := make([]models.NATRule, 0, totalEntries)
	position := 1

	for _, e := range preEntries {
		rules = append(rules, convertNATRuleEntry(e, position))
		position++
	}
	for _, e := range localEntries {
		rules = append(rules, convertNATRuleEntry(e, position))
		position++
	}
	for _, e := range postEntries {
		rules = append(rules, convertNATRuleEntry(e, position))
		position++
	}

	if len(rules) == 0 {
		return []models.NATRule{}, nil
	}

	// Fetch NAT rule hit counts
	hitCountResp, err := c.Op(ctx, "<show><rule-hit-count><vsys><vsys-name><entry name='vsys1'><rule-base><entry name='nat'><rules><all/></rules></entry></rule-base></entry></vsys-name></vsys></rule-hit-count></show>")
	if err == nil && hitCountResp.IsSuccess() {
		var hitResult struct {
			Entry []struct {
				Name      string `xml:"name,attr"`
				HitCount  int64  `xml:"hit-count"`
				LastHit   string `xml:"last-hit-timestamp"`
				FirstHit  string `xml:"first-hit-timestamp"`
				LastReset string `xml:"last-reset-timestamp"`
			} `xml:"rule-hit-count>vsys>entry>rule-base>entry>rules>entry"`
		}
		if xml.Unmarshal(hitCountResp.Result.Inner, &hitResult) == nil {
			type hitStats struct {
				count     int64
				lastHit   time.Time
				firstHit  time.Time
				lastReset time.Time
			}
			hitMap := make(map[string]hitStats)
			for _, h := range hitResult.Entry {
				stats := hitStats{count: h.HitCount}
				if h.LastHit != "" && h.LastHit != "0" {
					if ts, _ := strconv.ParseInt(h.LastHit, 10, 64); ts > 0 { //nolint:errcheck // intentional - default to zero time on parse error
						stats.lastHit = time.Unix(ts, 0)
					}
				}
				if h.FirstHit != "" && h.FirstHit != "0" {
					if ts, _ := strconv.ParseInt(h.FirstHit, 10, 64); ts > 0 { //nolint:errcheck // intentional - default to zero time on parse error
						stats.firstHit = time.Unix(ts, 0)
					}
				}
				if h.LastReset != "" && h.LastReset != "0" {
					if ts, _ := strconv.ParseInt(h.LastReset, 10, 64); ts > 0 { //nolint:errcheck // intentional - default to zero time on parse error
						stats.lastReset = time.Unix(ts, 0)
					}
				}
				hitMap[h.Name] = stats
			}
			for i := range rules {
				if hit, ok := hitMap[rules[i].Name]; ok {
					rules[i].HitCount = hit.count
					rules[i].LastHit = hit.lastHit
					rules[i].FirstHit = hit.firstHit
					rules[i].LastReset = hit.lastReset
				}
			}
		}
	}

	return rules, nil
}
