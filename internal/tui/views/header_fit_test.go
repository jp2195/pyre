package views

import (
	"testing"

	"charm.land/lipgloss/v2"
)

// TestRuleListHeaders_FitTheAvailableWidth checks the contract every
// FormatHeaderRow implementation is written against: given the available
// width, return a header that fits in it.
//
// A header wider than the panel does not overflow visibly, because lipgloss
// wraps it. It splits the column labels across two lines instead, which
// leaves the second line's labels sitting above unrelated data. The session
// table did exactly that on any terminal narrower than about 106 columns.
func TestRuleListHeaders_FitTheAvailableWidth(t *testing.T) {
	headers := map[string]func(int) string{
		"security":   formatSecurityHeader,
		"nat":        formatNATHeader,
		"interfaces": formatInterfaceHeader,
		"ipsec":      formatIPSecHeader,
		"gpusers":    formatGPUserHeader,
		"sessions":   formatSessionHeader,
	}

	// renderTable derives the available width by subtracting a fixed chrome
	// allowance from the terminal width, so mirror that here.
	const chrome = 12

	for name, format := range headers {
		for term := 80; term <= 200; term += 5 {
			available := term - chrome
			got := lipgloss.Width(format(available))
			if got > available {
				t.Errorf("%s header at terminal %d: %d cells for %d available, %d over",
					name, term, got, available, got-available)
				break
			}
		}
	}
}
