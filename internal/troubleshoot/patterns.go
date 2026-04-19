package troubleshoot

import (
	"regexp"
	"sync"
)

// PatternMatcher compiles and caches regex patterns for efficient matching.
// The common PAN-OS patterns enumerated in precompiledPatterns are compiled
// eagerly at package init (see precompiledRegexCache) and seeded into every
// new matcher; runtime-supplied patterns are lazily compiled via getOrCompile.
type PatternMatcher struct {
	mu       sync.RWMutex
	compiled map[string]*regexp.Regexp
}

// NewPatternMatcher creates a new pattern matcher seeded with the
// precompiled common-pattern set.
func NewPatternMatcher() *PatternMatcher {
	pm := &PatternMatcher{
		compiled: make(map[string]*regexp.Regexp, len(precompiledRegexCache)),
	}
	for src, re := range precompiledRegexCache {
		pm.compiled[src] = re
	}
	return pm
}

// Match checks if the output matches the given pattern.
func (pm *PatternMatcher) Match(pattern Pattern, output string) (*MatchResult, error) {
	re, err := pm.getOrCompile(pattern.Regex)
	if err != nil {
		return nil, err
	}

	// Single scan: FindStringSubmatchIndex gives us both the full match span
	// and any capture-group spans in one pass. Previously Match called
	// FindString then FindStringSubmatch, scanning twice.
	idx := re.FindStringSubmatchIndex(output)
	if idx == nil {
		return nil, nil
	}

	matches := make([]string, len(idx)/2)
	for i := range matches {
		start, end := idx[2*i], idx[2*i+1]
		if start < 0 || end < 0 {
			// Non-participating capture group.
			continue
		}
		matches[i] = output[start:end]
	}

	return &MatchResult{
		Pattern:     pattern,
		MatchedText: matches[0],
		Matches:     matches,
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

// precompiledPatterns holds the canonical regex source strings for the
// common PAN-OS patterns, addressable by name. precompiledRegexCache
// (below) maps each of these source strings to its compiled form.
//
// TODO: reviewer suggested folding these two vars into a single
// map[string]*regexp.Regexp. Kept separate for now because
// patterns_test.go dereferences precompiledPatterns.<name> as a named
// source string when checking that each canonical pattern compiles and
// matches its sample. Collapsing into one map would force the test to
// key by the raw regex string, losing the name-based assertion. Revisit
// once the test is reshaped (or move the name→source map into a test
// helper).
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

// precompiledRegexCache maps each common PAN-OS pattern source to its
// compiled form. Populated via regexp.MustCompile at package init so
// compilation happens once per process, not once per matcher instance.
// NewPatternMatcher copies this map into its per-instance cache.
var precompiledRegexCache = map[string]*regexp.Regexp{
	precompiledPatterns.sslHandshakeFailed:   regexp.MustCompile(precompiledPatterns.sslHandshakeFailed),
	precompiledPatterns.connectionRefused:    regexp.MustCompile(precompiledPatterns.connectionRefused),
	precompiledPatterns.authenticationFailed: regexp.MustCompile(precompiledPatterns.authenticationFailed),
	precompiledPatterns.peerUnreachable:      regexp.MustCompile(precompiledPatterns.peerUnreachable),
	precompiledPatterns.haStateNonFunctional: regexp.MustCompile(precompiledPatterns.haStateNonFunctional),
	precompiledPatterns.haSyncFailed:         regexp.MustCompile(precompiledPatterns.haSyncFailed),
	precompiledPatterns.haLinkDown:           regexp.MustCompile(precompiledPatterns.haLinkDown),
	precompiledPatterns.commitFailed:         regexp.MustCompile(precompiledPatterns.commitFailed),
	precompiledPatterns.validationError:      regexp.MustCompile(precompiledPatterns.validationError),
	precompiledPatterns.objectReference:      regexp.MustCompile(precompiledPatterns.objectReference),
	precompiledPatterns.configLocked:         regexp.MustCompile(precompiledPatterns.configLocked),
	precompiledPatterns.licenseExpired:       regexp.MustCompile(precompiledPatterns.licenseExpired),
	precompiledPatterns.highCPU:              regexp.MustCompile(precompiledPatterns.highCPU),
	precompiledPatterns.highMemory:           regexp.MustCompile(precompiledPatterns.highMemory),
	precompiledPatterns.oomKiller:            regexp.MustCompile(precompiledPatterns.oomKiller),
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
