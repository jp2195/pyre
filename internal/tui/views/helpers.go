package views

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jp2195/pyre/internal/auth"
)

// truncate truncates a string to maxLen, adding ... if needed
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// truncateEllipsis truncates with Unicode ellipsis
func truncateEllipsis(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return s[:maxLen]
	}
	return s[:maxLen-1] + "…"
}

// formatNumberWithCommas formats a number with thousand separators
func formatNumberWithCommas(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}

	var result strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}
	return result.String()
}

// formatBytes formats bytes into human readable format (KB, MB, GB, etc)
func formatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatPackets formats packet counts into human readable format (K, M, B)
func formatPackets(packets int64) string {
	if packets == 0 {
		return "0"
	}
	if packets < 1000 {
		return strconv.FormatInt(packets, 10)
	}
	if packets < 1000000 {
		return fmt.Sprintf("%.1fK", float64(packets)/1000)
	}
	if packets < 1000000000 {
		return fmt.Sprintf("%.1fM", float64(packets)/1000000)
	}
	return fmt.Sprintf("%.1fB", float64(packets)/1000000000)
}

// validateHost delegates to the app-wide host validator in internal/auth.
func validateHost(host string) string {
	return auth.ValidateHost(host)
}

// cleanValue normalizes ugly API values like "N/A", "ukn", "[n/a]" to empty
func cleanValue(s string) string {
	lower := strings.ToLower(strings.TrimSpace(s))
	if lower == "" || lower == "n/a" || lower == "ukn" || lower == "[n/a]" || lower == "unknown" {
		return ""
	}
	return s
}

// wrapText wraps text to the specified width.
func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}

	currentLine := words[0]
	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	lines = append(lines, currentLine)
	return lines
}
