package troubleshoot

import (
	"sync"
	"testing"
)

func TestNewPatternMatcher(t *testing.T) {
	pm := NewPatternMatcher()
	if pm == nil {
		t.Fatal("expected non-nil PatternMatcher")
	}
	if pm.compiled == nil {
		t.Error("expected compiled map to be initialized")
	}
}

func TestPatternMatcher_Match(t *testing.T) {
	pm := NewPatternMatcher()

	tests := []struct {
		name        string
		pattern     Pattern
		output      string
		expectMatch bool
		expectText  string
	}{
		{
			name: "simple match",
			pattern: Pattern{
				Regex: `error`,
			},
			output:      "Connection error occurred",
			expectMatch: true,
			expectText:  "error",
		},
		{
			name: "no match",
			pattern: Pattern{
				Regex: `error`,
			},
			output:      "Connection successful",
			expectMatch: false,
		},
		{
			name: "case insensitive match",
			pattern: Pattern{
				Regex: `(?i)error`,
			},
			output:      "Connection ERROR occurred",
			expectMatch: true,
			expectText:  "ERROR",
		},
		{
			name: "capture groups",
			pattern: Pattern{
				Regex: `cpu.*(\d+)%`,
			},
			output:      "cpu usage 85%",
			expectMatch: true,
			expectText:  "cpu usage 85%",
		},
		{
			name: "ssl handshake pattern",
			pattern: Pattern{
				Regex: precompiledPatterns.sslHandshakeFailed,
			},
			output:      "SSL handshake failed with peer",
			expectMatch: true,
		},
		{
			name: "authentication failed pattern",
			pattern: Pattern{
				Regex: precompiledPatterns.authenticationFailed,
			},
			output:      "Authentication failed for user admin",
			expectMatch: true,
		},
		{
			name: "high cpu pattern",
			pattern: Pattern{
				Regex: precompiledPatterns.highCPU,
			},
			output:      "System cpu 95% utilized",
			expectMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pm.Match(tt.pattern, tt.output)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectMatch {
				if result == nil {
					t.Error("expected match but got nil")
				} else if tt.expectText != "" && result.MatchedText != tt.expectText {
					t.Errorf("expected matched text %q, got %q", tt.expectText, result.MatchedText)
				}
			} else {
				if result != nil {
					t.Errorf("expected no match but got: %v", result)
				}
			}
		})
	}
}

func TestPatternMatcher_Match_InvalidRegex(t *testing.T) {
	pm := NewPatternMatcher()

	pattern := Pattern{
		Regex: `[invalid`,
	}

	_, err := pm.Match(pattern, "test output")
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestPatternMatcher_MatchAll(t *testing.T) {
	pm := NewPatternMatcher()

	patterns := []Pattern{
		{Regex: `error`, Severity: SeverityError},
		{Regex: `warning`, Severity: SeverityWarning},
		{Regex: `info`, Severity: SeverityInfo},
	}

	tests := []struct {
		name        string
		output      string
		expectCount int
	}{
		{
			name:        "no matches",
			output:      "all good here",
			expectCount: 0,
		},
		{
			name:        "one match",
			output:      "found an error in the log",
			expectCount: 1,
		},
		{
			name:        "multiple matches",
			output:      "error and warning and info messages",
			expectCount: 3,
		},
		{
			name:        "two matches",
			output:      "this has error and warning",
			expectCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := pm.MatchAll(patterns, tt.output)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(results) != tt.expectCount {
				t.Errorf("expected %d matches, got %d", tt.expectCount, len(results))
			}
		})
	}
}

func TestPatternMatcher_MatchAll_InvalidRegex(t *testing.T) {
	pm := NewPatternMatcher()

	patterns := []Pattern{
		{Regex: `valid`},
		{Regex: `[invalid`},
	}

	_, err := pm.MatchAll(patterns, "test output")
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestPatternMatcher_Caching(t *testing.T) {
	pm := NewPatternMatcher()

	pattern := Pattern{Regex: `test`}

	// First match - should compile and cache
	_, err := pm.Match(pattern, "test output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pm.mu.RLock()
	_, cached := pm.compiled[pattern.Regex]
	pm.mu.RUnlock()

	if !cached {
		t.Error("expected pattern to be cached")
	}

	// Second match - should use cache
	_, err = pm.Match(pattern, "another test")
	if err != nil {
		t.Fatalf("unexpected error on cached pattern: %v", err)
	}
}

func TestPatternMatcher_Concurrent(t *testing.T) {
	pm := NewPatternMatcher()

	patterns := []Pattern{
		{Regex: `error\d+`},
		{Regex: `warning\d+`},
		{Regex: `info\d+`},
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for _, p := range patterns {
				_, err := pm.Match(p, "error1 warning2 info3")
				if err != nil {
					t.Errorf("concurrent match error: %v", err)
				}
			}
		}(i)
	}
	wg.Wait()
}

func TestPrecompiledPatterns(t *testing.T) {
	pm := NewPatternMatcher()

	// Test that all precompiled patterns are valid regexes
	patterns := []struct {
		name   string
		regex  string
		sample string
	}{
		{"sslHandshakeFailed", precompiledPatterns.sslHandshakeFailed, "SSL handshake failed"},
		{"connectionRefused", precompiledPatterns.connectionRefused, "connection refused"},
		{"authenticationFailed", precompiledPatterns.authenticationFailed, "authentication failed"},
		{"peerUnreachable", precompiledPatterns.peerUnreachable, "peer unreachable"},
		{"haStateNonFunctional", precompiledPatterns.haStateNonFunctional, "state: suspended"},
		{"haSyncFailed", precompiledPatterns.haSyncFailed, "sync failed"},
		{"haLinkDown", precompiledPatterns.haLinkDown, "ha1 down"},
		{"commitFailed", precompiledPatterns.commitFailed, "commit failed"},
		{"validationError", precompiledPatterns.validationError, "validation error"},
		{"objectReference", precompiledPatterns.objectReference, "object not found"},
		{"configLocked", precompiledPatterns.configLocked, "config locked"},
		{"licenseExpired", precompiledPatterns.licenseExpired, "license expired"},
		{"highCPU", precompiledPatterns.highCPU, "cpu 95%"},
		{"highMemory", precompiledPatterns.highMemory, "memory 90%"},
		{"oomKiller", precompiledPatterns.oomKiller, "oom killer invoked"},
	}

	for _, tt := range patterns {
		t.Run(tt.name, func(t *testing.T) {
			pattern := Pattern{Regex: tt.regex}
			result, err := pm.Match(pattern, tt.sample)
			if err != nil {
				t.Errorf("pattern %s failed to compile: %v", tt.name, err)
			}
			if result == nil {
				t.Errorf("pattern %s did not match sample %q", tt.name, tt.sample)
			}
		})
	}
}

func TestCommonKBArticles(t *testing.T) {
	// Test that KB article URLs are non-empty and properly formatted
	articles := []struct {
		name string
		url  string
	}{
		{"panoramaSSL", commonKBArticles.panoramaSSL},
		{"panoramaConnect", commonKBArticles.panoramaConnect},
		{"haConfig", commonKBArticles.haConfig},
		{"haSync", commonKBArticles.haSync},
		{"commitFail", commonKBArticles.commitFail},
		{"licensing", commonKBArticles.licensing},
		{"resourceUsage", commonKBArticles.resourceUsage},
	}

	for _, tt := range articles {
		t.Run(tt.name, func(t *testing.T) {
			if tt.url == "" {
				t.Errorf("KB article %s is empty", tt.name)
			}
			if len(tt.url) < 10 {
				t.Errorf("KB article %s URL seems too short: %s", tt.name, tt.url)
			}
		})
	}
}

func TestMatchResult_Fields(t *testing.T) {
	pattern := Pattern{
		ID:          "test-pattern",
		Name:        "Test Pattern",
		Regex:       "error",
		Severity:    SeverityError,
		Message:     "Error found",
		KBArticles:  []string{"KB001"},
		Remediation: "Fix the error",
	}

	result := MatchResult{
		Pattern:     pattern,
		MatchedText: "error occurred",
	}

	if result.Pattern.ID != "test-pattern" {
		t.Errorf("expected Pattern.ID 'test-pattern', got %q", result.Pattern.ID)
	}
	if result.MatchedText != "error occurred" {
		t.Errorf("expected MatchedText 'error occurred', got %q", result.MatchedText)
	}
}

func TestPatternMatcher_MatchWithSeverities(t *testing.T) {
	pm := NewPatternMatcher()

	tests := []struct {
		name     string
		pattern  Pattern
		output   string
		severity Severity
	}{
		{
			name: "critical severity match",
			pattern: Pattern{
				Regex:    "CRITICAL",
				Severity: SeverityCritical,
			},
			output:   "CRITICAL error in system",
			severity: SeverityCritical,
		},
		{
			name: "error severity match",
			pattern: Pattern{
				Regex:    "ERROR",
				Severity: SeverityError,
			},
			output:   "ERROR in application",
			severity: SeverityError,
		},
		{
			name: "warning severity match",
			pattern: Pattern{
				Regex:    "WARN",
				Severity: SeverityWarning,
			},
			output:   "WARN: disk space low",
			severity: SeverityWarning,
		},
		{
			name: "info severity match",
			pattern: Pattern{
				Regex:    "INFO",
				Severity: SeverityInfo,
			},
			output:   "INFO: system started",
			severity: SeverityInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pm.Match(tt.pattern, tt.output)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected match")
			}
			if result.Pattern.Severity != tt.severity {
				t.Errorf("expected severity %s, got %s", tt.severity, result.Pattern.Severity)
			}
		})
	}
}

func TestPatternMatcher_MatchAll_WithMetadata(t *testing.T) {
	pm := NewPatternMatcher()

	patterns := []Pattern{
		{
			ID:          "p1",
			Name:        "Error Pattern",
			Regex:       "error",
			Severity:    SeverityError,
			Message:     "Error detected",
			KBArticles:  []string{"KB001", "KB002"},
			Remediation: "Check logs",
		},
		{
			ID:          "p2",
			Name:        "Warning Pattern",
			Regex:       "warning",
			Severity:    SeverityWarning,
			Message:     "Warning detected",
			KBArticles:  []string{"KB003"},
			Remediation: "Review settings",
		},
	}

	results, err := pm.MatchAll(patterns, "found error and warning in output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(results))
	}

	// Verify metadata is preserved
	for _, r := range results {
		if r.Pattern.ID == "" {
			t.Error("expected Pattern.ID to be set")
		}
		if r.Pattern.Name == "" {
			t.Error("expected Pattern.Name to be set")
		}
		if r.Pattern.Message == "" {
			t.Error("expected Pattern.Message to be set")
		}
		if len(r.Pattern.KBArticles) == 0 {
			t.Error("expected Pattern.KBArticles to be set")
		}
		if r.Pattern.Remediation == "" {
			t.Error("expected Pattern.Remediation to be set")
		}
	}
}

func TestPatternMatcher_EmptyPatterns(t *testing.T) {
	pm := NewPatternMatcher()

	results, err := pm.MatchAll([]Pattern{}, "some output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 matches for empty patterns, got %d", len(results))
	}
}

func TestPatternMatcher_EmptyOutput(t *testing.T) {
	pm := NewPatternMatcher()

	patterns := []Pattern{
		{Regex: "error"},
	}

	results, err := pm.MatchAll(patterns, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 matches for empty output, got %d", len(results))
	}
}

func TestPatternMatcher_MultilineOutput(t *testing.T) {
	pm := NewPatternMatcher()

	pattern := Pattern{
		Regex: `line\d+`,
	}

	output := `line1: first line
line2: second line
line3: third line`

	result, err := pm.Match(pattern, output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected match in multiline output")
	}
}
