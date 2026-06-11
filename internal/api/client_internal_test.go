package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestNewClient_TLSMinVersion(t *testing.T) {
	c, err := NewClient("fw.example", "KEY", ClientOptions{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	tr, ok := c.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", c.httpClient.Transport)
	}
	if tr.TLSClientConfig == nil {
		t.Fatal("expected TLSClientConfig to be set")
	}
	if tr.TLSClientConfig.MinVersion < tls.VersionTLS12 {
		t.Fatalf("MinVersion = %x, want >= TLS 1.2", tr.TLSClientConfig.MinVersion)
	}
}

func TestNewClient_OwnsTransport(t *testing.T) {
	c1, err := NewClient("a.example", "K1", ClientOptions{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c2, err := NewClient("b.example", "K2", ClientOptions{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c1.httpClient.Transport == c2.httpClient.Transport {
		t.Fatal("clients must not share http.Transport")
	}
	if c1.httpClient.Transport == http.DefaultTransport {
		t.Fatal("client must not use http.DefaultTransport")
	}
}

func TestNewClient_CustomCA(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(path, []byte(testCAPEM), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := NewClient("fw.example", "K", ClientOptions{CACertPath: path})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	tr := c.httpClient.Transport.(*http.Transport)
	if tr.TLSClientConfig.RootCAs == nil {
		t.Fatal("expected RootCAs populated from CA path")
	}
}

func TestNewClient_CustomCA_BadPath(t *testing.T) {
	// When the user explicitly sets CACertPath but the file cannot be read,
	// NewClient must return an error rather than silently falling back to
	// system roots (which would produce a confusing TLS handshake failure
	// later).
	c, err := NewClient("fw.example", "K", ClientOptions{CACertPath: "/nonexistent/ca.pem"})
	if err == nil {
		t.Fatal("expected error for unreadable CA bundle, got nil")
	}
	if c != nil {
		t.Fatal("expected nil client when CA load fails")
	}
}

func TestNewClient_CustomCA_EmptyPEM(t *testing.T) {
	// An empty (or otherwise PEM-less) file yields zero parsed
	// certificates. This must be reported as an error too.
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.pem")
	if err := os.WriteFile(path, []byte{}, 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := NewClient("fw.example", "K", ClientOptions{CACertPath: path})
	if err == nil {
		t.Fatal("expected error for empty CA bundle, got nil")
	}
	if c != nil {
		t.Fatal("expected nil client when CA load fails")
	}
}

func TestNewClient_NoCAPath_UsesSystemRoots(t *testing.T) {
	// Empty CACertPath must continue to work silently.
	c, err := NewClient("fw.example", "K", ClientOptions{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	tr := c.httpClient.Transport.(*http.Transport)
	if tr.TLSClientConfig == nil {
		t.Fatal("expected TLSClientConfig")
	}
	if tr.TLSClientConfig.RootCAs != nil {
		t.Fatal("expected RootCAs to be nil (use system roots) when CACertPath is empty")
	}
}

// testCAPEM is a self-signed test certificate for verifying RootCAs loading.
// Generated with:
//
//	openssl req -x509 -newkey rsa:2048 -days 3650 -nodes \
//	    -subj /CN=pyre-test-ca -keyout k.pem -out c.pem
const testCAPEM = `-----BEGIN CERTIFICATE-----
MIIDDzCCAfegAwIBAgIUeKMPuN83cAD39/DLG+Uz53UxYzowDQYJKoZIhvcNAQEL
BQAwFzEVMBMGA1UEAwwMcHlyZS10ZXN0LWNhMB4XDTI2MDQxOTEzMDMzM1oXDTM2
MDQxNjEzMDMzNFowFzEVMBMGA1UEAwwMcHlyZS10ZXN0LWNhMIIBIjANBgkqhkiG
9w0BAQEFAAOCAQ8AMIIBCgKCAQEAmlvIkd4JIm1P3vO8F30zMKgJLbR4+ikA780x
T6Ph6NfKSkcXqVCCbG1OJ5IsHL0uvTrALARxRTttuFAifF3xJcOAiZxBqcqtov9M
x1UsuThOVNnBKLDsLME4epS/QF3T6NZ9fs49offsdJSkIGGe+R132Bkm5deHYgtE
xnk7ksx6kkKSkQMZt5KlK8rnQnaH8EPTf4eri/0pFRnlGt9NUrECEhYkQ3o7M2Bw
Q8Xt5h7nta9JBNQbj6Xy0JXNNB1FUYGl4bZt5VxcG+46UWtCIl96A5Yk4v7rV03f
okXC+GUoEJybKSSL2Akbiir7zu61c5ewtPaBK9X6S9T2xaTf/QIDAQABo1MwUTAd
BgNVHQ4EFgQUWyV07zgAdNzXg0xLT5vAq436dd0wHwYDVR0jBBgwFoAUWyV07zgA
dNzXg0xLT5vAq436dd0wDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOC
AQEAXJjke/7+K2Mvy8daaOpAMVA7pIsd4eQmRDn5VacfkGrPvmjsj7QHf0PqJpSq
x7VBrjBiJidrM6BZy6h5lUJFrYbOFfNbtGwheijs0dl+YfFO5GvIyVKnnZH8bSEM
KsYFqASQcAZkc1VvtdrzKazvQV+7AOfigKHUr3AMBBAhVIyVrjORVzTa4AcB69dd
dIKLaCXmIfH4jHj9kT05fnldiclZ4YprgxuiZL+NbtKjIlP5WU4YSvHoUJzsn6/T
WOwIe7h27dqAtkAT+85PKvGei/8hpZmddRIl2D8Uo53R5JYlM1H3rpLpLJfhtaA0
qjFvRc97qd788ItTInXS3dqjRg==
-----END CERTIFICATE-----
`

// TestClient_ConcurrentTargetsNoInterference verifies that the per-request
// target parameter does not bleed across goroutines. Each concurrent request
// uses a distinct target serial; the handler echoes it back so we can assert
// the client sent the correct target for that goroutine.
func TestClient_ConcurrentTargetsNoInterference(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<response status="success"><result><target>%s</target></result></response>`, r.URL.Query().Get("target"))
	}))
	defer srv.Close()

	// Strip https:// scheme so NewClient can build its own URL.
	host := strings.TrimPrefix(srv.URL, "https://")
	c, err := NewClient(host, "K", ClientOptions{Insecure: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = c.Close() }()

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() {
			params := url.Values{"type": {"op"}, "cmd": {"<show/>"}}
			resp, reqErr := c.request(context.Background(), params, fmt.Sprintf("serial-%d", i))
			if reqErr != nil {
				t.Errorf("request failed: %v", reqErr)
				return
			}
			want := fmt.Sprintf("<target>serial-%d</target>", i)
			if !strings.Contains(string(resp.Result.Inner), want) {
				t.Errorf("target bled across goroutines; got %s, want contains %s", resp.Result.Inner, want)
			}
		})
	}
	wg.Wait()
}

// serveBody starts a TLS httptest server that returns the given body with a
// 200 OK status for every request. The returned teardown closes the server
// and the client's idle connections.
func serveBody(t *testing.T, body string) (*Client, func()) {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(body))
	}))
	host := strings.TrimPrefix(srv.URL, "https://")
	c, err := NewClient(host, "K", ClientOptions{Insecure: true})
	if err != nil {
		srv.Close()
		t.Fatalf("NewClient: %v", err)
	}
	return c, func() {
		_ = c.Close()
		srv.Close()
	}
}

// opParams is a small helper returning params for a trivial "op" request
// so tests can drive the full request() path.
func opParams() url.Values {
	return url.Values{"type": {"op"}, "cmd": {"<show/>"}}
}

func TestRequest_ValidSuccessResponse(t *testing.T) {
	c, done := serveBody(t, `<response status="success"><result><foo>ok</foo></result></response>`)
	defer done()

	resp, err := c.request(context.Background(), opParams(), "")
	if err != nil {
		t.Fatalf("request: unexpected error: %v", err)
	}
	if !resp.IsSuccess() {
		t.Fatalf("expected success, got status=%q code=%q", resp.Status, resp.Code)
	}
	if !strings.Contains(string(resp.Result.Inner), "<foo>ok</foo>") {
		t.Errorf("result inner = %q, want to contain <foo>ok</foo>", resp.Result.Inner)
	}
}

func TestRequest_ErrorResponse_PopulatesStatusAndMessage(t *testing.T) {
	// PAN-OS error style: status="error" code="13" with a <msg><line>…</line></msg>.
	// request() itself parses the envelope successfully (no error return); it
	// is the caller's job to surface the error via CheckResponse. Assert both
	// that the parsed envelope carries the diagnostic code and that
	// CheckResponse produces a typed *APIError with the sanitized message.
	body := `<response status="error" code="13"><msg><line>Permission denied</line></msg></response>`
	c, done := serveBody(t, body)
	defer done()

	resp, err := c.request(context.Background(), opParams(), "")
	if err != nil {
		t.Fatalf("request: unexpected transport error: %v", err)
	}
	if resp.IsSuccess() {
		t.Fatalf("expected non-success response, got status=%q", resp.Status)
	}
	if resp.Code != "13" {
		t.Errorf("resp.Code = %q, want %q", resp.Code, "13")
	}

	checkErr := CheckResponse(resp)
	if checkErr == nil {
		t.Fatal("CheckResponse returned nil for an error response")
	}
	var apiErr *APIError
	if !errors.As(checkErr, &apiErr) {
		t.Fatalf("CheckResponse err = %T, want *APIError", checkErr)
	}
	if apiErr.Code != "13" {
		t.Errorf("APIError.Code = %q, want %q", apiErr.Code, "13")
	}
	if !strings.Contains(apiErr.Error(), "Permission denied") {
		t.Errorf("APIError.Error() = %q, want to contain %q", apiErr.Error(), "Permission denied")
	}
}

func TestRequest_TruncatedXML_ReturnsError(t *testing.T) {
	// Body is an unterminated element — the XML decoder must surface this as
	// a parse error rather than panicking or returning a bogus success.
	c, done := serveBody(t, `<response status="success"><result><foo`)
	defer done()

	resp, err := c.request(context.Background(), opParams(), "")
	if err == nil {
		t.Fatalf("expected parse error for truncated XML, got resp=%+v", resp)
	}
	if !strings.Contains(err.Error(), "parsing response") {
		t.Errorf("err = %v, want to mention 'parsing response'", err)
	}
}

func TestRequest_EmptyBody_ReturnsError(t *testing.T) {
	c, done := serveBody(t, ``)
	defer done()

	resp, err := c.request(context.Background(), opParams(), "")
	if err == nil {
		t.Fatalf("expected error for empty body, got resp=%+v", resp)
	}
}

func TestRequest_DoctypeRejected_NotPanic(t *testing.T) {
	// Wire-level decodeXML rejects DOCTYPE directives; this must flow up to
	// request() as a returned error, not a panic.
	body := `<?xml version="1.0"?><!DOCTYPE response [<!ENTITY lol "lol">]><response status="success"><result/></response>`
	c, done := serveBody(t, body)
	defer done()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("request() panicked on DOCTYPE body: %v", r)
		}
	}()

	resp, err := c.request(context.Background(), opParams(), "")
	if err == nil {
		t.Fatalf("expected DOCTYPE rejection error, got resp=%+v", resp)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "doctype") &&
		!strings.Contains(strings.ToLower(err.Error()), "directive") {
		t.Errorf("err = %v, want to mention doctype/directive", err)
	}
}

func TestRequest_OversizedResponse_ReturnsExplicitError(t *testing.T) {
	// Stream just over maxResponseSize of padding. Pre-fix this truncates
	// silently and fails as an XML parse error; post-fix it must name the
	// size limit explicitly.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		chunk := bytes.Repeat([]byte(" "), 1024*1024)
		for range maxResponseSize/len(chunk) + 1 {
			if _, err := w.Write(chunk); err != nil {
				return // client hung up after hitting its limit
			}
		}
	}))
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "https://")
	c, err := NewClient(host, "K", ClientOptions{Insecure: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = c.Close() }()

	_, reqErr := c.request(context.Background(), opParams(), "")
	if reqErr == nil {
		t.Fatal("expected error for oversized response, got nil")
	}
	if !strings.Contains(reqErr.Error(), "exceeds") {
		t.Errorf("err = %v, want to mention 'exceeds'", reqErr)
	}
	if !strings.Contains(reqErr.Error(), "50MB") {
		t.Errorf("err = %v, want to mention '50MB'", reqErr)
	}
}
