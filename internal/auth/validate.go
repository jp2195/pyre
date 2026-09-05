package auth

import (
	"net"
	"regexp"
	"strings"
)

// hostnameRegex matches valid hostnames per RFC 952/1123.
var hostnameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?)*$`)

// ValidateHost checks whether host is a plausible IP address or hostname
// (optionally with :port). It returns a human-readable error message, or ""
// when valid. The empty host is considered valid so callers can apply their
// own required/optional policy.
//
// This is the single host validator for the app: the TUI forms and the
// CLI-flag/env credential path (ResolveCredentials) both use it, so a host
// that would change the request URL shape (slashes, @, ?, whitespace) is
// rejected before any URL is built from it.
func ValidateHost(host string) string {
	if host == "" {
		return ""
	}
	if strings.ContainsAny(host, " \t\n\r") {
		return "host must not contain whitespace"
	}
	// A pasted browser address is the most likely wrong answer here, and it
	// used to pass: "https://10.0.104.50" holds a single colon, so
	// net.SplitHostPort below reads the host as "https" and the rest as a
	// port, and "https" is a valid hostname. The client then prefixes
	// "https://" and appends "/api/", so the request went somewhere
	// meaningless and failed with a name-resolution error.
	if strings.Contains(host, "://") {
		return "enter the host only, without http:// or https://"
	}
	if strings.Contains(host, "/") {
		return "enter the host only, without a path"
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
