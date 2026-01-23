package troubleshoot

import (
	"regexp"
	"sync"
)

// PatternMatcher compiles and caches regex patterns for efficient matching.
type PatternMatcher struct {
	mu       sync.RWMutex
	compiled map[string]*regexp.Regexp
}

// NewPatternMatcher creates a new pattern matcher.
func NewPatternMatcher() *PatternMatcher {
	return &PatternMatcher{
		compiled: make(map[string]*regexp.Regexp),
	}
}

// Match checks if the output matches the given pattern.
func (pm *PatternMatcher) Match(pattern Pattern, output string) (*MatchResult, error) {
	re, err := pm.getOrCompile(pattern.Regex)
	if err != nil {
		return nil, err
	}

	match := re.FindString(output)
	if match == "" {
		return nil, nil
	}

	return &MatchResult{
		Pattern:     pattern,
		MatchedText: match,
		Matches:     re.FindStringSubmatch(output),
	}, nil
}

// MatchAll finds all patterns that match the given output.
func (pm *PatternMatcher) MatchAll(patterns []Pattern, output string) ([]MatchResult, error) {
	var results []MatchResult

	for _, pattern := range patterns {
		result, err := pm.Match(pattern, output)
		if err != nil {
			return nil, err
		}
		if result != nil {
			results = append(results, *result)
		}
	}

	return results, nil
}

// getOrCompile retrieves a compiled regex from cache or compiles it.
func (pm *PatternMatcher) getOrCompile(pattern string) (*regexp.Regexp, error) {
	pm.mu.RLock()
	if re, ok := pm.compiled[pattern]; ok {
		pm.mu.RUnlock()
		return re, nil
	}
	pm.mu.RUnlock()

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	pm.mu.Lock()
	pm.compiled[pattern] = re
	pm.mu.Unlock()

	return re, nil
}

// MatchResult contains the result of a pattern match.
type MatchResult struct {
	Pattern     Pattern
	MatchedText string
	Matches     []string // Capture groups
}

// PrecompiledPatterns contains commonly used PAN-OS patterns.
var PrecompiledPatterns = struct {
	SSLHandshakeFailed  string
	ConnectionRefused   string
	AuthenticationFailed string
	PeerUnreachable     string
	HAStateNonFunctional string
	HASyncFailed        string
	HALinkDown          string
	CommitFailed        string
	ValidationError     string
	ObjectReference     string
	ConfigLocked        string
	LicenseExpired      string
	HighCPU             string
	HighMemory          string
	OOMKiller           string
}{
	SSLHandshakeFailed:   `(?i)ssl.*handshake.*(fail|error)|tls.*error`,
	ConnectionRefused:    `(?i)connection.*refused|connect.*failed|unable.*to.*connect`,
	AuthenticationFailed: `(?i)auth.*fail|authentication.*error|invalid.*credentials`,
	PeerUnreachable:      `(?i)peer.*unreachable|cannot.*reach.*peer|peer.*timeout`,
	HAStateNonFunctional: `(?i)state:\s*(suspended|non-functional|initial)`,
	HASyncFailed:         `(?i)sync.*(fail|error)|synchronization.*problem`,
	HALinkDown:           `(?i)ha[12].*down|link.*down|monitoring.*failed`,
	CommitFailed:         `(?i)commit.*fail|result:\s*fail`,
	ValidationError:      `(?i)validation.*error|invalid.*configuration`,
	ObjectReference:      `(?i)object.*not.*found|reference.*error|undefined.*object`,
	ConfigLocked:         `(?i)config.*lock|locked.*by|configuration.*is.*locked`,
	LicenseExpired:       `(?i)license.*expir|expired:\s*yes|license.*invalid`,
	HighCPU:              `(?i)cpu.*([89]\d|100)%|cpu.*utilization.*high`,
	HighMemory:           `(?i)memory.*([89]\d|100)%|memory.*utilization.*high`,
	OOMKiller:            `(?i)oom.*kill|out.*of.*memory|memory.*exhausted`,
}

// CommonKBArticles contains common PAN-OS KB article URLs.
var CommonKBArticles = struct {
	PanoramaSSL       string
	PanoramaConnect   string
	HAConfig          string
	HASync            string
	CommitFail        string
	Licensing         string
	ResourceUsage     string
}{
	PanoramaSSL:     "https://knowledgebase.paloaltonetworks.com/KCSArticleDetail?id=kA10g000000ClGo",
	PanoramaConnect: "https://knowledgebase.paloaltonetworks.com/KCSArticleDetail?id=kA10g000000ClVr",
	HAConfig:        "https://knowledgebase.paloaltonetworks.com/KCSArticleDetail?id=kA10g000000ClH4",
	HASync:          "https://knowledgebase.paloaltonetworks.com/KCSArticleDetail?id=kA10g000000ClHE",
	CommitFail:      "https://knowledgebase.paloaltonetworks.com/KCSArticleDetail?id=kA10g000000ClIa",
	Licensing:       "https://knowledgebase.paloaltonetworks.com/KCSArticleDetail?id=kA10g000000ClJ5",
	ResourceUsage:   "https://knowledgebase.paloaltonetworks.com/KCSArticleDetail?id=kA10g000000ClJK",
}
