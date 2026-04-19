package config

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

// keyringService is the service name used for every Pyre keychain entry.
// Entries are keyed as "apikey:<host>" under this service so multiple
// firewalls can coexist without colliding.
const keyringService = "pyre"

// ErrCredentialNotFound is returned by GetAPIKey when no keychain entry
// exists for the requested host. Callers use this sentinel to decide
// whether to fall through to an interactive credential prompt.
var ErrCredentialNotFound = errors.New("credential not found in keychain")

// SetAPIKey stores an API key for host in the OS keychain.
func SetAPIKey(host, key string) error {
	return keyring.Set(keyringService, "apikey:"+host, key)
}

// GetAPIKey returns the API key stored for host, or ErrCredentialNotFound
// if no entry exists. Any other keychain error is wrapped.
func GetAPIKey(host string) (string, error) {
	v, err := keyring.Get(keyringService, "apikey:"+host)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrCredentialNotFound
	}
	if err != nil {
		return "", fmt.Errorf("keyring get: %w", err)
	}
	return v, nil
}

// DeleteAPIKey removes the API key stored for host. Missing entries are
// treated as success so callers can blindly Delete on disconnect.
func DeleteAPIKey(host string) error {
	err := keyring.Delete(keyringService, "apikey:"+host)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}
