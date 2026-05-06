package troubleshoot

import (
	"context"
	"fmt"
	"time"

	"github.com/jp2195/pyre/internal/models"
)

// APIClient is the narrow PAN-OS API surface the troubleshoot engine needs.
// *api.Client satisfies this interface implicitly, allowing tests to inject
// a fake without depending on the full api.Client type.
type APIClient interface {
	GetSystemInfo(ctx context.Context, target string) (*models.SystemInfo, error)
	GetSystemResources(ctx context.Context, target string) (*models.Resources, error)
	GetHAStatus(ctx context.Context, target string) (*models.HAStatus, error)
	GetSessionInfo(ctx context.Context, target string) (*models.SessionInfo, error)
}

// Engine executes troubleshooting runbooks.
type Engine struct {
	apiClient      APIClient
	registry       *Registry
	patternMatcher *PatternMatcher
	stepCallback   StepCallback
}

// StepCallback is called when a step starts or completes.
type StepCallback func(stepIndex int, step Step, status StepStatus, output string)

// NewEngine creates a new troubleshooting engine.
func NewEngine(apiClient APIClient, registry *Registry) *Engine {
	return &Engine{
		apiClient:      apiClient,
		registry:       registry,
		patternMatcher: NewPatternMatcher(),
	}
}

// SetStepCallback sets a callback function for step progress updates.
func (e *Engine) SetStepCallback(cb StepCallback) {
	e.stepCallback = cb
}

// Run executes a runbook by ID and returns the result.
func (e *Engine) Run(ctx context.Context, runbookID string) (*RunbookResult, error) {
	runbook, ok := e.registry.Get(runbookID)
	if !ok {
		return nil, fmt.Errorf("runbook not found: %s", runbookID)
	}

	return e.RunRunbook(ctx, runbook)
}

// RunRunbook executes a runbook directly.
func (e *Engine) RunRunbook(ctx context.Context, runbook *Runbook) (*RunbookResult, error) {
	result := NewRunbookResult(runbook)

	// Execute each step
	for i, step := range runbook.Steps {
		select {
		case <-ctx.Done():
			result.Error = ctx.Err()
			result.Finalize()
			return result, nil
		default:
		}

		stepResult := e.executeStep(ctx, i, step)
		result.AddStepResult(stepResult)

		// Stop if required step failed
		if step.Required && (stepResult.Status == StepStatusFailed || stepResult.Status == StepStatusError) {
			break
		}
	}

	result.Finalize()
	return result, nil
}

// executeStep runs a single step and returns the result.
func (e *Engine) executeStep(ctx context.Context, index int, step Step) StepResult {
	result := StepResult{
		Step:   step,
		Status: StepStatusRunning,
	}

	// Notify callback
	if e.stepCallback != nil {
		e.stepCallback(index, step, StepStatusRunning, "")
	}

	start := time.Now()

	var output string
	var err error

	// Apply per-step timeout so a hung API call cannot stall the whole runbook.
	// stepCtx bounds executeAPIStep only. MatchAll runs after the API call
	// returns and is not context-aware; pattern matching is fast enough today
	// that running it unbounded is intentional. If patterns ever scan large
	// outputs, thread ctx through PatternMatcher.MatchAll.
	stepCtx, cancel := context.WithTimeout(ctx, step.effectiveTimeout())
	defer cancel()

	switch step.Type {
	case StepTypeAPI:
		output, err = e.executeAPIStep(stepCtx, step)
	default:
		err = fmt.Errorf("unknown step type: %s", step.Type)
	}

	result.Duration = time.Since(start)

	if err != nil {
		result.Status = StepStatusError
		result.Error = err
	} else {
		result.Output = output
		// Match patterns
		matches, matchErr := e.patternMatcher.MatchAll(step.Patterns, output)
		if matchErr != nil {
			result.Status = StepStatusError
			result.Error = matchErr
		} else {
			result.Matches = matches
			// Determine status based on matches
			result.Status = StepStatusPassed
			for _, match := range matches {
				if match.Pattern.Severity == SeverityCritical || match.Pattern.Severity == SeverityError {
					result.Status = StepStatusFailed
					break
				}
			}
		}
	}

	// Notify callback
	if e.stepCallback != nil {
		e.stepCallback(index, step, result.Status, output)
	}

	return result
}

// executeAPIStep executes an API call step.
func (e *Engine) executeAPIStep(ctx context.Context, step Step) (string, error) {
	if e.apiClient == nil {
		return "", fmt.Errorf("API client not available")
	}

	// Map API calls to actual client methods
	switch step.APICall {
	case "system_info":
		info, err := e.apiClient.GetSystemInfo(ctx, "")
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Hostname: %s\nModel: %s\nVersion: %s\nUptime: %s",
			info.Hostname, info.Model, info.Version, info.Uptime), nil

	case "system_resources":
		res, err := e.apiClient.GetSystemResources(ctx, "")
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("CPU: %.1f%%\nMemory: %.1f%%\nLoad: %.2f/%.2f/%.2f",
			res.CPUPercent, res.MemoryPercent, res.Load1, res.Load5, res.Load15), nil

	case "ha_status":
		status, err := e.apiClient.GetHAStatus(ctx, "")
		if err != nil {
			return "", err
		}
		if !status.Enabled {
			return "HA not enabled", nil
		}
		return fmt.Sprintf("State: %s\nPeer: %s\nSync: %s",
			status.State, status.PeerState, status.SyncState), nil

	case "session_info":
		info, err := e.apiClient.GetSessionInfo(ctx, "")
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Active: %d\nMax: %d\nCPS: %d",
			info.ActiveCount, info.MaxCount, info.CPS), nil

	default:
		return "", fmt.Errorf("unknown API call: %s", step.APICall)
	}
}

// CanRun checks if a runbook can be executed with current connections.
func (e *Engine) CanRun(runbook *Runbook) (bool, string) {
	if e.apiClient == nil {
		return false, "API client not available"
	}
	return true, ""
}

// GetRegistry returns the runbook registry.
func (e *Engine) GetRegistry() *Registry {
	return e.registry
}
