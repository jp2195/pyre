package api

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jp2195/pyre/internal/models"
)

// Poll parameters for log queries. These are package-level vars (not consts)
// so tests can shrink them via t.Cleanup to keep runtime bounded. Production
// code never mutates them.
var (
	// logPollMaxAttempts is the maximum number of poll attempts for log queries.
	logPollMaxAttempts = 30
	// logPollInterval is the delay between log query poll attempts.
	logPollInterval = 500 * time.Millisecond
)

// logJobStatus classifies a PAN-OS log-query job state.
type logJobStatus int

const (
	logJobRunning logJobStatus = iota
	logJobDone
	logJobFailed
)

// classifyJobStatus extracts the PAN-OS job status from a LogGet response
// and classifies it. PAN-OS wraps the status inside <job><status>...</status></job>
// under the <result> element; we partially decode it here so the caller
// doesn't need to duplicate the schema.
func classifyJobStatus(resp *XMLResponse) (logJobStatus, string) {
	if resp == nil || len(resp.Result.Inner) == 0 {
		return logJobRunning, ""
	}
	var parsed struct {
		Status string `xml:"job>status"`
	}
	if err := decodeXML(bytes.NewReader(WrapInner(resp.Result.Inner)), &parsed); err != nil {
		return logJobRunning, ""
	}
	if parsed.Status == "" {
		return logJobRunning, ""
	}
	switch strings.ToUpper(parsed.Status) {
	case "FIN", "DONE":
		return logJobDone, parsed.Status
	case "FAIL", "CANC":
		return logJobFailed, parsed.Status
	default:
		return logJobRunning, parsed.Status
	}
}

// pollLogJob polls a PAN-OS log-query job until it reports completion, fails,
// or the attempt budget is exhausted. It returns the completed LogGet
// response so callers can decode their log-type-specific schema from
// resp.Result.Inner.
//
// The shared helper replaces three nearly-identical inline loops and adds:
//   - bounded retries (logPollMaxAttempts attempts, like the previous code)
//   - exponential backoff capped at 2s between polls
//   - FAIL / CANC classification returned as a real error
//   - a real timeout error when the attempt budget is exhausted
//   - a real error when consecutive transport/decode failures hit 3
func (c *Client) pollLogJob(ctx context.Context, jobID, target string) (*XMLResponse, error) {
	const maxConsecErrors = 3
	interval := logPollInterval

	var consecErr int
	var lastErr error
	for attempt := 1; attempt <= logPollMaxAttempts; attempt++ {
		resp, err := c.LogGet(ctx, jobID, target)
		if err == nil {
			err = CheckResponse(resp)
		}
		if err != nil {
			consecErr++
			lastErr = err
			if consecErr >= maxConsecErrors {
				return nil, fmt.Errorf("log poll failed after %d consecutive errors: %w", consecErr, err)
			}
		} else {
			consecErr = 0
			switch status, raw := classifyJobStatus(resp); status {
			case logJobDone:
				return resp, nil
			case logJobFailed:
				return nil, fmt.Errorf("log job %s reported failure: %s", jobID, SanitizeForDisplay(raw))
			case logJobRunning:
				// Exponential backoff, capped at 2s so we don't starve fast jobs.
				interval = min(interval*2, 2*time.Second)
			}
		}

		// Sleep between attempts (skipped after the final attempt).
		if attempt == logPollMaxAttempts {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("log poll exhausted %d attempts; last error: %w", logPollMaxAttempts, lastErr)
	}
	return nil, fmt.Errorf("log poll timed out after %d attempts", logPollMaxAttempts)
}

// submitAndPollLog submits a log query of the given type, parses the job ID
// from the response, and polls until the job reports completion. It returns
// the completed LogGet response so callers can decode their log-type-specific
// schema from resp.Result.Inner.
//
// The three GetXxxLogs functions share this submit-parse-poll preamble; only
// the decode-and-convert tail differs (different XML schemas, different model
// types). Extracting just the shared preamble keeps each caller's
// log-type-specific schema explicit and avoids forcing a generic over the
// substantively different per-entry XML shapes.
func (c *Client) submitAndPollLog(ctx context.Context, logType, query string, maxLogs int, target string) (*XMLResponse, error) {
	resp, err := c.Log(ctx, logType, maxLogs, query, target)
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	var jobResult struct {
		Job string `xml:"job"`
	}
	if err := decodeXML(bytes.NewReader(WrapInner(resp.Result.Inner)), &jobResult); err != nil {
		return nil, fmt.Errorf("parsing job response: %w", err)
	}
	if jobResult.Job == "" {
		return nil, fmt.Errorf("no job ID returned")
	}

	return c.pollLogJob(ctx, jobResult.Job, target)
}

// parseLogTime parses various PAN-OS time formats
func parseLogTime(timeStr string) time.Time {
	if timeStr == "" {
		return time.Time{}
	}
	if t, err := parsePANTime(timeStr); err == nil {
		return t
	}
	log.Printf("[API Warning] failed to parse log time %q: no matching layout", timeStr)
	return time.Time{}
}

// GetSystemLogs retrieves system logs with optional query filter
// Uses type=log API which returns a job ID, then polls for results
func (c *Client) GetSystemLogs(ctx context.Context, query string, maxLogs int, target string) ([]models.SystemLogEntry, error) {
	if maxLogs <= 0 {
		maxLogs = 100
	}

	resultResp, err := c.submitAndPollLog(ctx, "system", query, maxLogs, target)
	if err != nil {
		return nil, err
	}

	var statusResult struct {
		Logs struct {
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
	if err := decodeXML(bytes.NewReader(WrapInner(resultResp.Result.Inner)), &statusResult); err != nil {
		return nil, fmt.Errorf("parsing system log entries: %w", err)
	}

	logs := make([]models.SystemLogEntry, 0, len(statusResult.Logs.Entry))
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

	sanitizeAllStrings(&logs)
	return logs, nil
}

// GetTrafficLogs retrieves traffic logs with optional query filter
func (c *Client) GetTrafficLogs(ctx context.Context, query string, maxLogs int, target string) ([]models.TrafficLogEntry, error) {
	if maxLogs <= 0 {
		maxLogs = 100
	}

	resultResp, err := c.submitAndPollLog(ctx, "traffic", query, maxLogs, target)
	if err != nil {
		return nil, err
	}

	var statusResult struct {
		Logs struct {
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
	if err := decodeXML(bytes.NewReader(WrapInner(resultResp.Result.Inner)), &statusResult); err != nil {
		return nil, fmt.Errorf("parsing traffic log entries: %w", err)
	}

	logs := make([]models.TrafficLogEntry, 0, len(statusResult.Logs.Entry))
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

	sanitizeAllStrings(&logs)
	return logs, nil
}

// GetThreatLogs retrieves threat logs with optional query filter
func (c *Client) GetThreatLogs(ctx context.Context, query string, maxLogs int, target string) ([]models.ThreatLogEntry, error) {
	if maxLogs <= 0 {
		maxLogs = 100
	}

	resultResp, err := c.submitAndPollLog(ctx, "threat", query, maxLogs, target)
	if err != nil {
		return nil, err
	}

	var statusResult struct {
		Logs struct {
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
	if err := decodeXML(bytes.NewReader(WrapInner(resultResp.Result.Inner)), &statusResult); err != nil {
		return nil, fmt.Errorf("parsing threat log entries: %w", err)
	}

	logs := make([]models.ThreatLogEntry, 0, len(statusResult.Logs.Entry))
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

	sanitizeAllStrings(&logs)
	return logs, nil
}
