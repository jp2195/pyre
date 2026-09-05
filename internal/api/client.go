package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

// maxResponseSize is the maximum allowed response body size (50 MB).
const maxResponseSize = 50 * 1024 * 1024

// MaxConcurrentRequests caps how many API calls one client will have in
// flight at once.
//
// The PAN-OS management plane is slow and is shared with the web UI, and
// every call is recorded in the firewall's own log. Without a cap, opening
// the Overview dashboard fans out about sixteen requests simultaneously and
// a policy load issues roughly fifteen config calls, which is both a
// self-inflicted latency problem and a needlessly noisy audit trail. Four
// keeps the views responsive without behaving like a scraper.
//
// The cap is per client, and each connection has its own client, so it
// bounds load per firewall rather than across all of them.
const MaxConcurrentRequests = 4

// UserAgent identifies pyre in the firewall's API log. An unidentified
// client is a worse answer to "what made these calls?" than it needs to be.
// Exported so the keygen flow in internal/auth sends the same value.
const UserAgent = "pyre"

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
	baseURL    string        // 16 bytes (string header)
	apiKey     string        // 16 bytes (string header)
	httpClient *http.Client  // 8 bytes (pointer)
	sem        chan struct{} // 8 bytes (pointer); see MaxConcurrentRequests

	// deviceLoc is the zone the device keeps its clock in, learned from its
	// own reported time. Nil until GetSystemInfo has run. Atomic because
	// fetches run concurrently.
	deviceLoc atomic.Pointer[time.Location]
}

// deviceLocation returns the zone to interpret this device's timestamps in.
//
// PAN-OS emits bare wall clock with no offset, and `show system info` carries
// no time-zone field to interpret it with (verified on PA-440 / 11.2.10-h8),
// so the offset is derived from the device's own clock in learnDeviceClock.
// Until that has run, the device is assumed to share the operator's zone,
// which is both the common case and a far better guess than UTC.
//
// Deriving the offset from the clock rather than a zone name has a useful
// side effect: a device whose clock is skewed gets timestamps interpreted
// against its own clock, so "5m ago" stays true relative to the log the
// device wrote.
func (c *Client) deviceLocation() *time.Location {
	if loc := c.deviceLoc.Load(); loc != nil {
		return loc
	}
	return time.Local
}

// learnDeviceClock records the device's UTC offset from the wall clock it
// reports, so every later timestamp is interpreted in the device's zone.
// A value that does not parse leaves the previous assumption in place.
//
// On Panorama the offset tracks whichever managed device was last queried,
// which is what the views are showing.
func (c *Client) learnDeviceClock(reported string) {
	wall, err := parsePANTimeIn(reported, time.UTC)
	if err != nil {
		return
	}
	offset := wall.Sub(time.Now().UTC())
	c.deviceLoc.Store(time.FixedZone("", int(offset.Round(time.Second).Seconds())))
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

// NewTransport builds an *http.Transport with a hardened TLS config
// (MinVersion = TLS 1.2) applied from opts. It returns an error if
// opts.CACertPath is set but the CA bundle cannot be loaded, so
// configuration mistakes surface at connect time rather than as opaque TLS
// handshake failures on the first request.
//
// It is exported so the keygen flow in internal/auth shares the exact same
// TLS construction (and fail-closed CA handling) as the main API client.
// Each call returns a transport owned by its caller.
func NewTransport(opts ClientOptions) (*http.Transport, error) {
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
		TLSClientConfig: tlsCfg,
		// Honor HTTPS_PROXY / HTTP_PROXY / NO_PROXY. Every other Go HTTP
		// client on the machine does, and a firewall reached through a
		// corporate egress proxy is otherwise simply unreachable.
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}, nil
}

// BaseURL returns the PAN-OS API endpoint for host.
//
// A bare IPv6 literal has to be bracketed or the result is not a parseable
// URL at all: Go reads the trailing group as a port and every request fails
// with "invalid port" before a packet leaves the process. ValidateHost
// accepts bare IPv6 (net.ParseIP does), so the bracketing belongs here,
// where the URL is actually built.
//
// Hostnames, IPv4 literals, host:port, and already-bracketed IPv6 forms are
// returned unchanged.
func BaseURL(host string) string {
	if ip := net.ParseIP(host); ip != nil && ip.To4() == nil {
		host = "[" + host + "]"
	}
	return fmt.Sprintf("https://%s/api/", host)
}

// NewClient builds a PAN-OS XML API client for host using apiKey for
// authentication. The client owns its *http.Transport; callers must invoke
// Close to release idle connections when finished.
//
// An error is returned when opts.CACertPath is set but the CA bundle cannot
// be loaded (unreadable file or PEM contains zero certificates). When
// CACertPath is empty, NewClient uses system roots and never fails.
func NewClient(host, apiKey string, opts ClientOptions) (*Client, error) {
	tr, err := NewTransport(opts)
	if err != nil {
		return nil, err
	}
	return &Client{
		baseURL: BaseURL(host),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		},
		sem: make(chan struct{}, MaxConcurrentRequests),
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
	// Wait for a slot before doing anything else. Queuing here rather than
	// at the call sites means every path is bounded, including the fan-outs
	// that build a dashboard. Nothing holds a slot while issuing another
	// request, so this cannot deadlock.
	if c.sem != nil {
		select {
		case c.sem <- struct{}{}:
			defer func() { <-c.sem }()
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

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
	req.Header.Set("User-Agent", UserAgent)

	resp, err := c.httpClient.Do(req)
	duration := time.Since(start)

	if err != nil {
		log.Printf("[API Error] request failed after %dms: %v", duration.Milliseconds(), err)
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }() //nolint:errcheck // best effort cleanup

	// Read one byte past the cap so an at-limit response is distinguishable
	// from an over-limit one, and report the latter explicitly instead of
	// letting truncated XML surface as a confusing parse error.
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		log.Printf("[API Error] reading response: %v", err)
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if len(body) > maxResponseSize {
		log.Printf("[API Error] response exceeded %d byte limit", maxResponseSize)
		return nil, fmt.Errorf("response exceeds %dMB limit", maxResponseSize/(1024*1024))
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
		params.Set("nlogs", strconv.Itoa(nlogs))
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
