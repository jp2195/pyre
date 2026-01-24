package api

import (
	"context"
	"encoding/xml"

	"github.com/jp2195/pyre/internal/models"
)

func (c *Client) GetHAStatus(ctx context.Context) (*models.HAStatus, error) {
	resp, err := c.Op(ctx, "<show><high-availability><state></state></high-availability></show>")
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	// Handle empty result
	if len(resp.Result.Inner) == 0 {
		return &models.HAStatus{Enabled: false}, nil
	}

	var result struct {
		Enabled string `xml:"enabled"`
		Group   struct {
			LocalInfo struct {
				State string `xml:"state"`
			} `xml:"local-info"`
			PeerInfo struct {
				State string `xml:"state"`
			} `xml:"peer-info"`
			RunningSyncEnabled string `xml:"running-sync-enabled"`
			RunningSyncState   string `xml:"running-sync"`
		} `xml:"group"`
	}
	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil {
		// Return disabled HA if parsing fails
		return &models.HAStatus{Enabled: false}, nil
	}

	return &models.HAStatus{
		Enabled:   result.Enabled == "yes",
		State:     result.Group.LocalInfo.State,
		PeerState: result.Group.PeerInfo.State,
		SyncState: result.Group.RunningSyncState,
	}, nil
}
