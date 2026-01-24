package api

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"strconv"
	"strings"

	"github.com/jp2195/pyre/internal/models"
)

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
	// Try Advanced Routing commands first (PAN-OS 10.2+), then legacy
	commands := []string{
		// Advanced Routing mode (PAN-OS 10.2+)
		"<show><advanced-routing><route></route></advanced-routing></show>",
		// Legacy routing mode
		"<show><routing><route></route></routing></show>",
	}

	var resp *XMLResponse
	var err error
	var isAdvancedRouting bool
	for _, cmd := range commands {
		resp, err = c.Op(ctx, cmd)
		if err == nil && resp.IsSuccess() && len(resp.Result.Inner) > 0 {
			// Check if response indicates deprecated command
			inner := string(resp.Result.Inner)
			if !strings.Contains(inner, "deprecated") && !strings.Contains(inner, "Command deprecated") {
				isAdvancedRouting = strings.Contains(cmd, "advanced-routing")
				break
			}
		}
	}

	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	if len(resp.Result.Inner) == 0 {
		return []models.RouteEntry{}, nil
	}

	// Advanced Routing returns JSON wrapped in XML
	if isAdvancedRouting {
		return parseAdvancedRoutingJSON(resp.Result.Inner)
	}

	// Legacy XML parsing
	return parseLegacyRoutingXML(resp.Result.Inner)
}

// parseAdvancedRoutingJSON parses the JSON format from Advanced Routing mode
func parseAdvancedRoutingJSON(inner []byte) ([]models.RouteEntry, error) {
	// Extract JSON from <json>...</json> wrapper
	var jsonWrapper struct {
		JSON string `xml:"json"`
	}
	if err := xml.Unmarshal(WrapInner(inner), &jsonWrapper); err != nil {
		return []models.RouteEntry{}, nil
	}

	if jsonWrapper.JSON == "" {
		return []models.RouteEntry{}, nil
	}

	// Parse the JSON structure: map[logicalRouter]map[prefix][]routeInfo
	var routeData map[string]map[string][]struct {
		Prefix    string `json:"prefix"`
		Protocol  string `json:"protocol"`
		VRFName   string `json:"vrfName"`
		Distance  int    `json:"distance"`
		Metric    int    `json:"metric"`
		Installed bool   `json:"installed"`
		Selected  bool   `json:"selected"`
		Uptime    string `json:"uptime"`
		Nexthops  []struct {
			InterfaceName     string `json:"interfaceName"`
			IP                string `json:"ip"`
			Active            bool   `json:"active"`
			DirectlyConnected bool   `json:"directlyConnected"`
		} `json:"nexthops"`
	}

	if err := json.Unmarshal([]byte(jsonWrapper.JSON), &routeData); err != nil {
		return []models.RouteEntry{}, nil
	}

	var routes []models.RouteEntry
	for vrName, prefixes := range routeData {
		for _, routeInfos := range prefixes {
			for _, r := range routeInfos {
				// Get the first nexthop info
				var nexthop, iface string
				if len(r.Nexthops) > 0 {
					nh := r.Nexthops[0]
					iface = nh.InterfaceName
					if nh.IP != "" {
						nexthop = nh.IP
					} else if nh.DirectlyConnected {
						nexthop = "directly connected"
					}
				}

				routes = append(routes, models.RouteEntry{
					Destination:   r.Prefix,
					Nexthop:       nexthop,
					Metric:        r.Metric,
					Interface:     iface,
					Protocol:      r.Protocol,
					VirtualRouter: vrName,
					Flags:         r.Uptime,
					Age:           0,
				})
			}
		}
	}

	return routes, nil
}

// parseLegacyRoutingXML parses the XML format from legacy routing mode
func parseLegacyRoutingXML(inner []byte) ([]models.RouteEntry, error) {
	// Define entry structure supporting multiple PAN-OS response formats
	type routeEntry struct {
		Destination   string `xml:"destination"`
		Nexthop       string `xml:"nexthop"`
		NextHop       string `xml:"next-hop"`
		Metric        int    `xml:"metric"`
		Interface     string `xml:"interface"`
		Flags         string `xml:"flags"`
		Age           string `xml:"age"`
		VirtualRouter string `xml:"virtual-router"`
		VR            string `xml:"vr"`
		Protocol      string `xml:"protocol"`
		RouteTable    string `xml:"route-table"`
	}

	var entries []routeEntry

	// Try multiple parsing approaches
	parseAttempts := []func() []routeEntry{
		// Direct entry elements
		func() []routeEntry {
			var r struct {
				Entry []routeEntry `xml:"entry"`
			}
			if xml.Unmarshal(WrapInner(inner), &r) == nil {
				return r.Entry
			}
			return nil
		},
		// With route-table wrapper
		func() []routeEntry {
			var r struct {
				Entry []routeEntry `xml:"route-table>entry"`
			}
			if xml.Unmarshal(WrapInner(inner), &r) == nil {
				return r.Entry
			}
			return nil
		},
		// With dp (dataplane) entries
		func() []routeEntry {
			var r struct {
				DP struct {
					Entry []routeEntry `xml:"entry"`
				} `xml:"dp"`
			}
			if xml.Unmarshal(WrapInner(inner), &r) == nil {
				return r.DP.Entry
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

	routes := make([]models.RouteEntry, 0, len(entries))
	for _, e := range entries {
		nexthop := e.Nexthop
		if nexthop == "" {
			nexthop = e.NextHop
		}

		vr := e.VirtualRouter
		if vr == "" {
			vr = e.VR
		}

		protocol := e.Protocol
		if protocol == "" {
			protocol = "static"
			flags := strings.ToLower(e.Flags)
			if strings.Contains(flags, "c") {
				protocol = "connected"
			} else if strings.Contains(flags, "l") {
				protocol = "local"
			} else if strings.Contains(flags, "b") {
				protocol = "bgp"
			} else if strings.Contains(flags, "o") {
				protocol = "ospf"
			}
		}

		var ageSeconds int
		if e.Age != "" {
			if age, err := strconv.Atoi(e.Age); err == nil {
				ageSeconds = age
			}
		}

		routes = append(routes, models.RouteEntry{
			Destination:   e.Destination,
			Nexthop:       nexthop,
			Metric:        e.Metric,
			Interface:     e.Interface,
			Protocol:      protocol,
			VirtualRouter: vr,
			Flags:         e.Flags,
			Age:           ageSeconds,
		})
	}

	return routes, nil
}
