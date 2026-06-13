package api

import (
	"bytes"
	"context"
	"fmt"
	"log"
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
	if unmarshalErr := decodeXML(bytes.NewReader(WrapInner(inner)), &withWrapper); unmarshalErr == nil && len(withWrapper.Entry) > 0 {
		entries = withWrapper.Entry
	} else {
		// Try parsing without wrapper (entries directly in result)
		var withoutWrapper struct {
			Entry []securityRuleEntry `xml:"entry"`
		}
		if decodeXML(bytes.NewReader(WrapInner(inner)), &withoutWrapper) == nil {
			entries = withoutWrapper.Entry
		}
	}

	return entries
}

// convertSecurityRuleEntry converts a parsed XML entry to a SecurityRule model
func convertSecurityRuleEntry(e securityRuleEntry, position int, ruleBase models.RuleBase) models.SecurityRule {
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
		RuleBase:             ruleBase,
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

// hitStats holds parsed hit count statistics for a rule.
type hitStats struct {
	count     int64
	lastHit   time.Time
	firstHit  time.Time
	lastReset time.Time
}

// parseUnixTimestamp parses a string unix timestamp into a time.Time.
// Returns zero time for empty strings, "0", or unparseable values.
func parseUnixTimestamp(s string) time.Time {
	if s == "" || s == "0" {
		return time.Time{}
	}
	if ts, _ := strconv.ParseInt(s, 10, 64); ts > 0 { //nolint:errcheck // intentional - default to zero time on parse error
		return time.Unix(ts, 0)
	}
	return time.Time{}
}

// parseRuleHitCounts parses the XML response from a rule hit count op command
// and returns a map of rule name to hit statistics.
func parseRuleHitCounts(inner []byte) map[string]hitStats {
	var hitResult struct {
		Entry []struct {
			Name      string `xml:"name,attr"`
			HitCount  int64  `xml:"hit-count"`
			LastHit   string `xml:"last-hit-timestamp"`
			FirstHit  string `xml:"first-hit-timestamp"`
			LastReset string `xml:"last-reset-timestamp"`
		} `xml:"rule-hit-count>vsys>entry>rule-base>entry>rules>entry"`
	}
	if decodeXML(bytes.NewReader(WrapInner(inner)), &hitResult) != nil {
		return nil
	}

	hitMap := make(map[string]hitStats, len(hitResult.Entry))
	for _, h := range hitResult.Entry {
		hitMap[h.Name] = hitStats{
			count:     h.HitCount,
			lastHit:   parseUnixTimestamp(h.LastHit),
			firstHit:  parseUnixTimestamp(h.FirstHit),
			lastReset: parseUnixTimestamp(h.LastReset),
		}
	}
	return hitMap
}

// fetchRulesFromPaths tries to fetch rules from multiple XPaths, using the provided parse function.
func fetchRulesFromPaths[T any](c *Client, ctx context.Context, xpaths []string, target string, parse func([]byte) []T) []T {
	for _, xpath := range xpaths {
		resp, err := c.Show(ctx, xpath, target)
		if err == nil && resp.IsSuccess() && len(resp.Result.Inner) > 0 {
			if entries := parse(resp.Result.Inner); len(entries) > 0 {
				return entries
			}
		}
		// Try Get if Show didn't work
		resp, err = c.Get(ctx, xpath, target)
		if err == nil && resp.IsSuccess() && len(resp.Result.Inner) > 0 {
			if entries := parse(resp.Result.Inner); len(entries) > 0 {
				return entries
			}
		}
	}
	return nil
}

// rulebasePaths returns the candidate XPaths for one rulebase location
// ("pre-rulebase", "rulebase", or "post-rulebase") of the given policy kind
// ("security" or "nat"). The plain local rulebase has an extra vsys-less
// fallback; pre/post instead have the Panorama-pushed path.
func rulebasePaths(location, kind string) []string {
	if location == "rulebase" {
		return []string{
			fmt.Sprintf("/config/devices/entry[@name='localhost.localdomain']/vsys/entry[@name='vsys1']/rulebase/%s/rules", kind),
			fmt.Sprintf("/config/devices/entry/vsys/entry[@name='vsys1']/rulebase/%s/rules", kind),
			fmt.Sprintf("/config/devices/entry/vsys/entry/rulebase/%s/rules", kind),
		}
	}
	return []string{
		fmt.Sprintf("/config/devices/entry[@name='localhost.localdomain']/vsys/entry[@name='vsys1']/%s/%s/rules", location, kind),
		fmt.Sprintf("/config/devices/entry/vsys/entry[@name='vsys1']/%s/%s/rules", location, kind),
		fmt.Sprintf("/config/panorama/vsys/entry[@name='vsys1']/%s/%s/rules", location, kind),
	}
}

// rulebaseSpec parameterizes fetchRulebase over a policy kind.
type rulebaseSpec[TEntry, TModel any] struct {
	// kind is the xpath segment and hit-count rule-base name: "security" or
	// "nat". Compile-time constant — never user input.
	kind     string
	parse    func([]byte) []TEntry
	convert  func(TEntry, int, models.RuleBase) TModel
	ruleName func(TModel) string
	applyHit func(*TModel, hitStats)
}

// fetchRulebase retrieves a complete policy rulebase (pre, local, post — in
// evaluation order, with 1-based positions), then best-effort decorates the
// rules with hit-count statistics. Hit-count failures are logged warnings,
// never errors: the rules themselves are still useful without stats.
func fetchRulebase[TEntry, TModel any](c *Client, ctx context.Context, target string, spec rulebaseSpec[TEntry, TModel]) ([]TModel, error) {
	pre := fetchRulesFromPaths(c, ctx, rulebasePaths("pre-rulebase", spec.kind), target, spec.parse)
	local := fetchRulesFromPaths(c, ctx, rulebasePaths("rulebase", spec.kind), target, spec.parse)
	post := fetchRulesFromPaths(c, ctx, rulebasePaths("post-rulebase", spec.kind), target, spec.parse)

	rules := make([]TModel, 0, len(pre)+len(local)+len(post))
	position := 1
	for _, group := range []struct {
		entries []TEntry
		base    models.RuleBase
	}{
		{pre, models.RuleBasePre},
		{local, models.RuleBaseLocal},
		{post, models.RuleBasePost},
	} {
		for _, e := range group.entries {
			rules = append(rules, spec.convert(e, position, group.base))
			position++
		}
	}
	if len(rules) == 0 {
		return []TModel{}, nil
	}

	cmd := fmt.Sprintf("<show><rule-hit-count><vsys><vsys-name><entry name='vsys1'><rule-base><entry name='%s'><rules><all/></rules></entry></rule-base></entry></vsys-name></vsys></rule-hit-count></show>", spec.kind)
	hitCountResp, err := c.Op(ctx, cmd, target)
	switch {
	case err != nil:
		log.Printf("[API Warning] failed to fetch %s rule hit counts: %v", spec.kind, err)
	case !hitCountResp.IsSuccess():
		log.Printf("[API Warning] %s rule hit count request returned non-success: %s", spec.kind, hitCountResp.Error())
	default:
		if hitMap := parseRuleHitCounts(hitCountResp.Result.Inner); hitMap != nil {
			for i := range rules {
				if hit, ok := hitMap[spec.ruleName(rules[i])]; ok {
					spec.applyHit(&rules[i], hit)
				}
			}
		}
	}

	return rules, nil
}

func (c *Client) GetSecurityPolicies(ctx context.Context, target string) ([]models.SecurityRule, error) {
	return fetchRulebase(c, ctx, target, rulebaseSpec[securityRuleEntry, models.SecurityRule]{
		kind:     "security",
		parse:    parseSecurityRuleEntries,
		convert:  convertSecurityRuleEntry,
		ruleName: func(r models.SecurityRule) string { return r.Name },
		applyHit: func(r *models.SecurityRule, h hitStats) {
			r.HitCount, r.LastHit, r.FirstHit, r.LastReset = h.count, h.lastHit, h.firstHit, h.lastReset
		},
	})
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
	if unmarshalErr := decodeXML(bytes.NewReader(WrapInner(inner)), &withWrapper); unmarshalErr == nil && len(withWrapper.Entry) > 0 {
		entries = withWrapper.Entry
	} else {
		// Try parsing without wrapper
		var withoutWrapper struct {
			Entry []natRuleEntry `xml:"entry"`
		}
		if decodeXML(bytes.NewReader(WrapInner(inner)), &withoutWrapper) == nil {
			entries = withoutWrapper.Entry
		}
	}

	return entries
}

// convertNATRuleEntry converts a parsed XML entry to a NATRule model
func convertNATRuleEntry(e natRuleEntry, position int, ruleBase models.RuleBase) models.NATRule {
	rule := models.NATRule{
		Name:          e.Name,
		Position:      position,
		Disabled:      e.Disabled == "yes",
		Description:   e.Description,
		Tags:          e.Tag.Member,
		RuleBase:      ruleBase,
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

// GetNATRules retrieves NAT policy rules from the firewall
func (c *Client) GetNATRules(ctx context.Context, target string) ([]models.NATRule, error) {
	return fetchRulebase(c, ctx, target, rulebaseSpec[natRuleEntry, models.NATRule]{
		kind:     "nat",
		parse:    parseNATRuleEntries,
		convert:  convertNATRuleEntry,
		ruleName: func(r models.NATRule) string { return r.Name },
		applyHit: func(r *models.NATRule, h hitStats) {
			r.HitCount, r.LastHit, r.FirstHit, r.LastReset = h.count, h.lastHit, h.firstHit, h.lastReset
		},
	})
}
