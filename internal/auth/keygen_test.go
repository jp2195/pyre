package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/joshuamontgomery/pyre/internal/auth"
	"github.com/joshuamontgomery/pyre/internal/testutil"
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
