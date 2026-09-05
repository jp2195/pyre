package tui

// epoch_test.go – the other half of the stale-data problem.
//
// Clearing the views on a switch stops *old* data being displayed under the
// new device's name, but it does nothing about a request that was already in
// flight when the switch happened. That response arrives afterwards, looks
// exactly like a fresh one, and repopulates the view with the previous
// device's data. The window is not small: the PAN-OS management plane is slow
// and a policy load issues a dozen-plus calls.

import (
	"testing"

	"github.com/jp2195/pyre/internal/config"
	"github.com/jp2195/pyre/internal/models"
)

func TestStaleResponse_IsDropped(t *testing.T) {
	m := newTestModel(t, ViewPolicies)
	before := m.connEpoch

	// A device switch clears the views and invalidates anything in flight.
	m.resetViewData()
	if m.connEpoch == before {
		t.Fatal("resetViewData did not invalidate in-flight requests")
	}

	// The previous device's policy fetch now lands.
	updated, _ := m.Update(epochMsg{
		epoch: before,
		inner: PoliciesMsg{Policies: []models.SecurityRule{{Name: "previous-device-rule"}}},
	})
	nm := updated.(Model)

	if nm.policies.HasData() {
		t.Error("a response from before the switch repopulated the view")
	}
}

func TestCurrentResponse_IsApplied(t *testing.T) {
	m := newTestModel(t, ViewPolicies)
	m.resetViewData()

	updated, _ := m.Update(epochMsg{
		epoch: m.connEpoch,
		inner: PoliciesMsg{Policies: []models.SecurityRule{{Name: "current-rule"}}},
	})
	nm := updated.(Model)

	if !nm.policies.HasData() {
		t.Error("a response for the current connection was dropped")
	}
}

// TestFetchCommandsCarryTheEpoch checks the tagging actually happens in the
// production fetch path, not just that Update knows how to read a tag.
func TestFetchCommandsCarryTheEpoch(t *testing.T) {
	m := newTestModel(t, ViewPolicies)
	// Port 1 refuses immediately, so the command completes without a server
	// and without waiting out a timeout. The inner message carries the
	// resulting error; what matters here is the envelope.
	if _, err := m.session.AddConnection("127.0.0.1:1", &config.ConnectionConfig{}, "k"); err != nil {
		t.Fatalf("AddConnection: %v", err)
	}
	m.resetViewData()

	cmd := m.fetchPolicies()
	if cmd == nil {
		t.Fatal("fetchPolicies returned no command")
	}
	msg := cmd()

	tagged, ok := msg.(epochMsg)
	if !ok {
		t.Fatalf("fetch produced %T, want it wrapped so stale responses can be dropped", msg)
	}
	if tagged.epoch != m.connEpoch {
		t.Errorf("epoch = %d, want %d", tagged.epoch, m.connEpoch)
	}
	if _, ok := tagged.inner.(PoliciesMsg); !ok {
		t.Errorf("inner = %T, want PoliciesMsg", tagged.inner)
	}
}
