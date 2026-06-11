package auth_test

import (
	"context"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jp2195/pyre/internal/api"
	"github.com/jp2195/pyre/internal/auth"
	"github.com/jp2195/pyre/internal/testutil"
)

func TestGenerateAPIKey_Success(t *testing.T) {
	mock := testutil.NewMockPANOS()
	defer mock.Close()

	result, err := auth.GenerateAPIKey(
		context.Background(),
		mock.Host(),
		"admin",
		"admin",
		api.ClientOptions{Insecure: true},
	)

	if err != nil {
		t.Fatalf("GenerateAPIKey failed: %v", err)
	}

	if result.Error != nil {
		t.Fatalf("unexpected error in result: %v", result.Error)
	}

	if result.APIKey == "" {
		t.Error("expected API key to be set")
	}

	if result.APIKey != "LUFRPT1234567890abcdef==" {
		t.Errorf("unexpected API key: %s", result.APIKey)
	}
}

func TestGenerateAPIKey_InvalidCredentials(t *testing.T) {
	mock := testutil.NewMockPANOS()
	defer mock.Close()

	result, err := auth.GenerateAPIKey(
		context.Background(),
		mock.Host(),
		"admin",
		"wrongpassword",
		api.ClientOptions{Insecure: true},
	)

	if err != nil {
		t.Fatalf("GenerateAPIKey failed with error: %v", err)
	}

	if result.Error == nil {
		t.Fatal("expected error for invalid credentials")
	}

	if result.Error.Error() != "Invalid credentials" {
		t.Errorf("unexpected error message: %s", result.Error.Error())
	}
}

func TestGenerateAPIKey_UnreachableHost(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := auth.GenerateAPIKey(
		ctx,
		"192.0.2.1:12345", // Non-routable test address
		"admin",
		"admin",
		api.ClientOptions{Insecure: true},
	)

	if err == nil {
		t.Error("expected error for unreachable host")
	}
}

func TestGenerateAPIKey_OversizedResponseRejected(t *testing.T) {
	// 2MB XML comment before the valid response: an unbounded read would
	// parse this successfully; the 1MB cap truncates mid-comment and the
	// XML decoder must surface an error instead of returning a key.
	padding := strings.Repeat(" ", 2*1024*1024)
	body := "<!--" + padding + `--><response status="success"><result><key>LUFRPT-big==</key></result></response>`

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "https://")
	_, err := auth.GenerateAPIKey(context.Background(), host, "admin", "admin", api.ClientOptions{Insecure: true})
	if err == nil {
		t.Fatal("expected error for oversized keygen response, got nil")
	}
}

// keygenTLSServer serves a fixed successful keygen response over TLS and
// returns the server plus a PEM file path containing its certificate, usable
// as a CA bundle (httptest's cert is self-signed, so it is its own CA).
func keygenTLSServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<response status="success"><result><key>LUFRPT-ca==</key></result></response>`))
	}))
	caPath := filepath.Join(t.TempDir(), "ca.pem")
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	if err := os.WriteFile(caPath, caPEM, 0o600); err != nil {
		srv.Close()
		t.Fatalf("writing CA fixture: %v", err)
	}
	return srv, caPath
}

func TestGenerateAPIKey_CustomCA(t *testing.T) {
	srv, caPath := keygenTLSServer(t)
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "https://")
	result, err := auth.GenerateAPIKey(context.Background(), host, "admin", "admin",
		api.ClientOptions{CACertPath: caPath})
	if err != nil {
		t.Fatalf("GenerateAPIKey with CA bundle: %v", err)
	}
	if result.APIKey != "LUFRPT-ca==" {
		t.Errorf("APIKey = %q, want LUFRPT-ca==", result.APIKey)
	}
}

func TestGenerateAPIKey_UntrustedCert_FailsWithoutCAOrInsecure(t *testing.T) {
	// Default (verified) TLS against httptest's self-signed cert must fail:
	// proves keygen is fail-closed when neither Insecure nor a CA is given.
	srv, _ := keygenTLSServer(t)
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "https://")
	_, err := auth.GenerateAPIKey(context.Background(), host, "admin", "admin",
		api.ClientOptions{})
	if err == nil {
		t.Fatal("expected TLS verification failure against untrusted cert, got nil")
	}
}

func TestGenerateAPIKey_BadCABundle_FailsClosed(t *testing.T) {
	// An unreadable CA bundle must error before any request is sent.
	_, err := auth.GenerateAPIKey(context.Background(), "127.0.0.1:1", "admin", "admin",
		api.ClientOptions{CACertPath: "/nonexistent/ca.pem"})
	if err == nil {
		t.Fatal("expected error for unreadable CA bundle, got nil")
	}
	if !strings.Contains(err.Error(), "CA bundle") {
		t.Errorf("err = %v, want to mention the CA bundle", err)
	}
}
