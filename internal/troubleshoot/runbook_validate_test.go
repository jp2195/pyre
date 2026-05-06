package troubleshoot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validRunbook() Runbook {
	return Runbook{
		ID:   "rb1",
		Name: "Test Runbook",
		Steps: []Step{
			{
				ID:      "step1",
				Name:    "Step 1",
				Type:    StepTypeAPI,
				APICall: "system_info",
				Patterns: []Pattern{
					{ID: "p1", Regex: `error\d+`, Severity: SeverityError},
				},
			},
		},
	}
}

func TestRunbook_Validate_HappyPath(t *testing.T) {
	rb := validRunbook()
	if err := rb.Validate(); err != nil {
		t.Fatalf("expected valid runbook, got error: %v", err)
	}
}

func TestRunbook_Validate_EmptyID(t *testing.T) {
	rb := validRunbook()
	rb.ID = ""
	err := rb.Validate()
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
	if !strings.Contains(err.Error(), "ID") {
		t.Errorf("expected error mentioning ID, got %v", err)
	}
}

func TestRunbook_Validate_EmptyName(t *testing.T) {
	rb := validRunbook()
	rb.Name = ""
	err := rb.Validate()
	if err == nil {
		t.Fatal("expected error for empty Name")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Errorf("expected error mentioning name, got %v", err)
	}
}

func TestRunbook_Validate_ZeroSteps(t *testing.T) {
	rb := validRunbook()
	rb.Steps = nil
	err := rb.Validate()
	if err == nil {
		t.Fatal("expected error for zero steps")
	}
	if !strings.Contains(err.Error(), "step") {
		t.Errorf("expected error mentioning step, got %v", err)
	}
}

func TestStep_Validate_EmptyID(t *testing.T) {
	s := Step{Type: StepTypeAPI}
	err := s.Validate()
	if err == nil {
		t.Fatal("expected error for empty step ID")
	}
}

func TestStep_Validate_UnknownType(t *testing.T) {
	s := Step{ID: "s1", Type: StepType("bogus")}
	err := s.Validate()
	if err == nil {
		t.Fatal("expected error for unknown step type")
	}
	if !strings.Contains(err.Error(), "unknown type") {
		t.Errorf("expected error mentioning unknown type, got %v", err)
	}
}

func TestStep_Validate_BadRegex(t *testing.T) {
	s := Step{
		ID:      "s1",
		Type:    StepTypeAPI,
		APICall: "system_info",
		Patterns: []Pattern{
			{ID: "p1", Regex: "[invalid"},
		},
	}
	err := s.Validate()
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
	if !strings.Contains(err.Error(), "regex") {
		t.Errorf("expected error mentioning regex, got %v", err)
	}
}

func TestStep_Validate_HappyPath(t *testing.T) {
	s := Step{
		ID:      "s1",
		Type:    StepTypeAPI,
		APICall: "system_info",
		Patterns: []Pattern{
			{ID: "p1", Regex: `ok`},
		},
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("expected valid step, got: %v", err)
	}
}

func TestStep_Validate_APIRequiresKnownCall(t *testing.T) {
	tests := []struct {
		name    string
		step    Step
		wantErr string
	}{
		{"empty api_call", Step{ID: "s1", Type: StepTypeAPI}, "non-empty api_call"},
		{"unknown api_call", Step{ID: "s2", Type: StepTypeAPI, APICall: "bogus"}, "unknown api_call"},
		{"valid system_info", Step{ID: "s3", Type: StepTypeAPI, APICall: "system_info"}, ""},
		{"valid ha_status", Step{ID: "s4", Type: StepTypeAPI, APICall: "ha_status"}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.step.Validate()
			switch {
			case tc.wantErr == "" && err != nil:
				t.Errorf("unexpected err: %v", err)
			case tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)):
				t.Errorf("err = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestRegistry_LoadFromFile_ReadsDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rb.yaml")
	content := `id: disk-rb
name: Disk Runbook
category: testing
steps:
  - id: s1
    name: Step 1
    type: api
    api_call: system_info
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp runbook: %v", err)
	}

	r := NewRegistry()
	if err := r.LoadFromFile(path); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}
	rb, ok := r.Get("disk-rb")
	if !ok {
		t.Fatal("expected disk-rb to be registered")
	}
	if rb.Name != "Disk Runbook" {
		t.Errorf("expected Name 'Disk Runbook', got %q", rb.Name)
	}
}

func TestRegistry_LoadFromFile_RejectsInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	// No ID, no Name, no steps
	if err := os.WriteFile(path, []byte("description: empty\n"), 0o600); err != nil {
		t.Fatalf("write temp runbook: %v", err)
	}

	r := NewRegistry()
	if err := r.LoadFromFile(path); err == nil {
		t.Fatal("expected LoadFromFile to reject invalid runbook")
	}
	if r.Count() != 0 {
		t.Errorf("expected 0 registered runbooks, got %d", r.Count())
	}
}

func TestRegistry_LoadFromFile_MissingFile(t *testing.T) {
	r := NewRegistry()
	err := r.LoadFromFile(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestEmbeddedRunbooks_AllValidate(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadEmbedded(); err != nil {
		t.Fatalf("LoadEmbedded failed: %v", err)
	}
	if r.Count() < 1 {
		t.Fatalf("expected at least 1 embedded runbook, got %d", r.Count())
	}
	// Already validated by LoadEmbedded; re-validate for belt-and-suspenders.
	for _, rb := range r.List() {
		if err := rb.Validate(); err != nil {
			t.Errorf("embedded runbook %q failed validation: %v", rb.ID, err)
		}
	}
}
