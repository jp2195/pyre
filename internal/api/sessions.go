package api

import (
	"context"
	"encoding/xml"
	"fmt"
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
		// Ignore parse error - time format may vary, zero time acceptable
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
