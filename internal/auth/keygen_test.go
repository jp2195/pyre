package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
		true,
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
		true,
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
		true,
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
	_, err := auth.GenerateAPIKey(context.Background(), host, "admin", "admin", true)
	if err == nil {
		t.Fatal("expected error for oversized keygen response, got nil")
	}
}
