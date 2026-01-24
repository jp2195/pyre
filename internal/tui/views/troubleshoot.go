package views

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jp2195/pyre/internal/troubleshoot"
)

// TroubleshootMode represents the current mode of the troubleshoot view.
type TroubleshootMode int

const (
	TroubleshootModeList TroubleshootMode = iota
	TroubleshootModeRunning
	TroubleshootModeResult
)

// TroubleshootModel represents the troubleshooting view.
type TroubleshootModel struct {
	runbooks []*troubleshoot.Runbook
	selected int
	mode     TroubleshootMode
	result   *troubleshoot.RunbookResult
	hasSSH   bool
	width    int
	height   int

	// Running state
	currentStep  int
	stepStatuses []troubleshoot.StepStatus
	stepOutputs  []string

	// SSH connection state
	sshConfigured bool // SSH settings exist in config
	sshConnecting bool
	sshError      error

	// Error
	err error
}

// NewTroubleshootModel creates a new troubleshoot model.
func NewTroubleshootModel() TroubleshootModel {
	return TroubleshootModel{
		runbooks: make([]*troubleshoot.Runbook, 0),
		mode:     TroubleshootModeList,
	}
}

// SetSize sets the dimensions of the view.
func (m TroubleshootModel) SetSize(width, height int) TroubleshootModel {
	m.width = width
	m.height = height
	return m
}

// SetRunbooks sets the available runbooks.
func (m TroubleshootModel) SetRunbooks(runbooks []*troubleshoot.Runbook) TroubleshootModel {
	// Sort by category and name
	sort.Slice(runbooks, func(i, j int) bool {
		if runbooks[i].Category == runbooks[j].Category {
			return runbooks[i].Name < runbooks[j].Name
		}
		return runbooks[i].Category < runbooks[j].Category
	})
	m.runbooks = runbooks
	return m
}

// SetSSHAvailable indicates whether SSH is available.
func (m TroubleshootModel) SetSSHAvailable(available bool) TroubleshootModel {
	m.hasSSH = available
	return m
}

// SetSSHConfigured indicates whether SSH settings exist in config.
func (m TroubleshootModel) SetSSHConfigured(configured bool) TroubleshootModel {
	m.sshConfigured = configured
	return m
}

// SetSSHConnecting sets the SSH connecting state.
func (m TroubleshootModel) SetSSHConnecting(connecting bool) TroubleshootModel {
	m.sshConnecting = connecting
	return m
}

// SetSSHError sets the SSH error.
func (m TroubleshootModel) SetSSHError(err error) TroubleshootModel {
	m.sshError = err
	return m
}

// SetRunning transitions to running mode.
func (m TroubleshootModel) SetRunning(runbook *troubleshoot.Runbook) TroubleshootModel {
	m.mode = TroubleshootModeRunning
	m.currentStep = 0
	m.stepStatuses = make([]troubleshoot.StepStatus, len(runbook.Steps))
	m.stepOutputs = make([]string, len(runbook.Steps))
	for i := range m.stepStatuses {
		m.stepStatuses[i] = troubleshoot.StepStatusPending
	}
	return m
}

// UpdateStepProgress updates the progress of a step.
func (m TroubleshootModel) UpdateStepProgress(stepIndex int, status troubleshoot.StepStatus, output string) TroubleshootModel {
	if stepIndex >= 0 && stepIndex < len(m.stepStatuses) {
		m.stepStatuses[stepIndex] = status
		m.stepOutputs[stepIndex] = output
		m.currentStep = stepIndex
	}
	return m
}

// SetResult sets the runbook execution result.
func (m TroubleshootModel) SetResult(result *troubleshoot.RunbookResult, err error) TroubleshootModel {
	m.result = result
	m.err = err
	m.mode = TroubleshootModeResult
	return m
}

// SetError sets an error message.
func (m TroubleshootModel) SetError(err error) TroubleshootModel {
	m.err = err
	return m
}

// ClearResult clears the result and returns to list mode.
func (m TroubleshootModel) ClearResult() TroubleshootModel {
	m.result = nil
	m.err = nil
	m.mode = TroubleshootModeList
	return m
}

// Selected returns the selected runbook.
func (m TroubleshootModel) Selected() *troubleshoot.Runbook {
	if m.selected >= 0 && m.selected < len(m.runbooks) {
		return m.runbooks[m.selected]
	}
	return nil
}

// Mode returns the current mode.
func (m TroubleshootModel) Mode() TroubleshootMode {
	return m.mode
}

// Update handles input events.
func (m TroubleshootModel) Update(msg tea.Msg) (TroubleshootModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.mode {
		case TroubleshootModeList:
			switch msg.String() {
			case "up", "k":
				if m.selected > 0 {
					m.selected--
				}
			case "down", "j":
				if m.selected < len(m.runbooks)-1 {
					m.selected++
				}
			case "home", "g":
				m.selected = 0
			case "end", "G":
				if len(m.runbooks) > 0 {
					m.selected = len(m.runbooks) - 1
				}
			}

		case TroubleshootModeResult:
			switch msg.String() {
			case "esc", "q":
				m = m.ClearResult()
			}
		}
	}
	return m, nil
}

// View renders the troubleshoot view.
func (m TroubleshootModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	switch m.mode {
	case TroubleshootModeList:
		return m.renderList()
	case TroubleshootModeRunning:
		return m.renderRunning()
	case TroubleshootModeResult:
		return m.renderResult()
	}

	return ""
}

func (m TroubleshootModel) renderList() string {
	titleStyle := ViewTitleStyle.MarginBottom(1)
	panelStyle := ViewPanelStyle.Width(m.width - 4)
	selectedStyle := TableRowSelectedStyle.Bold(true)
	normalStyle := DetailValueStyle
	sshRequiredStyle := StatusWarningStyle
	categoryStyle := DetailDimStyle.Italic(true)
	descStyle := DetailDimStyle

	var b strings.Builder
	b.WriteString(titleStyle.Render("Troubleshooting Runbooks"))
	b.WriteString("\n\n")

	if len(m.runbooks) == 0 {
		b.WriteString(descStyle.Render("No runbooks available"))
	} else {
		currentCategory := ""

		for i, rb := range m.runbooks {
			// Category header
			if rb.Category != currentCategory {
				if currentCategory != "" {
					b.WriteString("\n")
				}
				currentCategory = rb.Category
				b.WriteString(categoryStyle.Render(strings.ToUpper(rb.Category)))
				b.WriteString("\n")
			}

			// Runbook entry
			style := normalStyle
			prefix := "  "
			if i == m.selected {
				style = selectedStyle
				prefix = "> "
			}

			// SSH indicator
			sshIndicator := ""
			if rb.RequiresSSH {
				if m.hasSSH {
					sshIndicator = " [SSH]"
				} else {
					sshIndicator = sshRequiredStyle.Render(" [SSH*]")
				}
			}

			name := style.Render(prefix + rb.Name + sshIndicator)
			b.WriteString(name)
			b.WriteString("\n")

			// Show description for selected item
			if i == m.selected {
				b.WriteString(descStyle.Render("    " + rb.Description))
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("\n")

	// SSH status messages
	if m.sshConnecting {
		b.WriteString(SeverityLowStyle.Render("Connecting to SSH..."))
		b.WriteString("\n")
	} else if m.sshError != nil {
		b.WriteString(ErrorMsgStyle.Render("SSH connection failed: " + m.sshError.Error()))
		b.WriteString("\n")
		b.WriteString(sshRequiredStyle.Render("Press R to retry SSH connection"))
		b.WriteString("\n")
	} else if !m.hasSSH {
		if m.sshConfigured {
			// SSH is configured but not connected yet
			b.WriteString(sshRequiredStyle.Render("* SSH not connected - some runbooks unavailable"))
			b.WriteString("\n")
			b.WriteString(sshRequiredStyle.Render("Press R to connect SSH"))
			b.WriteString("\n")
		} else {
			// SSH is not configured at all
			b.WriteString(descStyle.Render("* SSH not configured - some runbooks unavailable"))
			b.WriteString("\n")
			b.WriteString(descStyle.Render("  Add SSH settings to ~/.pyre.yaml or set PYRE_SSH_USERNAME"))
			b.WriteString("\n")
		}
	}

	b.WriteString(descStyle.Render("Press Enter to run selected runbook, j/k to navigate"))

	return panelStyle.Render(b.String())
}

func (m TroubleshootModel) renderRunning() string {
	titleStyle := ViewTitleStyle.MarginBottom(1)
	panelStyle := ViewPanelStyle.Width(m.width - 4)
	runningStyle := StatusWarningStyle
	passedStyle := StatusActiveStyle
	failedStyle := StatusInactiveStyle
	pendingStyle := StatusMutedStyle

	var b strings.Builder

	runbook := m.Selected()
	if runbook == nil {
		return panelStyle.Render("No runbook selected")
	}

	b.WriteString(titleStyle.Render("Running: " + runbook.Name))
	b.WriteString("\n\n")

	for i, step := range runbook.Steps {
		var status string
		var style lipgloss.Style

		if i < len(m.stepStatuses) {
			switch m.stepStatuses[i] {
			case troubleshoot.StepStatusRunning:
				status = "[...] "
				style = runningStyle
			case troubleshoot.StepStatusPassed:
				status = "[OK]  "
				style = passedStyle
			case troubleshoot.StepStatusFailed:
				status = "[FAIL]"
				style = failedStyle
			case troubleshoot.StepStatusError:
				status = "[ERR] "
				style = failedStyle
			case troubleshoot.StepStatusSkipped:
				status = "[SKIP]"
				style = pendingStyle
			default:
				status = "[ ]   "
				style = pendingStyle
			}
		} else {
			status = "[ ]   "
			style = pendingStyle
		}

		b.WriteString(style.Render(status + " " + step.Name))
		b.WriteString("\n")
	}

	return panelStyle.Render(b.String())
}

func (m TroubleshootModel) renderResult() string {
	titleStyle := ViewTitleStyle.MarginBottom(1)
	panelStyle := ViewPanelStyle.Width(m.width - 4)
	passedStyle := ActionAllowStyle
	failedStyle := ActionDenyStyle
	stepPassedStyle := StatusActiveStyle
	stepFailedStyle := StatusInactiveStyle
	labelStyle := DetailLabelStyle
	criticalStyle := SeverityCriticalStyle
	errorStyle := ErrorMsgStyle
	warningStyle := WarningMsgStyle
	infoStyle := SeverityLowStyle
	linkStyle := TagStyle.Underline(true)

	var b strings.Builder

	if m.err != nil {
		b.WriteString(failedStyle.Render("Error: " + m.err.Error()))
		b.WriteString("\n\n")
		b.WriteString(labelStyle.Render("Press Esc to go back"))
		return panelStyle.Render(b.String())
	}

	if m.result == nil {
		return panelStyle.Render("No result available")
	}

	// Show result-level error (e.g., SSH not available)
	if m.result.Error != nil {
		b.WriteString(titleStyle.Render(m.result.Runbook.Name))
		b.WriteString("  ")
		b.WriteString(failedStyle.Render("FAILED"))
		b.WriteString("\n\n")
		b.WriteString(failedStyle.Render("Error: " + m.result.Error.Error()))
		b.WriteString("\n\n")
		b.WriteString(labelStyle.Render("Press Esc to go back"))
		return panelStyle.Render(b.String())
	}

	// Title and status
	statusText := "PASSED"
	statusStyle := passedStyle
	if !m.result.Passed {
		statusText = "FAILED"
		statusStyle = failedStyle
	}

	b.WriteString(titleStyle.Render(m.result.Runbook.Name))
	b.WriteString("  ")
	b.WriteString(statusStyle.Render(statusText))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render(fmt.Sprintf("Duration: %s", m.result.Duration.Round(1))))
	b.WriteString("\n\n")

	// Steps summary
	b.WriteString(labelStyle.Render("Steps:"))
	b.WriteString("\n")

	for _, step := range m.result.Steps {
		var icon string
		var style lipgloss.Style

		switch step.Status {
		case troubleshoot.StepStatusPassed:
			icon = "[OK]  "
			style = stepPassedStyle
		case troubleshoot.StepStatusFailed:
			icon = "[FAIL]"
			style = stepFailedStyle
		case troubleshoot.StepStatusError:
			icon = "[ERR] "
			style = stepFailedStyle
		case troubleshoot.StepStatusSkipped:
			icon = "[SKIP]"
			style = labelStyle
		default:
			icon = "[ ]   "
			style = labelStyle
		}

		b.WriteString(style.Render("  " + icon + " " + step.Step.Name))
		b.WriteString("\n")
	}

	// Issues
	if len(m.result.Issues) > 0 {
		b.WriteString("\n")
		b.WriteString(labelStyle.Render(fmt.Sprintf("Issues Found: %d", len(m.result.Issues))))
		b.WriteString("\n\n")

		for _, issue := range m.result.Issues {
			var severityStyle lipgloss.Style
			var severityIcon string

			switch issue.Severity {
			case troubleshoot.SeverityCritical:
				severityStyle = criticalStyle
				severityIcon = "[!!!]"
			case troubleshoot.SeverityError:
				severityStyle = errorStyle
				severityIcon = "[!!]"
			case troubleshoot.SeverityWarning:
				severityStyle = warningStyle
				severityIcon = "[!]"
			default:
				severityStyle = infoStyle
				severityIcon = "[i]"
			}

			b.WriteString(severityStyle.Render(severityIcon + " " + issue.Message))
			b.WriteString("\n")

			if issue.Remediation != "" {
				b.WriteString(labelStyle.Render("    Fix: "))
				b.WriteString(issue.Remediation)
				b.WriteString("\n")
			}

			for _, kb := range issue.KBArticles {
				b.WriteString(labelStyle.Render("    KB: "))
				b.WriteString(linkStyle.Render(kb))
				b.WriteString("\n")
			}

			b.WriteString("\n")
		}
	} else {
		b.WriteString("\n")
		b.WriteString(passedStyle.Render("No issues detected"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Press Esc to go back"))

	return panelStyle.Render(b.String())
}
