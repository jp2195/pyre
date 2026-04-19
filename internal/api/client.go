package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

// maxResponseSize is the maximum allowed response body size (50 MB).
const maxResponseSize = 50 * 1024 * 1024

// debugLogging enables per-request API trace logging when PYRE_DEBUG=1 (or
// "true") is set in the environment at process start. It is evaluated once
// because toggling it at runtime across goroutines would require a mutex or
// atomic, and debug logging is an opt-in developer tool, not a runtime knob.
var debugLogging = os.Getenv("PYRE_DEBUG") == "1" || os.Getenv("PYRE_DEBUG") == "true"

// debugf writes a trace line to the standard logger when debugLogging is on.
// Per-request logs may include PAN-OS config paths and op command bodies, so
// they are off by default.
func debugf(format string, args ...any) {
	if !debugLogging {
		return
	}
	log.Printf(format, args...)
}

// Client represents a PAN-OS API client.
// Fields are ordered for optimal memory alignment on 64-bit systems.
//
// The client is stateless with respect to Panorama target routing: each
// request method accepts an explicit target serial argument rather than
// consulting shared mutable state. This eliminates cross-goroutine bleed
// when multiple fetches run concurrently with different targets.
type Client struct {
	baseURL    string       // 16 bytes (string header)
	apiKey     string       // 16 bytes (string header)
	httpClient *http.Client // 8 bytes (pointer)
}

// ClientOptions carries optional knobs for NewClient. Zero value is safe:
// verified TLS, system roots, no custom CA.
type ClientOptions struct {
	// Insecure disables TLS certificate verification. Required for
	// firewalls that present self-signed certs without a trusted CA bundle.
	Insecure bool
	// CACertPath is the path to an optional PEM-encoded CA bundle used to
	// verify the firewall's certificate. If empty, system roots are used.
	// If the file is set but cannot be read or contains no parseable
	// certificates, NewClient returns an error rather than silently
	// falling back to system roots.
	CACertPath string
}

// newTransport builds an *http.Transport owned by a single Client with a
// hardened TLS config (MinVersion = TLS 1.2) applied from opts. It returns an
// error if opts.CACertPath is set but the CA bundle cannot be loaded, so
// configuration mistakes surface at connect time rather than as opaque TLS
// handshake failures on the first request.
func newTransport(opts ClientOptions) (*http.Transport, error) {
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if opts.Insecure {
		// #nosec G402 -- InsecureSkipVerify required for self-signed firewall certificates when user opts in
		tlsCfg.InsecureSkipVerify = true //nolint:gosec
	} else if opts.CACertPath != "" {
		pem, err := os.ReadFile(opts.CACertPath) // #nosec G304 -- path comes from user config
		if err != nil {
			return nil, fmt.Errorf("reading CA bundle %q: %w", opts.CACertPath, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("no certificates parsed from CA bundle %q", opts.CACertPath)
		}
		tlsCfg.RootCAs = pool
	}
	return &http.Transport{
		TLSClientConfig:       tlsCfg,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}, nil
}

// NewClient builds a PAN-OS XML API client for host using apiKey for
// authentication. The client owns its *http.Transport; callers must invoke
// Close to release idle connections when finished.
//
// An error is returned when opts.CACertPath is set but the CA bundle cannot
// be loaded (unreadable file or PEM contains zero certificates). When
// CACertPath is empty, NewClient uses system roots and never fails.
func NewClient(host, apiKey string, opts ClientOptions) (*Client, error) {
	tr, err := newTransport(opts)
	if err != nil {
		return nil, err
	}
	return &Client{
		baseURL: fmt.Sprintf("https://%s/api/", host),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		},
	}, nil
}

// Close releases resources associated with the client.
// It closes this client's idle HTTP connections (not DefaultTransport's).
func (c *Client) Close() error {
	if c.httpClient == nil {
		return nil
	}
	if tr, ok := c.httpClient.Transport.(*http.Transport); ok {
		tr.CloseIdleConnections()
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
		return SanitizeForDisplay(r.Msg.Line)
	}
	return fmt.Sprintf("API error: status=%s code=%s", r.Status, r.Code)
}

// request performs a PAN-OS XML API call. target is the Panorama-managed
// device serial the call should be routed to; pass "" for standalone
// firewalls and Panorama-local queries. Target is per-request to avoid the
// races that come with client-scoped mutable state.
func (c *Client) request(ctx context.Context, params url.Values, target string) (*XMLResponse, error) {
	start := time.Now()

	// Inject target parameter for Panorama routing
	if target != "" {
		params.Set("target", target)
	}

	// Log request (sanitized - no API key). Gated behind PYRE_DEBUG.
	debugf("[API Request] type=%s action=%s xpath=%s target=%s",
		params.Get("type"),
		params.Get("action"),
		params.Get("xpath"),
		target,
	)
	if cmd := params.Get("cmd"); cmd != "" {
		debugf("[API Request] cmd=%s", truncateLog(cmd, 500))
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
	if err := decodeXML(bytes.NewReader(body), &xmlResp); err != nil {
		log.Printf("[API Error] parsing XML after %dms: %v", duration.Milliseconds(), err)
		log.Printf("[API Error] body preview: %s", truncateLog(string(body), 500))
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	// Log response with timing. Gated behind PYRE_DEBUG.
	debugf("[API Response] status=%s code=%s duration=%dms size=%d bytes",
		xmlResp.Status,
		xmlResp.Code,
		duration.Milliseconds(),
		len(body),
	)
	if !xmlResp.IsSuccess() {
		debugf("[API Response] error: %s", SanitizeForDisplay(xmlResp.Msg.Line))
	}
	if len(xmlResp.Result.Inner) > 0 {
		debugf("[API Response] body preview: %s", truncateLog(string(xmlResp.Result.Inner), 1000))
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

// Op issues an operational ("op") command. target is the Panorama-managed
// device serial, or "" for a standalone firewall or a Panorama-local op.
func (c *Client) Op(ctx context.Context, cmd, target string) (*XMLResponse, error) {
	params := url.Values{}
	params.Set("type", "op")
	params.Set("cmd", cmd)
	return c.request(ctx, params, target)
}

// Get fetches config at xpath. Pass target="" for standalone firewalls.
func (c *Client) Get(ctx context.Context, xpath, target string) (*XMLResponse, error) {
	params := url.Values{}
	params.Set("type", "config")
	params.Set("action", "get")
	params.Set("xpath", xpath)
	return c.request(ctx, params, target)
}

// Show fetches runtime config at xpath. Pass target="" for standalone firewalls.
func (c *Client) Show(ctx context.Context, xpath, target string) (*XMLResponse, error) {
	params := url.Values{}
	params.Set("type", "config")
	params.Set("action", "show")
	params.Set("xpath", xpath)
	return c.request(ctx, params, target)
}

// Log submits a log query. Returns a job ID that can be polled for results.
func (c *Client) Log(ctx context.Context, logType string, nlogs int, query, target string) (*XMLResponse, error) {
	params := url.Values{}
	params.Set("type", "log")
	params.Set("log-type", logType)
	if nlogs > 0 {
		params.Set("nlogs", fmt.Sprintf("%d", nlogs))
	}
	if query != "" {
		params.Set("query", query)
	}
	return c.request(ctx, params, target)
}

// LogGet retrieves results of a log query job.
func (c *Client) LogGet(ctx context.Context, jobID, target string) (*XMLResponse, error) {
	params := url.Values{}
	params.Set("type", "log")
	params.Set("action", "get")
	params.Set("job-id", jobID)
	return c.request(ctx, params, target)
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
		Message: SanitizeForDisplay(resp.Msg.Line),
	}
}

// WrapInner wraps the inner XML content in a root element for proper parsing.
// This is needed because Result.Inner contains raw XML without a wrapper.
func WrapInner(inner []byte) []byte {
	return append(append([]byte("<root>"), inner...), []byte("</root>")...)
}
