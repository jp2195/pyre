package api

import (
	"context"
	"encoding/xml"
	"strings"
	"time"

	"github.com/jp2195/pyre/internal/models"
)

// GetIPSecTunnels retrieves IPSec VPN tunnel status
func (c *Client) GetIPSecTunnels(ctx context.Context) ([]models.IPSecTunnel, error) {
	resp, err := c.Op(ctx, "<show><vpn><ipsec-sa></ipsec-sa></vpn></show>")
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	if len(resp.Result.Inner) == 0 {
		return []models.IPSecTunnel{}, nil
	}

	var result struct {
		Entry []struct {
			Name       string `xml:"name"`
			Gateway    string `xml:"gateway"`
			LocalSPI   string `xml:"local-spi"`
			RemoteSPI  string `xml:"peer-spi"`
			Protocol   string `xml:"protocol"`
			Encryption string `xml:"enc-algo"`
			Auth       string `xml:"auth-algo"`
			TunnelIf   string `xml:"tunnel-interface"`
			State      string `xml:"state"`
			Lifesize   string `xml:"lifesize"`
			Lifetime   string `xml:"lifetime"`
		} `xml:"entries>entry"`
	}

	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil {
		return []models.IPSecTunnel{}, nil
	}

	tunnels := make([]models.IPSecTunnel, 0, len(result.Entry))
	for _, e := range result.Entry {
		state := "down"
		if strings.Contains(strings.ToLower(e.State), "active") || strings.Contains(strings.ToLower(e.State), "up") {
			state = "up"
		} else if strings.Contains(strings.ToLower(e.State), "init") {
			state = "init"
		}

		tunnels = append(tunnels, models.IPSecTunnel{
			Name:       e.Name,
			Gateway:    e.Gateway,
			State:      state,
			LocalSPI:   e.LocalSPI,
			RemoteSPI:  e.RemoteSPI,
			Protocol:   e.Protocol,
			Encryption: e.Encryption,
			Auth:       e.Auth,
			Uptime:     e.Lifetime,
		})
	}

	return tunnels, nil
}

// GetGlobalProtectUsers retrieves detailed GlobalProtect user information
func (c *Client) GetGlobalProtectUsers(ctx context.Context) ([]models.GlobalProtectUser, error) {
	resp, err := c.Op(ctx, "<show><global-protect-gateway><current-user></current-user></global-protect-gateway></show>")
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	if len(resp.Result.Inner) == 0 {
		return []models.GlobalProtectUser{}, nil
	}

	var result struct {
		Entry []struct {
			Username  string `xml:"username"`
			Domain    string `xml:"domain"`
			Computer  string `xml:"computer"`
			Client    string `xml:"client"`
			VirtualIP string `xml:"virtual-ip"`
			PublicIP  string `xml:"public-ip"`
			LoginTime string `xml:"login-time"`
			Gateway   string `xml:"gateway"`
			Region    string `xml:"source-region"`
		} `xml:"entry"`
	}

	if err := xml.Unmarshal(WrapInner(resp.Result.Inner), &result); err != nil {
		return []models.GlobalProtectUser{}, nil
	}

	users := make([]models.GlobalProtectUser, 0, len(result.Entry))
	for _, e := range result.Entry {
		user := models.GlobalProtectUser{
			Username:     e.Username,
			Domain:       e.Domain,
			Computer:     e.Computer,
			Client:       e.Client,
			VirtualIP:    e.VirtualIP,
			ClientIP:     e.PublicIP,
			Gateway:      e.Gateway,
			SourceRegion: e.Region,
		}

		// Parse login time
		for _, layout := range []string{"2006/01/02 15:04:05", "Mon Jan 2 15:04:05 2006"} {
			if t, err := time.Parse(layout, e.LoginTime); err == nil {
				user.LoginTime = t
				user.Duration = formatDuration(time.Since(t))
				break
			}
		}

		users = append(users, user)
	}

	return users, nil
}
