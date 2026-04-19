package api

import (
	"context"
	"crypto/tls"
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
		i := i
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
