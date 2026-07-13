package api

import (
	"bytes"
	"context"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jp2195/pyre/internal/models"
)

// memberList is the recurring PAN-OS <x><member>...</member></x> shape.
type memberList struct {
	Member []string `xml:"member"`
}

// securityRuleEntry defines the XML structure for security rule parsing
type securityRuleEntry struct {
	Name           string     `xml:"name,attr"`
	Disabled       string     `xml:"disabled"`
	Description    string     `xml:"description"`
	RuleType       string     `xml:"rule-type"`
	Action         string     `xml:"action"`
	Tag            memberList `xml:"tag"`
	From           memberList `xml:"from"`
	To             memberList `xml:"to"`
	Source         memberList `xml:"source"`
	SourceUser     memberList `xml:"source-user"`
	NegateSource   string     `xml:"negate-source"`
	Destination    memberList `xml:"destination"`
	NegateDest     string     `xml:"negate-destination"`
	Application    memberList `xml:"application"`
	Service        memberList `xml:"service"`
	Category       memberList `xml:"category"`
	LogStart       string     `xml:"log-start"`
	LogEnd         string     `xml:"log-end"`
	LogSetting     string     `xml:"log-setting"`
	ProfileSetting struct {
		Group    memberList `xml:"group"`
		Profiles struct {
			Virus            memberList `xml:"virus"`
			Spyware          memberList `xml:"spyware"`
			Vulnerability    memberList `xml:"vulnerability"`
			URLFiltering     memberList `xml:"url-filtering"`
			FileBlocking     memberList `xml:"file-blocking"`
			WildFireAnalysis memberList `xml:"wildfire-analysis"`
		} `xml:"profiles"`
	} `xml:"profile-setting"`
}

// parseRuleEntries parses a rulebase XML response, tolerating both the
// <rules><entry>... wrapper and bare <entry> shapes PAN-OS emits.
func parseRuleEntries[T any](inner []byte) []T {
	var withWrapper struct {
		Entry []T `xml:"rules>entry"`
	}
	if err := decodeXML(bytes.NewReader(WrapInner(inner)), &withWrapper); err == nil && len(withWrapper.Entry) > 0 {
		return withWrapper.Entry
	}
	var withoutWrapper struct {
		Entry []T `xml:"entry"`
	}
	if decodeXML(bytes.NewReader(WrapInner(inner)), &withoutWrapper) == nil {
		return withoutWrapper.Entry
	}
	return nil
}

// rulebasePaths returns the candidate XPaths for the pre/local/post
// rulebases of the given kind ("security" or "nat"). One definition —
// the security and NAT lists previously only differed by this segment.
func rulebasePaths(kind string) (pre, local, post []string) {
	pre = []string{
		// Standalone firewall paths
		"/config/devices/entry[@name='localhost.localdomain']/vsys/entry[@name='vsys1']/pre-rulebase/" + kind + "/rules",
		"/config/devices/entry/vsys/entry[@name='vsys1']/pre-rulebase/" + kind + "/rules",
		// Panorama-pushed rules on managed firewalls
		"/config/panorama/vsys/entry[@name='vsys1']/pre-rulebase/" + kind + "/rules",
	}
	local = []string{
		"/config/devices/entry[@name='localhost.localdomain']/vsys/entry[@name='vsys1']/rulebase/" + kind + "/rules",
		"/config/devices/entry/vsys/entry[@name='vsys1']/rulebase/" + kind + "/rules",
		"/config/devices/entry/vsys/entry/rulebase/" + kind + "/rules",
	}
	post = []string{
		"/config/devices/entry[@name='localhost.localdomain']/vsys/entry[@name='vsys1']/post-rulebase/" + kind + "/rules",
		"/config/devices/entry/vsys/entry[@name='vsys1']/post-rulebase/" + kind + "/rules",
		"/config/panorama/vsys/entry[@name='vsys1']/post-rulebase/" + kind + "/rules",
	}
	return pre, local, post
}

// fetchAllRulebases fetches pre/local/post rulebases concurrently (they are
// independent requests) and converts entries in evaluation order.
func fetchAllRulebases[E, R any](ctx context.Context, c *Client, target, kind string, convert func(E, int, models.RuleBase) R) []R {
	prePaths, localPaths, postPaths := rulebasePaths(kind)

	var preEntries, localEntries, postEntries []E
	var wg sync.WaitGroup
	wg.Go(func() { preEntries = fetchRulesFromPaths(c, ctx, prePaths, target, parseRuleEntries[E]) })
	wg.Go(func() { localEntries = fetchRulesFromPaths(c, ctx, localPaths, target, parseRuleEntries[E]) })
	wg.Go(func() { postEntries = fetchRulesFromPaths(c, ctx, postPaths, target, parseRuleEntries[E]) })
	wg.Wait()

	rules := make([]R, 0, len(preEntries)+len(localEntries)+len(postEntries))
	position := 1
	for _, e := range preEntries {
		rules = append(rules, convert(e, position, models.RuleBasePre))
		position++
	}
	for _, e := range localEntries {
		rules = append(rules, convert(e, position, models.RuleBaseLocal))
		position++
	}
	for _, e := range postEntries {
		rules = append(rules, convert(e, position, models.RuleBasePost))
		position++
	}
	return rules
}

// fetchRuleHitCounts fetches hit-count stats for the named rulebase
// ("security" or "nat"). Failures are logged and return nil — hit counts
// are enrichment, never fatal.
func (c *Client) fetchRuleHitCounts(ctx context.Context, target, ruleBaseName string) map[string]hitStats {
	cmd := "<show><rule-hit-count><vsys><vsys-name><entry name='vsys1'><rule-base><entry name='" + ruleBaseName + "'><rules><all/></rules></entry></rule-base></entry></vsys-name></vsys></rule-hit-count></show>"
	resp, err := c.Op(ctx, cmd, target)
	if err != nil {
		log.Printf("[API Warning] failed to fetch %s rule hit counts: %v", ruleBaseName, err)
		return nil
	}
	if !resp.IsSuccess() {
		log.Printf("[API Warning] %s rule hit count request returned non-success: %s", ruleBaseName, resp.Error())
		return nil
	}
	return parseRuleHitCounts(resp.Result.Inner)
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
	if decodeXML(bytes.NewReader(inner), &hitResult) != nil {
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

func (c *Client) GetSecurityPolicies(ctx context.Context, target string) ([]models.SecurityRule, error) {
	rules := fetchAllRulebases(ctx, c, target, "security", convertSecurityRuleEntry)
	if len(rules) == 0 {
		return []models.SecurityRule{}, nil
	}

	if hitMap := c.fetchRuleHitCounts(ctx, target, "security"); hitMap != nil {
		for i := range rules {
			if hit, ok := hitMap[rules[i].Name]; ok {
				rules[i].HitCount = hit.count
				rules[i].LastHit = hit.lastHit
				rules[i].FirstHit = hit.firstHit
				rules[i].LastReset = hit.lastReset
			}
		}
	}
	return rules, nil
}

// natRuleEntry defines the XML structure for NAT rule parsing
type natRuleEntry struct {
	Name              string     `xml:"name,attr"`
	Disabled          string     `xml:"disabled"`
	Description       string     `xml:"description"`
	NATType           string     `xml:"nat-type"`
	Tag               memberList `xml:"tag"`
	From              memberList `xml:"from"`
	To                memberList `xml:"to"`
	Source            memberList `xml:"source"`
	Destination       memberList `xml:"destination"`
	Service           string     `xml:"service"`
	ToInterface       string     `xml:"to-interface"`
	SourceTranslation struct {
		DynamicIPAndPort struct {
			InterfaceAddress struct {
				Interface string `xml:"interface"`
				IP        string `xml:"ip"`
			} `xml:"interface-address"`
			TranslatedAddress memberList `xml:"translated-address"`
		} `xml:"dynamic-ip-and-port"`
		DynamicIP struct {
			TranslatedAddress memberList `xml:"translated-address"`
			Fallback          struct {
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
	rules := fetchAllRulebases(ctx, c, target, "nat", convertNATRuleEntry)
	if len(rules) == 0 {
		return []models.NATRule{}, nil
	}

	if hitMap := c.fetchRuleHitCounts(ctx, target, "nat"); hitMap != nil {
		for i := range rules {
			if hit, ok := hitMap[rules[i].Name]; ok {
				rules[i].HitCount = hit.count
				rules[i].LastHit = hit.lastHit
				rules[i].FirstHit = hit.firstHit
				rules[i].LastReset = hit.lastReset
			}
		}
	}
	return rules, nil
}
