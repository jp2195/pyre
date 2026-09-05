package views

import (
	"testing"

	"github.com/jp2195/pyre/internal/config"
)

// TestEditForm_PreservesUneditedFields guards against the edit form silently
// downgrading a connection's TLS posture. The form exposes host, username,
// type, and insecure; ca_cert_path is not editable, so rebuilding the config
// from the visible fields alone drops a pinned CA and sends the next session
// to system roots without saying so.
func TestEditForm_PreservesUneditedFields(t *testing.T) {
	original := config.ConnectionConfig{
		Username:   "admin",
		Type:       "firewall",
		Insecure:   false,
		CACertPath: "/etc/pyre/corp-ca.pem",
	}

	m := NewEditConnectionForm("fw.example.com", original)
	got := m.GetConfig()

	if got.CACertPath != "/etc/pyre/corp-ca.pem" {
		t.Errorf("CACertPath = %q, want it preserved through an edit", got.CACertPath)
	}
	if got.Username != "admin" {
		t.Errorf("Username = %q, want admin", got.Username)
	}
	if got.Insecure {
		t.Error("Insecure = true, want false")
	}
}

// TestEditForm_TogglingInsecureKeepsCACertPath checks the same invariant on
// the path where the user does change something.
func TestEditForm_TogglingInsecureKeepsCACertPath(t *testing.T) {
	m := NewEditConnectionForm("fw.example.com", config.ConnectionConfig{
		CACertPath: "/etc/pyre/corp-ca.pem",
	})

	got := m.ToggleInsecure().GetConfig()

	if !got.Insecure {
		t.Error("Insecure = false after toggle, want true")
	}
	if got.CACertPath != "/etc/pyre/corp-ca.pem" {
		t.Errorf("CACertPath = %q, want it preserved", got.CACertPath)
	}
}
