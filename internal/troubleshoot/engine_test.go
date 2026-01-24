package troubleshoot

import (
	"context"
	"testing"
)

func TestNewEngine(t *testing.T) {
	registry := NewRegistry()
	engine := NewEngine(nil, nil, registry)

	if engine == nil {
		t.Fatal("expected non-nil engine")
	}
	if engine.registry != registry {
		t.Error("expected registry to match")
	}
	if engine.patternMatcher == nil {
		t.Error("expected pattern matcher to be initialized")
	}
}

func TestEngine_SetStepCallback(t *testing.T) {
	registry := NewRegistry()
	engine := NewEngine(nil, nil, registry)

	callback := func(stepIndex int, step Step, status StepStatus, output string) {
		// Callback set
	}

	engine.SetStepCallback(callback)
	if engine.stepCallback == nil {
		t.Error("expected callback to be set")
	}
}

func TestEngine_GetRegistry(t *testing.T) {
	registry := NewRegistry()
	engine := NewEngine(nil, nil, registry)

	if engine.GetRegistry() != registry {
		t.Error("expected GetRegistry to return the registry")
	}
}

func TestEngine_HasSSH(t *testing.T) {
	registry := NewRegistry()

	// No SSH client
	engine := NewEngine(nil, nil, registry)
	if engine.HasSSH() {
		t.Error("expected HasSSH to be false with nil SSH client")
	}
}

func TestEngine_CanRun(t *testing.T) {
	registry := NewRegistry()

	t.Run("no API client", func(t *testing.T) {
		engine := NewEngine(nil, nil, registry)
		runbook := &Runbook{ID: "test"}

		canRun, reason := engine.CanRun(runbook)
		if canRun {
			t.Error("expected CanRun to be false with no API client")
		}
		if reason != "API client not available" {
			t.Errorf("expected reason 'API client not available', got %q", reason)
		}
	})

	t.Run("requires SSH but SSH nil", func(t *testing.T) {
		// Use a non-nil API client placeholder (can't actually call it)
		engine := NewEngine(nil, nil, registry)
		runbook := &Runbook{ID: "test", RequiresSSH: true}

		// Since apiClient is nil, it should fail at API check first
		canRun, reason := engine.CanRun(runbook)
		if canRun {
			t.Error("expected CanRun to be false")
		}
		// Will fail at API client check before SSH check
		if reason == "" {
			t.Error("expected non-empty reason")
		}
	})
}

func TestEngine_Run_RunbookNotFound(t *testing.T) {
	registry := NewRegistry()
	engine := NewEngine(nil, nil, registry)

	ctx := context.Background()
	_, err := engine.Run(ctx, "nonexistent")

	if err == nil {
		t.Error("expected error for nonexistent runbook")
	}
}

func TestEngine_Run_NoSSHForSSHRunbook(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&Runbook{
		ID:          "ssh-runbook",
		Name:        "SSH Runbook",
		RequiresSSH: true,
		Steps: []Step{
			{ID: "step1", Type: StepTypeSSH, Command: "show clock"},
		},
	})

	engine := NewEngine(nil, nil, registry)

	ctx := context.Background()
	result, err := engine.Run(ctx, "ssh-runbook")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Error == nil {
		t.Error("expected result error for missing SSH")
	}
	if result.Passed {
		t.Error("expected Passed to be false")
	}
}

func TestEngine_RunRunbook_ContextCancellation(t *testing.T) {
	registry := NewRegistry()
	runbook := &Runbook{
		ID: "test-runbook",
		Steps: []Step{
			{ID: "step1", Type: StepTypeAPI, APICall: "system_info"},
		},
	}

	engine := NewEngine(nil, nil, registry)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := engine.RunRunbook(ctx, runbook)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Error == nil {
		t.Error("expected result error for cancelled context")
	}
}

func TestEngine_ExecuteStep_UnknownType(t *testing.T) {
	registry := NewRegistry()
	runbook := &Runbook{
		ID: "test-runbook",
		Steps: []Step{
			{ID: "step1", Type: StepType("unknown")},
		},
	}

	engine := NewEngine(nil, nil, registry)

	ctx := context.Background()
	result, err := engine.RunRunbook(ctx, runbook)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Steps) != 1 {
		t.Fatalf("expected 1 step result, got %d", len(result.Steps))
	}
	if result.Steps[0].Status != StepStatusError {
		t.Errorf("expected status Error for unknown type, got %s", result.Steps[0].Status)
	}
}

func TestEngine_ExecuteAPIStep_NoClient(t *testing.T) {
	registry := NewRegistry()
	runbook := &Runbook{
		ID: "api-runbook",
		Steps: []Step{
			{ID: "step1", Type: StepTypeAPI, APICall: "system_info"},
		},
	}

	engine := NewEngine(nil, nil, registry)

	ctx := context.Background()
	result, err := engine.RunRunbook(ctx, runbook)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Steps) != 1 {
		t.Fatalf("expected 1 step result, got %d", len(result.Steps))
	}
	if result.Steps[0].Status != StepStatusError {
		t.Errorf("expected status Error for no API client, got %s", result.Steps[0].Status)
	}
}

func TestEngine_ExecuteSSHStep_NoClient(t *testing.T) {
	registry := NewRegistry()
	runbook := &Runbook{
		ID:          "ssh-runbook",
		RequiresSSH: false, // Don't check at runbook level
		Steps: []Step{
			{ID: "step1", Type: StepTypeSSH, Command: "show clock"},
		},
	}

	engine := NewEngine(nil, nil, registry)

	ctx := context.Background()
	result, err := engine.RunRunbook(ctx, runbook)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Steps) != 1 {
		t.Fatalf("expected 1 step result, got %d", len(result.Steps))
	}
	if result.Steps[0].Status != StepStatusError {
		t.Errorf("expected status Error for no SSH client, got %s", result.Steps[0].Status)
	}
}

func TestEngine_RequiredStepFailure(t *testing.T) {
	registry := NewRegistry()
	runbook := &Runbook{
		ID: "multi-step",
		Steps: []Step{
			{ID: "step1", Type: StepTypeAPI, APICall: "system_info", Required: true},
			{ID: "step2", Type: StepTypeAPI, APICall: "ha_status"},
		},
	}

	engine := NewEngine(nil, nil, registry)

	ctx := context.Background()
	result, err := engine.RunRunbook(ctx, runbook)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First step should fail (no API client), second should not run
	if len(result.Steps) != 1 {
		t.Errorf("expected 1 step (stopped at required failure), got %d", len(result.Steps))
	}
}

func TestEngine_NonRequiredStepFailure(t *testing.T) {
	registry := NewRegistry()
	runbook := &Runbook{
		ID: "multi-step",
		Steps: []Step{
			{ID: "step1", Type: StepTypeAPI, APICall: "system_info", Required: false},
			{ID: "step2", Type: StepTypeAPI, APICall: "ha_status", Required: false},
		},
	}

	engine := NewEngine(nil, nil, registry)

	ctx := context.Background()
	result, err := engine.RunRunbook(ctx, runbook)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both steps should run (non-required)
	if len(result.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(result.Steps))
	}
}

func TestEngine_StepCallback(t *testing.T) {
	registry := NewRegistry()
	runbook := &Runbook{
		ID: "callback-test",
		Steps: []Step{
			{ID: "step1", Type: StepType("test")},
		},
	}

	engine := NewEngine(nil, nil, registry)

	callbackCount := 0
	var receivedStatuses []StepStatus

	engine.SetStepCallback(func(stepIndex int, step Step, status StepStatus, output string) {
		callbackCount++
		receivedStatuses = append(receivedStatuses, status)
	})

	ctx := context.Background()
	_, _ = engine.RunRunbook(ctx, runbook)

	// Callback should be called twice: once for running, once for final status
	if callbackCount != 2 {
		t.Errorf("expected callback called 2 times, got %d", callbackCount)
	}

	if len(receivedStatuses) != 2 {
		t.Fatalf("expected 2 statuses, got %d", len(receivedStatuses))
	}

	if receivedStatuses[0] != StepStatusRunning {
		t.Errorf("expected first status Running, got %s", receivedStatuses[0])
	}
}

func TestEngine_PatternMatching(t *testing.T) {
	registry := NewRegistry()

	// Create a mock test that exercises pattern matching
	// Since we can't easily mock API/SSH, we'll test the pattern matching part
	pm := NewPatternMatcher()

	patterns := []Pattern{
		{ID: "p1", Regex: "error", Severity: SeverityError},
		{ID: "p2", Regex: "warning", Severity: SeverityWarning},
	}

	matches, err := pm.MatchAll(patterns, "found an error and warning here")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}

	// Verify engine uses pattern matcher correctly
	engine := NewEngine(nil, nil, registry)
	if engine.patternMatcher == nil {
		t.Error("expected engine to have pattern matcher")
	}
}
