package troubleshoot

import (
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("expected non-nil Registry")
	}
	if r.runbooks == nil {
		t.Error("expected runbooks map to be initialized")
	}
	if r.Count() != 0 {
		t.Errorf("expected empty registry, got %d runbooks", r.Count())
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()

	runbook := &Runbook{
		ID:          "test-runbook",
		Name:        "Test Runbook",
		Description: "A test runbook",
		Category:    "testing",
	}

	r.Register(runbook)

	if r.Count() != 1 {
		t.Errorf("expected 1 runbook, got %d", r.Count())
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()

	runbook := &Runbook{
		ID:          "test-runbook",
		Name:        "Test Runbook",
		Description: "A test runbook",
		Category:    "testing",
	}

	r.Register(runbook)

	// Test getting existing runbook
	got, ok := r.Get("test-runbook")
	if !ok {
		t.Error("expected to find test-runbook")
	}
	if got.Name != "Test Runbook" {
		t.Errorf("expected name 'Test Runbook', got %q", got.Name)
	}

	// Test getting non-existent runbook
	_, ok = r.Get("nonexistent")
	if ok {
		t.Error("expected not to find nonexistent runbook")
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()

	// Empty registry
	list := r.List()
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d", len(list))
	}

	// Add runbooks
	r.Register(&Runbook{ID: "rb1", Name: "Runbook 1", Category: "cat1"})
	r.Register(&Runbook{ID: "rb2", Name: "Runbook 2", Category: "cat2"})
	r.Register(&Runbook{ID: "rb3", Name: "Runbook 3", Category: "cat1"})

	list = r.List()
	if len(list) != 3 {
		t.Errorf("expected 3 runbooks, got %d", len(list))
	}
}

func TestRegistry_ListByCategory(t *testing.T) {
	r := NewRegistry()

	r.Register(&Runbook{ID: "rb1", Name: "Runbook 1", Category: "network"})
	r.Register(&Runbook{ID: "rb2", Name: "Runbook 2", Category: "security"})
	r.Register(&Runbook{ID: "rb3", Name: "Runbook 3", Category: "network"})
	r.Register(&Runbook{ID: "rb4", Name: "Runbook 4", Category: "ha"})

	// Test network category
	networkRunbooks := r.ListByCategory("network")
	if len(networkRunbooks) != 2 {
		t.Errorf("expected 2 network runbooks, got %d", len(networkRunbooks))
	}

	// Test security category
	securityRunbooks := r.ListByCategory("security")
	if len(securityRunbooks) != 1 {
		t.Errorf("expected 1 security runbook, got %d", len(securityRunbooks))
	}

	// Test non-existent category
	emptyRunbooks := r.ListByCategory("nonexistent")
	if len(emptyRunbooks) != 0 {
		t.Errorf("expected 0 runbooks for nonexistent category, got %d", len(emptyRunbooks))
	}
}

func TestRegistry_Categories(t *testing.T) {
	r := NewRegistry()

	// Empty registry
	cats := r.Categories()
	if len(cats) != 0 {
		t.Errorf("expected no categories, got %d", len(cats))
	}

	// Add runbooks with categories
	r.Register(&Runbook{ID: "rb1", Category: "network"})
	r.Register(&Runbook{ID: "rb2", Category: "security"})
	r.Register(&Runbook{ID: "rb3", Category: "network"}) // Duplicate
	r.Register(&Runbook{ID: "rb4", Category: ""})        // Empty category

	cats = r.Categories()
	if len(cats) != 2 {
		t.Errorf("expected 2 unique categories, got %d", len(cats))
	}

	// Verify categories
	categorySet := make(map[string]bool)
	for _, c := range cats {
		categorySet[c] = true
	}
	if !categorySet["network"] {
		t.Error("expected 'network' category")
	}
	if !categorySet["security"] {
		t.Error("expected 'security' category")
	}
}

func TestRegistry_Count(t *testing.T) {
	r := NewRegistry()

	if r.Count() != 0 {
		t.Errorf("expected 0, got %d", r.Count())
	}

	r.Register(&Runbook{ID: "rb1"})
	if r.Count() != 1 {
		t.Errorf("expected 1, got %d", r.Count())
	}

	r.Register(&Runbook{ID: "rb2"})
	if r.Count() != 2 {
		t.Errorf("expected 2, got %d", r.Count())
	}

	// Overwriting same ID shouldn't increase count
	r.Register(&Runbook{ID: "rb1", Name: "Updated"})
	if r.Count() != 2 {
		t.Errorf("expected 2 after overwrite, got %d", r.Count())
	}
}

func TestRegistry_LoadEmbedded(t *testing.T) {
	r := NewRegistry()

	err := r.LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded failed: %v", err)
	}

	// Should have loaded at least some runbooks
	if r.Count() == 0 {
		t.Log("No embedded runbooks found (this may be expected if no runbook files exist)")
	}
}

func TestRunbook_Fields(t *testing.T) {
	runbook := Runbook{
		ID:          "test-id",
		Name:        "Test Name",
		Description: "Test Description",
		Category:    "test-category",
		Tags:        []string{"tag1", "tag2"},
		RequiresSSH: true,
		Steps: []Step{
			{
				ID:       "step1",
				Name:     "Step 1",
				Type:     StepTypeSSH,
				Command:  "show system info",
				Required: true,
			},
			{
				ID:      "step2",
				Name:    "Step 2",
				Type:    StepTypeAPI,
				APICall: "system_info",
			},
		},
	}

	if runbook.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got %q", runbook.ID)
	}
	if runbook.Name != "Test Name" {
		t.Errorf("expected Name 'Test Name', got %q", runbook.Name)
	}
	if !runbook.RequiresSSH {
		t.Error("expected RequiresSSH to be true")
	}
	if len(runbook.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(runbook.Steps))
	}
	if runbook.Steps[0].Type != StepTypeSSH {
		t.Errorf("expected first step type SSH, got %s", runbook.Steps[0].Type)
	}
	if runbook.Steps[1].Type != StepTypeAPI {
		t.Errorf("expected second step type API, got %s", runbook.Steps[1].Type)
	}
}

func TestStep_Fields(t *testing.T) {
	step := Step{
		ID:          "step-id",
		Name:        "Step Name",
		Description: "Step description",
		Type:        StepTypeSSH,
		Command:     "show clock",
		Required:    true,
		Patterns: []Pattern{
			{
				ID:          "pattern1",
				Name:        "Error Pattern",
				Regex:       "error",
				Severity:    SeverityError,
				Message:     "Error detected",
				KBArticles:  []string{"KB001"},
				Remediation: "Fix the error",
			},
		},
	}

	if step.ID != "step-id" {
		t.Errorf("expected ID 'step-id', got %q", step.ID)
	}
	if step.Type != StepTypeSSH {
		t.Errorf("expected type SSH, got %s", step.Type)
	}
	if !step.Required {
		t.Error("expected Required to be true")
	}
	if len(step.Patterns) != 1 {
		t.Errorf("expected 1 pattern, got %d", len(step.Patterns))
	}
}

func TestPattern_Fields(t *testing.T) {
	pattern := Pattern{
		ID:          "pattern-id",
		Name:        "Pattern Name",
		Regex:       `error\s+\d+`,
		Severity:    SeverityCritical,
		Message:     "Critical error found",
		KBArticles:  []string{"KB001", "KB002"},
		Remediation: "Follow KB001 to resolve",
	}

	if pattern.ID != "pattern-id" {
		t.Errorf("expected ID 'pattern-id', got %q", pattern.ID)
	}
	if pattern.Severity != SeverityCritical {
		t.Errorf("expected severity Critical, got %s", pattern.Severity)
	}
	if len(pattern.KBArticles) != 2 {
		t.Errorf("expected 2 KB articles, got %d", len(pattern.KBArticles))
	}
}

func TestSeverityConstants(t *testing.T) {
	tests := []struct {
		severity Severity
		expected string
	}{
		{SeverityInfo, "info"},
		{SeverityWarning, "warning"},
		{SeverityError, "error"},
		{SeverityCritical, "critical"},
	}

	for _, tt := range tests {
		if string(tt.severity) != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, string(tt.severity))
		}
	}
}

func TestStepTypeConstants(t *testing.T) {
	tests := []struct {
		stepType StepType
		expected string
	}{
		{StepTypeAPI, "api"},
		{StepTypeSSH, "ssh"},
	}

	for _, tt := range tests {
		if string(tt.stepType) != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, string(tt.stepType))
		}
	}
}
