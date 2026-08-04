package views

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/auth"
)

func TestNewLoginModel(t *testing.T) {
	creds := &auth.Credentials{}
	m := NewLoginModel(creds)

	if m.FocusedField() != FieldHost {
		t.Errorf("expected focusedField=FieldHost, got %d", m.FocusedField())
	}
	if m.Host() != "" {
		t.Error("expected empty host")
	}
}

func TestNewLoginModel_WithHost(t *testing.T) {
	creds := &auth.Credentials{Host: "10.0.0.1"}
	m := NewLoginModel(creds)

	if m.Host() != "10.0.0.1" {
		t.Errorf("expected host '10.0.0.1', got %q", m.Host())
	}
	if m.FocusedField() != FieldUsername {
		t.Error("expected focus to move to username when host is pre-filled")
	}
}

func TestLoginModel_SetSize(t *testing.T) {
	creds := &auth.Credentials{}
	m := NewLoginModel(creds)
	m = m.SetSize(100, 50)

	if m.width != 100 {
		t.Errorf("expected width=100, got %d", m.width)
	}
	if m.height != 50 {
		t.Errorf("expected height=50, got %d", m.height)
	}
}

func TestLoginModel_SetError(t *testing.T) {
	creds := &auth.Credentials{}
	m := NewLoginModel(creds)

	err := errors.New("test error")
	m = m.SetError(err)

	if m.err != err {
		t.Error("expected error to be set")
	}
}

func TestLoginModel_NextField(t *testing.T) {
	creds := &auth.Credentials{}
	m := NewLoginModel(creds)

	// Start at host
	if m.FocusedField() != FieldHost {
		t.Errorf("expected FieldHost, got %d", m.FocusedField())
	}

	// Next to username
	m = m.NextField()
	if m.FocusedField() != FieldUsername {
		t.Errorf("expected FieldUsername, got %d", m.FocusedField())
	}

	// Next to password
	m = m.NextField()
	if m.FocusedField() != FieldPassword {
		t.Errorf("expected FieldPassword, got %d", m.FocusedField())
	}

	// Next to insecure
	m = m.NextField()
	if m.FocusedField() != FieldInsecure {
		t.Errorf("expected FieldInsecure, got %d", m.FocusedField())
	}

	// Next wraps to host
	m = m.NextField()
	if m.FocusedField() != FieldHost {
		t.Errorf("expected FieldHost after wrap, got %d", m.FocusedField())
	}
}

func TestLoginModel_Host(t *testing.T) {
	creds := &auth.Credentials{}
	m := NewLoginModel(creds)

	m.hostInput.SetValue("  10.0.0.1  ")
	if m.Host() != "10.0.0.1" {
		t.Errorf("expected trimmed host '10.0.0.1', got %q", m.Host())
	}
}

func TestLoginModel_Username(t *testing.T) {
	creds := &auth.Credentials{}
	m := NewLoginModel(creds)

	m.usernameInput.SetValue("  admin  ")
	if m.Username() != "admin" {
		t.Errorf("expected trimmed username 'admin', got %q", m.Username())
	}
}

func TestLoginModel_Password(t *testing.T) {
	creds := &auth.Credentials{}
	m := NewLoginModel(creds)

	m.passwordInput.SetValue("secret123")
	if m.Password() != "secret123" {
		t.Errorf("expected password 'secret123', got %q", m.Password())
	}
}

func TestLoginModel_CanSubmit(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		username string
		password string
		want     bool
	}{
		{"all empty", "", "", "", false},
		{"only host", "10.0.0.1", "", "", false},
		{"host and username", "10.0.0.1", "admin", "", false},
		{"all filled", "10.0.0.1", "admin", "password", true},
		{"missing host", "", "admin", "password", false},
		{"missing username", "10.0.0.1", "", "password", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds := &auth.Credentials{}
			m := NewLoginModel(creds)
			m.hostInput.SetValue(tt.host)
			m.usernameInput.SetValue(tt.username)
			m.passwordInput.SetValue(tt.password)

			if got := m.CanSubmit(); got != tt.want {
				t.Errorf("CanSubmit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoginModel_Update(t *testing.T) {
	creds := &auth.Credentials{}
	m := NewLoginModel(creds)

	// Type in host field
	msg := tea.KeyPressMsg{Code: 'a', Text: "a"}
	updated, _ := m.Update(msg)

	// The input should have processed the key
	// Verify the update returned a valid model
	if updated.View() == "" {
		t.Error("expected non-empty view after update")
	}
}

func TestLoginModel_View(t *testing.T) {
	creds := &auth.Credentials{}
	m := NewLoginModel(creds)
	m = m.SetSize(100, 50)

	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestLoginModel_View_WithError(t *testing.T) {
	creds := &auth.Credentials{}
	m := NewLoginModel(creds)
	m = m.SetSize(100, 50)
	m = m.SetError(errors.New("authentication failed"))

	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestLoginField_Constants(t *testing.T) {
	if FieldHost != 0 {
		t.Errorf("expected FieldHost=0, got %d", FieldHost)
	}
	if FieldUsername != 1 {
		t.Errorf("expected FieldUsername=1, got %d", FieldUsername)
	}
	if FieldPassword != 2 {
		t.Errorf("expected FieldPassword=2, got %d", FieldPassword)
	}
	if FieldInsecure != 3 {
		t.Errorf("expected FieldInsecure=3, got %d", FieldInsecure)
	}
}

func TestLoginModel_Focus(t *testing.T) {
	creds := &auth.Credentials{}
	m := NewLoginModel(creds)

	// Initially host should be focused
	if !m.hostInput.Focused() {
		t.Error("expected host input to be focused initially")
	}

	// After NextField, username should be focused
	m = m.NextField()
	if !m.usernameInput.Focused() {
		t.Error("expected username input to be focused after NextField")
	}
	if m.hostInput.Focused() {
		t.Error("expected host input to be blurred after NextField")
	}

	// After NextField, password should be focused
	m = m.NextField()
	if !m.passwordInput.Focused() {
		t.Error("expected password input to be focused after second NextField")
	}
}

// TestLoginView_GeometryIsStableAcrossStates pins that the centered login box
// keeps identical dimensions whether it is idle, authenticating, or showing an
// error. The status text used to be rendered inline, so pressing Enter grew
// the box (the in-flight warning was the widest line in the form) and
// lipgloss.Place re-centered it — the whole form visibly jumped sideways and
// up at the exact moment the user was watching for feedback.
func TestLoginView_GeometryIsStableAcrossStates(t *testing.T) {
	InitStyles()

	build := func() LoginModel {
		m := NewLoginModel(&auth.Credentials{Host: "fw.example.com", Username: "admin"})
		return m.SetSize(120, 40)
	}

	// View() returns a full-terminal canvas via lipgloss.Place, so measure the
	// bordered box inside it: its left edge and its extent.
	dims := func(view string) (left, w, h int) {
		left = 1 << 30
		for line := range strings.SplitSeq(view, "\n") {
			plain := strings.TrimRight(stripANSI(line), " ")
			if plain == "" {
				continue
			}
			h++
			indent := len([]rune(plain)) - len([]rune(strings.TrimLeft(plain, " ")))
			left = min(left, indent)
			w = max(w, len([]rune(plain)))
		}
		return left, w, h
	}

	idleL, idleW, idleH := dims(build().View())

	authL, authW, authH := dims(build().SetSubmitting(true).SetSpinner("|").View())
	if authL != idleL || authW != idleW || authH != idleH {
		t.Errorf("authenticating box: left=%d w=%d h=%d; idle: left=%d w=%d h=%d; "+
			"the form jumps on Enter", authL, authW, authH, idleL, idleW, idleH)
	}

	errL, errW, errH := dims(build().SetError(errFakeLogin{}).View())
	if errL != idleL || errW != idleW || errH != idleH {
		t.Errorf("error box: left=%d w=%d h=%d; idle: left=%d w=%d h=%d; "+
			"the form jumps on failure", errL, errW, errH, idleL, idleW, idleH)
	}
}

type errFakeLogin struct{}

func (errFakeLogin) Error() string { return "invalid credentials" }
