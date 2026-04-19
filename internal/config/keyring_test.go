package config

import (
	"errors"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestKeyring_RoundTrip(t *testing.T) {
	keyring.MockInit()

	const host = "fw.example.com"
	const key = "super-secret-key"

	if err := SetAPIKey(host, key); err != nil {
		t.Fatalf("SetAPIKey returned error: %v", err)
	}

	got, err := GetAPIKey(host)
	if err != nil {
		t.Fatalf("GetAPIKey returned error: %v", err)
	}
	if got != key {
		t.Fatalf("GetAPIKey = %q, want %q", got, key)
	}

	if err := DeleteAPIKey(host); err != nil {
		t.Fatalf("DeleteAPIKey returned error: %v", err)
	}

	if _, err := GetAPIKey(host); !errors.Is(err, ErrCredentialNotFound) {
		t.Fatalf("GetAPIKey after delete = %v, want ErrCredentialNotFound", err)
	}
}

func TestKeyring_GetMissingReturnsSentinel(t *testing.T) {
	keyring.MockInit()

	_, err := GetAPIKey("missing.example.com")
	if !errors.Is(err, ErrCredentialNotFound) {
		t.Fatalf("GetAPIKey on missing host = %v, want ErrCredentialNotFound", err)
	}
}

func TestKeyring_DeleteMissingIsNoOp(t *testing.T) {
	keyring.MockInit()

	if err := DeleteAPIKey("missing.example.com"); err != nil {
		t.Fatalf("DeleteAPIKey on missing host returned %v, want nil", err)
	}
}
