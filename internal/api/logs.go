package api

import (
	"context"
	"encoding/xml"
	"fmt"
	"log"
	"time"

	"github.com/jp2195/pyre/internal/models"
)

const (
	// logPollMaxAttempts is the maximum number of poll attempts for log queries.
	logPollMaxAttempts = 30
	// logPollInterval is the delay between log query poll attempts.
	logPollInterval = 500 * time.Millisecond
)

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
	log.Printf("[API Warning] failed to parse log time %q: no matching layout", timeStr)
	return time.Time{}
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
	for i := 0; i < logPollMaxAttempts; i++ {
		time.Sleep(logPollInterval)

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
	for i := 0; i < logPollMaxAttempts; i++ {
		time.Sleep(logPollInterval)

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
	for i := 0; i < logPollMaxAttempts; i++ {
		time.Sleep(logPollInterval)

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
