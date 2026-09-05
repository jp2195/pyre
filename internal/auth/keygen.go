package auth

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jp2195/pyre/internal/api"
)

type KeygenResult struct {
	APIKey string
	Error  error
}

type keygenResponse struct {
	XMLName xml.Name `xml:"response"`
	Status  string   `xml:"status,attr"`
	Result  struct {
		Key string `xml:"key"`
	} `xml:"result"`
	Msg struct {
		Line string `xml:"line"`
	} `xml:"msg"`
}

// maxKeygenResponseSize caps the keygen response read. Real keygen responses
// are well under 4KB; 1MB leaves generous headroom while preventing an
// unverified endpoint from streaming an unbounded body during login.
const maxKeygenResponseSize = 1 << 20

// GenerateAPIKey performs the PAN-OS keygen exchange for host using the
// supplied credentials. TLS behavior is governed by opts exactly as in
// api.NewClient: verified by default, custom CA via opts.CACertPath
// (fail-closed), or opts.Insecure to skip verification.
func GenerateAPIKey(ctx context.Context, host, username, password string, opts api.ClientOptions) (*KeygenResult, error) {
	tr, err := api.NewTransport(opts)
	if err != nil {
		return nil, fmt.Errorf("configuring keygen TLS: %w", err)
	}
	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: tr,
	}

	// Use POST with form body to keep credentials out of URLs/logs.
	// Shares api.BaseURL so keygen and the API client agree on host
	// formatting, including bracketing bare IPv6 literals.
	reqURL := api.BaseURL(host)
	formData := url.Values{}
	formData.Set("type", "keygen")
	formData.Set("user", username)
	formData.Set("password", password)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating keygen request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", api.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("keygen request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }() //nolint:errcheck // best effort cleanup

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxKeygenResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading keygen response: %w", err)
	}

	var xmlResp keygenResponse
	if err := api.DecodeXML(bytes.NewReader(body), &xmlResp); err != nil {
		return nil, fmt.Errorf("parsing keygen response: %w", err)
	}

	if xmlResp.Status != "success" {
		// Sanitize before surfacing: PAN-OS (or a MITM) could embed ANSI
		// escapes or control bytes in <msg><line>, which would otherwise
		// flow unchanged into the TUI login error pane.
		errMsg := api.SanitizeForDisplay(xmlResp.Msg.Line)
		if errMsg == "" {
			errMsg = "authentication failed"
		}
		return &KeygenResult{Error: fmt.Errorf("%s", errMsg)}, nil
	}

	if xmlResp.Result.Key == "" {
		return nil, fmt.Errorf("empty API key in response")
	}

	return &KeygenResult{APIKey: xmlResp.Result.Key}, nil
}
