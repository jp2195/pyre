package troubleshoot

import (
	"errors"
	"testing"
	"time"
)

func TestNewRunbookResult(t *testing.T) {
	runbook := &Runbook{
		ID:   "test-rb",
		Name: "Test Runbook",
		Steps: []Step{
			{ID: "step1"},
			{ID: "step2"},
		},
	}

	result := NewRunbookResult(runbook)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Runbook != runbook {
		t.Error("expected runbook reference to match")
	}
	if result.Steps == nil {
		t.Error("expected Steps to be initialized")
	}
	if result.Issues == nil {
		t.Error("expected Issues to be initialized")
	}
	if result.StartTime.IsZero() {
		t.Error("expected StartTime to be set")
	}
}

func TestRunbookResult_AddStepResult(t *testing.T) {
	runbook := &Runbook{ID: "test-rb"}
	result := NewRunbookResult(runbook)

	stepResult := StepResult{
		Step:   Step{ID: "step1", Name: "Step 1"},
		Status: StepStatusPassed,
		Output: "test output",
		Matches: []MatchResult{
			{
				Pattern: Pattern{
					ID:          "p1",
					Name:        "Warning Pattern",
					Severity:    SeverityWarning,
					Message:     "Warning message",
					KBArticles:  []string{"KB001"},
					Remediation: "Fix it",
				},
				MatchedText: "warning text",
			},
		},
	}

	result.AddStepResult(stepResult)

	if len(result.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(result.Steps))
	}
	if len(result.Issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(result.Issues))
	}

	issue := result.Issues[0]
	if issue.StepID != "step1" {
		t.Errorf("expected StepID 'step1', got %q", issue.StepID)
	}
	if issue.Severity != SeverityWarning {
		t.Errorf("expected severity Warning, got %s", issue.Severity)
	}
	if issue.MatchedText != "warning text" {
		t.Errorf("expected matched text 'warning text', got %q", issue.MatchedText)
	}
}

func TestRunbookResult_Finalize(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*RunbookResult)
		wantPassed bool
	}{
		{
			name: "passed with no issues",
			setup: func(r *RunbookResult) {
				r.AddStepResult(StepResult{
					Step:   Step{ID: "step1"},
					Status: StepStatusPassed,
				})
			},
			wantPassed: true,
		},
		{
			name: "failed with failed step",
			setup: func(r *RunbookResult) {
				r.AddStepResult(StepResult{
					Step:   Step{ID: "step1"},
					Status: StepStatusFailed,
				})
			},
			wantPassed: false,
		},
		{
			name: "failed with error step",
			setup: func(r *RunbookResult) {
				r.AddStepResult(StepResult{
					Step:   Step{ID: "step1"},
					Status: StepStatusError,
				})
			},
			wantPassed: false,
		},
		{
			name: "failed with runbook error",
			setup: func(r *RunbookResult) {
				r.Error = errors.New("runbook error")
			},
			wantPassed: false,
		},
		{
			name: "failed with no steps",
			setup: func(r *RunbookResult) {
				// No steps added
			},
			wantPassed: false,
		},
		{
			name: "failed with critical issue",
			setup: func(r *RunbookResult) {
				r.AddStepResult(StepResult{
					Step:   Step{ID: "step1"},
					Status: StepStatusPassed,
					Matches: []MatchResult{
						{Pattern: Pattern{Severity: SeverityCritical}},
					},
				})
			},
			wantPassed: false,
		},
		{
			name: "failed with error severity issue",
			setup: func(r *RunbookResult) {
				r.AddStepResult(StepResult{
					Step:   Step{ID: "step1"},
					Status: StepStatusPassed,
					Matches: []MatchResult{
						{Pattern: Pattern{Severity: SeverityError}},
					},
				})
			},
			wantPassed: false,
		},
		{
			name: "passed with warning issue",
			setup: func(r *RunbookResult) {
				r.AddStepResult(StepResult{
					Step:   Step{ID: "step1"},
					Status: StepStatusPassed,
					Matches: []MatchResult{
						{Pattern: Pattern{Severity: SeverityWarning}},
					},
				})
			},
			wantPassed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runbook := &Runbook{ID: "test-rb"}
			result := NewRunbookResult(runbook)
			tt.setup(result)
			result.Finalize()

			if result.Passed != tt.wantPassed {
				t.Errorf("expected Passed=%v, got %v", tt.wantPassed, result.Passed)
			}
			if result.EndTime.IsZero() {
				t.Error("expected EndTime to be set")
			}
			if result.Duration == 0 {
				t.Error("expected Duration to be non-zero")
			}
		})
	}
}

func TestRunbookResult_HasIssues(t *testing.T) {
	runbook := &Runbook{ID: "test-rb"}

	// No issues
	result := NewRunbookResult(runbook)
	if result.HasIssues() {
		t.Error("expected no issues")
	}

	// With issues
	result.AddStepResult(StepResult{
		Step: Step{ID: "step1"},
		Matches: []MatchResult{
			{Pattern: Pattern{Severity: SeverityWarning}},
		},
	})
	if !result.HasIssues() {
		t.Error("expected to have issues")
	}
}

func TestRunbookResult_IssuesBySeverity(t *testing.T) {
	runbook := &Runbook{ID: "test-rb"}
	result := NewRunbookResult(runbook)

	// Add steps with various severity issues
	result.AddStepResult(StepResult{
		Step: Step{ID: "step1"},
		Matches: []MatchResult{
			{Pattern: Pattern{ID: "p1", Severity: SeverityCritical}},
			{Pattern: Pattern{ID: "p2", Severity: SeverityError}},
			{Pattern: Pattern{ID: "p3", Severity: SeverityWarning}},
			{Pattern: Pattern{ID: "p4", Severity: SeverityInfo}},
		},
	})
	result.AddStepResult(StepResult{
		Step: Step{ID: "step2"},
		Matches: []MatchResult{
			{Pattern: Pattern{ID: "p5", Severity: SeverityWarning}},
		},
	})

	critical := result.IssuesBySeverity(SeverityCritical)
	if len(critical) != 1 {
		t.Errorf("expected 1 critical, got %d", len(critical))
	}

	errors := result.IssuesBySeverity(SeverityError)
	if len(errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(errors))
	}

	warnings := result.IssuesBySeverity(SeverityWarning)
	if len(warnings) != 2 {
		t.Errorf("expected 2 warnings, got %d", len(warnings))
	}

	infos := result.IssuesBySeverity(SeverityInfo)
	if len(infos) != 1 {
		t.Errorf("expected 1 info, got %d", len(infos))
	}
}

func TestRunbookResult_ConvenienceMethods(t *testing.T) {
	runbook := &Runbook{ID: "test-rb"}
	result := NewRunbookResult(runbook)

	result.AddStepResult(StepResult{
		Step: Step{ID: "step1"},
		Matches: []MatchResult{
			{Pattern: Pattern{Severity: SeverityCritical}},
			{Pattern: Pattern{Severity: SeverityError}},
			{Pattern: Pattern{Severity: SeverityError}},
			{Pattern: Pattern{Severity: SeverityWarning}},
			{Pattern: Pattern{Severity: SeverityInfo}},
		},
	})

	if len(result.CriticalIssues()) != 1 {
		t.Errorf("expected 1 critical, got %d", len(result.CriticalIssues()))
	}
	if len(result.ErrorIssues()) != 2 {
		t.Errorf("expected 2 errors, got %d", len(result.ErrorIssues()))
	}
	if len(result.WarningIssues()) != 1 {
		t.Errorf("expected 1 warning, got %d", len(result.WarningIssues()))
	}
	if len(result.InfoIssues()) != 1 {
		t.Errorf("expected 1 info, got %d", len(result.InfoIssues()))
	}
}

func TestRunbookResult_Summary(t *testing.T) {
	runbook := &Runbook{ID: "test-rb"}

	// Passed summary
	result := NewRunbookResult(runbook)
	result.AddStepResult(StepResult{Step: Step{ID: "step1"}, Status: StepStatusPassed})
	result.Finalize()

	summary := result.Summary()
	if summary == "" {
		t.Error("expected non-empty summary")
	}
	if len(summary) < 10 {
		t.Error("expected meaningful summary")
	}

	// Failed summary with issues
	result2 := NewRunbookResult(runbook)
	result2.AddStepResult(StepResult{
		Step:   Step{ID: "step1"},
		Status: StepStatusFailed,
		Matches: []MatchResult{
			{Pattern: Pattern{Severity: SeverityCritical}},
			{Pattern: Pattern{Severity: SeverityError}},
			{Pattern: Pattern{Severity: SeverityWarning}},
		},
	})
	result2.Finalize()

	summary2 := result2.Summary()
	if summary2 == "" {
		t.Error("expected non-empty summary")
	}
}

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		status   StepStatus
		expected string
	}{
		{StepStatusPassed, "OK"},
		{StepStatusFailed, "FAIL"},
		{StepStatusError, "ERR"},
		{StepStatusSkipped, "SKIP"},
		{StepStatusRunning, "..."},
		{StepStatusPending, " "},
		{StepStatus("unknown"), " "},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := StatusIcon(tt.status)
			if got != tt.expected {
				t.Errorf("StatusIcon(%s) = %q, want %q", tt.status, got, tt.expected)
			}
		})
	}
}

func TestSeverityIcon(t *testing.T) {
	tests := []struct {
		severity Severity
		expected string
	}{
		{SeverityCritical, "[!!!]"},
		{SeverityError, "[!!]"},
		{SeverityWarning, "[!]"},
		{SeverityInfo, "[i]"},
		{Severity("unknown"), "[ ]"},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			got := SeverityIcon(tt.severity)
			if got != tt.expected {
				t.Errorf("SeverityIcon(%s) = %q, want %q", tt.severity, got, tt.expected)
			}
		})
	}
}

func TestStepStatusConstants(t *testing.T) {
	tests := []struct {
		status   StepStatus
		expected string
	}{
		{StepStatusPending, "pending"},
		{StepStatusRunning, "running"},
		{StepStatusPassed, "passed"},
		{StepStatusFailed, "failed"},
		{StepStatusSkipped, "skipped"},
		{StepStatusError, "error"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, string(tt.status))
		}
	}
}

func TestStepResult_Fields(t *testing.T) {
	step := Step{ID: "step1", Name: "Test Step"}
	stepResult := StepResult{
		Step:     step,
		Status:   StepStatusPassed,
		Output:   "test output",
		Error:    nil,
		Duration: 100 * time.Millisecond,
		Matches: []MatchResult{
			{Pattern: Pattern{ID: "p1"}, MatchedText: "match1"},
		},
	}

	if stepResult.Step.ID != "step1" {
		t.Errorf("expected step ID 'step1', got %q", stepResult.Step.ID)
	}
	if stepResult.Status != StepStatusPassed {
		t.Errorf("expected status Passed, got %s", stepResult.Status)
	}
	if stepResult.Output != "test output" {
		t.Errorf("expected output 'test output', got %q", stepResult.Output)
	}
	if stepResult.Duration != 100*time.Millisecond {
		t.Errorf("expected duration 100ms, got %v", stepResult.Duration)
	}
	if len(stepResult.Matches) != 1 {
		t.Errorf("expected 1 match, got %d", len(stepResult.Matches))
	}
}

func TestIssue_Fields(t *testing.T) {
	issue := Issue{
		StepID:      "step1",
		StepName:    "Step Name",
		PatternID:   "pattern1",
		PatternName: "Pattern Name",
		Severity:    SeverityError,
		Message:     "Error message",
		MatchedText: "matched text",
		KBArticles:  []string{"KB001", "KB002"},
		Remediation: "Fix it like this",
	}

	if issue.StepID != "step1" {
		t.Errorf("expected StepID 'step1', got %q", issue.StepID)
	}
	if issue.Severity != SeverityError {
		t.Errorf("expected severity Error, got %s", issue.Severity)
	}
	if len(issue.KBArticles) != 2 {
		t.Errorf("expected 2 KB articles, got %d", len(issue.KBArticles))
	}
}
