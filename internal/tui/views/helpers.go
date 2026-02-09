package views

import (
	"fmt"
	"net"
	"regexp"
	"strings"
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

// minInt returns the minimum of two ints
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// maxInt returns the maximum of two ints
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// formatNumberWithCommas formats a number with thousand separators
func formatNumberWithCommas(n int64) string {
	s := fmt.Sprintf("%d", n)
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
		return fmt.Sprintf("%d", packets)
	}
	if packets < 1000000 {
		return fmt.Sprintf("%.1fK", float64(packets)/1000)
	}
	if packets < 1000000000 {
		return fmt.Sprintf("%.1fM", float64(packets)/1000000)
	}
	return fmt.Sprintf("%.1fB", float64(packets)/1000000000)
}

// hostnameRegex matches valid hostnames per RFC 952/1123
var hostnameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?)*$`)

// validateHost checks if a string is a valid IP address or hostname.
// Returns an error message string if invalid, or empty string if valid.
func validateHost(host string) string {
	if host == "" {
		return ""
	}
	if strings.ContainsAny(host, " \t\n\r") {
		return "host must not contain whitespace"
	}
	// Valid IP address
	if net.ParseIP(host) != nil {
		return ""
	}
	// host:port form - strip port and validate host part
	if h, _, err := net.SplitHostPort(host); err == nil {
		if net.ParseIP(h) != nil {
			return ""
		}
		if hostnameRegex.MatchString(h) {
			return ""
		}
		return "invalid hostname or IP address"
	}
	// Valid hostname
	if len(host) > 253 {
		return "hostname too long (max 253 characters)"
	}
	if hostnameRegex.MatchString(host) {
		return ""
	}
	return "invalid hostname or IP address"
}

// cleanValue normalizes ugly API values like "N/A", "ukn", "[n/a]" to empty
func cleanValue(s string) string {
	lower := strings.ToLower(strings.TrimSpace(s))
	if lower == "" || lower == "n/a" || lower == "ukn" || lower == "[n/a]" || lower == "unknown" {
		return ""
	}
	return s
}
