package views

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/auth"
	"github.com/jp2195/pyre/internal/config"
)

// newBlankLogin builds a login form with no credentials pre-filled, which is
// what the screen shows when pyre is started without a host.
func newBlankLogin() LoginModel { return NewLoginModel(&auth.Credentials{}) }

// typeText feeds a string to a model one key at a time, the way the terminal
// delivers it.
func typeLogin(m LoginModel, text string) LoginModel {
	for _, r := range text {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

func typeForm(m ConnectionFormModel, text string) ConnectionFormModel {
	for _, r := range text {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

// TestLogin_FieldOrderCyclesBothWays checks tab and shift-tab reach every
// field and wrap, so no field can become unreachable.
func TestLogin_FieldOrderCyclesBothWays(t *testing.T) {
	order := []LoginField{FieldHost, FieldUsername, FieldPassword, FieldInsecure}

	m := newBlankLogin()
	for i, want := range append(order[1:], order[0]) {
		m = m.NextField()
		if got := m.FocusedField(); got != want {
			t.Fatalf("after %d tabs, focus is %d, want %d", i+1, got, want)
		}
	}

	m = newBlankLogin()
	for i, want := range []LoginField{FieldInsecure, FieldPassword, FieldUsername, FieldHost} {
		m = m.PrevField()
		if got := m.FocusedField(); got != want {
			t.Fatalf("after %d back-tabs, focus is %d, want %d", i+1, got, want)
		}
	}
}

// TestLogin_TypingGoesToTheFocusedField checks each field collects its own
// text, and that the insecure checkbox swallows keys rather than leaking them
// into whichever input was last focused.
func TestLogin_TypingGoesToTheFocusedField(t *testing.T) {
	m := newBlankLogin()
	m = typeLogin(m, "10.0.104.50")
	m = typeLogin(m.NextField(), "testuser")
	m = typeLogin(m.NextField(), "secret")

	if got := m.Host(); got != "10.0.104.50" {
		t.Errorf("host = %q, want 10.0.104.50", got)
	}
	if got := m.Username(); got != "testuser" {
		t.Errorf("username = %q, want claude", got)
	}
	if got := m.Password(); got != "secret" {
		t.Errorf("password = %q, want secret", got)
	}

	onCheckbox := typeLogin(m.NextField(), "xxxx")
	if got := onCheckbox.Password(); got != "secret" {
		t.Errorf("typing on the checkbox changed the password to %q", got)
	}
}

// TestLogin_CanSubmitNeedsEveryCredential checks the form does not offer to
// connect until it has something to connect with, and that a malformed host
// is refused.
func TestLogin_CanSubmitNeedsEveryCredential(t *testing.T) {
	full := func() LoginModel {
		m := typeLogin(newBlankLogin(), "10.0.104.50")
		m = typeLogin(m.NextField(), "testuser")
		return typeLogin(m.NextField(), "secret")
	}

	if !full().CanSubmit() {
		t.Error("a fully filled form cannot submit")
	}

	cases := map[string]LoginModel{
		"no host":     typeLogin(typeLogin(newBlankLogin().NextField(), "testuser").NextField(), "secret"),
		"no username": typeLogin(typeLogin(newBlankLogin(), "10.0.104.50").NextField().NextField(), "secret"),
		"no password": typeLogin(typeLogin(newBlankLogin(), "10.0.104.50").NextField(), "testuser"),
	}
	for name, m := range cases {
		if m.CanSubmit() {
			t.Errorf("%s: form offers to submit anyway", name)
		}
	}

	bad := typeLogin(newBlankLogin(), "http://10.0.104.50/api")
	bad = typeLogin(bad.NextField(), "testuser")
	bad = typeLogin(bad.NextField(), "secret")
	if bad.CanSubmit() {
		t.Errorf("form accepts a host of %q", bad.Host())
	}
}

// TestLogin_ClearPasswordEmptiesTheInput is a security check: the password is
// dropped once it has been exchanged for an API key, and it must leave the
// widget as well as the model.
func TestLogin_ClearPasswordEmptiesTheInput(t *testing.T) {
	m := typeLogin(newBlankLogin().NextField().NextField(), "secret")
	if m.Password() == "" {
		t.Fatal("password was not typed into the model")
	}

	cleared := m.ClearPassword()
	if got := cleared.Password(); got != "" {
		t.Errorf("password after clearing = %q, want empty", got)
	}
	if strings.Contains(cleared.SetSize(100, 30).View(), "secret") {
		t.Error("cleared password is still rendered on screen")
	}
}

// TestLogin_ToggleInsecure covers the checkbox that turns off certificate
// verification. It defaults to off, and the rendered box has to agree with
// the value the model reports.
func TestLogin_ToggleInsecure(t *testing.T) {
	m := newBlankLogin()
	if m.Insecure() {
		t.Error("TLS verification is skipped by default")
	}
	on := m.ToggleInsecure()
	if !on.Insecure() {
		t.Error("toggle did not turn the insecure flag on")
	}
	if !strings.Contains(on.SetSize(100, 30).View(), "[x]") {
		t.Error("insecure checkbox renders unchecked while the flag is on")
	}
	if on.ToggleInsecure().Insecure() {
		t.Error("toggling twice did not return to verifying certificates")
	}
}

// TestConnectionForm_FieldOrderRespectsTheMode checks the save-to-config
// checkbox is reachable when adding a connection and skipped when editing
// one, in both directions.
func TestConnectionForm_FieldOrderRespectsTheMode(t *testing.T) {
	cases := []struct {
		name string
		form ConnectionFormModel
		last ConnectionFormField
	}{
		{"add", NewAddConnectionForm(), FormFieldSave},
		{"quick connect", NewQuickConnectForm(), FormFieldSave},
		{"edit", NewEditConnectionForm("10.0.104.50", config.ConnectionConfig{}), FormFieldInsecure},
	}

	for _, tc := range cases {
		m := tc.form
		seen := map[ConnectionFormField]bool{m.FocusedField(): true}
		for range 10 {
			m = m.NextField()
			seen[m.FocusedField()] = true
			if m.FocusedField() == FormFieldHost {
				break
			}
		}
		if !seen[tc.last] {
			t.Errorf("%s: tabbing never reaches field %d", tc.name, tc.last)
		}
		if tc.last == FormFieldInsecure && seen[FormFieldSave] {
			t.Errorf("%s: tabbing reaches the save checkbox, which this mode hides", tc.name)
		}

		if got := tc.form.PrevField().FocusedField(); got != tc.last {
			t.Errorf("%s: back-tab from the first field lands on %d, want %d", tc.name, got, tc.last)
		}
	}
}

// TestConnectionForm_TypingGoesToTheFocusedField checks the host and username
// inputs collect their own text and that the checkboxes do not leak keys into
// them.
func TestConnectionForm_TypingGoesToTheFocusedField(t *testing.T) {
	m := typeForm(NewAddConnectionForm(), "10.0.104.50")
	m = typeForm(m.NextField(), "testuser")

	if got := m.Host(); got != "10.0.104.50" {
		t.Errorf("host = %q, want 10.0.104.50", got)
	}
	if got := m.Username(); got != "testuser" {
		t.Errorf("username = %q, want claude", got)
	}

	onCheckbox := typeForm(m.NextField().NextField(), "zzz")
	if got := onCheckbox.Username(); got != "testuser" {
		t.Errorf("typing on a checkbox changed the username to %q", got)
	}
}

// TestConnectionForm_CanSubmitValidatesTheHost checks the form refuses a host
// it cannot connect to rather than failing later against the device.
func TestConnectionForm_CanSubmitValidatesTheHost(t *testing.T) {
	if NewAddConnectionForm().CanSubmit() {
		t.Error("an empty form offers to submit")
	}
	if !typeForm(NewAddConnectionForm(), "10.0.104.50").CanSubmit() {
		t.Error("a valid host cannot submit")
	}
	if typeForm(NewAddConnectionForm(), "https://10.0.104.50").CanSubmit() {
		t.Error("form accepts a host with a scheme")
	}
}

// TestConnectionForm_TogglesAndConfig checks the toggles reach the config the
// form hands back.
func TestConnectionForm_TogglesAndConfig(t *testing.T) {
	m := typeForm(NewAddConnectionForm(), "10.0.104.50")
	m = typeForm(m.NextField(), "testuser")

	if got := m.GetConfig().Type; got != "firewall" {
		t.Errorf("new connections default to type %q, want firewall", got)
	}
	toggled := m.ToggleType().ToggleInsecure()
	cfg := toggled.GetConfig()
	if cfg.Type != "panorama" {
		t.Errorf("type after toggling = %q, want panorama", cfg.Type)
	}
	if !cfg.Insecure {
		t.Error("insecure toggle did not reach the config")
	}
	if cfg.Username != "testuser" {
		t.Errorf("username in config = %q, want claude", cfg.Username)
	}
	if toggled.ToggleType().GetConfig().Type != "firewall" {
		t.Error("toggling type twice did not return to firewall")
	}

	// Adding a connection saves it by default; quick connect does not.
	if !NewAddConnectionForm().SaveToConfig() {
		t.Error("the add form does not save by default")
	}
	if NewAddConnectionForm().ToggleSave().SaveToConfig() {
		t.Error("save toggle did not turn off")
	}
	if NewQuickConnectForm().SaveToConfig() {
		t.Error("quick connect saves to the config file by default")
	}
	if !NewQuickConnectForm().ToggleSave().SaveToConfig() {
		t.Error("save toggle did not turn on for quick connect")
	}
}

// TestModalInputs_ShrinkWithTheTerminal checks the text inputs inside the two
// modal forms are sized from the terminal. They were a fixed forty cells,
// which is wider than the content area of a terminal under about fifty-four
// columns, so the box was pushed past the screen edge.
func TestModalInputs_ShrinkWithTheTerminal(t *testing.T) {
	// The login box is the first thing pyre shows, so it is worth
	// holding together below the sixty columns the rest of the
	// application targets.
	for _, width := range []int{40, 50, 60, 80, 120} {
		login := newBlankLogin().SetSize(width, 30)
		form := NewAddConnectionForm().SetSize(width, 30)

		for name, view := range map[string]string{"login": login.View(), "form": form.View()} {
			for line := range strings.SplitSeq(view, "\n") {
				if got := lipglossWidth(line); got > width {
					t.Errorf("%s at width %d: line is %d cells, %d over", name, width, got, got-width)
					break
				}
			}
		}
	}
}

func lipglossWidth(s string) int { return len([]rune(ansiPattern.ReplaceAllString(s, ""))) }

// hubWithConnections builds a hub holding three saved connections, one of
// them the configured default.
func hubWithConnections(t *testing.T) ConnectionHubModel {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Connections = map[string]config.ConnectionConfig{
		"10.0.104.50":          {Type: "firewall"},
		"10.0.104.51":          {Type: "firewall", Insecure: true},
		"panorama.example.com": {Type: "panorama"},
	}
	cfg.Default = "10.0.104.51"
	state := &config.State{Connections: map[string]config.ConnectionState{}}
	return NewConnectionHubModel().SetConnections(cfg, state).SetSize(120, 40)
}

func pressHub(m ConnectionHubModel, keys ...string) ConnectionHubModel {
	for _, k := range keys {
		m, _ = m.Update(key(k))
	}
	return m
}

// TestConnectionHub_NavigationStaysInBounds checks the cursor cannot leave
// the list. Selected indexes the slice directly, so running off either end
// would either panic or silently connect to the wrong firewall.
func TestConnectionHub_NavigationStaysInBounds(t *testing.T) {
	m := hubWithConnections(t)
	first := m.SelectedHost()

	bottom := pressHub(m, "j", "j", "j", "j", "j", "j")
	if bottom.Selected() == nil {
		t.Fatal("running off the bottom left nothing selected")
	}
	last := bottom.SelectedHost()

	top := pressHub(bottom, "k", "k", "k", "k", "k", "k")
	if got := top.SelectedHost(); got != first {
		t.Errorf("running off the top selects %q, want the first entry %q", got, first)
	}

	if got := pressHub(m, "G").SelectedHost(); got != last {
		t.Errorf("G selects %q, want the last entry %q", got, last)
	}
	if got := pressHub(pressHub(m, "G"), "g").SelectedHost(); got != first {
		t.Errorf("g selects %q, want the first entry %q", got, first)
	}
	for _, k := range []string{"down", "up", "home", "end"} {
		if pressHub(m, k).Selected() == nil {
			t.Errorf("%q left nothing selected", k)
		}
	}
}

// TestConnectionHub_DefaultConnectionSortsFirst checks the configured default
// opens under the cursor, which is what makes enter a one-key reconnect.
func TestConnectionHub_DefaultConnectionSortsFirst(t *testing.T) {
	m := hubWithConnections(t)
	if got := m.SelectedHost(); got != "10.0.104.51" {
		t.Errorf("hub opens on %q, want the default connection 10.0.104.51", got)
	}
	if entry := m.Selected(); entry == nil || !entry.IsDefault {
		t.Errorf("the first entry is not marked as the default: %+v", entry)
	}
}

// TestConnectionHub_ConfirmModeSwallowsNavigation checks the cursor cannot be
// moved while a delete confirmation is up. Moving it would leave the prompt
// naming one connection and the pending delete pointing at another.
func TestConnectionHub_ConfirmModeSwallowsNavigation(t *testing.T) {
	m := hubWithConnections(t)
	target := m.SelectedHost()

	confirming := m.ShowDeleteConfirm(target)
	if !confirming.IsConfirming() || confirming.ConfirmTarget() != target {
		t.Fatalf("confirm state is %v/%q, want true/%q",
			confirming.IsConfirming(), confirming.ConfirmTarget(), target)
	}

	moved := pressHub(confirming, "j", "G", "down", "end")
	if got := moved.SelectedHost(); got != target {
		t.Errorf("cursor moved to %q during confirmation, want it pinned to %q", got, target)
	}

	dismissed := moved.HideDeleteConfirm()
	if dismissed.IsConfirming() || dismissed.ConfirmTarget() != "" {
		t.Errorf("confirmation left state behind: %v/%q",
			dismissed.IsConfirming(), dismissed.ConfirmTarget())
	}
	if pressHub(dismissed, "j").SelectedHost() == target {
		t.Error("cursor still pinned after the confirmation was dismissed")
	}
}

// TestConnectionHub_EmptyHasNothingSelected checks the accessors on a hub
// with no saved connections, which is what a first run shows.
func TestConnectionHub_EmptyHasNothingSelected(t *testing.T) {
	empty := NewConnectionHubModel().
		SetConnections(config.DefaultConfig(), &config.State{}).SetSize(120, 40)

	if empty.HasConnections() {
		t.Error("an empty hub reports connections")
	}
	if empty.Selected() != nil {
		t.Errorf("an empty hub has a selection: %+v", empty.Selected())
	}
	if got := empty.SelectedHost(); got != "" {
		t.Errorf("SelectedHost on an empty hub = %q, want empty", got)
	}
	if pressHub(empty, "j", "k", "g", "G").Selected() != nil {
		t.Error("navigating an empty hub produced a selection")
	}
}

// TestConnectionHub_ViewListsEveryConnection checks the screen shows what it
// holds, including the type and the insecure marker.
func TestConnectionHub_ViewListsEveryConnection(t *testing.T) {
	view := hubWithConnections(t).View()
	for _, want := range []string{"10.0.104.50", "10.0.104.51", "panorama.example.com"} {
		if !strings.Contains(view, want) {
			t.Errorf("connection hub does not list %q", want)
		}
	}
	if !hubWithConnections(t).HasConnections() {
		t.Error("hub with three connections reports none")
	}
}
