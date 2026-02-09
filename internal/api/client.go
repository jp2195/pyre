package api

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// maxResponseSize is the maximum allowed response body size (50 MB).
const maxResponseSize = 50 * 1024 * 1024

// Client represents a PAN-OS API client.
// Fields are ordered for optimal memory alignment on 64-bit systems.
type Client struct {
	baseURL      string       // 16 bytes (string header)
	apiKey       string       // 16 bytes (string header)
	targetSerial string       // 16 bytes (string header) - For Panorama: routes API calls to specific device
	httpClient   *http.Client // 8 bytes (pointer)
}

type ClientOption func(*Client)

func WithInsecure(insecure bool) ClientOption {
	return func(c *Client) {
		if insecure {
			c.httpClient.Transport = &http.Transport{
				// #nosec G402 -- InsecureSkipVerify required for self-signed firewall certificates when user enables --insecure
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			}
		}
	}
}

func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

func NewClient(host, apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		baseURL: fmt.Sprintf("https://%s/api/", host),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Close releases resources associated with the client.
// It closes idle HTTP connections to free up system resources.
func (c *Client) Close() error {
	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
	}
	return nil
}

type XMLResponse struct {
	Status string `xml:"status,attr"`
	Code   string `xml:"code,attr"`
	Result struct {
		Inner []byte `xml:",innerxml"`
	} `xml:"result"`
	Msg struct {
		Line string `xml:"line"`
	} `xml:"msg"`
}

func (r *XMLResponse) IsSuccess() bool {
	return r.Status == "success"
}

func (r *XMLResponse) Error() string {
	if r.Msg.Line != "" {
		return r.Msg.Line
	}
	return fmt.Sprintf("API error: status=%s code=%s", r.Status, r.Code)
}

// SetTarget sets the target serial number for Panorama API routing.
func (c *Client) SetTarget(serial string) {
	c.targetSerial = serial
}

// ClearTarget clears the target serial number.
func (c *Client) ClearTarget() {
	c.targetSerial = ""
}

// GetTarget returns the current target serial number.
func (c *Client) GetTarget() string {
	return c.targetSerial
}

func (c *Client) request(ctx context.Context, params url.Values) (*XMLResponse, error) {
	start := time.Now()

	// Inject target parameter for Panorama routing
	if c.targetSerial != "" {
		params.Set("target", c.targetSerial)
	}

	// Log request (sanitized - no API key)
	log.Printf("[API Request] type=%s action=%s xpath=%s target=%s",
		params.Get("type"),
		params.Get("action"),
		params.Get("xpath"),
		c.targetSerial,
	)
	if cmd := params.Get("cmd"); cmd != "" {
		log.Printf("[API Request] cmd=%s", truncateLog(cmd, 500))
	}

	reqURL := c.baseURL + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		log.Printf("[API Error] creating request: %v", err)
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Use X-PAN-KEY header instead of query parameter (PAN-OS 8.0+)
	// This prevents API key from appearing in server/proxy logs
	req.Header.Set("X-PAN-KEY", c.apiKey) // NOT logged

	resp, err := c.httpClient.Do(req)
	duration := time.Since(start)

	if err != nil {
		log.Printf("[API Error] request failed after %dms: %v", duration.Milliseconds(), err)
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }() //nolint:errcheck // best effort cleanup

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		log.Printf("[API Error] reading response: %v", err)
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var xmlResp XMLResponse
	if err := xml.Unmarshal(body, &xmlResp); err != nil {
		log.Printf("[API Error] parsing XML after %dms: %v", duration.Milliseconds(), err)
		log.Printf("[API Error] body preview: %s", truncateLog(string(body), 500))
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	// Log response with timing
	log.Printf("[API Response] status=%s code=%s duration=%dms size=%d bytes",
		xmlResp.Status,
		xmlResp.Code,
		duration.Milliseconds(),
		len(body),
	)
	if !xmlResp.IsSuccess() {
		log.Printf("[API Response] error: %s", xmlResp.Msg.Line)
	}
	if len(xmlResp.Result.Inner) > 0 {
		log.Printf("[API Response] body preview: %s", truncateLog(string(xmlResp.Result.Inner), 1000))
	}

	return &xmlResp, nil
}

// truncateLog truncates a string to maxLen characters, appending a truncation indicator if needed.
func truncateLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...[truncated]"
}

func (c *Client) Op(ctx context.Context, cmd string) (*XMLResponse, error) {
	params := url.Values{}
	params.Set("type", "op")
	params.Set("cmd", cmd)
	return c.request(ctx, params)
}

func (c *Client) Get(ctx context.Context, xpath string) (*XMLResponse, error) {
	params := url.Values{}
	params.Set("type", "config")
	params.Set("action", "get")
	params.Set("xpath", xpath)
	return c.request(ctx, params)
}

func (c *Client) Show(ctx context.Context, xpath string) (*XMLResponse, error) {
	params := url.Values{}
	params.Set("type", "config")
	params.Set("action", "show")
	params.Set("xpath", xpath)
	return c.request(ctx, params)
}

// Log submits a log query. Returns a job ID that can be polled for results.
func (c *Client) Log(ctx context.Context, logType string, nlogs int, query string) (*XMLResponse, error) {
	params := url.Values{}
	params.Set("type", "log")
	params.Set("log-type", logType)
	if nlogs > 0 {
		params.Set("nlogs", fmt.Sprintf("%d", nlogs))
	}
	if query != "" {
		params.Set("query", query)
	}
	return c.request(ctx, params)
}

// LogGet retrieves results of a log query job.
func (c *Client) LogGet(ctx context.Context, jobID string) (*XMLResponse, error) {
	params := url.Values{}
	params.Set("type", "log")
	params.Set("action", "get")
	params.Set("job-id", jobID)
	return c.request(ctx, params)
}

type APIError struct {
	Status  string
	Code    string
	Message string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("API error: status=%s code=%s", e.Status, e.Code)
}

func CheckResponse(resp *XMLResponse) error {
	if resp.IsSuccess() {
		return nil
	}
	return &APIError{
		Status:  resp.Status,
		Code:    resp.Code,
		Message: resp.Msg.Line,
	}
}

// WrapInner wraps the inner XML content in a root element for proper parsing.
// This is needed because Result.Inner contains raw XML without a wrapper.
func WrapInner(inner []byte) []byte {
	return append(append([]byte("<root>"), inner...), []byte("</root>")...)
}
