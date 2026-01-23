package troubleshoot

import (
	"context"
	"fmt"
	"time"

	"github.com/joshuamontgomery/pyre/internal/api"
	"github.com/joshuamontgomery/pyre/internal/ssh"
)

// Engine executes troubleshooting runbooks.
type Engine struct {
	apiClient      *api.Client
	sshClient      *ssh.Client
	registry       *Registry
	patternMatcher *PatternMatcher
	stepCallback   StepCallback
}

// StepCallback is called when a step starts or completes.
type StepCallback func(stepIndex int, step Step, status StepStatus, output string)

// NewEngine creates a new troubleshooting engine.
func NewEngine(apiClient *api.Client, sshClient *ssh.Client, registry *Registry) *Engine {
	return &Engine{
		apiClient:      apiClient,
		sshClient:      sshClient,
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

	// Check SSH requirement
	if runbook.RequiresSSH && (e.sshClient == nil || !e.sshClient.IsConnected()) {
		result.Error = fmt.Errorf("runbook requires SSH but SSH is not connected")
		result.Finalize()
		return result, nil
	}

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

	switch step.Type {
	case StepTypeSSH:
		output, err = e.executeSSHStep(ctx, step)
	case StepTypeAPI:
		output, err = e.executeAPIStep(ctx, step)
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

// executeSSHStep executes an SSH command step.
func (e *Engine) executeSSHStep(ctx context.Context, step Step) (string, error) {
	if e.sshClient == nil {
		return "", fmt.Errorf("SSH client not available")
	}

	if !e.sshClient.IsConnected() {
		return "", fmt.Errorf("SSH not connected")
	}

	cmdResult, err := e.sshClient.Execute(ctx, step.Command)
	if err != nil {
		return "", err
	}

	if cmdResult.Error != nil {
		return cmdResult.Stdout + cmdResult.Stderr, cmdResult.Error
	}

	return cmdResult.Stdout + cmdResult.Stderr, nil
}

// executeAPIStep executes an API call step.
func (e *Engine) executeAPIStep(ctx context.Context, step Step) (string, error) {
	if e.apiClient == nil {
		return "", fmt.Errorf("API client not available")
	}

	// Map API calls to actual client methods
	switch step.APICall {
	case "system_info":
		info, err := e.apiClient.GetSystemInfo(ctx)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Hostname: %s\nModel: %s\nVersion: %s\nUptime: %s",
			info.Hostname, info.Model, info.Version, info.Uptime), nil

	case "system_resources":
		res, err := e.apiClient.GetSystemResources(ctx)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("CPU: %.1f%%\nMemory: %.1f%%\nLoad: %.2f/%.2f/%.2f",
			res.CPUPercent, res.MemoryPercent, res.Load1, res.Load5, res.Load15), nil

	case "ha_status":
		status, err := e.apiClient.GetHAStatus(ctx)
		if err != nil {
			return "", err
		}
		if !status.Enabled {
			return "HA not enabled", nil
		}
		return fmt.Sprintf("State: %s\nPeer: %s\nSync: %s",
			status.State, status.PeerState, status.SyncState), nil

	case "session_info":
		info, err := e.apiClient.GetSessionInfo(ctx)
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

	if runbook.RequiresSSH {
		if e.sshClient == nil {
			return false, "SSH not configured"
		}
		if !e.sshClient.IsConnected() {
			return false, "SSH not connected"
		}
	}

	return true, ""
}

// GetRegistry returns the runbook registry.
func (e *Engine) GetRegistry() *Registry {
	return e.registry
}

// HasSSH returns true if SSH is available.
func (e *Engine) HasSSH() bool {
	return e.sshClient != nil && e.sshClient.IsConnected()
}
