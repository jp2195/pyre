package api

import (
	"context"
	"encoding/xml"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jp2195/pyre/internal/models"
)

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
		log.Printf("[API Warning] failed to parse session info XML: %v", err)
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
		log.Printf("[API Warning] failed to parse sessions list XML: %v", err)
		return []models.Session{}, nil
	}

	sessions := make([]models.Session, 0, len(result.Entry))
	for _, e := range result.Entry {
		var startTime time.Time
		// Ignore parse error - time format may vary, zero time acceptable
		if e.StartTime != "" {
			startTime, _ = time.Parse("Mon Jan 2 15:04:05 2006", e.StartTime) //nolint:errcheck // intentional - zero time acceptable
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

// GetSessionByID retrieves detailed information for a specific session.
func (c *Client) GetSessionByID(ctx context.Context, id int64) (*models.SessionDetail, error) {
	cmd := fmt.Sprintf("<show><session><id>%d</id></session></show>", id)

	resp, err := c.Op(ctx, cmd)
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	if len(resp.Result.Inner) == 0 {
		return nil, fmt.Errorf("session %d not found", id)
	}

	var result struct {
		// Basic
		Idx         int64  `xml:"idx"`
		State       string `xml:"state"`
		Application string `xml:"application"`
		Type        string `xml:"type"`
		Vsys        string `xml:"vsys"`

		// Source/Dest
		Source      string `xml:"source"`
		Sport       int    `xml:"sport"`
		Destination string `xml:"dst"`
		Dport       int    `xml:"dport"`
		SourceZone  string `xml:"from"`
		DestZone    string `xml:"to"`
		Proto       string `xml:"proto"`
		Srcuser     string `xml:"srcuser"`
		SrcHostID   string `xml:"src-host-id"`

		// NAT
		XSource string `xml:"xsource"`
		XSport  int    `xml:"xsport"`
		XDst    string `xml:"xdst"`
		XDport  int    `xml:"xdport"`
		NATRule string `xml:"nat-rule"`

		// Security
		SecurityRule   string `xml:"security-rule"`
		URLCategory    string `xml:"url-category"`
		URLFilterRule  string `xml:"url-filter-rule"`
		DecryptionRule string `xml:"decryption-rule"`

		// Traffic
		ToClientPkts   int64 `xml:"s2c-pkts"`
		ToServerPkts   int64 `xml:"c2s-pkts"`
		ToClientBytes  int64 `xml:"s2c-bytes"`
		ToServerBytes  int64 `xml:"c2s-bytes"`
		TotalByteCount int64 `xml:"total-byte-count"`

		// Timing
		StartTime string `xml:"start-time"`
		Timeout   int    `xml:"timeout"`
		TTL       int    `xml:"ttl"`
		Idle      string `xml:"idle"`

		// Flags
		Offloaded     string `xml:"offloaded"`
		DecryptMirror string `xml:"decrypt-mirror"`
		L7Processing  string `xml:"layer7-processing"`
	}

	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil {
		return nil, fmt.Errorf("failed to parse session detail: %w", err)
	}

	detail := &models.SessionDetail{
		ID:          result.Idx,
		State:       result.State,
		Application: result.Application,
		Protocol:    protoToName(result.Proto),
		Type:        result.Type,

		SourceIP:     result.Source,
		SourcePort:   result.Sport,
		DestIP:       result.Destination,
		DestPort:     result.Dport,
		SourceZone:   result.SourceZone,
		DestZone:     result.DestZone,
		SourceUser:   result.Srcuser,
		SourceHostID: result.SrcHostID,

		NATSourceIP:   result.XSource,
		NATSourcePort: result.XSport,
		NATDestIP:     result.XDst,
		NATDestPort:   result.XDport,
		NATRule:       result.NATRule,

		SecurityRule:     result.SecurityRule,
		URLCategory:      result.URLCategory,
		URLFilteringRule: result.URLFilterRule,
		DecryptionRule:   result.DecryptionRule,

		PacketsToClient: result.ToClientPkts,
		PacketsToServer: result.ToServerPkts,
		BytesToClient:   result.ToClientBytes,
		BytesToServer:   result.ToServerBytes,
		TotalBytes:      result.TotalByteCount,
		TotalPackets:    result.ToClientPkts + result.ToServerPkts,

		Timeout:    result.Timeout,
		TimeToLive: result.TTL,

		Offloaded:      strings.ToLower(result.Offloaded) == "yes" || strings.ToLower(result.Offloaded) == "true",
		DecryptMirror:  strings.ToLower(result.DecryptMirror) == "yes" || strings.ToLower(result.DecryptMirror) == "true",
		LayerSevenInfo: result.L7Processing,
	}

	// Parse start time
	if result.StartTime != "" {
		if t, err := time.Parse("Mon Jan 2 15:04:05 2006", result.StartTime); err == nil {
			detail.StartTime = t
		}
	}

	// Parse idle time (format: "XXs" or similar)
	if result.Idle != "" {
		idleStr := strings.TrimSuffix(result.Idle, "s")
		if idle, err := strconv.Atoi(idleStr); err == nil {
			detail.IdleTime = idle
		}
	}

	return detail, nil
}
