package models

import "time"

// SourceTranslationType defines the type of source NAT translation
type SourceTranslationType string

const (
	SourceTransNone          SourceTranslationType = "none"
	SourceTransDynamicIPPort SourceTranslationType = "dynamic-ip-and-port"
	SourceTransDynamicIP     SourceTranslationType = "dynamic-ip"
	SourceTransStaticIP      SourceTranslationType = "static-ip"
)

// NATRule represents a NAT policy rule
type NATRule struct {
	Name        string
	Position    int
	Disabled    bool
	Description string
	Tags        []string
	RuleBase    RuleBase // pre, local, post - indicates rule origin

	// Match criteria
	SourceZones   []string
	DestZones     []string
	Sources       []string // Source addresses/groups
	Destinations  []string // Destination addresses/groups
	Services      []string
	DestInterface string // Destination interface (for interface-based NAT)

	// Source Translation
	SourceTransType   SourceTranslationType // "dynamic-ip-and-port", "static-ip", "dynamic-ip", "none"
	TranslatedSource  string                // Translated source address or interface
	SourceInterfaceIP bool                  // True if using interface IP for source NAT
	TranslatedSrcPort string                // Translated source port (for static IP)

	// Destination Translation
	TranslatedDest     string // Translated destination address
	TranslatedDestPort string // Translated destination port

	// NAT type indicators
	NATType      string // "ipv4", "nat64", "nptv6"
	ActiveActive bool   // Active-Active device binding

	// Statistics
	HitCount  int64
	LastHit   time.Time
	FirstHit  time.Time
	LastReset time.Time
}
