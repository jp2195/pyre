package api

import (
	"bytes"
	"context"
	"log"
	"strings"
	"time"

	"github.com/jp2195/pyre/internal/models"
)

// GetIPSecTunnels retrieves IPSec VPN tunnel status
func (c *Client) GetIPSecTunnels(ctx context.Context, target string) ([]models.IPSecTunnel, error) {
	resp, err := c.Op(ctx, "<show><vpn><ipsec-sa></ipsec-sa></vpn></show>", target)
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
			Name         string `xml:"name"`
			Gateway      string `xml:"gateway"`
			LocalSPI     string `xml:"local-spi"`
			RemoteSPI    string `xml:"peer-spi"`
			Protocol     string `xml:"protocol"`
			Encryption   string `xml:"enc-algo"`
			Auth         string `xml:"auth-algo"`
			TunnelIf     string `xml:"tunnel-interface"`
			State        string `xml:"state"`
			Lifesize     string `xml:"lifesize"`
			Lifetime     string `xml:"lifetime"`
			EncapBytes   int64  `xml:"encap-bytes"`
			DecapBytes   int64  `xml:"decap-bytes"`
			EncapPackets int64  `xml:"encap-pkts"`
			DecapPackets int64  `xml:"decap-pkts"`
		} `xml:"entries>entry"`
	}

	if err := decodeXML(bytes.NewReader(WrapInner(resp.Result.Inner)), &result); err != nil {
		log.Printf("[API Warning] failed to parse IPSec tunnels XML: %v", err)
		return []models.IPSecTunnel{}, nil
	}

	tunnels := make([]models.IPSecTunnel, 0, len(result.Entry))
	for _, e := range result.Entry {
		lowerState := strings.ToLower(e.State)
		state := "down"
		if strings.Contains(lowerState, "active") ||
			strings.Contains(lowerState, "up") ||
			strings.Contains(lowerState, "established") {
			state = "up"
		} else if strings.Contains(lowerState, "init") {
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
			BytesOut:   e.EncapBytes, // Encap = outbound traffic
			BytesIn:    e.DecapBytes, // Decap = inbound traffic
			PacketsOut: e.EncapPackets,
			PacketsIn:  e.DecapPackets,
		})
	}

	return tunnels, nil
}

// GetGlobalProtectUsers retrieves detailed GlobalProtect user information
func (c *Client) GetGlobalProtectUsers(ctx context.Context, target string) ([]models.GlobalProtectUser, error) {
	resp, err := c.Op(ctx, "<show><global-protect-gateway><current-user></current-user></global-protect-gateway></show>", target)
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

	if err := decodeXML(bytes.NewReader(WrapInner(resp.Result.Inner)), &result); err != nil {
		log.Printf("[API Warning] failed to parse GlobalProtect users XML: %v", err)
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
		if t, err := parsePANTime(e.LoginTime); err == nil {
			user.LoginTime = t
			user.Duration = formatDuration(time.Since(t))
		}

		users = append(users, user)
	}

	return users, nil
}
