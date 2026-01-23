package models

import "time"

// RuleType indicates the type of security rule
type RuleType string

const (
	RuleTypeUniversal  RuleType = "universal"
	RuleTypeIntrazone  RuleType = "intrazone"
	RuleTypeInterzone  RuleType = "interzone"
)

type SecurityRule struct {
	Name         string
	Position     int
	Disabled     bool
	Description  string
	Tags         []string
	RuleType     RuleType  // universal, intrazone, interzone
	Action       string    // allow, deny, drop, reset-client, reset-server, reset-both

	// Source criteria
	SourceZones  []string
	Sources      []string  // addresses/groups
	SourceUsers  []string
	NegateSource bool

	// Destination criteria
	DestZones    []string
	Destinations []string  // addresses/groups
	NegateDest   bool

	// Service/Application
	Applications []string
	Services     []string
	URLCategories []string

	// Security profiles
	Profile         string   // profile group name
	ProfileType     string   // "group" or "profiles"
	AntivirusProfile    string
	VulnerabilityProfile string
	SpywareProfile      string
	URLFilteringProfile string
	FileBlockingProfile string
	WildFireProfile     string

	// Logging
	LogStart     bool
	LogEnd       bool
	LogForwarding string

	// Rule usage statistics
	HitCount     int64
	LastHit      time.Time
	FirstHit     time.Time
	LastReset    time.Time
	AppsSeen     int       // number of unique apps seen
}
