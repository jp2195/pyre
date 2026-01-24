package api

import (
	"context"
	"encoding/xml"
	"strings"

	"github.com/jp2195/pyre/internal/models"
)

func (c *Client) GetInterfaces(ctx context.Context) ([]models.Interface, error) {
	resp, err := c.Op(ctx, "<show><interface>all</interface></show>")
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	// Handle empty result
	if len(resp.Result.Inner) == 0 {
		return []models.Interface{}, nil
	}

	var result struct {
		Ifnet struct {
			Entry []struct {
				Name  string `xml:"name"`
				Zone  string `xml:"zone"`
				Vsys  string `xml:"vsys"`
				IP    string `xml:"ip"`
				State string `xml:"state"`
				Speed string `xml:"speed"`
				Fwd   string `xml:"fwd"` // virtual router
				MTU   int    `xml:"mtu"`
				Mode  string `xml:"mode"`
				Tag   int    `xml:"tag"`
			} `xml:"entry"`
		} `xml:"ifnet"`
		HW struct {
			Entry []struct {
				Name   string `xml:"name"`
				State  string `xml:"state"`
				Speed  string `xml:"speed"`
				Duplex string `xml:"duplex"`
				MAC    string `xml:"mac"`
				Mode   string `xml:"mode"`
				Type   string `xml:"type"`
			} `xml:"entry"`
		} `xml:"hw"`
	}
	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil {
		// Return empty list rather than error
		return []models.Interface{}, nil
	}

	// Build HW map for hardware details
	hwMap := make(map[string]struct {
		state  string
		speed  string
		duplex string
		mac    string
		mode   string
		ifType string
	})
	for _, hw := range result.HW.Entry {
		hwMap[hw.Name] = struct {
			state  string
			speed  string
			duplex string
			mac    string
			mode   string
			ifType string
		}{hw.State, hw.Speed, hw.Duplex, hw.MAC, hw.Mode, hw.Type}
	}

	interfaces := make([]models.Interface, 0, len(result.Ifnet.Entry))
	for _, e := range result.Ifnet.Entry {
		iface := models.Interface{
			Name:          e.Name,
			Zone:          e.Zone,
			Vsys:          e.Vsys,
			IP:            e.IP,
			State:         e.State,
			Speed:         e.Speed,
			VirtualRouter: e.Fwd,
			MTU:           e.MTU,
			Mode:          e.Mode,
			Tag:           e.Tag,
		}

		// Determine interface type from name
		switch {
		case strings.HasPrefix(e.Name, "ethernet"):
			iface.Type = "ethernet"
		case strings.HasPrefix(e.Name, "loopback"):
			iface.Type = "loopback"
		case strings.HasPrefix(e.Name, "tunnel"):
			iface.Type = "tunnel"
		case strings.HasPrefix(e.Name, "vlan"):
			iface.Type = "vlan"
		case strings.HasPrefix(e.Name, "ae"):
			iface.Type = "aggregate"
		}

		// Merge hardware details
		if hw, ok := hwMap[e.Name]; ok {
			if iface.State == "" {
				iface.State = hw.state
			}
			if iface.Speed == "" {
				iface.Speed = hw.speed
			}
			iface.Duplex = hw.duplex
			iface.MAC = hw.mac
			if iface.Mode == "" {
				iface.Mode = hw.mode
			}
			if iface.Type == "" && hw.ifType != "" {
				iface.Type = hw.ifType
			}
		}
		interfaces = append(interfaces, iface)
	}

	// Fetch counters separately
	c.fetchInterfaceCounters(ctx, interfaces)

	return interfaces, nil
}

// fetchInterfaceCounters fetches counter data for interfaces
func (c *Client) fetchInterfaceCounters(ctx context.Context, interfaces []models.Interface) {
	resp, err := c.Op(ctx, "<show><counter><interface>all</interface></counter></show>")
	if err != nil || CheckResponse(resp) != nil {
		return
	}

	var result struct {
		Ifnet struct {
			Entry []struct {
				Name     string `xml:"name"`
				Ibytes   int64  `xml:"ibytes"`
				Obytes   int64  `xml:"obytes"`
				Ipackets int64  `xml:"ipackets"`
				Opackets int64  `xml:"opackets"`
				Ierrors  int64  `xml:"ierrors"`
				Idrops   int64  `xml:"idrops"`
				Oerrors  int64  `xml:"oerrors"`
				Odrops   int64  `xml:"odrops"`
			} `xml:"entry"`
		} `xml:"ifnet"`
	}
	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil {
		return
	}

	// Build counter map
	counterMap := make(map[string]struct {
		bytesIn, bytesOut     int64
		packetsIn, packetsOut int64
		errorsIn, errorsOut   int64
		dropsIn, dropsOut     int64
	})
	for _, e := range result.Ifnet.Entry {
		counterMap[e.Name] = struct {
			bytesIn, bytesOut     int64
			packetsIn, packetsOut int64
			errorsIn, errorsOut   int64
			dropsIn, dropsOut     int64
		}{e.Ibytes, e.Obytes, e.Ipackets, e.Opackets, e.Ierrors, e.Oerrors, e.Idrops, e.Odrops}
	}

	// Update interfaces with counters
	for i := range interfaces {
		if c, ok := counterMap[interfaces[i].Name]; ok {
			interfaces[i].BytesIn = c.bytesIn
			interfaces[i].BytesOut = c.bytesOut
			interfaces[i].PacketsIn = c.packetsIn
			interfaces[i].PacketsOut = c.packetsOut
			interfaces[i].ErrorsIn = c.errorsIn
			interfaces[i].ErrorsOut = c.errorsOut
			interfaces[i].DropsIn = c.dropsIn
			interfaces[i].DropsOut = c.dropsOut
		}
	}
}
