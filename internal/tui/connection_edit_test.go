package tui

// connection_edit_test.go – regression tests for the saved-connection edit
// and login paths, both of which quietly discarded user intent.

import (
	"testing"

	"github.com/jp2195/pyre/internal/config"
	"github.com/jp2195/pyre/internal/tui/views"
)

// TestConnectionFormSubmit_EditRenamesHost covers a silent no-op: the edit
// form lets the user change the host, but the handler called UpdateConnection
// under the *new* key, which does not exist, and discarded the error. Neither
// the rename nor any other edit made in the same submission was saved.
func TestConnectionFormSubmit_EditRenamesHost(t *testing.T) {
	m := newTestModel(t, ViewConnectionForm)
	m.config.Connections["old.example.com"] = config.ConnectionConfig{
		Username:   "admin",
		Type:       "firewall",
		CACertPath: "/etc/pyre/corp-ca.pem",
	}
	m.config.Default = "old.example.com"

	updated, _ := m.Update(ConnectionFormSubmitMsg{
		OriginalHost: "old.example.com",
		Host:         "new.example.com",
		Config: config.ConnectionConfig{
			Username:   "operator",
			Type:       "firewall",
			CACertPath: "/etc/pyre/corp-ca.pem",
		},
		SaveToConfig: true,
		Mode:         views.FormModeEdit,
	})
	nm := updated.(Model)

	if _, ok := nm.config.Connections["old.example.com"]; ok {
		t.Error("old host still present after rename")
	}
	got, ok := nm.config.Connections["new.example.com"]
	if !ok {
		t.Fatal("renamed host missing from config")
	}
	if got.Username != "operator" {
		t.Errorf("Username = %q, want operator (edits in the same submission must persist)", got.Username)
	}
	if got.CACertPath != "/etc/pyre/corp-ca.pem" {
		t.Errorf("CACertPath = %q, want it carried across the rename", got.CACertPath)
	}
	if nm.config.Default != "new.example.com" {
		t.Errorf("Default = %q, want it to follow the rename", nm.config.Default)
	}
}

// TestConnectionFormSubmit_EditInPlace checks the non-rename path still
// updates the existing entry rather than creating a duplicate.
func TestConnectionFormSubmit_EditInPlace(t *testing.T) {
	m := newTestModel(t, ViewConnectionForm)
	m.config.Connections["fw.example.com"] = config.ConnectionConfig{Username: "admin"}

	updated, _ := m.Update(ConnectionFormSubmitMsg{
		OriginalHost: "fw.example.com",
		Host:         "fw.example.com",
		Config:       config.ConnectionConfig{Username: "operator"},
		SaveToConfig: true,
		Mode:         views.FormModeEdit,
	})
	nm := updated.(Model)

	if len(nm.config.Connections) != 1 {
		t.Errorf("connection count = %d, want 1", len(nm.config.Connections))
	}
	if got := nm.config.Connections["fw.example.com"].Username; got != "operator" {
		t.Errorf("Username = %q, want operator", got)
	}
}

// TestLoginSuccess_HonorsInsecureToggle covers a mismatch that produced a
// login which appeared to succeed and then failed every subsequent request:
// the insecure checkbox on the login screen was passed to keygen but the API
// client was built from the saved connection config, which still said verify.
func TestLoginSuccess_HonorsInsecureToggle(t *testing.T) {
	m := newTestModel(t, ViewLogin)
	m.selectedConnection = "fw.example.com"
	m.selectedConnectionConfig = config.ConnectionConfig{Insecure: false}

	updated, _ := m.Update(LoginSuccessMsg{
		Host:     "fw.example.com",
		APIKey:   "k",
		Username: "admin",
		Insecure: true,
	})
	nm := updated.(Model)

	conn := nm.session.GetActiveConnection()
	if conn == nil {
		t.Fatal("expected an active connection")
	}
	if !conn.Config.Insecure {
		t.Error("connection was built with TLS verification on, but the user ticked insecure at login")
	}
}

// TestConnectionFormSubmit_SaveOverExistingHostUpdatesIt pins the add path's
// behavior when the host is already saved. Ticking "save" means "store these
// settings for this host", so the entry is updated. It previously hit
// AddConnection's already-exists error, which was discarded, so the save
// silently did nothing.
func TestConnectionFormSubmit_SaveOverExistingHostUpdatesIt(t *testing.T) {
	m := newTestModel(t, ViewConnectionForm)
	m.config.Connections["fw.example.com"] = config.ConnectionConfig{Username: "admin"}

	updated, _ := m.Update(ConnectionFormSubmitMsg{
		Host:         "fw.example.com",
		Config:       config.ConnectionConfig{Username: "operator"},
		SaveToConfig: true,
		Mode:         views.FormModeQuickConnect,
	})
	nm := updated.(Model)

	if got := nm.config.Connections["fw.example.com"].Username; got != "operator" {
		t.Errorf("Username = %q, want operator", got)
	}
	if nm.currentView != ViewLogin {
		t.Errorf("currentView = %v, want ViewLogin: saving must not block connecting", nm.currentView)
	}
	if nm.err != nil {
		t.Errorf("unexpected error surfaced: %v", nm.err)
	}
}

// TestConnectionFormSubmit_RenameOntoExistingHostIsRefused checks the one
// case that must not proceed: a rename whose new name already belongs to a
// different connection would otherwise clobber it.
func TestConnectionFormSubmit_RenameOntoExistingHostIsRefused(t *testing.T) {
	m := newTestModel(t, ViewConnectionForm)
	m.config.Connections["a.example.com"] = config.ConnectionConfig{Username: "admin-a"}
	m.config.Connections["b.example.com"] = config.ConnectionConfig{Username: "admin-b"}

	updated, _ := m.Update(ConnectionFormSubmitMsg{
		OriginalHost: "a.example.com",
		Host:         "b.example.com",
		Config:       config.ConnectionConfig{Username: "admin-a"},
		SaveToConfig: true,
		Mode:         views.FormModeEdit,
	})
	nm := updated.(Model)

	if nm.err == nil {
		t.Error("expected an error when renaming onto an existing connection")
	}
	if got := nm.config.Connections["b.example.com"].Username; got != "admin-b" {
		t.Errorf("b.example.com Username = %q, want admin-b (must not be clobbered)", got)
	}
	if _, ok := nm.config.Connections["a.example.com"]; !ok {
		t.Error("a.example.com was removed despite the refused rename")
	}
}
