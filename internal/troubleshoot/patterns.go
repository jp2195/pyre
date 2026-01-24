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

// precompiledPatterns contains commonly used PAN-OS patterns.
// These are package-private as they are only used within the troubleshoot package.
var precompiledPatterns = struct {
	sslHandshakeFailed   string
	connectionRefused    string
	authenticationFailed string
	peerUnreachable      string
	haStateNonFunctional string
	haSyncFailed         string
	haLinkDown           string
	commitFailed         string
	validationError      string
	objectReference      string
	configLocked         string
	licenseExpired       string
	highCPU              string
	highMemory           string
	oomKiller            string
}{
	sslHandshakeFailed:   `(?i)ssl.*handshake.*(fail|error)|tls.*error`,
	connectionRefused:    `(?i)connection.*refused|connect.*failed|unable.*to.*connect`,
	authenticationFailed: `(?i)auth.*fail|authentication.*error|invalid.*credentials`,
	peerUnreachable:      `(?i)peer.*unreachable|cannot.*reach.*peer|peer.*timeout`,
	haStateNonFunctional: `(?i)state:\s*(suspended|non-functional|initial)`,
	haSyncFailed:         `(?i)sync.*(fail|error)|synchronization.*problem`,
	haLinkDown:           `(?i)ha[12].*down|link.*down|monitoring.*failed`,
	commitFailed:         `(?i)commit.*fail|result:\s*fail`,
	validationError:      `(?i)validation.*error|invalid.*configuration`,
	objectReference:      `(?i)object.*not.*found|reference.*error|undefined.*object`,
	configLocked:         `(?i)config.*lock|locked.*by|configuration.*is.*locked`,
	licenseExpired:       `(?i)license.*expir|expired:\s*yes|license.*invalid`,
	highCPU:              `(?i)cpu.*([89]\d|100)%|cpu.*utilization.*high`,
	highMemory:           `(?i)memory.*([89]\d|100)%|memory.*utilization.*high`,
	oomKiller:            `(?i)oom.*kill|out.*of.*memory|memory.*exhausted`,
}

// commonKBArticles contains common PAN-OS KB article URLs.
// These are package-private as they are only used within the troubleshoot package.
var commonKBArticles = struct {
	panoramaSSL     string
	panoramaConnect string
	haConfig        string
	haSync          string
	commitFail      string
	licensing       string
	resourceUsage   string
}{
	panoramaSSL:     "https://knowledgebase.paloaltonetworks.com/KCSArticleDetail?id=kA10g000000ClGo",
	panoramaConnect: "https://knowledgebase.paloaltonetworks.com/KCSArticleDetail?id=kA10g000000ClVr",
	haConfig:        "https://knowledgebase.paloaltonetworks.com/KCSArticleDetail?id=kA10g000000ClH4",
	haSync:          "https://knowledgebase.paloaltonetworks.com/KCSArticleDetail?id=kA10g000000ClHE",
	commitFail:      "https://knowledgebase.paloaltonetworks.com/KCSArticleDetail?id=kA10g000000ClIa",
	licensing:       "https://knowledgebase.paloaltonetworks.com/KCSArticleDetail?id=kA10g000000ClJ5",
	resourceUsage:   "https://knowledgebase.paloaltonetworks.com/KCSArticleDetail?id=kA10g000000ClJK",
}
