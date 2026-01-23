package api

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	baseURL      string
	apiKey       string
	httpClient   *http.Client
	targetSerial string // For Panorama: routes API calls to specific device
}

type ClientOption func(*Client)

func WithInsecure(insecure bool) ClientOption {
	return func(c *Client) {
		if insecure {
			c.httpClient.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
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
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
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
	params.Set("key", c.apiKey)

	// Inject target parameter for Panorama routing
	if c.targetSerial != "" {
		params.Set("target", c.targetSerial)
	}

	reqURL := c.baseURL + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var xmlResp XMLResponse
	if err := xml.Unmarshal(body, &xmlResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &xmlResp, nil
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
