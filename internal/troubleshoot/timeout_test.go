package troubleshoot

import (
	"testing"
	"time"
)

func TestStep_EffectiveTimeout_Default(t *testing.T) {
	s := Step{ID: "s1", Type: StepTypeAPI}
	if got := s.effectiveTimeout(); got != defaultStepTimeout {
		t.Errorf("expected default %s, got %s", defaultStepTimeout, got)
	}
}

func TestStep_EffectiveTimeout_Zero(t *testing.T) {
	s := Step{ID: "s1", Type: StepTypeAPI, Timeout: 0}
	if got := s.effectiveTimeout(); got != defaultStepTimeout {
		t.Errorf("expected default %s for zero Timeout, got %s", defaultStepTimeout, got)
	}
}

func TestStep_EffectiveTimeout_Negative(t *testing.T) {
	// Negative timeouts are treated as unset and fall through to the default,
	// per the ">0" contract in effectiveTimeout.
	s := Step{ID: "s1", Type: StepTypeAPI, Timeout: -1 * time.Second}
	if got := s.effectiveTimeout(); got != defaultStepTimeout {
		t.Errorf("expected default %s for negative Timeout, got %s", defaultStepTimeout, got)
	}
}

func TestStep_EffectiveTimeout_Custom(t *testing.T) {
	want := 250 * time.Millisecond
	s := Step{ID: "s1", Type: StepTypeAPI, Timeout: want}
	if got := s.effectiveTimeout(); got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestDefaultStepTimeout_Is30s(t *testing.T) {
	// Pin the documented default so accidental regressions show up in tests.
	if defaultStepTimeout != 30*time.Second {
		t.Errorf("expected defaultStepTimeout to be 30s, got %s", defaultStepTimeout)
	}
}

// NOTE: End-to-end verification that executeStep honors the derived
// context deadline (i.e. slow httptest server + Step{Timeout: 50ms}
// returning context.DeadlineExceeded) is deferred to an integration
// suite. The api.Client currently composes its base URL as
// https://<host>/api/, which is awkward to point at an httptest.Server
// from within this package without leaking test-only wiring into the
// client. The effectiveTimeout() assertions above plus the context.
// WithTimeout wiring in engine.executeStep cover the logic here.
