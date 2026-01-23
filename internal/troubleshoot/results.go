package troubleshoot

import (
	"fmt"
	"time"
)

// StepStatus represents the status of a step execution.
type StepStatus string

const (
	StepStatusPending  StepStatus = "pending"
	StepStatusRunning  StepStatus = "running"
	StepStatusPassed   StepStatus = "passed"
	StepStatusFailed   StepStatus = "failed"
	StepStatusSkipped  StepStatus = "skipped"
	StepStatusError    StepStatus = "error"
)

// RunbookResult contains the complete result of a runbook execution.
type RunbookResult struct {
	Runbook   *Runbook
	Steps     []StepResult
	Issues    []Issue
	Passed    bool
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Error     error
}

// StepResult contains the result of a single step execution.
type StepResult struct {
	Step     Step
	Status   StepStatus
	Output   string
	Error    error
	Matches  []MatchResult
	Duration time.Duration
}

// Issue represents a detected problem with remediation info.
type Issue struct {
	StepID      string
	StepName    string
	PatternID   string
	PatternName string
	Severity    Severity
	Message     string
	MatchedText string
	KBArticles  []string
	Remediation string
}

// NewRunbookResult creates a new runbook result.
func NewRunbookResult(runbook *Runbook) *RunbookResult {
	return &RunbookResult{
		Runbook:   runbook,
		Steps:     make([]StepResult, 0, len(runbook.Steps)),
		Issues:    make([]Issue, 0),
		StartTime: time.Now(),
	}
}

// AddStepResult adds a step result and extracts any issues.
func (r *RunbookResult) AddStepResult(result StepResult) {
	r.Steps = append(r.Steps, result)

	// Extract issues from pattern matches
	for _, match := range result.Matches {
		issue := Issue{
			StepID:      result.Step.ID,
			StepName:    result.Step.Name,
			PatternID:   match.Pattern.ID,
			PatternName: match.Pattern.Name,
			Severity:    match.Pattern.Severity,
			Message:     match.Pattern.Message,
			MatchedText: match.MatchedText,
			KBArticles:  match.Pattern.KBArticles,
			Remediation: match.Pattern.Remediation,
		}
		r.Issues = append(r.Issues, issue)
	}
}

// Finalize marks the result as complete.
func (r *RunbookResult) Finalize() {
	r.EndTime = time.Now()
	r.Duration = r.EndTime.Sub(r.StartTime)

	// Determine overall pass/fail
	r.Passed = true

	// Fail if there's an error (e.g., SSH not available)
	if r.Error != nil {
		r.Passed = false
		return
	}

	// Fail if no steps were executed
	if len(r.Steps) == 0 {
		r.Passed = false
		return
	}

	for _, step := range r.Steps {
		if step.Status == StepStatusFailed || step.Status == StepStatusError {
			r.Passed = false
			break
		}
	}

	// Also fail if there are critical or error severity issues
	for _, issue := range r.Issues {
		if issue.Severity == SeverityCritical || issue.Severity == SeverityError {
			r.Passed = false
			break
		}
	}
}

// HasIssues returns true if any issues were detected.
func (r *RunbookResult) HasIssues() bool {
	return len(r.Issues) > 0
}

// IssuesBySeverity returns issues filtered by severity.
func (r *RunbookResult) IssuesBySeverity(severity Severity) []Issue {
	var filtered []Issue
	for _, issue := range r.Issues {
		if issue.Severity == severity {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

// CriticalIssues returns all critical severity issues.
func (r *RunbookResult) CriticalIssues() []Issue {
	return r.IssuesBySeverity(SeverityCritical)
}

// ErrorIssues returns all error severity issues.
func (r *RunbookResult) ErrorIssues() []Issue {
	return r.IssuesBySeverity(SeverityError)
}

// WarningIssues returns all warning severity issues.
func (r *RunbookResult) WarningIssues() []Issue {
	return r.IssuesBySeverity(SeverityWarning)
}

// InfoIssues returns all info severity issues.
func (r *RunbookResult) InfoIssues() []Issue {
	return r.IssuesBySeverity(SeverityInfo)
}

// Summary returns a brief text summary of the result.
func (r *RunbookResult) Summary() string {
	status := "PASSED"
	if !r.Passed {
		status = "FAILED"
	}

	critical := len(r.CriticalIssues())
	errors := len(r.ErrorIssues())
	warnings := len(r.WarningIssues())

	return fmt.Sprintf("%s - %d critical, %d errors, %d warnings (took %s)",
		status, critical, errors, warnings, r.Duration.Round(time.Millisecond))
}

// StatusIcon returns an icon/emoji for the status.
func StatusIcon(status StepStatus) string {
	switch status {
	case StepStatusPassed:
		return "OK"
	case StepStatusFailed:
		return "FAIL"
	case StepStatusError:
		return "ERR"
	case StepStatusSkipped:
		return "SKIP"
	case StepStatusRunning:
		return "..."
	default:
		return " "
	}
}

// SeverityIcon returns an icon for the severity level.
func SeverityIcon(severity Severity) string {
	switch severity {
	case SeverityCritical:
		return "[!!!]"
	case SeverityError:
		return "[!!]"
	case SeverityWarning:
		return "[!]"
	case SeverityInfo:
		return "[i]"
	default:
		return "[ ]"
	}
}
