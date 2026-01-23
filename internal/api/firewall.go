package api

import (
	"context"
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/joshuamontgomery/pyre/internal/models"
)

func (c *Client) GetSystemInfo(ctx context.Context) (*models.SystemInfo, error) {
	resp, err := c.Op(ctx, "<show><system><info></info></system></show>")
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	// Handle empty result
	if len(resp.Result.Inner) == 0 {
		return &models.SystemInfo{}, nil
	}

	// Comprehensive system info parsing
	type sysInfo struct {
		Hostname          string `xml:"hostname"`
		Model             string `xml:"model"`
		Serial            string `xml:"serial"`
		SWVersion         string `xml:"sw-version"`
		Uptime            string `xml:"uptime"`
		DeviceName        string `xml:"devicename"`
		MultiVsys         string `xml:"multi-vsys"`
		OperationalMode   string `xml:"operational-mode"`
		IPAddress         string `xml:"ip-address"`
		Netmask           string `xml:"netmask"`
		DefaultGateway    string `xml:"default-gateway"`
		IPv6Address       string `xml:"ipv6-address"`
		MACAddress        string `xml:"mac-address"`
		Domain            string `xml:"domain"`
		TimeZone          string `xml:"time-zone"`
		Time              string `xml:"time"`
		AppVersion        string `xml:"app-version"`
		AppReleaseDate    string `xml:"app-release-date"`
		ThreatVersion     string `xml:"threat-version"`
		ThreatReleaseDate string `xml:"threat-release-date"`
		AVVersion         string `xml:"av-version"`
		AVReleaseDate     string `xml:"av-release-date"`
		WFVersion         string `xml:"wildfire-version"`
		WFReleaseDate     string `xml:"wildfire-release-date"`
		URLVersion        string `xml:"url-filtering-version"`
	}

	var si sysInfo

	// Try parsing with <system> wrapper first
	var result struct {
		System sysInfo `xml:"system"`
	}
	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err == nil && result.System.Hostname != "" {
		si = result.System
	} else {
		// Try without wrapper
		if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &si); err != nil || si.Hostname == "" {
			return &models.SystemInfo{}, nil
		}
	}

	info := &models.SystemInfo{
		Hostname:            si.Hostname,
		Model:               si.Model,
		Serial:              si.Serial,
		Version:             si.SWVersion,
		Uptime:              si.Uptime,
		IPAddress:           si.IPAddress,
		Netmask:             si.Netmask,
		Gateway:             si.DefaultGateway,
		IPv6Address:         si.IPv6Address,
		MACAddress:          si.MACAddress,
		Domain:              si.Domain,
		TimeZone:            si.TimeZone,
		AppVersion:          si.AppVersion,
		AppReleaseDate:      si.AppReleaseDate,
		ThreatVersion:       si.ThreatVersion,
		ThreatReleaseDate:   si.ThreatReleaseDate,
		AntivirusVersion:    si.AVVersion,
		AntivirusDate:       si.AVReleaseDate,
		WildFireVersion:     si.WFVersion,
		WildFireDate:        si.WFReleaseDate,
		URLFilteringVersion: si.URLVersion,
		MultiVsys:           si.MultiVsys == "on",
		OperationalMode:     si.OperationalMode,
	}

	// Parse current time
	if si.Time != "" {
		layouts := []string{
			"Mon Jan 2 15:04:05 2006",
			"Mon Jan 02 15:04:05 2006",
			"2006-01-02 15:04:05",
		}
		for _, layout := range layouts {
			if t, err := time.Parse(layout, si.Time); err == nil {
				info.CurrentTime = t
				break
			}
		}
	}

	return info, nil
}

// GetLoggedInAdmins returns the list of currently logged in administrators
func (c *Client) GetLoggedInAdmins(ctx context.Context) ([]models.LoggedInAdmin, error) {
	resp, err := c.Op(ctx, "<show><admins></admins></show>")
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	if len(resp.Result.Inner) == 0 {
		return []models.LoggedInAdmin{}, nil
	}

	var result struct {
		Entry []struct {
			Admin    string `xml:"admin"`
			From     string `xml:"from"`
			Type     string `xml:"type"`
			Time     string `xml:"session-start"`
			IdleTime string `xml:"idle-for"`
		} `xml:"admins>entry"`
	}

	// Try multiple parsing approaches
	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil || len(result.Entry) == 0 {
		// Try alternate structure
		var alt struct {
			Entry []struct {
				Admin    string `xml:"admin"`
				From     string `xml:"from"`
				Type     string `xml:"type"`
				Time     string `xml:"session-start"`
				IdleTime string `xml:"idle-for"`
			} `xml:"entry"`
		}
		if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &alt); err == nil {
			result.Entry = alt.Entry
		}
	}

	admins := make([]models.LoggedInAdmin, 0, len(result.Entry))
	for _, e := range result.Entry {
		admin := models.LoggedInAdmin{
			Username: e.Admin,
			From:     e.From,
			Type:     e.Type,
			IdleTime: e.IdleTime,
		}
		// Parse session start time
		if e.Time != "" {
			layouts := []string{
				"01/02/2006 15:04:05",
				"2006-01-02 15:04:05",
				"Mon Jan 2 15:04:05 2006",
			}
			for _, layout := range layouts {
				if t, err := time.Parse(layout, e.Time); err == nil {
					admin.SessionStart = t
					break
				}
			}
		}
		admins = append(admins, admin)
	}

	return admins, nil
}

// Regex patterns for parsing top output
var (
	loadAvgRegex = regexp.MustCompile(`load average:\s*([\d.]+)[,\s]+([\d.]+)[,\s]+([\d.]+)`)
	// Multiple patterns for CPU idle - different PAN-OS versions have different formats
	// Format 1: "91.7 id" or "91.7  id"
	// Format 2: "91.7%id" or "91.7% id"
	cpuIdlePatterns = []*regexp.Regexp{
		regexp.MustCompile(`([\d.]+)\s*%?\s*id[,\s]`),  // "91.7 id," or "91.7%id,"
		regexp.MustCompile(`([\d.]+)\s*%\s*id`),        // "91.7% id"
		regexp.MustCompile(`([\d.]+)\s+id`),            // "91.7 id"
		regexp.MustCompile(`id[,:\s]+([\d.]+)`),        // "id, 91.7" or "id: 91.7"
	}
)

func (c *Client) GetSystemResources(ctx context.Context) (*models.Resources, error) {
	resp, err := c.Op(ctx, "<show><system><resources></resources></system></show>")
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	output := string(resp.Result.Inner)
	resources := &models.Resources{}

	// Parse load average using regex
	if matches := loadAvgRegex.FindStringSubmatch(output); len(matches) >= 4 {
		resources.Load1, _ = strconv.ParseFloat(matches[1], 64)
		resources.Load5, _ = strconv.ParseFloat(matches[2], 64)
		resources.Load15, _ = strconv.ParseFloat(matches[3], 64)
	}

	lines := strings.Split(output, "\n")
	cpuFound := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse CPU - look for Cpu line and extract idle percentage
		if !cpuFound && (strings.HasPrefix(line, "%Cpu") || strings.HasPrefix(line, "Cpu")) {
			// Try each pattern until one matches
			for _, pattern := range cpuIdlePatterns {
				if matches := pattern.FindStringSubmatch(line); len(matches) >= 2 {
					idle, err := strconv.ParseFloat(matches[1], 64)
					if err == nil && idle >= 0 && idle <= 100 {
						resources.CPUPercent = 100 - idle
						cpuFound = true
						break
					}
				}
			}

			// Fallback: try to find "us" (user) percentage and estimate from that
			if !cpuFound {
				usPattern := regexp.MustCompile(`([\d.]+)\s*%?\s*us`)
				if matches := usPattern.FindStringSubmatch(line); len(matches) >= 2 {
					user, err := strconv.ParseFloat(matches[1], 64)
					if err == nil && user >= 0 && user <= 100 {
						// This is just user CPU, actual total is higher but better than 100%
						resources.CPUPercent = user
						cpuFound = true
					}
				}
			}
		}

		// Parse memory - look for lines with memory info
		if strings.HasPrefix(line, "Mem:") || strings.HasPrefix(line, "KiB Mem") || strings.HasPrefix(line, "MiB Mem") {
			var total, used float64
			fields := strings.Fields(line)
			for i, f := range fields {
				// Handle format: "16384000 total," or "total: 16384000"
				cleanField := strings.TrimRight(f, ",%")
				if (cleanField == "total" || f == "total," || f == "total") && i > 0 {
					total, _ = strconv.ParseFloat(strings.TrimRight(fields[i-1], ",%"), 64)
				}
				if (cleanField == "used" || f == "used," || f == "used") && i > 0 {
					used, _ = strconv.ParseFloat(strings.TrimRight(fields[i-1], ",%"), 64)
				}
			}
			if total > 0 {
				resources.MemoryPercent = (used / total) * 100
			}
		}
	}

	return resources, nil
}

func (c *Client) GetSessionInfo(ctx context.Context) (*models.SessionInfo, error) {
	resp, err := c.Op(ctx, "<show><session><info></info></session></show>")
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	// Handle empty result
	if len(resp.Result.Inner) == 0 {
		return &models.SessionInfo{}, nil
	}

	var result struct {
		NumActive  int   `xml:"num-active"`
		NumMax     int   `xml:"num-max"`
		Throughput int64 `xml:"kbps"`
		CPS        int   `xml:"cps"`
		NumTCP     int   `xml:"num-tcp"`
		NumUDP     int   `xml:"num-udp"`
		NumICMP    int   `xml:"num-icmp"`
	}
	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil {
		// Return empty info rather than error
		return &models.SessionInfo{}, nil
	}

	return &models.SessionInfo{
		ActiveCount:    result.NumActive,
		MaxCount:       result.NumMax,
		ThroughputKbps: result.Throughput,
		CPS:            result.CPS,
		TCPSessions:    result.NumTCP,
		UDPSessions:    result.NumUDP,
		ICMPSessions:   result.NumICMP,
	}, nil
}

func (c *Client) GetSessions(ctx context.Context, filter string) ([]models.Session, error) {
	cmd := "<show><session><all></all></session></show>"
	if filter != "" {
		cmd = fmt.Sprintf("<show><session><all><filter><%s/></filter></all></session></show>", filter)
	}

	resp, err := c.Op(ctx, cmd)
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	// Handle empty result (no sessions)
	if len(resp.Result.Inner) == 0 {
		return []models.Session{}, nil
	}

	var result struct {
		Entry []struct {
			ID          int64  `xml:"idx"`
			Vsys        string `xml:"vsys"`
			Application string `xml:"application"`
			State       string `xml:"state"`
			Type        string `xml:"type"`
			SrcIP       string `xml:"source"`
			SrcPort     int    `xml:"sport"`
			DstIP       string `xml:"dst"`
			DstPort     int    `xml:"dport"`
			SrcZone     string `xml:"from"`
			DstZone     string `xml:"to"`
			NatSrcIP    string `xml:"xsource"`
			NatSrcPort  int    `xml:"xsport"`
			Proto       string `xml:"proto"`
			Rule        string `xml:"security-rule"`
			StartTime   string `xml:"start-time"`
			BytesIn     int64  `xml:"total-byte-count"`
		} `xml:"entry"`
	}
	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil {
		// If parsing fails, return empty list rather than error
		return []models.Session{}, nil
	}

	sessions := make([]models.Session, 0, len(result.Entry))
	for _, e := range result.Entry {
		var startTime time.Time
		if e.StartTime != "" {
			startTime, _ = time.Parse("Mon Jan 2 15:04:05 2006", e.StartTime)
		}
		// Convert protocol number to name
		proto := protoToName(e.Proto)
		sessions = append(sessions, models.Session{
			ID:            e.ID,
			State:         e.State,
			Application:   e.Application,
			Protocol:      proto,
			SourceIP:      e.SrcIP,
			SourcePort:    e.SrcPort,
			DestIP:        e.DstIP,
			DestPort:      e.DstPort,
			SourceZone:    e.SrcZone,
			DestZone:      e.DstZone,
			NATSourceIP:   e.NatSrcIP,
			NATSourcePort: e.NatSrcPort,
			BytesIn:       e.BytesIn,
			StartTime:     startTime,
			Rule:          e.Rule,
		})
	}

	return sessions, nil
}

func (c *Client) GetSecurityPolicies(ctx context.Context) ([]models.SecurityRule, error) {
	// Try multiple xpaths - device name varies between configurations
	xpaths := []string{
		"/config/devices/entry[@name='localhost.localdomain']/vsys/entry[@name='vsys1']/rulebase/security/rules",
		"/config/devices/entry/vsys/entry[@name='vsys1']/rulebase/security/rules",
		"/config/devices/entry/vsys/entry/rulebase/security/rules",
	}

	var resp *XMLResponse
	var err error
	for _, xpath := range xpaths {
		resp, err = c.Show(ctx, xpath)
		if err == nil && resp.IsSuccess() && len(resp.Result.Inner) > 0 {
			break
		}
		// Try Get if Show didn't work
		resp, err = c.Get(ctx, xpath)
		if err == nil && resp.IsSuccess() && len(resp.Result.Inner) > 0 {
			break
		}
	}

	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	// Handle empty result
	if len(resp.Result.Inner) == 0 {
		return []models.SecurityRule{}, nil
	}

	// Define the entry structure with all available fields
	type ruleEntry struct {
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
		Destination struct {
			Member []string `xml:"member"`
		} `xml:"destination"`
		NegateDest string `xml:"negate-destination"`
		Application struct {
			Member []string `xml:"member"`
		} `xml:"application"`
		Service struct {
			Member []string `xml:"member"`
		} `xml:"service"`
		Category struct {
			Member []string `xml:"member"`
		} `xml:"category"`
		LogStart      string `xml:"log-start"`
		LogEnd        string `xml:"log-end"`
		LogSetting    string `xml:"log-setting"`
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

	var entries []ruleEntry

	// Try parsing with <rules> wrapper (xpath ends at /rules, so response includes the element)
	var withWrapper struct {
		Entry []ruleEntry `xml:"rules>entry"`
	}
	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &withWrapper); err == nil && len(withWrapper.Entry) > 0 {
		entries = withWrapper.Entry
	} else {
		// Try parsing without wrapper (entries directly in result)
		var withoutWrapper struct {
			Entry []ruleEntry `xml:"entry"`
		}
		if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &withoutWrapper); err == nil {
			entries = withoutWrapper.Entry
		}
	}

	if len(entries) == 0 {
		return []models.SecurityRule{}, nil
	}

	rules := make([]models.SecurityRule, 0, len(entries))
	for i, e := range entries {
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

		rule := models.SecurityRule{
			Name:         e.Name,
			Position:     i + 1,
			Disabled:     e.Disabled == "yes",
			Description:  e.Description,
			Tags:         e.Tag.Member,
			RuleType:     ruleType,
			Action:       e.Action,
			SourceZones:  e.From.Member,
			Sources:      e.Source.Member,
			SourceUsers:  e.SourceUser.Member,
			NegateSource: e.NegateSource == "yes",
			DestZones:    e.To.Member,
			Destinations: e.Destination.Member,
			NegateDest:   e.NegateDest == "yes",
			Applications: e.Application.Member,
			Services:     e.Service.Member,
			URLCategories: e.Category.Member,
			Profile:      profileGroup,
			ProfileType:  profileType,
			AntivirusProfile:     avProfile,
			VulnerabilityProfile: vulnProfile,
			SpywareProfile:       spyProfile,
			URLFilteringProfile:  urlProfile,
			FileBlockingProfile:  fbProfile,
			WildFireProfile:      wfProfile,
			LogStart:      e.LogStart == "yes",
			LogEnd:        e.LogEnd == "yes",
			LogForwarding: e.LogSetting,
		}
		rules = append(rules, rule)
	}

	// Fetch rule hit counts with extended stats
	hitCountResp, err := c.Op(ctx, "<show><rule-hit-count><vsys><vsys-name><entry name='vsys1'><rule-base><entry name='security'><rules><all/></rules></entry></rule-base></entry></vsys-name></vsys></rule-hit-count></show>")
	if err == nil && hitCountResp.IsSuccess() {
		var hitResult struct {
			Entry []struct {
				Name          string `xml:"name,attr"`
				HitCount      int64  `xml:"hit-count"`
				LastHit       string `xml:"last-hit-timestamp"`
				FirstHit      string `xml:"first-hit-timestamp"`
				LastReset     string `xml:"last-reset-timestamp"`
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
					if ts, _ := strconv.ParseInt(h.LastHit, 10, 64); ts > 0 {
						stats.lastHit = time.Unix(ts, 0)
					}
				}
				if h.FirstHit != "" && h.FirstHit != "0" {
					if ts, _ := strconv.ParseInt(h.FirstHit, 10, 64); ts > 0 {
						stats.firstHit = time.Unix(ts, 0)
					}
				}
				if h.LastReset != "" && h.LastReset != "0" {
					if ts, _ := strconv.ParseInt(h.LastReset, 10, 64); ts > 0 {
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

func (c *Client) GetHAStatus(ctx context.Context) (*models.HAStatus, error) {
	resp, err := c.Op(ctx, "<show><high-availability><state></state></high-availability></show>")
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	// Handle empty result
	if len(resp.Result.Inner) == 0 {
		return &models.HAStatus{Enabled: false}, nil
	}

	var result struct {
		Enabled string `xml:"enabled"`
		Group   struct {
			LocalInfo struct {
				State string `xml:"state"`
			} `xml:"local-info"`
			PeerInfo struct {
				State string `xml:"state"`
			} `xml:"peer-info"`
			RunningSyncEnabled string `xml:"running-sync-enabled"`
			RunningSyncState   string `xml:"running-sync"`
		} `xml:"group"`
	}
	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil {
		// Return disabled HA if parsing fails
		return &models.HAStatus{Enabled: false}, nil
	}

	return &models.HAStatus{
		Enabled:   result.Enabled == "yes",
		State:     result.Group.LocalInfo.State,
		PeerState: result.Group.PeerInfo.State,
		SyncState: result.Group.RunningSyncState,
	}, nil
}

func (c *Client) GetInterfaces(ctx context.Context) ([]models.Interface, error) {
	resp, err := c.Op(ctx, "<show><interface>all</interface></show>")
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	// Handle empty result
	if len(resp.Result.Inner) == 0 {
		return []models.Interface{}, nil
	}

	var result struct {
		Ifnet struct {
			Entry []struct {
				Name   string `xml:"name"`
				Zone   string `xml:"zone"`
				Vsys   string `xml:"vsys"`
				IP     string `xml:"ip"`
				State  string `xml:"state"`
				Speed  string `xml:"speed"`
				Fwd    string `xml:"fwd"` // virtual router
				MTU    int    `xml:"mtu"`
				Mode   string `xml:"mode"`
				Tag    int    `xml:"tag"`
			} `xml:"entry"`
		} `xml:"ifnet"`
		HW struct {
			Entry []struct {
				Name   string `xml:"name"`
				State  string `xml:"state"`
				Speed  string `xml:"speed"`
				Duplex string `xml:"duplex"`
				MAC    string `xml:"mac"`
				Mode   string `xml:"mode"`
				Type   string `xml:"type"`
			} `xml:"entry"`
		} `xml:"hw"`
	}
	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil {
		// Return empty list rather than error
		return []models.Interface{}, nil
	}

	// Build HW map for hardware details
	hwMap := make(map[string]struct {
		state  string
		speed  string
		duplex string
		mac    string
		mode   string
		ifType string
	})
	for _, hw := range result.HW.Entry {
		hwMap[hw.Name] = struct {
			state  string
			speed  string
			duplex string
			mac    string
			mode   string
			ifType string
		}{hw.State, hw.Speed, hw.Duplex, hw.MAC, hw.Mode, hw.Type}
	}

	interfaces := make([]models.Interface, 0, len(result.Ifnet.Entry))
	for _, e := range result.Ifnet.Entry {
		iface := models.Interface{
			Name:          e.Name,
			Zone:          e.Zone,
			Vsys:          e.Vsys,
			IP:            e.IP,
			State:         e.State,
			Speed:         e.Speed,
			VirtualRouter: e.Fwd,
			MTU:           e.MTU,
			Mode:          e.Mode,
			Tag:           e.Tag,
		}

		// Determine interface type from name
		switch {
		case strings.HasPrefix(e.Name, "ethernet"):
			iface.Type = "ethernet"
		case strings.HasPrefix(e.Name, "loopback"):
			iface.Type = "loopback"
		case strings.HasPrefix(e.Name, "tunnel"):
			iface.Type = "tunnel"
		case strings.HasPrefix(e.Name, "vlan"):
			iface.Type = "vlan"
		case strings.HasPrefix(e.Name, "ae"):
			iface.Type = "aggregate"
		}

		// Merge hardware details
		if hw, ok := hwMap[e.Name]; ok {
			if iface.State == "" {
				iface.State = hw.state
			}
			if iface.Speed == "" {
				iface.Speed = hw.speed
			}
			iface.Duplex = hw.duplex
			iface.MAC = hw.mac
			if iface.Mode == "" {
				iface.Mode = hw.mode
			}
			if iface.Type == "" && hw.ifType != "" {
				iface.Type = hw.ifType
			}
		}
		interfaces = append(interfaces, iface)
	}

	// Fetch counters separately
	c.fetchInterfaceCounters(ctx, interfaces)

	return interfaces, nil
}

// fetchInterfaceCounters fetches counter data for interfaces
func (c *Client) fetchInterfaceCounters(ctx context.Context, interfaces []models.Interface) {
	resp, err := c.Op(ctx, "<show><counter><interface>all</interface></counter></show>")
	if err != nil || CheckResponse(resp) != nil {
		return
	}

	var result struct {
		Ifnet struct {
			Entry []struct {
				Name     string `xml:"name"`
				Ibytes   int64  `xml:"ibytes"`
				Obytes   int64  `xml:"obytes"`
				Ipackets int64  `xml:"ipackets"`
				Opackets int64  `xml:"opackets"`
				Ierrors  int64  `xml:"ierrors"`
				Idrops   int64  `xml:"idrops"`
				Oerrors  int64  `xml:"oerrors"`
				Odrops   int64  `xml:"odrops"`
			} `xml:"entry"`
		} `xml:"ifnet"`
	}
	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil {
		return
	}

	// Build counter map
	counterMap := make(map[string]struct {
		bytesIn, bytesOut     int64
		packetsIn, packetsOut int64
		errorsIn, errorsOut   int64
		dropsIn, dropsOut     int64
	})
	for _, e := range result.Ifnet.Entry {
		counterMap[e.Name] = struct {
			bytesIn, bytesOut     int64
			packetsIn, packetsOut int64
			errorsIn, errorsOut   int64
			dropsIn, dropsOut     int64
		}{e.Ibytes, e.Obytes, e.Ipackets, e.Opackets, e.Ierrors, e.Oerrors, e.Idrops, e.Odrops}
	}

	// Update interfaces with counters
	for i := range interfaces {
		if c, ok := counterMap[interfaces[i].Name]; ok {
			interfaces[i].BytesIn = c.bytesIn
			interfaces[i].BytesOut = c.bytesOut
			interfaces[i].PacketsIn = c.packetsIn
			interfaces[i].PacketsOut = c.packetsOut
			interfaces[i].ErrorsIn = c.errorsIn
			interfaces[i].ErrorsOut = c.errorsOut
			interfaces[i].DropsIn = c.dropsIn
			interfaces[i].DropsOut = c.dropsOut
		}
	}
}

func (c *Client) GetThreatSummary(ctx context.Context) (*models.ThreatSummary, error) {
	// Query threat logs for the last hour to get a summary
	resp, err := c.Op(ctx, "<show><counter><global><name>flow_threat_*</name></global></counter></show>")
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	summary := &models.ThreatSummary{}

	var result struct {
		Entry []struct {
			Name    string `xml:"name"`
			Value   int64  `xml:"value"`
			Rate    int64  `xml:"rate"`
			Aspect  string `xml:"aspect"`
			Desc    string `xml:"desc"`
			Severity string `xml:"severity"`
		} `xml:"global>counters>entry"`
	}
	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil {
		// Fallback: try to parse simple counter format
		// Return empty summary if parsing fails (device may not have threat prevention)
		return summary, nil
	}

	for _, e := range result.Entry {
		if strings.Contains(e.Name, "threat") || strings.Contains(e.Desc, "threat") {
			summary.TotalThreats += e.Value
			switch strings.ToLower(e.Severity) {
			case "critical":
				summary.CriticalCount += e.Value
			case "high":
				summary.HighCount += e.Value
			case "medium":
				summary.MediumCount += e.Value
			case "low", "informational":
				summary.LowCount += e.Value
			}
			if strings.Contains(e.Name, "block") || strings.Contains(e.Desc, "block") {
				summary.BlockedCount += e.Value
			} else {
				summary.AlertedCount += e.Value
			}
		}
	}

	return summary, nil
}

func (c *Client) GetGlobalProtectInfo(ctx context.Context) (*models.GlobalProtectInfo, error) {
	resp, err := c.Op(ctx, "<show><global-protect-gateway><current-user></current-user></global-protect-gateway></show>")
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	info := &models.GlobalProtectInfo{}

	var result struct {
		Entry []struct {
			Username string `xml:"username"`
			Domain   string `xml:"domain"`
			Computer string `xml:"computer"`
			Client   string `xml:"client"`
			VirtualIP string `xml:"virtual-ip"`
			LoginTime string `xml:"login-time"`
		} `xml:"entry"`
	}
	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil {
		return info, nil
	}

	info.ActiveUsers = len(result.Entry)
	info.TotalUsers = len(result.Entry)

	return info, nil
}

func (c *Client) GetLicenseInfo(ctx context.Context) ([]models.LicenseInfo, error) {
	resp, err := c.Op(ctx, "<request><license><info></info></license></request>")
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	var result struct {
		Entry []struct {
			Feature     string `xml:"feature"`
			Description string `xml:"description"`
			Expires     string `xml:"expires"`
			Expired     string `xml:"expired"`
		} `xml:"licenses>entry"`
	}
	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil {
		return nil, fmt.Errorf("parsing license info: %w", err)
	}

	licenses := make([]models.LicenseInfo, 0, len(result.Entry))
	for _, e := range result.Entry {
		lic := models.LicenseInfo{
			Feature:     e.Feature,
			Description: e.Description,
			Expires:     e.Expires,
			Expired:     e.Expired == "yes",
		}
		// Calculate days left if we have an expiration date
		if e.Expires != "" && e.Expires != "Never" {
			if expTime, err := time.Parse("January 02, 2006", e.Expires); err == nil {
				lic.DaysLeft = int(time.Until(expTime).Hours() / 24)
			}
		}
		licenses = append(licenses, lic)
	}

	return licenses, nil
}

func (c *Client) GetJobs(ctx context.Context) ([]models.Job, error) {
	resp, err := c.Op(ctx, "<show><jobs><all></all></jobs></show>")
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	if len(resp.Result.Inner) == 0 {
		return []models.Job{}, nil
	}

	var result struct {
		Entry []struct {
			ID        int    `xml:"id"`
			Type      string `xml:"type"`
			Status    string `xml:"status"`
			Result    string `xml:"result"`
			Progress  string `xml:"progress"`
			Details   string `xml:"details>line"`
			TEnq      string `xml:"tenq"`      // Time enqueued
			TDeq      string `xml:"tdeq"`      // Time dequeued (started)
			Tfin      string `xml:"tfin"`      // Time finished
			User      string `xml:"user"`
			Stoppable string `xml:"stoppable"`
		} `xml:"job"`
	}

	// Try multiple parsing approaches
	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil || len(result.Entry) == 0 {
		// Try alternate structure with entry wrapper
		var alt struct {
			Entry []struct {
				ID        int    `xml:"id"`
				Type      string `xml:"type"`
				Status    string `xml:"status"`
				Result    string `xml:"result"`
				Progress  string `xml:"progress"`
				Details   string `xml:"details>line"`
				TEnq      string `xml:"tenq"`
				TDeq      string `xml:"tdeq"`
				Tfin      string `xml:"tfin"`
				User      string `xml:"user"`
			} `xml:"entry"`
		}
		if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &alt); err == nil {
			for _, e := range alt.Entry {
				result.Entry = append(result.Entry, struct {
					ID        int    `xml:"id"`
					Type      string `xml:"type"`
					Status    string `xml:"status"`
					Result    string `xml:"result"`
					Progress  string `xml:"progress"`
					Details   string `xml:"details>line"`
					TEnq      string `xml:"tenq"`
					TDeq      string `xml:"tdeq"`
					Tfin      string `xml:"tfin"`
					User      string `xml:"user"`
					Stoppable string `xml:"stoppable"`
				}{
					ID:       e.ID,
					Type:     e.Type,
					Status:   e.Status,
					Result:   e.Result,
					Progress: e.Progress,
					Details:  e.Details,
					TEnq:     e.TEnq,
					TDeq:     e.TDeq,
					Tfin:     e.Tfin,
					User:     e.User,
				})
			}
		}
	}

	jobs := make([]models.Job, 0, len(result.Entry))
	for _, e := range result.Entry {
		job := models.Job{
			ID:      e.ID,
			Type:    e.Type,
			Status:  e.Status,
			Result:  e.Result,
			Message: e.Details,
			User:    e.User,
		}

		// Parse progress
		if e.Progress != "" {
			job.Progress, _ = strconv.Atoi(strings.TrimSuffix(e.Progress, "%"))
		}

		// Parse timestamps - PAN-OS typically uses format like "2024/01/15 10:30:45"
		timeLayouts := []string{
			"2006/01/02 15:04:05",
			"2006-01-02 15:04:05",
			"Mon Jan 2 15:04:05 2006",
		}

		for _, layout := range timeLayouts {
			if e.TDeq != "" {
				if t, err := time.Parse(layout, e.TDeq); err == nil {
					job.StartTime = t
					break
				}
			}
		}
		for _, layout := range timeLayouts {
			if e.Tfin != "" {
				if t, err := time.Parse(layout, e.Tfin); err == nil {
					job.EndTime = t
					break
				}
			}
		}

		jobs = append(jobs, job)
	}

	// Sort by ID descending (most recent first)
	for i := 0; i < len(jobs)-1; i++ {
		for j := i + 1; j < len(jobs); j++ {
			if jobs[j].ID > jobs[i].ID {
				jobs[i], jobs[j] = jobs[j], jobs[i]
			}
		}
	}

	return jobs, nil
}

// protoToName converts IP protocol number to name
func protoToName(proto string) string {
	switch proto {
	case "6":
		return "tcp"
	case "17":
		return "udp"
	case "1":
		return "icmp"
	case "47":
		return "gre"
	case "50":
		return "esp"
	case "51":
		return "ah"
	case "58":
		return "icmp6"
	case "89":
		return "ospf"
	default:
		if proto == "" {
			return "tcp"
		}
		return proto
	}
}

// GetSystemLogs retrieves system logs with optional query filter
// Uses type=log API which returns a job ID, then polls for results
func (c *Client) GetSystemLogs(ctx context.Context, query string, maxLogs int) ([]models.SystemLogEntry, error) {
	if maxLogs <= 0 {
		maxLogs = 100
	}

	// Submit log query - this returns a job ID
	resp, err := c.Log(ctx, "system", maxLogs, query)
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	// Parse job ID from response
	var jobResult struct {
		Job string `xml:"job"`
	}
	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &jobResult); err != nil {
		return nil, fmt.Errorf("parsing job response: %w", err)
	}

	if jobResult.Job == "" {
		return nil, fmt.Errorf("no job ID returned")
	}

	// Poll for results with timeout
	var logs []models.SystemLogEntry
	for i := 0; i < 30; i++ { // Max 30 attempts (15 seconds)
		time.Sleep(500 * time.Millisecond)

		resultResp, err := c.LogGet(ctx, jobResult.Job)
		if err != nil {
			continue
		}
		if err := CheckResponse(resultResp); err != nil {
			continue
		}

		// Check job status
		var statusResult struct {
			Status string `xml:"job>status"`
			Logs   struct {
				Entry []struct {
					Time        string `xml:"time_generated"`
					Type        string `xml:"type"`
					Subtype     string `xml:"subtype"`
					Severity    string `xml:"severity"`
					Description string `xml:"opaque"`
					EventID     string `xml:"eventid"`
					Serial      string `xml:"serial"`
					DeviceName  string `xml:"device_name"`
				} `xml:"entry"`
			} `xml:"log>logs"`
		}
		if err := xml.Unmarshal(WrapInner(resultResp.Result.Inner), &statusResult); err != nil {
			continue
		}

		if statusResult.Status == "FIN" {
			// Parse the log entries
			for _, e := range statusResult.Logs.Entry {
				entry := models.SystemLogEntry{
					Severity:    e.Severity,
					Description: e.Description,
				}
				if e.Subtype != "" {
					entry.Type = fmt.Sprintf("%s/%s", e.Type, e.Subtype)
				} else {
					entry.Type = e.Type
				}
				entry.Time = parseLogTime(e.Time)
				logs = append(logs, entry)
			}
			return logs, nil
		}
	}

	return logs, nil // Return what we have even if incomplete
}

// GetTrafficLogs retrieves traffic logs with optional query filter
func (c *Client) GetTrafficLogs(ctx context.Context, query string, maxLogs int) ([]models.TrafficLogEntry, error) {
	if maxLogs <= 0 {
		maxLogs = 100
	}

	// Submit log query
	resp, err := c.Log(ctx, "traffic", maxLogs, query)
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	var jobResult struct {
		Job string `xml:"job"`
	}
	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &jobResult); err != nil {
		return nil, fmt.Errorf("parsing job response: %w", err)
	}

	if jobResult.Job == "" {
		return nil, fmt.Errorf("no job ID returned")
	}

	// Poll for results
	var logs []models.TrafficLogEntry
	for i := 0; i < 30; i++ {
		time.Sleep(500 * time.Millisecond)

		resultResp, err := c.LogGet(ctx, jobResult.Job)
		if err != nil {
			continue
		}
		if err := CheckResponse(resultResp); err != nil {
			continue
		}

		var statusResult struct {
			Status string `xml:"job>status"`
			Logs   struct {
				Entry []struct {
					Time        string `xml:"time_generated"`
					ReceiveTime string `xml:"receive_time"`
					Serial      string `xml:"serial"`
					Type        string `xml:"type"`
					Subtype     string `xml:"subtype"`
					SrcIP       string `xml:"src"`
					DstIP       string `xml:"dst"`
					SrcPort     int    `xml:"sport"`
					DstPort     int    `xml:"dport"`
					NATSrcIP    string `xml:"natsrc"`
					NATDstIP    string `xml:"natdst"`
					NATSrcPort  int    `xml:"natsport"`
					NATDstPort  int    `xml:"natdport"`
					SrcZone     string `xml:"from"`
					DstZone     string `xml:"to"`
					Rule        string `xml:"rule"`
					App         string `xml:"app"`
					Action      string `xml:"action"`
					Bytes       int64  `xml:"bytes"`
					BytesSent   int64  `xml:"bytes_sent"`
					BytesRecv   int64  `xml:"bytes_received"`
					Packets     int64  `xml:"packets"`
					PktsSent    int64  `xml:"pkts_sent"`
					PktsRecv    int64  `xml:"pkts_received"`
					SessionID   int64  `xml:"sessionid"`
					SessionEnd  string `xml:"session_end_reason"`
					Duration    int64  `xml:"elapsed"`
					User        string `xml:"srcuser"`
					Protocol    string `xml:"proto"`
					Category    string `xml:"category"`
					Vsys        string `xml:"vsys"`
					DeviceName  string `xml:"device_name"`
				} `xml:"entry"`
			} `xml:"log>logs"`
		}
		if err := xml.Unmarshal(WrapInner(resultResp.Result.Inner), &statusResult); err != nil {
			continue
		}

		if statusResult.Status == "FIN" {
			for _, e := range statusResult.Logs.Entry {
				entry := models.TrafficLogEntry{
					Serial:        e.Serial,
					Type:          e.Type,
					Subtype:       e.Subtype,
					SourceIP:      e.SrcIP,
					DestIP:        e.DstIP,
					SourcePort:    e.SrcPort,
					DestPort:      e.DstPort,
					NATSourceIP:   e.NATSrcIP,
					NATDestIP:     e.NATDstIP,
					NATSourcePort: e.NATSrcPort,
					NATDestPort:   e.NATDstPort,
					SourceZone:    e.SrcZone,
					DestZone:      e.DstZone,
					Rule:          e.Rule,
					Application:   e.App,
					Action:        e.Action,
					Bytes:         e.Bytes,
					BytesSent:     e.BytesSent,
					BytesRecv:     e.BytesRecv,
					Packets:       e.Packets,
					PacketsSent:   e.PktsSent,
					PacketsRecv:   e.PktsRecv,
					SessionID:     e.SessionID,
					SessionEnd:    e.SessionEnd,
					Duration:      e.Duration,
					User:          e.User,
					Protocol:      protoToName(e.Protocol),
					Category:      e.Category,
					VirtualSystem: e.Vsys,
					DeviceName:    e.DeviceName,
					Time:          parseLogTime(e.Time),
					ReceiveTime:   parseLogTime(e.ReceiveTime),
				}
				logs = append(logs, entry)
			}
			return logs, nil
		}
	}

	return logs, nil
}

// GetThreatLogs retrieves threat logs with optional query filter
func (c *Client) GetThreatLogs(ctx context.Context, query string, maxLogs int) ([]models.ThreatLogEntry, error) {
	if maxLogs <= 0 {
		maxLogs = 100
	}

	// Submit log query
	resp, err := c.Log(ctx, "threat", maxLogs, query)
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	var jobResult struct {
		Job string `xml:"job"`
	}
	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &jobResult); err != nil {
		return nil, fmt.Errorf("parsing job response: %w", err)
	}

	if jobResult.Job == "" {
		return nil, fmt.Errorf("no job ID returned")
	}

	// Poll for results
	var logs []models.ThreatLogEntry
	for i := 0; i < 30; i++ {
		time.Sleep(500 * time.Millisecond)

		resultResp, err := c.LogGet(ctx, jobResult.Job)
		if err != nil {
			continue
		}
		if err := CheckResponse(resultResp); err != nil {
			continue
		}

		var statusResult struct {
			Status string `xml:"job>status"`
			Logs   struct {
				Entry []struct {
					Time        string `xml:"time_generated"`
					ReceiveTime string `xml:"receive_time"`
					Serial      string `xml:"serial"`
					Type        string `xml:"type"`
					Subtype     string `xml:"subtype"`
					SrcIP       string `xml:"src"`
					DstIP       string `xml:"dst"`
					SrcPort     int    `xml:"sport"`
					DstPort     int    `xml:"dport"`
					NATSrcIP    string `xml:"natsrc"`
					NATDstIP    string `xml:"natdst"`
					NATSrcPort  int    `xml:"natsport"`
					NATDstPort  int    `xml:"natdport"`
					SrcZone     string `xml:"from"`
					DstZone     string `xml:"to"`
					Rule        string `xml:"rule"`
					App         string `xml:"app"`
					Action      string `xml:"action"`
					SessionID   int64  `xml:"sessionid"`
					User        string `xml:"srcuser"`
					ThreatID    int64  `xml:"threatid"`
					ThreatName  string `xml:"threat"`
					ThreatCat   string `xml:"thr_category"`
					Severity    string `xml:"severity"`
					Direction   string `xml:"direction"`
					URL         string `xml:"misc"`
					Filename    string `xml:"filename"`
					FileHash    string `xml:"filedigest"`
					ContentType string `xml:"contenttype"`
					Vsys        string `xml:"vsys"`
					DeviceName  string `xml:"device_name"`
					ReportID    int64  `xml:"reportid"`
					PCAP        string `xml:"pcap_id"`
				} `xml:"entry"`
			} `xml:"log>logs"`
		}
		if err := xml.Unmarshal(WrapInner(resultResp.Result.Inner), &statusResult); err != nil {
			continue
		}

		if statusResult.Status == "FIN" {
			for _, e := range statusResult.Logs.Entry {
				entry := models.ThreatLogEntry{
					Serial:         e.Serial,
					Type:           e.Type,
					Subtype:        e.Subtype,
					SourceIP:       e.SrcIP,
					DestIP:         e.DstIP,
					SourcePort:     e.SrcPort,
					DestPort:       e.DstPort,
					NATSourceIP:    e.NATSrcIP,
					NATDestIP:      e.NATDstIP,
					NATSourcePort:  e.NATSrcPort,
					NATDestPort:    e.NATDstPort,
					SourceZone:     e.SrcZone,
					DestZone:       e.DstZone,
					Rule:           e.Rule,
					Application:    e.App,
					Action:         e.Action,
					SessionID:      e.SessionID,
					User:           e.User,
					ThreatID:       e.ThreatID,
					ThreatName:     e.ThreatName,
					ThreatCategory: e.ThreatCat,
					Severity:       e.Severity,
					Direction:      e.Direction,
					URL:            e.URL,
					Filename:       e.Filename,
					FileHash:       e.FileHash,
					ContentType:    e.ContentType,
					VirtualSystem:  e.Vsys,
					DeviceName:     e.DeviceName,
					ReportID:       e.ReportID,
					PCAP:           e.PCAP,
					Time:           parseLogTime(e.Time),
					ReceiveTime:    parseLogTime(e.ReceiveTime),
				}
				logs = append(logs, entry)
			}
			return logs, nil
		}
	}

	return logs, nil
}

// parseLogTime parses various PAN-OS time formats
func parseLogTime(timeStr string) time.Time {
	if timeStr == "" {
		return time.Time{}
	}
	layouts := []string{
		"2006/01/02 15:04:05",
		"2006-01-02 15:04:05",
		"Mon Jan 2 15:04:05 2006",
		"01/02/2006 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, timeStr); err == nil {
			return t
		}
	}
	return time.Time{}
}

// GetDiskUsage retrieves disk usage information
func (c *Client) GetDiskUsage(ctx context.Context) ([]models.DiskUsage, error) {
	resp, err := c.Op(ctx, "<show><system><disk-space></disk-space></system></show>")
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	// The response is typically plain text output from 'df -h'
	output := string(resp.Result.Inner)
	lines := strings.Split(output, "\n")

	var disks []models.DiskUsage
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Filesystem") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 6 {
			pctStr := strings.TrimSuffix(fields[4], "%")
			pct, _ := strconv.ParseFloat(pctStr, 64)

			disk := models.DiskUsage{
				Filesystem: fields[0],
				Size:       fields[1],
				Used:       fields[2],
				Available:  fields[3],
				Percent:    pct,
				MountPoint: fields[5],
			}
			disks = append(disks, disk)
		}
	}

	return disks, nil
}

// GetEnvironmentals retrieves hardware environmental sensor data
func (c *Client) GetEnvironmentals(ctx context.Context) ([]models.Environmental, error) {
	resp, err := c.Op(ctx, "<show><system><environmentals></environmentals></system></show>")
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	if len(resp.Result.Inner) == 0 {
		return []models.Environmental{}, nil
	}

	var result struct {
		Power struct {
			Slot []struct {
				Entry []struct {
					Description string `xml:"description"`
					Alarm       string `xml:"alarm"`
				} `xml:"entry"`
			} `xml:"slot"`
		} `xml:"power"`
		Thermal struct {
			Slot []struct {
				Entry []struct {
					Description string `xml:"description"`
					DegreesC    string `xml:"DegreesC"`
					Alarm       string `xml:"alarm"`
				} `xml:"entry"`
			} `xml:"slot"`
		} `xml:"thermal"`
		Fan struct {
			Slot []struct {
				Entry []struct {
					Description string `xml:"description"`
					RPMs        string `xml:"RPMs"`
					Alarm       string `xml:"alarm"`
				} `xml:"entry"`
			} `xml:"slot"`
		} `xml:"fan"`
	}

	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil {
		return []models.Environmental{}, nil
	}

	var envs []models.Environmental

	// Process power supplies
	for _, slot := range result.Power.Slot {
		for _, e := range slot.Entry {
			alarm := strings.ToLower(e.Alarm) == "true"
			status := "normal"
			if alarm {
				status = "critical"
			}
			envs = append(envs, models.Environmental{
				Component: e.Description,
				Status:    status,
				Value:     "OK",
				Alarm:     alarm,
			})
		}
	}

	// Process thermal sensors
	for _, slot := range result.Thermal.Slot {
		for _, e := range slot.Entry {
			alarm := strings.ToLower(e.Alarm) == "true"
			status := "normal"
			if alarm {
				status = "critical"
			}
			envs = append(envs, models.Environmental{
				Component: e.Description,
				Status:    status,
				Value:     e.DegreesC + "C",
				Alarm:     alarm,
			})
		}
	}

	// Process fans
	for _, slot := range result.Fan.Slot {
		for _, e := range slot.Entry {
			alarm := strings.ToLower(e.Alarm) == "true"
			status := "normal"
			if alarm {
				status = "critical"
			}
			envs = append(envs, models.Environmental{
				Component: e.Description,
				Status:    status,
				Value:     e.RPMs + " RPM",
				Alarm:     alarm,
			})
		}
	}

	return envs, nil
}

// GetCertificates retrieves certificate information
func (c *Client) GetCertificates(ctx context.Context) ([]models.Certificate, error) {
	resp, err := c.Op(ctx, "<show><sslmgr-store><certificate><all></all></certificate></sslmgr-store></show>")
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	if len(resp.Result.Inner) == 0 {
		return []models.Certificate{}, nil
	}

	var result struct {
		Entry []struct {
			Name           string `xml:"name,attr"`
			Subject        string `xml:"subject"`
			Issuer         string `xml:"issuer"`
			NotValidBefore string `xml:"not-valid-before"`
			NotValidAfter  string `xml:"not-valid-after"`
			SerialNum      string `xml:"serial-number"`
			Algorithm      string `xml:"algorithm"`
		} `xml:"certificate>entry"`
	}

	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil {
		// Try alternate structure
		var alt struct {
			Entry []struct {
				Name           string `xml:"name,attr"`
				Subject        string `xml:"subject"`
				Issuer         string `xml:"issuer"`
				NotValidBefore string `xml:"not-valid-before"`
				NotValidAfter  string `xml:"not-valid-after"`
				SerialNum      string `xml:"serial-number"`
				Algorithm      string `xml:"algorithm"`
			} `xml:"entry"`
		}
		if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &alt); err != nil {
			return []models.Certificate{}, nil
		}
		result.Entry = alt.Entry
	}

	certs := make([]models.Certificate, 0, len(result.Entry))
	for _, e := range result.Entry {
		cert := models.Certificate{
			Name:         e.Name,
			Subject:      e.Subject,
			Issuer:       e.Issuer,
			SerialNumber: e.SerialNum,
			Algorithm:    e.Algorithm,
		}

		// Parse dates
		dateLayouts := []string{
			"Jan 2 15:04:05 2006 MST",
			"2006-01-02 15:04:05",
			"Mon Jan 2 15:04:05 2006",
		}
		for _, layout := range dateLayouts {
			if t, err := time.Parse(layout, e.NotValidBefore); err == nil {
				cert.NotBefore = t
				break
			}
		}
		for _, layout := range dateLayouts {
			if t, err := time.Parse(layout, e.NotValidAfter); err == nil {
				cert.NotAfter = t
				break
			}
		}

		// Calculate days left and status
		if !cert.NotAfter.IsZero() {
			cert.DaysLeft = int(time.Until(cert.NotAfter).Hours() / 24)
			if cert.DaysLeft < 0 {
				cert.Status = "expired"
			} else if cert.DaysLeft < 30 {
				cert.Status = "expiring"
			} else {
				cert.Status = "valid"
			}
		}

		certs = append(certs, cert)
	}

	return certs, nil
}

// GetARPTable retrieves the ARP table
func (c *Client) GetARPTable(ctx context.Context) ([]models.ARPEntry, error) {
	resp, err := c.Op(ctx, "<show><arp><entry name='all'/></arp></show>")
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	if len(resp.Result.Inner) == 0 {
		return []models.ARPEntry{}, nil
	}

	var result struct {
		Entry []struct {
			IP        string `xml:"ip"`
			MAC       string `xml:"mac"`
			Interface string `xml:"interface"`
			Status    string `xml:"status"`
			TTL       int    `xml:"ttl"`
			Port      string `xml:"port"`
		} `xml:"entries>entry"`
	}

	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil {
		return []models.ARPEntry{}, nil
	}

	entries := make([]models.ARPEntry, 0, len(result.Entry))
	for _, e := range result.Entry {
		entries = append(entries, models.ARPEntry{
			IP:        e.IP,
			MAC:       e.MAC,
			Interface: e.Interface,
			Status:    e.Status,
			TTL:       e.TTL,
			Port:      e.Port,
		})
	}

	return entries, nil
}

// GetRoutingTable retrieves the routing table
func (c *Client) GetRoutingTable(ctx context.Context) ([]models.RouteEntry, error) {
	resp, err := c.Op(ctx, "<show><routing><route></route></routing></show>")
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	if len(resp.Result.Inner) == 0 {
		return []models.RouteEntry{}, nil
	}

	var result struct {
		Entry []struct {
			Destination   string `xml:"destination"`
			Nexthop       string `xml:"nexthop"`
			Metric        int    `xml:"metric"`
			Interface     string `xml:"interface"`
			Flags         string `xml:"flags"`
			Age           int    `xml:"age"`
			VirtualRouter string `xml:"virtual-router"`
		} `xml:"entry"`
	}

	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil {
		return []models.RouteEntry{}, nil
	}

	routes := make([]models.RouteEntry, 0, len(result.Entry))
	for _, e := range result.Entry {
		// Determine protocol from flags
		protocol := "static"
		flags := strings.ToLower(e.Flags)
		if strings.Contains(flags, "c") || strings.Contains(flags, "connected") {
			protocol = "connected"
		} else if strings.Contains(flags, "b") || strings.Contains(flags, "bgp") {
			protocol = "bgp"
		} else if strings.Contains(flags, "o") || strings.Contains(flags, "ospf") {
			protocol = "ospf"
		}

		routes = append(routes, models.RouteEntry{
			Destination:   e.Destination,
			Nexthop:       e.Nexthop,
			Metric:        e.Metric,
			Interface:     e.Interface,
			Protocol:      protocol,
			VirtualRouter: e.VirtualRouter,
			Flags:         e.Flags,
			Age:           e.Age,
		})
	}

	return routes, nil
}

// GetIPSecTunnels retrieves IPSec VPN tunnel status
func (c *Client) GetIPSecTunnels(ctx context.Context) ([]models.IPSecTunnel, error) {
	resp, err := c.Op(ctx, "<show><vpn><ipsec-sa></ipsec-sa></vpn></show>")
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	if len(resp.Result.Inner) == 0 {
		return []models.IPSecTunnel{}, nil
	}

	var result struct {
		Entry []struct {
			Name       string `xml:"name"`
			Gateway    string `xml:"gateway"`
			LocalSPI   string `xml:"local-spi"`
			RemoteSPI  string `xml:"peer-spi"`
			Protocol   string `xml:"protocol"`
			Encryption string `xml:"enc-algo"`
			Auth       string `xml:"auth-algo"`
			TunnelIf   string `xml:"tunnel-interface"`
			State      string `xml:"state"`
			Lifesize   string `xml:"lifesize"`
			Lifetime   string `xml:"lifetime"`
		} `xml:"entries>entry"`
	}

	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil {
		return []models.IPSecTunnel{}, nil
	}

	tunnels := make([]models.IPSecTunnel, 0, len(result.Entry))
	for _, e := range result.Entry {
		state := "down"
		if strings.Contains(strings.ToLower(e.State), "active") || strings.Contains(strings.ToLower(e.State), "up") {
			state = "up"
		} else if strings.Contains(strings.ToLower(e.State), "init") {
			state = "init"
		}

		tunnels = append(tunnels, models.IPSecTunnel{
			Name:       e.Name,
			Gateway:    e.Gateway,
			State:      state,
			LocalSPI:   e.LocalSPI,
			RemoteSPI:  e.RemoteSPI,
			Protocol:   e.Protocol,
			Encryption: e.Encryption,
			Auth:       e.Auth,
			Uptime:     e.Lifetime,
		})
	}

	return tunnels, nil
}

// GetGlobalProtectUsers retrieves detailed GlobalProtect user information
func (c *Client) GetGlobalProtectUsers(ctx context.Context) ([]models.GlobalProtectUser, error) {
	resp, err := c.Op(ctx, "<show><global-protect-gateway><current-user></current-user></global-protect-gateway></show>")
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	if len(resp.Result.Inner) == 0 {
		return []models.GlobalProtectUser{}, nil
	}

	var result struct {
		Entry []struct {
			Username  string `xml:"username"`
			Domain    string `xml:"domain"`
			Computer  string `xml:"computer"`
			Client    string `xml:"client"`
			VirtualIP string `xml:"virtual-ip"`
			PublicIP  string `xml:"public-ip"`
			LoginTime string `xml:"login-time"`
			Gateway   string `xml:"gateway"`
			Region    string `xml:"source-region"`
		} `xml:"entry"`
	}

	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil {
		return []models.GlobalProtectUser{}, nil
	}

	users := make([]models.GlobalProtectUser, 0, len(result.Entry))
	for _, e := range result.Entry {
		user := models.GlobalProtectUser{
			Username:     e.Username,
			Domain:       e.Domain,
			Computer:     e.Computer,
			Client:       e.Client,
			VirtualIP:    e.VirtualIP,
			ClientIP:     e.PublicIP,
			Gateway:      e.Gateway,
			SourceRegion: e.Region,
		}

		// Parse login time
		for _, layout := range []string{"2006/01/02 15:04:05", "Mon Jan 2 15:04:05 2006"} {
			if t, err := time.Parse(layout, e.LoginTime); err == nil {
				user.LoginTime = t
				user.Duration = formatDuration(time.Since(t))
				break
			}
		}

		users = append(users, user)
	}

	return users, nil
}

// GetPendingChanges retrieves pending configuration changes
func (c *Client) GetPendingChanges(ctx context.Context) ([]models.PendingChange, error) {
	resp, err := c.Op(ctx, "<show><config><list><changes></changes></list></config></show>")
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	if len(resp.Result.Inner) == 0 {
		return []models.PendingChange{}, nil
	}

	var result struct {
		Entry []struct {
			Admin       string `xml:"admin"`
			Location    string `xml:"xpath"`
			Action      string `xml:"action"`
			Description string `xml:"info"`
			Time        string `xml:"time"`
		} `xml:"journal>entry"`
	}

	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil {
		// Try alternate structure
		var alt struct {
			Entry []struct {
				Admin       string `xml:"admin"`
				Location    string `xml:"xpath"`
				Action      string `xml:"action"`
				Description string `xml:"info"`
				Time        string `xml:"time"`
			} `xml:"entry"`
		}
		if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &alt); err != nil {
			return []models.PendingChange{}, nil
		}
		result.Entry = alt.Entry
	}

	changes := make([]models.PendingChange, 0, len(result.Entry))
	for _, e := range result.Entry {
		change := models.PendingChange{
			User:        e.Admin,
			Location:    e.Location,
			Type:        e.Action,
			Description: e.Description,
		}

		// Parse time
		for _, layout := range []string{"2006/01/02 15:04:05", "Mon Jan 2 15:04:05 2006"} {
			if t, err := time.Parse(layout, e.Time); err == nil {
				change.Time = t
				break
			}
		}

		changes = append(changes, change)
	}

	return changes, nil
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 24 {
		days := hours / 24
		hours = hours % 24
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
