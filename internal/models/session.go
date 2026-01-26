package models

import "time"

type Session struct {
	ID            int64
	State         string
	Application   string
	Protocol      string // tcp, udp, icmp
	SourceIP      string
	SourcePort    int
	DestIP        string
	DestPort      int
	SourceZone    string
	DestZone      string
	NATSourceIP   string
	NATSourcePort int
	User          string
	BytesIn       int64
	BytesOut      int64
	StartTime     time.Time
	Rule          string
}

// SessionDetail contains extended session information fetched on-demand.
type SessionDetail struct {
	ID int64

	// Basic info
	State       string
	Application string
	Protocol    string
	Type        string // flow, predict

	// Source/Destination
	SourceIP     string
	SourcePort   int
	DestIP       string
	DestPort     int
	SourceZone   string
	DestZone     string
	SourceUser   string
	SourceHostID string

	// NAT Info
	NATSourceIP   string
	NATSourcePort int
	NATDestIP     string
	NATDestPort   int
	NATRule       string

	// Security
	SecurityRule     string
	URLCategory      string
	URLFilteringRule string
	DecryptionRule   string

	// Traffic Stats
	PacketsToClient int64
	PacketsToServer int64
	BytesToClient   int64
	BytesToServer   int64
	TotalPackets    int64
	TotalBytes      int64

	// Timing
	StartTime  time.Time
	Timeout    int // seconds
	TimeToLive int // seconds remaining
	IdleTime   int // seconds

	// Flags
	Offloaded      bool
	DecryptMirror  bool
	LayerSevenInfo string
}
