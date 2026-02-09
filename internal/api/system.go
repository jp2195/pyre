package api

import (
	"context"
	"encoding/xml"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jp2195/pyre/internal/models"
)

// Regex patterns for parsing top output
var (
	loadAvgRegex = regexp.MustCompile(`load average:\s*([\d.]+)[,\s]+([\d.]+)[,\s]+([\d.]+)`)
	// Multiple patterns for CPU idle - different PAN-OS versions have different formats
	// Format 1: "91.7 id" or "91.7  id"
	// Format 2: "91.7%id" or "91.7% id"
	cpuIdlePatterns = []*regexp.Regexp{
		regexp.MustCompile(`([\d.]+)\s*%?\s*id[,\s]`), // "91.7 id," or "91.7%id,"
		regexp.MustCompile(`([\d.]+)\s*%\s*id`),       // "91.7% id"
		regexp.MustCompile(`([\d.]+)\s+id`),           // "91.7 id"
	}
	// CPU user/system patterns for us + sy calculation (more accurate for PAN-OS management CPU)
	cpuUsPattern = regexp.MustCompile(`([\d.]+)\s*%?\s*us`)
	cpuSyPattern = regexp.MustCompile(`([\d.]+)\s*%?\s*sy`)
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
		parsed := false
		for _, layout := range layouts {
			if t, err := time.Parse(layout, si.Time); err == nil {
				info.CurrentTime = t
				parsed = true
				break
			}
		}
		if !parsed {
			log.Printf("[API Warning] failed to parse system time %q: no matching layout", si.Time)
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
	// Ignore parse errors - optional fields, zero value acceptable if parsing fails
	if matches := loadAvgRegex.FindStringSubmatch(output); len(matches) >= 4 {
		resources.Load1, _ = strconv.ParseFloat(matches[1], 64)  //nolint:errcheck // intentional - zero value acceptable
		resources.Load5, _ = strconv.ParseFloat(matches[2], 64)  //nolint:errcheck // intentional - zero value acceptable
		resources.Load15, _ = strconv.ParseFloat(matches[3], 64) //nolint:errcheck // intentional - zero value acceptable
	}

	lines := strings.Split(output, "\n")
	cpuFound := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse CPU - look for Cpu line and extract CPU usage
		// PAN-OS management CPU is best represented by us (user) + sy (system)
		if !cpuFound && (strings.HasPrefix(line, "%Cpu") || strings.HasPrefix(line, "Cpu")) {
			// Primary method: us + sy (matches PAN-OS "Management CPU" metric)
			var userCPU, sysCPU float64
			var foundUser, foundSys bool

			if matches := cpuUsPattern.FindStringSubmatch(line); len(matches) >= 2 {
				if val, err := strconv.ParseFloat(matches[1], 64); err == nil && val >= 0 && val <= 100 {
					userCPU = val
					foundUser = true
				}
			}
			if matches := cpuSyPattern.FindStringSubmatch(line); len(matches) >= 2 {
				if val, err := strconv.ParseFloat(matches[1], 64); err == nil && val >= 0 && val <= 100 {
					sysCPU = val
					foundSys = true
				}
			}

			if foundUser && foundSys {
				resources.CPUPercent = userCPU + sysCPU
				cpuFound = true
			} else if foundUser {
				// Fallback to just user CPU if system not found
				resources.CPUPercent = userCPU
				cpuFound = true
			}

			// Fallback: try idle-based calculation if us+sy didn't work
			if !cpuFound {
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
			}
		}

		// Parse memory - look for lines with memory info
		if strings.HasPrefix(line, "Mem:") || strings.HasPrefix(line, "KiB Mem") || strings.HasPrefix(line, "MiB Mem") {
			var total, used float64
			fields := strings.Fields(line)
			for i, f := range fields {
				// Handle format: "16384000 total," or "total: 16384000"
				// Ignore parse errors - fields may have unexpected format, zero value acceptable
				cleanField := strings.TrimRight(f, ",%")
				if (cleanField == "total" || f == "total," || f == "total") && i > 0 {
					total, _ = strconv.ParseFloat(strings.TrimRight(fields[i-1], ",%"), 64) //nolint:errcheck // intentional
				}
				if (cleanField == "used" || f == "used," || f == "used") && i > 0 {
					used, _ = strconv.ParseFloat(strings.TrimRight(fields[i-1], ",%"), 64) //nolint:errcheck // intentional
				}
			}
			if total > 0 {
				resources.MemoryPercent = (used / total) * 100
			}
		}
	}

	// Set ManagementCPU to Load1 (1-minute load average) as the quick health indicator
	resources.ManagementCPU = resources.Load1

	return resources, nil
}

// GetDataPlaneResources fetches dataplane CPU utilization from the resource monitor
func (c *Client) GetDataPlaneResources(ctx context.Context) (float64, error) {
	resp, err := c.Op(ctx, "<show><running><resource-monitor><hour><last>1</last></hour></resource-monitor></running></show>")
	if err != nil {
		return 0, err
	}
	if err := CheckResponse(resp); err != nil {
		return 0, err
	}

	// Parse the resource monitor response
	// Structure: <resource-monitor><data-processors><dp0><hour><cpu-load-average><entry>...
	type cpuEntry struct {
		CoreID string `xml:"coreid"`
		Value  string `xml:"value"`
	}

	type dpHour struct {
		CPULoadAverage []cpuEntry `xml:"cpu-load-average>entry"`
	}

	type dataProcessor struct {
		Hour dpHour `xml:"hour"`
	}

	type resourceMonitor struct {
		DP0 dataProcessor `xml:"data-processors>dp0"`
		DP1 dataProcessor `xml:"data-processors>dp1"`
	}

	var result struct {
		ResourceMonitor resourceMonitor `xml:"resource-monitor"`
	}

	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil {
		return 0, fmt.Errorf("failed to parse resource monitor: %w", err)
	}

	// Calculate average CPU across all cores from dp0
	// If dp1 exists (multi-DP systems), include it as well
	var totalCPU float64
	var coreCount int

	for _, entry := range result.ResourceMonitor.DP0.Hour.CPULoadAverage {
		if val, err := strconv.ParseFloat(entry.Value, 64); err == nil {
			totalCPU += val
			coreCount++
		}
	}

	// Check for dp1 (some platforms have multiple data processors)
	for _, entry := range result.ResourceMonitor.DP1.Hour.CPULoadAverage {
		if val, err := strconv.ParseFloat(entry.Value, 64); err == nil {
			totalCPU += val
			coreCount++
		}
	}

	if coreCount == 0 {
		return 0, nil
	}

	return totalCPU / float64(coreCount), nil
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
