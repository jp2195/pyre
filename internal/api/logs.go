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

// Poll parameters for log queries. These are package-level vars (not consts)
// so tests can shrink them via t.Cleanup to keep runtime bounded. Production
// code never mutates them.
var (
	// logPollMaxAttempts is the maximum number of poll attempts for log queries.
	logPollMaxAttempts = 30
	// logPollInterval is the delay between log query poll attempts.
	logPollInterval = 500 * time.Millisecond
)

const (
	// maxLogRows is the device's hard ceiling for nlogs. PAN-OS answers
	// 5001 with HTTP 400, so this is enforced client-side.
	maxLogRows = 5000
	// defaultLogRows is one page. Traffic rows run about 4.9 KiB each, so
	// 500 is roughly 2.4 MiB and a second or two on the wire, while a full
	// 5000 would be 24.6 MiB against a 50 MB response cap.
	defaultLogRows = 500
)

// LogQuery describes one page of a log fetch.
type LogQuery struct {
	// Query is a raw PAN-OS log expression, exactly as the operator typed
	// it. It is passed through unvalidated: the device owns this grammar
	// and rejects bad input with a better message than we could write.
	Query string
	// Since bounds the page below. The zero value means no lower bound.
	// It is a time.Time rather than a string because the bound must be
	// formatted in the device's zone, which only the client knows.
	Since time.Time
	// Max is rows per page, clamped to [1, maxLogRows].
	Max int
	// Skip is how many matching rows to step over, for paging.
	Skip int
}

// normalized returns a copy with Max and Skip forced into the range the
// device accepts.
func (q LogQuery) normalized() LogQuery {
	if q.Max <= 0 {
		q.Max = defaultLogRows
	}
	q.Max = min(q.Max, maxLogRows)
	q.Skip = max(q.Skip, 0)
	return q
}

// LogPage is one page of results. HasMore is a heuristic, not a count: the
// device reports no total anywhere in its response, so a page that came back
// exactly full is the only evidence that more rows may exist. Query is the
// assembled expression that was sent, which the view shows when a query
// matches nothing.
type LogPage[T any] struct {
	Entries []T
	HasMore bool
	Query   string
	// Warning is a non-fatal problem with this fetch that the operator
	// needs to see -- today, that the time bound had to be written without
	// knowing the device's UTC offset. Empty when there is nothing to say.
	Warning string
}

// buildLogQuery assembles the device-side expression. The time bound is
// formatted in the device's own wall clock: a zoneless PAN-OS layout means
// whatever the device thinks the time is, so formatting in our zone would put
// every bound out by the device's offset. The operator's clause is wrapped in
// parentheses so a top-level `or` cannot escape the time bound; PAN-OS accepts
// redundant parentheses.
func (c *Client) buildLogQuery(q LogQuery) string {
	var clauses []string
	if !q.Since.IsZero() {
		clauses = append(clauses, fmt.Sprintf("(receive_time geq '%s')",
			q.Since.In(c.deviceLocation()).Format("2006/01/02 15:04:05")))
	}
	if user := strings.TrimSpace(q.Query); user != "" {
		clauses = append(clauses, "("+user+")")
	}
	return strings.Join(clauses, " and ")
}

// clockUnknownWarning is what the operator is told when a time bound had to
// be written without knowing the device's UTC offset.
const clockUnknownWarning = "device clock unknown: the time bound was written in this computer's zone, so the window may be off by the firewall's UTC offset"

// ensureBoundZone makes sure the device's UTC offset is known before a bound
// is formatted against it, and reports a warning when it still is not.
//
// PAN-OS interprets a bound in its own wall clock and reports no time zone
// anywhere, so the offset is derived by comparing the clock the device
// reports against ours (learnDeviceClock, driven by GetSystemInfo). Until
// that has run deviceLocation assumes the operator's zone, which is a fine
// default for reading timestamps back but the wrong thing to write into a
// bound: the window silently shifts by the device's offset on the one path
// the design calls non-negotiable. Learning it costs one op command, once
// per client, and only when a bound is actually being sent.
//
// When the device cannot be asked, the bound is still written. Dropping the
// time clause would answer "the last hour" with everything, which is its own
// silent wrong answer; the warning makes the guess visible instead.
func (c *Client) ensureBoundZone(ctx context.Context, q LogQuery, target string) string {
	if q.Since.IsZero() || c.deviceLoc.Load() != nil {
		return ""
	}
	// One probe at a time. The three log tabs fetch concurrently, so without
	// this they each find the zone unknown and each ask the device for the
	// same answer. Whoever gets the lock second finds it already learned.
	c.deviceLocMu.Lock()
	defer c.deviceLocMu.Unlock()
	if c.deviceLoc.Load() != nil {
		return ""
	}
	if _, err := c.GetSystemInfo(ctx, target); err != nil {
		debugf("[API] could not learn the device clock for a log time bound: %v", err)
	}
	if c.deviceLoc.Load() != nil {
		return ""
	}
	return clockUnknownWarning
}

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
func (c *Client) submitAndPollLog(ctx context.Context, logType string, q LogQuery, sent, target string) (*XMLResponse, error) {
	resp, err := c.Log(ctx, logType, q.Max, q.Skip, sent, target)
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
func (c *Client) parseLogTime(timeStr string) time.Time {
	if timeStr == "" {
		return time.Time{}
	}
	if t, err := c.parsePANTime(timeStr); err == nil {
		return t
	}
	log.Printf("[API Warning] failed to parse log time %q: no matching layout", timeStr)
	return time.Time{}
}

// GetSystemLogs retrieves one page of system logs.
func (c *Client) GetSystemLogs(ctx context.Context, q LogQuery, target string) (LogPage[models.SystemLogEntry], error) {
	q = q.normalized()
	// Before the bound is formatted, not after: it has to be written in the
	// device's wall clock, and that zone is learned rather than reported.
	warning := c.ensureBoundZone(ctx, q, target)
	sent := c.buildLogQuery(q)

	resultResp, err := c.submitAndPollLog(ctx, "system", q, sent, target)
	if err != nil {
		return LogPage[models.SystemLogEntry]{}, err
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
		return LogPage[models.SystemLogEntry]{}, fmt.Errorf("parsing system log entries: %w", err)
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
		entry.Time = c.parseLogTime(e.Time)
		logs = append(logs, entry)
	}

	return LogPage[models.SystemLogEntry]{
		Entries: logs,
		HasMore: len(logs) == q.Max,
		Query:   sent,
		Warning: warning,
	}, nil
}

// GetTrafficLogs retrieves one page of traffic logs.
func (c *Client) GetTrafficLogs(ctx context.Context, q LogQuery, target string) (LogPage[models.TrafficLogEntry], error) {
	q = q.normalized()
	// Before the bound is formatted, not after: it has to be written in the
	// device's wall clock, and that zone is learned rather than reported.
	warning := c.ensureBoundZone(ctx, q, target)
	sent := c.buildLogQuery(q)

	resultResp, err := c.submitAndPollLog(ctx, "traffic", q, sent, target)
	if err != nil {
		return LogPage[models.TrafficLogEntry]{}, err
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
		return LogPage[models.TrafficLogEntry]{}, fmt.Errorf("parsing traffic log entries: %w", err)
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
			Time:          c.parseLogTime(e.Time),
			ReceiveTime:   c.parseLogTime(e.ReceiveTime),
		}
		logs = append(logs, entry)
	}

	return LogPage[models.TrafficLogEntry]{
		Entries: logs,
		HasMore: len(logs) == q.Max,
		Query:   sent,
		Warning: warning,
	}, nil
}

// GetThreatLogs retrieves one page of threat logs.
func (c *Client) GetThreatLogs(ctx context.Context, q LogQuery, target string) (LogPage[models.ThreatLogEntry], error) {
	q = q.normalized()
	// Before the bound is formatted, not after: it has to be written in the
	// device's wall clock, and that zone is learned rather than reported.
	warning := c.ensureBoundZone(ctx, q, target)
	sent := c.buildLogQuery(q)

	resultResp, err := c.submitAndPollLog(ctx, "threat", q, sent, target)
	if err != nil {
		return LogPage[models.ThreatLogEntry]{}, err
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
				ThreatID    string `xml:"threatid"`
				// PAN-OS 11.x sends the readable name in threat_name, and threatid
				// itself carries a name-shaped value ("Proxy:mask.test-dns.net")
				// with the numeric id moved to <tid>. Verified on a PA-440 running
				// 11.2.10-h8. Decoding threatid as int64 failed the whole batch.
				ThreatName string `xml:"threat_name"`
				// LegacyName is the pre-11.x <threat> element. No pre-11.x device
				// was available to verify it, so it is only ever a fallback.
				LegacyName  string `xml:"threat"`
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
		return LogPage[models.ThreatLogEntry]{}, fmt.Errorf("parsing threat log entries: %w", err)
	}

	logs := make([]models.ThreatLogEntry, 0, len(statusResult.Logs.Entry))
	for _, e := range statusResult.Logs.Entry {
		// Resolve the readable name across the three shapes PAN-OS has used:
		// 11.x threat_name, pre-11.x <threat>, and 11.x threatid when it is
		// name-shaped rather than numeric.
		threatName := e.ThreatName
		if threatName == "" {
			threatName = e.LegacyName
		}
		if threatName == "" && e.ThreatID != "" {
			if _, err := strconv.ParseInt(e.ThreatID, 10, 64); err != nil {
				threatName = e.ThreatID
			}
		}

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
			ThreatName:     threatName,
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
			Time:           c.parseLogTime(e.Time),
			ReceiveTime:    c.parseLogTime(e.ReceiveTime),
		}
		logs = append(logs, entry)
	}

	return LogPage[models.ThreatLogEntry]{
		Entries: logs,
		HasMore: len(logs) == q.Max,
		Query:   sent,
		Warning: warning,
	}, nil
}
