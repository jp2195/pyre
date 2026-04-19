package api

import (
	"context"
	"bytes"
	"strconv"
	"strings"
	"time"

	"github.com/jp2195/pyre/internal/models"
)

func (c *Client) GetThreatSummary(ctx context.Context, target string) (*models.ThreatSummary, error) {
	// Query threat logs for the last hour to get a summary
	resp, err := c.Op(ctx, "<show><counter><global><name>flow_threat_*</name></global></counter></show>", target)
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	summary := &models.ThreatSummary{}

	var result struct {
		Entry []struct {
			Name     string `xml:"name"`
			Value    int64  `xml:"value"`
			Rate     int64  `xml:"rate"`
			Aspect   string `xml:"aspect"`
			Desc     string `xml:"desc"`
			Severity string `xml:"severity"`
		} `xml:"global>counters>entry"`
	}
	if err := decodeXML(bytes.NewReader(WrapInner(resp.Result.Inner)), &result); err != nil {
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

func (c *Client) GetGlobalProtectInfo(ctx context.Context, target string) (*models.GlobalProtectInfo, error) {
	resp, err := c.Op(ctx, "<show><global-protect-gateway><current-user></current-user></global-protect-gateway></show>", target)
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	info := &models.GlobalProtectInfo{}

	var result struct {
		Entry []struct {
			Username  string `xml:"username"`
			Domain    string `xml:"domain"`
			Computer  string `xml:"computer"`
			Client    string `xml:"client"`
			VirtualIP string `xml:"virtual-ip"`
			LoginTime string `xml:"login-time"`
		} `xml:"entry"`
	}
	if err := decodeXML(bytes.NewReader(WrapInner(resp.Result.Inner)), &result); err != nil {
		return info, nil
	}

	info.ActiveUsers = len(result.Entry)
	info.TotalUsers = len(result.Entry)

	return info, nil
}

// jobEntry is the shared XML structure for job entries across different PAN-OS response formats.
type jobEntry struct {
	ID        int    `xml:"id"`
	Type      string `xml:"type"`
	Status    string `xml:"status"`
	Result    string `xml:"result"`
	Progress  string `xml:"progress"`
	Details   string `xml:"details>line"`
	TEnq      string `xml:"tenq"` // Time enqueued
	TDeq      string `xml:"tdeq"` // Time dequeued (started)
	Tfin      string `xml:"tfin"` // Time finished
	User      string `xml:"user"`
	Stoppable string `xml:"stoppable"`
}

// parseJobTimestamp tries multiple time layouts to parse a PAN-OS job timestamp.
func parseJobTimestamp(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{
		"2006/01/02 15:04:05",
		"2006-01-02 15:04:05",
		"Mon Jan 2 15:04:05 2006",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func (c *Client) GetJobs(ctx context.Context, target string) ([]models.Job, error) {
	resp, err := c.Op(ctx, "<show><jobs><all></all></jobs></show>", target)
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	if len(resp.Result.Inner) == 0 {
		return []models.Job{}, nil
	}

	var entries []jobEntry

	// Try <job> wrapper first (most common)
	var jobResult struct {
		Entry []jobEntry `xml:"job"`
	}
	if decodeXML(bytes.NewReader(WrapInner(resp.Result.Inner)), &jobResult) == nil && len(jobResult.Entry) > 0 {
		entries = jobResult.Entry
	}

	// Fall back to <entry> wrapper
	if len(entries) == 0 {
		var entryResult struct {
			Entry []jobEntry `xml:"entry"`
		}
		if decodeXML(bytes.NewReader(WrapInner(resp.Result.Inner)), &entryResult) == nil {
			entries = entryResult.Entry
		}
	}

	jobs := make([]models.Job, 0, len(entries))
	for _, e := range entries {
		job := models.Job{
			ID:      e.ID,
			Type:    e.Type,
			Status:  e.Status,
			Result:  e.Result,
			Message: e.Details,
			User:    e.User,
		}

		if e.Progress != "" {
			job.Progress, _ = strconv.Atoi(strings.TrimSuffix(e.Progress, "%")) //nolint:errcheck // intentional - default to 0 on parse error
		}

		job.StartTime = parseJobTimestamp(e.TDeq)
		job.EndTime = parseJobTimestamp(e.Tfin)

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

// GetDiskUsage retrieves disk usage information
func (c *Client) GetDiskUsage(ctx context.Context, target string) ([]models.DiskUsage, error) {
	resp, err := c.Op(ctx, "<show><system><disk-space></disk-space></system></show>", target)
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
			pct, _ := strconv.ParseFloat(pctStr, 64) //nolint:errcheck // intentional - default to 0 on parse error

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
//
//nolint:misspell // "environmentals" is the PAN-OS XML API tag name
func (c *Client) GetEnvironmentals(ctx context.Context, target string) ([]models.Environmental, error) {
	resp, err := c.Op(ctx, "<show><system><environmentals></environmentals></system></show>", target)
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	if len(resp.Result.Inner) == 0 {
		return []models.Environmental{}, nil
	}

	// Environmental entry common structure
	type envEntry struct {
		Description string `xml:"description"`
		DegreesC    string `xml:"DegreesC"`
		RPMs        string `xml:"RPMs"`
		Alarm       string `xml:"alarm"`
	}

	// Slot wrapper that captures any slot element (Slot1, Slot2, slot, etc.)
	type slotWrapper struct {
		Entry []envEntry `xml:"entry"`
	}

	// Use a flexible structure that captures slot elements with any name
	// PAN-OS uses both <slot> and <Slot1>, <Slot2>, etc. depending on model
	var envs []models.Environmental

	// Parse power section
	type powerSection struct {
		Slots []slotWrapper `xml:",any"`
	}
	var powerResult struct {
		Power powerSection `xml:"power"`
	}
	if decodeXML(bytes.NewReader(WrapInner(resp.Result.Inner)), &powerResult) == nil {
		for _, slot := range powerResult.Power.Slots {
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
	}

	// Parse thermal section
	type thermalSection struct {
		Slots []slotWrapper `xml:",any"`
	}
	var thermalResult struct {
		Thermal thermalSection `xml:"thermal"`
	}
	if decodeXML(bytes.NewReader(WrapInner(resp.Result.Inner)), &thermalResult) == nil {
		for _, slot := range thermalResult.Thermal.Slots {
			for _, e := range slot.Entry {
				alarm := strings.ToLower(e.Alarm) == "true"
				status := "normal"
				if alarm {
					status = "critical"
				}
				value := e.DegreesC
				if value != "" && !strings.HasSuffix(value, "C") {
					value += "C"
				}
				if value == "" {
					value = "N/A"
				}
				envs = append(envs, models.Environmental{
					Component: e.Description,
					Status:    status,
					Value:     value,
					Alarm:     alarm,
				})
			}
		}
	}

	// Parse fan section
	type fanSection struct {
		Slots []slotWrapper `xml:",any"`
	}
	var fanResult struct {
		Fan fanSection `xml:"fan"`
	}
	if decodeXML(bytes.NewReader(WrapInner(resp.Result.Inner)), &fanResult) == nil {
		for _, slot := range fanResult.Fan.Slots {
			for _, e := range slot.Entry {
				alarm := strings.ToLower(e.Alarm) == "true"
				status := "normal"
				if alarm {
					status = "critical"
				}
				value := e.RPMs
				if value != "" && !strings.Contains(value, "RPM") {
					value += " RPM"
				}
				if value == "" {
					value = "N/A"
				}
				envs = append(envs, models.Environmental{
					Component: e.Description,
					Status:    status,
					Value:     value,
					Alarm:     alarm,
				})
			}
		}
	}

	return envs, nil
}

// GetCertificates retrieves certificate information
func (c *Client) GetCertificates(ctx context.Context, target string) ([]models.Certificate, error) {
	resp, err := c.Op(ctx, "<show><sslmgr-store><certificate><all></all></certificate></sslmgr-store></show>", target)
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

	if err := decodeXML(bytes.NewReader(WrapInner(resp.Result.Inner)), &result); err != nil {
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
		if err := decodeXML(bytes.NewReader(WrapInner(resp.Result.Inner)), &alt); err != nil {
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
