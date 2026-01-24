package api

import (
	"context"
	"encoding/xml"
	"strings"
	"time"

	"github.com/jp2195/pyre/internal/models"
)

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

// GetNATPoolInfo retrieves NAT IP pool utilization information
func (c *Client) GetNATPoolInfo(ctx context.Context) ([]models.NATPoolInfo, error) {
	// Try multiple API commands - different PAN-OS versions use different paths
	// nat-rule-ippool works for both dedicated pools AND interface-based DIPP
	commands := []string{
		"<show><running><nat-rule-ippool></nat-rule-ippool></running></show>",
		"<show><running><ippool></ippool></running></show>",
		"<show><running><nat><ippool></ippool></nat></running></show>",
	}

	var resp *XMLResponse
	var err error
	for _, cmd := range commands {
		resp, err = c.Op(ctx, cmd)
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

	if len(resp.Result.Inner) == 0 {
		return []models.NATPoolInfo{}, nil
	}

	// Define entry structure supporting multiple PAN-OS response formats
	type poolEntry struct {
		RuleName   string `xml:"rule"`
		Name       string `xml:"name,attr"`
		NameElem   string `xml:"name"`
		Type       string `xml:"pool-type"`
		PoolType   string `xml:"type"`
		StartIP    string `xml:"start-ip"`
		EndIP      string `xml:"end-ip"`
		Allocated  int64  `xml:"num-used-addresses"`
		Available  int64  `xml:"num-available-addresses"`
		PortsUsed  int64  `xml:"num-used-ports"`
		PortsAvail int64  `xml:"num-available-ports"`
		// Alternative field names used in some versions
		UsedAddr   int64 `xml:"used-addresses"`
		AvailAddr  int64 `xml:"available-addresses"`
		UsedPorts  int64 `xml:"used-ports"`
		AvailPorts int64 `xml:"available-ports"`
		// nat-rule-ippool specific fields
		DP          string `xml:"dp"`
		AllocatedIP string `xml:"allocated-ip"`
		PortRange   string `xml:"port-range"`
		NumPorts    int64  `xml:"num-ports"`
		NumUsed     int64  `xml:"num-used"`
		NumFree     int64  `xml:"num-free"`
	}

	var entries []poolEntry

	// Try multiple parsing approaches for different response formats
	parseAttempts := []func() []poolEntry{
		// Direct entry elements
		func() []poolEntry {
			var r struct {
				Entry []poolEntry `xml:"entry"`
			}
			if xml.Unmarshal(WrapInner(resp.Result.Inner), &r) == nil {
				return r.Entry
			}
			return nil
		},
		// With ippool wrapper
		func() []poolEntry {
			var r struct {
				Entry []poolEntry `xml:"ippool>entry"`
			}
			if xml.Unmarshal(WrapInner(resp.Result.Inner), &r) == nil {
				return r.Entry
			}
			return nil
		},
		// With nat-rule-ippool wrapper
		func() []poolEntry {
			var r struct {
				Entry []poolEntry `xml:"nat-rule-ippool>entry"`
			}
			if xml.Unmarshal(WrapInner(resp.Result.Inner), &r) == nil {
				return r.Entry
			}
			return nil
		},
		// With rules wrapper (common in nat-rule-ippool)
		func() []poolEntry {
			var r struct {
				Entry []poolEntry `xml:"rules>entry"`
			}
			if xml.Unmarshal(WrapInner(resp.Result.Inner), &r) == nil {
				return r.Entry
			}
			return nil
		},
		// With dp (data plane) entries
		func() []poolEntry {
			var r struct {
				DP []struct {
					Entry []poolEntry `xml:"entry"`
				} `xml:"dp"`
			}
			if xml.Unmarshal(WrapInner(resp.Result.Inner), &r) == nil && len(r.DP) > 0 {
				return r.DP[0].Entry
			}
			return nil
		},
	}

	for _, parse := range parseAttempts {
		if result := parse(); len(result) > 0 {
			entries = result
			break
		}
	}

	pools := make([]models.NATPoolInfo, 0, len(entries))
	for _, e := range entries {
		pool := models.NATPoolInfo{
			RuleName: e.RuleName,
			Type:     e.Type,
		}

		// Use Name (attr or element) if RuleName is empty
		if pool.RuleName == "" {
			if e.Name != "" {
				pool.RuleName = e.Name
			} else if e.NameElem != "" {
				pool.RuleName = e.NameElem
			}
		}

		// Use PoolType if Type is empty
		if pool.Type == "" && e.PoolType != "" {
			pool.Type = e.PoolType
		}

		// Determine used/available from whichever fields are populated
		portsUsed := e.PortsUsed
		if portsUsed == 0 {
			portsUsed = e.UsedPorts
		}
		if portsUsed == 0 {
			portsUsed = e.NumUsed
		}
		portsAvail := e.PortsAvail
		if portsAvail == 0 {
			portsAvail = e.AvailPorts
		}
		if portsAvail == 0 {
			portsAvail = e.NumFree
		}
		// If we have NumPorts (total), calculate available
		if portsAvail == 0 && e.NumPorts > 0 && portsUsed > 0 {
			portsAvail = e.NumPorts - portsUsed
		}

		addrUsed := e.Allocated
		if addrUsed == 0 {
			addrUsed = e.UsedAddr
		}
		addrAvail := e.Available
		if addrAvail == 0 {
			addrAvail = e.AvailAddr
		}

		// DIPP (Dynamic IP and Port Pool) uses ports
		// DIP (Dynamic IP Pool) uses addresses
		poolType := strings.ToUpper(pool.Type)
		if strings.Contains(poolType, "DIPP") || strings.Contains(poolType, "DIP-PORT") ||
			portsUsed > 0 || portsAvail > 0 || e.NumPorts > 0 {
			pool.Used = portsUsed
			pool.Available = portsAvail
			if pool.Type == "" {
				pool.Type = "DIPP"
			}
		} else {
			pool.Used = addrUsed
			pool.Available = addrAvail
			if pool.Type == "" {
				pool.Type = "DIP"
			}
		}

		// Calculate utilization percentage
		total := pool.Used + pool.Available
		if total > 0 {
			pool.Percent = float64(pool.Used) / float64(total) * 100
		}

		// Only include pools with actual data
		if pool.RuleName != "" && total > 0 {
			pools = append(pools, pool)
		}
	}

	return pools, nil
}
