package auth

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
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

func GenerateAPIKey(ctx context.Context, host, username, password string, insecure bool) (*KeygenResult, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure}, //nolint:gosec // G402: InsecureSkipVerify required for self-signed firewall certificates when user enables --insecure
		},
	}

	// Use POST with form body to keep credentials out of URLs/logs
	reqURL := fmt.Sprintf("https://%s/api/", host)
	formData := url.Values{}
	formData.Set("type", "keygen")
	formData.Set("user", username)
	formData.Set("password", password)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating keygen request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("keygen request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }() //nolint:errcheck // best effort cleanup

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading keygen response: %w", err)
	}

	var xmlResp keygenResponse
	if err := xml.Unmarshal(body, &xmlResp); err != nil {
		return nil, fmt.Errorf("parsing keygen response: %w", err)
	}

	if xmlResp.Status != "success" {
		errMsg := xmlResp.Msg.Line
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

type KeygenError struct {
	Message string
	Cause   error
}

func (e *KeygenError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *KeygenError) Unwrap() error {
	return e.Cause
}

// IsAuthenticationError checks if the error indicates an authentication failure.
// Uses errors.As to properly unwrap and check for KeygenError types.
func IsAuthenticationError(err error) bool {
	if err == nil {
		return false
	}

	// Check if this is a KeygenError with authentication-related message
	var keygenErr *KeygenError
	if errors.As(err, &keygenErr) {
		msg := strings.ToLower(keygenErr.Message)
		return strings.Contains(msg, "invalid credential") ||
			strings.Contains(msg, "authentication failed") ||
			strings.Contains(msg, "invalid username") ||
			strings.Contains(msg, "invalid password")
	}

	// Fallback: check the error message directly for common auth failure strings
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "authentication failed") ||
		strings.Contains(msg, "invalid credentials") ||
		strings.Contains(msg, "invalid username or password")
}

// IsConnectionError checks if the error is a connection-related KeygenError.
// Uses errors.As to properly unwrap error chains.
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}
	var keygenErr *KeygenError
	return errors.As(err, &keygenErr)
}
