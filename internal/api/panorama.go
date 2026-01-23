package api

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/joshuamontgomery/pyre/internal/models"
)

// GetManagedDevices fetches all devices managed by Panorama.
func (c *Client) GetManagedDevices(ctx context.Context) ([]models.ManagedDevice, error) {
	resp, err := c.Op(ctx, "<show><devices><all></all></devices></show>")
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	var result struct {
		Devices struct {
			Entry []struct {
				Name        string `xml:"name,attr"`
				Serial      string `xml:"serial"`
				Hostname    string `xml:"hostname"`
				IPAddress   string `xml:"ip-address"`
				Model       string `xml:"model"`
				SWVersion   string `xml:"sw-version"`
				HAState     string `xml:"ha>state"`
				Connected   string `xml:"connected"`
				DeviceGroup string `xml:"device-group"`
			} `xml:"entry"`
		} `xml:"devices"`
	}
	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil {
		return nil, fmt.Errorf("parsing managed devices: %w", err)
	}

	devices := make([]models.ManagedDevice, 0, len(result.Devices.Entry))
	for _, e := range result.Devices.Entry {
		serial := e.Serial
		if serial == "" {
			serial = e.Name // Sometimes serial is in name attr
		}
		devices = append(devices, models.ManagedDevice{
			Serial:      serial,
			Hostname:    e.Hostname,
			IPAddress:   e.IPAddress,
			Model:       e.Model,
			SWVersion:   e.SWVersion,
			HAState:     e.HAState,
			Connected:   e.Connected == "yes",
			DeviceGroup: e.DeviceGroup,
		})
	}

	return devices, nil
}

// IsPanoramaModel returns true if the model string indicates a Panorama appliance.
func IsPanoramaModel(model string) bool {
	lower := strings.ToLower(model)
	return strings.Contains(lower, "panorama") || strings.HasPrefix(model, "M-")
}
