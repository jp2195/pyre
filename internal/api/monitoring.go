package api

import (
	"bytes"
	"cmp"
	"context"
	"log"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jp2195/pyre/internal/models"
)

// threatSummarySample caps how many threat log entries the summary examines.
// The log query is capped server-side, so the result describes the most
// recent threats rather than every threat the device has ever recorded.
const threatSummarySample = 100

// GetThreatSummary aggregates recent threat log entries by severity and action.
//
// This used to read dataplane global counters matching flow_threat_*, which
// was wrong twice over on a PA-440 running 11.2.10-h8. The filtered counter
// command fails outright there, so the panel showed nothing at all. And the
// counters it would have read carry a severity of drop / info / warn, which
// never matches a threat severity, so every severity bucket was zero while
// the total accumulated raw packet counts since boot. The threat log is the
// device's actual record of threats and carries real severities and actions.
func (c *Client) GetThreatSummary(ctx context.Context, target string) (*models.ThreatSummary, error) {
	logs, err := c.GetThreatLogs(ctx, "", threatSummarySample, target)
	if err != nil {
		return nil, err
	}

	summary := &models.ThreatSummary{SampleLimit: threatSummarySample}
	for _, l := range logs {
		summary.TotalThreats++
		switch strings.ToLower(l.Severity) {
		case "critical":
			summary.CriticalCount++
		case "high":
			summary.HighCount++
		case "medium":
			summary.MediumCount++
		case "low", "informational":
			summary.LowCount++
		}
		if threatActionStopped(l.Action) {
			summary.BlockedCount++
		} else {
			summary.AlertedCount++
		}
	}
	return summary, nil
}

// threatActionStopped reports whether a threat log action intervened in the
// traffic rather than merely recording it.
//
// The vocabulary is PAN-OS's. alert and allow observe; everything else
// intervenes, including sinkhole, which answers a DNS query with a controlled
// address instead of the real one. Listing the observe-only actions rather
// than the intervening ones means an action this code has not seen before
// counts as an intervention, which is the safer way to be wrong in a panel
// that tells an operator whether something was stopped.
func threatActionStopped(action string) bool {
	switch strings.ToLower(action) {
	case "", "alert", "allow", "continue", "override":
		return false
	}
	return true
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
		log.Printf("[API Warning] failed to parse GlobalProtect gateway users: %v", err)
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

// jobTimeOfDayLayout matches the bare dequeue time PAN-OS reports for jobs.
const jobTimeOfDayLayout = "15:04:05"

// parseJobStart resolves a job's start time from the enqueue and dequeue
// fields.
//
// PAN-OS reports tenq as a full datetime but tdeq as a bare time of day
// ("21:43:45"), which matches no layout on its own, so every job's start time
// used to come back as the zero time. The date is taken from tenq; a start
// that lands before the enqueue belongs to the following day, which is what
// happens to a job queued at 23:59.
func (c *Client) parseJobStart(enqueued, dequeued string) time.Time {
	enq := c.parseJobTimestamp(enqueued)
	deq := c.parseJobTimestamp(dequeued)
	if !deq.IsZero() {
		return deq
	}
	if enq.IsZero() || dequeued == "" {
		return enq
	}
	// Scoped to this parser on purpose: a bare time of day must not be in
	// the shared layout list, or any malformed timestamp anywhere would
	// "parse" into year zero instead of reporting an error.
	tod, err := time.ParseInLocation(jobTimeOfDayLayout, dequeued, c.deviceLocation())
	if err != nil {
		return enq
	}
	start := time.Date(enq.Year(), enq.Month(), enq.Day(),
		tod.Hour(), tod.Minute(), tod.Second(), 0, c.deviceLocation())
	if start.Before(enq) {
		start = start.AddDate(0, 0, 1)
	}
	return start
}

// parseJobTimestamp tries multiple time layouts to parse a PAN-OS job timestamp.
func (c *Client) parseJobTimestamp(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if t, err := c.parsePANTime(s); err == nil {
		return t
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

		// PAN-OS reuses the progress field: it holds a percentage while a
		// job runs, and the completion timestamp once it has finished. A
		// finished job is 100% whatever the field says.
		if pct, err := strconv.Atoi(strings.TrimSuffix(e.Progress, "%")); err == nil {
			job.Progress = pct
		} else if strings.EqualFold(e.Status, "FIN") {
			job.Progress = 100
		}

		job.StartTime = c.parseJobStart(e.TEnq, e.TDeq)
		job.EndTime = c.parseJobTimestamp(e.Tfin)

		jobs = append(jobs, job)
	}

	// Sort by ID descending (most recent first).
	slices.SortFunc(jobs, func(a, b models.Job) int {
		return cmp.Compare(b.ID, a.ID)
	})

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

	// The response is plain text output from 'df -h', CDATA-wrapped on real
	// hardware — decode it as text so the header check below actually fires.
	output := InnerText(resp.Result.Inner)
	lines := strings.Split(output, "\n")

	// Non-nil even when empty: the dashboard treats a nil slice as
	// "still fetching", which would spin its loading indicator forever.
	disks := []models.DiskUsage{}
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
				Filesystem: SanitizeForDisplay(fields[0]),
				Size:       SanitizeForDisplay(fields[1]),
				Used:       SanitizeForDisplay(fields[2]),
				Available:  SanitizeForDisplay(fields[3]),
				Percent:    pct,
				MountPoint: SanitizeForDisplay(fields[5]),
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
	// Non-nil even when empty — see the note in GetDiskUsage.
	envs := []models.Environmental{}

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
		if t, err := c.parsePANTime(e.NotValidBefore); err == nil {
			cert.NotBefore = t
		}
		if t, err := c.parsePANTime(e.NotValidAfter); err == nil {
			cert.NotAfter = t
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
