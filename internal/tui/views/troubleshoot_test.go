package views

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jp2195/pyre/internal/troubleshoot"
)

func TestNewTroubleshootModel(t *testing.T) {
	m := NewTroubleshootModel()

	if m.selected != 0 {
		t.Errorf("expected selected=0, got %d", m.selected)
	}
	if m.mode != TroubleshootModeList {
		t.Errorf("expected mode=TroubleshootModeList, got %d", m.mode)
	}
	if len(m.runbooks) != 0 {
		t.Errorf("expected 0 runbooks, got %d", len(m.runbooks))
	}
}

func TestTroubleshootModel_SetSize(t *testing.T) {
	m := NewTroubleshootModel()
	m = m.SetSize(100, 50)

	if m.width != 100 {
		t.Errorf("expected width=100, got %d", m.width)
	}
	if m.height != 50 {
		t.Errorf("expected height=50, got %d", m.height)
	}
}

func TestTroubleshootModel_SetRunbooks(t *testing.T) {
	m := NewTroubleshootModel()

	runbooks := []*troubleshoot.Runbook{
		{Name: "Beta", Category: "Network"},
		{Name: "Alpha", Category: "Network"},
		{Name: "Gamma", Category: "System"},
	}

	m = m.SetRunbooks(runbooks)

	if len(m.runbooks) != 3 {
		t.Errorf("expected 3 runbooks, got %d", len(m.runbooks))
	}

	// Should be sorted by category then name
	if m.runbooks[0].Name != "Alpha" {
		t.Errorf("expected first runbook to be 'Alpha', got %q", m.runbooks[0].Name)
	}
	if m.runbooks[1].Name != "Beta" {
		t.Errorf("expected second runbook to be 'Beta', got %q", m.runbooks[1].Name)
	}
	if m.runbooks[2].Name != "Gamma" {
		t.Errorf("expected third runbook to be 'Gamma', got %q", m.runbooks[2].Name)
	}
}

func TestTroubleshootModel_SetSSHAvailable(t *testing.T) {
	m := NewTroubleshootModel()

	if m.hasSSH {
		t.Error("expected hasSSH=false initially")
	}

	m = m.SetSSHAvailable(true)
	if !m.hasSSH {
		t.Error("expected hasSSH=true after SetSSHAvailable(true)")
	}

	m = m.SetSSHAvailable(false)
	if m.hasSSH {
		t.Error("expected hasSSH=false after SetSSHAvailable(false)")
	}
}

func TestTroubleshootModel_SetSSHConfigured(t *testing.T) {
	m := NewTroubleshootModel()

	if m.sshConfigured {
		t.Error("expected sshConfigured=false initially")
	}

	m = m.SetSSHConfigured(true)
	if !m.sshConfigured {
		t.Error("expected sshConfigured=true")
	}
}

func TestTroubleshootModel_SetSSHConnecting(t *testing.T) {
	m := NewTroubleshootModel()

	if m.sshConnecting {
		t.Error("expected sshConnecting=false initially")
	}

	m = m.SetSSHConnecting(true)
	if !m.sshConnecting {
		t.Error("expected sshConnecting=true")
	}
}

func TestTroubleshootModel_SetSSHError(t *testing.T) {
	m := NewTroubleshootModel()

	if m.sshError != nil {
		t.Error("expected sshError=nil initially")
	}

	err := errors.New("connection refused")
	m = m.SetSSHError(err)
	if m.sshError != err {
		t.Error("expected sshError to be set")
	}
}

func TestTroubleshootModel_SetRunning(t *testing.T) {
	m := NewTroubleshootModel()

	runbook := &troubleshoot.Runbook{
		Name: "Test",
		Steps: []troubleshoot.Step{
			{Name: "Step 1"},
			{Name: "Step 2"},
			{Name: "Step 3"},
		},
	}

	m = m.SetRunning(runbook)

	if m.mode != TroubleshootModeRunning {
		t.Errorf("expected mode=TroubleshootModeRunning, got %d", m.mode)
	}
	if m.currentStep != 0 {
		t.Errorf("expected currentStep=0, got %d", m.currentStep)
	}
	if len(m.stepStatuses) != 3 {
		t.Errorf("expected 3 step statuses, got %d", len(m.stepStatuses))
	}
	if len(m.stepOutputs) != 3 {
		t.Errorf("expected 3 step outputs, got %d", len(m.stepOutputs))
	}

	// All statuses should be pending
	for i, status := range m.stepStatuses {
		if status != troubleshoot.StepStatusPending {
			t.Errorf("expected step %d status=Pending, got %v", i, status)
		}
	}
}

func TestTroubleshootModel_UpdateStepProgress(t *testing.T) {
	m := NewTroubleshootModel()

	runbook := &troubleshoot.Runbook{
		Steps: []troubleshoot.Step{
			{Name: "Step 1"},
			{Name: "Step 2"},
		},
	}
	m = m.SetRunning(runbook)

	// Update first step
	m = m.UpdateStepProgress(0, troubleshoot.StepStatusRunning, "Running...")
	if m.stepStatuses[0] != troubleshoot.StepStatusRunning {
		t.Error("expected step 0 to be running")
	}
	if m.stepOutputs[0] != "Running..." {
		t.Errorf("expected output 'Running...', got %q", m.stepOutputs[0])
	}
	if m.currentStep != 0 {
		t.Errorf("expected currentStep=0, got %d", m.currentStep)
	}

	// Update second step
	m = m.UpdateStepProgress(1, troubleshoot.StepStatusPassed, "Done!")
	if m.stepStatuses[1] != troubleshoot.StepStatusPassed {
		t.Error("expected step 1 to be passed")
	}
	if m.currentStep != 1 {
		t.Errorf("expected currentStep=1, got %d", m.currentStep)
	}

	// Invalid step index should be ignored
	m = m.UpdateStepProgress(-1, troubleshoot.StepStatusPassed, "bad")
	if m.currentStep != 1 {
		t.Error("expected currentStep to remain 1")
	}

	m = m.UpdateStepProgress(100, troubleshoot.StepStatusPassed, "bad")
	if m.currentStep != 1 {
		t.Error("expected currentStep to remain 1")
	}
}

func TestTroubleshootModel_SetResult(t *testing.T) {
	m := NewTroubleshootModel()

	result := &troubleshoot.RunbookResult{
		Passed: true,
	}
	err := errors.New("test error")

	m = m.SetResult(result, err)

	if m.result != result {
		t.Error("expected result to be set")
	}
	if m.err != err {
		t.Error("expected err to be set")
	}
	if m.mode != TroubleshootModeResult {
		t.Errorf("expected mode=TroubleshootModeResult, got %d", m.mode)
	}
}

func TestTroubleshootModel_SetError(t *testing.T) {
	m := NewTroubleshootModel()

	err := errors.New("test error")
	m = m.SetError(err)

	if m.err != err {
		t.Error("expected err to be set")
	}
}

func TestTroubleshootModel_ClearResult(t *testing.T) {
	m := NewTroubleshootModel()

	// Set up result state
	m.result = &troubleshoot.RunbookResult{}
	m.err = errors.New("test")
	m.mode = TroubleshootModeResult

	m = m.ClearResult()

	if m.result != nil {
		t.Error("expected result to be nil")
	}
	if m.err != nil {
		t.Error("expected err to be nil")
	}
	if m.mode != TroubleshootModeList {
		t.Errorf("expected mode=TroubleshootModeList, got %d", m.mode)
	}
}

func TestTroubleshootModel_Selected(t *testing.T) {
	m := NewTroubleshootModel()

	// No runbooks
	if m.Selected() != nil {
		t.Error("expected nil when no runbooks")
	}

	// With runbooks
	runbooks := []*troubleshoot.Runbook{
		{Name: "First"},
		{Name: "Second"},
	}
	m = m.SetRunbooks(runbooks)

	selected := m.Selected()
	if selected == nil {
		t.Fatal("expected non-nil selected")
	}
	if selected.Name != "First" {
		t.Errorf("expected selected name 'First', got %q", selected.Name)
	}

	// Move selection
	m.selected = 1
	selected = m.Selected()
	if selected.Name != "Second" {
		t.Errorf("expected selected name 'Second', got %q", selected.Name)
	}

	// Out of bounds
	m.selected = 100
	if m.Selected() != nil {
		t.Error("expected nil for out of bounds selection")
	}
}

func TestTroubleshootModel_Mode(t *testing.T) {
	m := NewTroubleshootModel()

	if m.Mode() != TroubleshootModeList {
		t.Errorf("expected initial mode=TroubleshootModeList, got %d", m.Mode())
	}

	m.mode = TroubleshootModeRunning
	if m.Mode() != TroubleshootModeRunning {
		t.Errorf("expected mode=TroubleshootModeRunning, got %d", m.Mode())
	}

	m.mode = TroubleshootModeResult
	if m.Mode() != TroubleshootModeResult {
		t.Errorf("expected mode=TroubleshootModeResult, got %d", m.Mode())
	}
}

func TestTroubleshootModel_Update_Navigation(t *testing.T) {
	m := NewTroubleshootModel()
	m = m.SetSize(100, 50)

	runbooks := []*troubleshoot.Runbook{
		{Name: "First"},
		{Name: "Second"},
		{Name: "Third"},
	}
	m = m.SetRunbooks(runbooks)

	// Move down with j
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.selected != 1 {
		t.Errorf("expected selected=1 after j, got %d", m.selected)
	}

	// Move down with down arrow
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.selected != 2 {
		t.Errorf("expected selected=2 after down, got %d", m.selected)
	}

	// At end, shouldn't go further
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.selected != 2 {
		t.Errorf("expected selected to stay at 2, got %d", m.selected)
	}

	// Move up with k
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.selected != 1 {
		t.Errorf("expected selected=1 after k, got %d", m.selected)
	}

	// Move up with up arrow
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.selected != 0 {
		t.Errorf("expected selected=0 after up, got %d", m.selected)
	}

	// At start, shouldn't go negative
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.selected != 0 {
		t.Errorf("expected selected to stay at 0, got %d", m.selected)
	}

	// Go to end with G
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	if m.selected != 2 {
		t.Errorf("expected selected=2 after G, got %d", m.selected)
	}

	// Go to start with g
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	if m.selected != 0 {
		t.Errorf("expected selected=0 after g, got %d", m.selected)
	}

	// End key
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	if m.selected != 2 {
		t.Errorf("expected selected=2 after End, got %d", m.selected)
	}

	// Home key
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyHome})
	if m.selected != 0 {
		t.Errorf("expected selected=0 after Home, got %d", m.selected)
	}
}

func TestTroubleshootModel_Update_ResultMode(t *testing.T) {
	m := NewTroubleshootModel()
	m = m.SetSize(100, 50)
	m.mode = TroubleshootModeResult
	m.result = &troubleshoot.RunbookResult{}

	// Esc should clear result
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != TroubleshootModeList {
		t.Errorf("expected mode=TroubleshootModeList after Esc, got %d", m.mode)
	}

	// Reset and try q
	m.mode = TroubleshootModeResult
	m.result = &troubleshoot.RunbookResult{}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if m.mode != TroubleshootModeList {
		t.Errorf("expected mode=TroubleshootModeList after q, got %d", m.mode)
	}
}

func TestTroubleshootModel_View_ZeroWidth(t *testing.T) {
	m := NewTroubleshootModel()
	// Don't set size

	view := m.View()
	if view != "Loading..." {
		t.Errorf("expected 'Loading...' with zero width, got %q", view)
	}
}

func TestTroubleshootModel_View_ListMode(t *testing.T) {
	m := NewTroubleshootModel()
	m = m.SetSize(100, 50)

	// Empty list
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}

	// With runbooks
	runbooks := []*troubleshoot.Runbook{
		{Name: "Test Runbook", Description: "A test", Category: "Testing"},
	}
	m = m.SetRunbooks(runbooks)

	view = m.View()
	if view == "" {
		t.Error("expected non-empty view with runbooks")
	}
}

func TestTroubleshootModel_View_RunningMode(t *testing.T) {
	m := NewTroubleshootModel()
	m = m.SetSize(100, 50)

	runbook := &troubleshoot.Runbook{
		Name: "Test",
		Steps: []troubleshoot.Step{
			{Name: "Step 1"},
		},
	}
	m = m.SetRunbooks([]*troubleshoot.Runbook{runbook})
	m = m.SetRunning(runbook)

	view := m.View()
	if view == "" {
		t.Error("expected non-empty view in running mode")
	}
}

func TestTroubleshootModel_View_ResultMode(t *testing.T) {
	m := NewTroubleshootModel()
	m = m.SetSize(100, 50)

	runbook := &troubleshoot.Runbook{
		Name: "Test",
		Steps: []troubleshoot.Step{
			{Name: "Step 1"},
		},
	}
	m = m.SetRunbooks([]*troubleshoot.Runbook{runbook})

	result := &troubleshoot.RunbookResult{
		Runbook: runbook,
		Passed:  true,
		Steps: []troubleshoot.StepResult{
			{Step: troubleshoot.Step{Name: "Step 1"}, Status: troubleshoot.StepStatusPassed, Output: "OK"},
		},
	}
	m = m.SetResult(result, nil)

	view := m.View()
	if view == "" {
		t.Error("expected non-empty view in result mode")
	}
}

func TestTroubleshootMode_Constants(t *testing.T) {
	if TroubleshootModeList != 0 {
		t.Errorf("expected TroubleshootModeList=0, got %d", TroubleshootModeList)
	}
	if TroubleshootModeRunning != 1 {
		t.Errorf("expected TroubleshootModeRunning=1, got %d", TroubleshootModeRunning)
	}
	if TroubleshootModeResult != 2 {
		t.Errorf("expected TroubleshootModeResult=2, got %d", TroubleshootModeResult)
	}
}
