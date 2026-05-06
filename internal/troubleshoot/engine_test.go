package troubleshoot

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jp2195/pyre/internal/models"
)

type fakeAPIClient struct {
	systemInfo      *models.SystemInfo
	systemResources *models.Resources
	haStatus        *models.HAStatus
	sessionInfo     *models.SessionInfo
	err             error
}

func (f *fakeAPIClient) GetSystemInfo(_ context.Context, _ string) (*models.SystemInfo, error) {
	return f.systemInfo, f.err
}
func (f *fakeAPIClient) GetSystemResources(_ context.Context, _ string) (*models.Resources, error) {
	return f.systemResources, f.err
}
func (f *fakeAPIClient) GetHAStatus(_ context.Context, _ string) (*models.HAStatus, error) {
	return f.haStatus, f.err
}
func (f *fakeAPIClient) GetSessionInfo(_ context.Context, _ string) (*models.SessionInfo, error) {
	return f.sessionInfo, f.err
}

func TestNewEngine(t *testing.T) {
	registry := NewRegistry()
	engine := NewEngine(nil, registry)

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
	engine := NewEngine(nil, registry)

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
	engine := NewEngine(nil, registry)

	if engine.GetRegistry() != registry {
		t.Error("expected GetRegistry to return the registry")
	}
}

func TestEngine_CanRun(t *testing.T) {
	registry := NewRegistry()

	t.Run("no API client", func(t *testing.T) {
		engine := NewEngine(nil, registry)
		runbook := &Runbook{ID: "test"}

		canRun, reason := engine.CanRun(runbook)
		if canRun {
			t.Error("expected CanRun to be false with no API client")
		}
		if reason != "API client not available" {
			t.Errorf("expected reason 'API client not available', got %q", reason)
		}
	})
}

func TestEngine_Run_RunbookNotFound(t *testing.T) {
	registry := NewRegistry()
	engine := NewEngine(nil, registry)

	ctx := context.Background()
	_, err := engine.Run(ctx, "nonexistent")

	if err == nil {
		t.Error("expected error for nonexistent runbook")
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

	engine := NewEngine(nil, registry)

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

	engine := NewEngine(nil, registry)

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

	engine := NewEngine(nil, registry)

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

func TestEngine_RequiredStepFailure(t *testing.T) {
	registry := NewRegistry()
	runbook := &Runbook{
		ID: "multi-step",
		Steps: []Step{
			{ID: "step1", Type: StepTypeAPI, APICall: "system_info", Required: true},
			{ID: "step2", Type: StepTypeAPI, APICall: "ha_status"},
		},
	}

	engine := NewEngine(nil, registry)

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

	engine := NewEngine(nil, registry)

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

	engine := NewEngine(nil, registry)

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

func TestExecuteAPIStep_SystemInfo(t *testing.T) {
	fake := &fakeAPIClient{
		systemInfo: &models.SystemInfo{
			Hostname: "fw01",
			Model:    "PA-440",
			Version:  "11.1.0",
			Uptime:   "5 days",
		},
	}
	eng := NewEngine(fake, NewRegistry())
	step := Step{ID: "info", Type: StepTypeAPI, APICall: "system_info"}

	out, err := eng.executeAPIStep(context.Background(), step)
	if err != nil {
		t.Fatalf("executeAPIStep err: %v", err)
	}
	for _, want := range []string{"fw01", "PA-440", "11.1.0", "5 days"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got: %s", want, out)
		}
	}
}

func TestExecuteAPIStep_SystemResources(t *testing.T) {
	fake := &fakeAPIClient{
		systemResources: &models.Resources{
			CPUPercent:    42.5,
			MemoryPercent: 67.3,
			Load1:         1.2,
			Load5:         0.9,
			Load15:        0.7,
		},
	}
	eng := NewEngine(fake, NewRegistry())
	step := Step{ID: "res", Type: StepTypeAPI, APICall: "system_resources"}

	out, err := eng.executeAPIStep(context.Background(), step)
	if err != nil {
		t.Fatalf("executeAPIStep err: %v", err)
	}
	for _, want := range []string{"42.5", "67.3", "1.20", "CPU", "Memory"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got: %s", want, out)
		}
	}
}

func TestExecuteAPIStep_HAStatus(t *testing.T) {
	fake := &fakeAPIClient{
		haStatus: &models.HAStatus{
			Enabled:   true,
			State:     "active",
			PeerState: "passive",
			SyncState: "synchronized",
		},
	}
	eng := NewEngine(fake, NewRegistry())
	step := Step{ID: "ha", Type: StepTypeAPI, APICall: "ha_status"}

	out, err := eng.executeAPIStep(context.Background(), step)
	if err != nil {
		t.Fatalf("executeAPIStep err: %v", err)
	}
	for _, want := range []string{"active", "passive", "synchronized"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got: %s", want, out)
		}
	}
}

func TestExecuteAPIStep_HADisabled(t *testing.T) {
	fake := &fakeAPIClient{
		haStatus: &models.HAStatus{Enabled: false},
	}
	eng := NewEngine(fake, NewRegistry())
	step := Step{ID: "ha", Type: StepTypeAPI, APICall: "ha_status"}

	out, err := eng.executeAPIStep(context.Background(), step)
	if err != nil {
		t.Fatalf("executeAPIStep err: %v", err)
	}
	if !strings.Contains(out, "HA not enabled") {
		t.Errorf("expected 'HA not enabled' in output, got: %s", out)
	}
}

func TestExecuteAPIStep_SessionInfo(t *testing.T) {
	fake := &fakeAPIClient{
		sessionInfo: &models.SessionInfo{
			ActiveCount: 12345,
			MaxCount:    250000,
			CPS:         678,
		},
	}
	eng := NewEngine(fake, NewRegistry())
	step := Step{ID: "sess", Type: StepTypeAPI, APICall: "session_info"}

	out, err := eng.executeAPIStep(context.Background(), step)
	if err != nil {
		t.Fatalf("executeAPIStep err: %v", err)
	}
	for _, want := range []string{"12345", "250000", "678"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got: %s", want, out)
		}
	}
}

func TestExecuteAPIStep_APIError(t *testing.T) {
	wantErr := errors.New("boom")
	fake := &fakeAPIClient{err: wantErr}
	eng := NewEngine(fake, NewRegistry())
	step := Step{ID: "info", Type: StepTypeAPI, APICall: "system_info"}

	_, err := eng.executeAPIStep(context.Background(), step)
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped error %v, got %v", wantErr, err)
	}
}

func TestEngine_PatternMatching(t *testing.T) {
	registry := NewRegistry()

	// Create a mock test that exercises pattern matching
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
	engine := NewEngine(nil, registry)
	if engine.patternMatcher == nil {
		t.Error("expected engine to have pattern matcher")
	}
}
