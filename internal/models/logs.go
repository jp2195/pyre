package models

import "time"

// LogType represents the type of firewall log
type LogType string

const (
	LogTypeSystem  LogType = "system"
	LogTypeTraffic LogType = "traffic"
	LogTypeThreat  LogType = "threat"
)

// TrafficLogEntry represents a traffic log entry from the firewall
type TrafficLogEntry struct {
	Time        time.Time
	ReceiveTime time.Time
	Serial      string
	Type        string // traffic
	Subtype     string // start, end, drop, deny

	// Source/Destination
	SourceIP      string
	DestIP        string
	SourcePort    int
	DestPort      int
	NATSourceIP   string
	NATDestIP     string
	NATSourcePort int
	NATDestPort   int

	// Zones
	SourceZone string
	DestZone   string

	// Identifiers
	Rule        string
	Application string
	User        string
	SessionID   int64

	// Action and result
	Action     string // allow, deny, drop
	SessionEnd string // aged-out, tcp-fin, tcp-rst, etc.

	// Metrics
	Bytes       int64
	BytesSent   int64
	BytesRecv   int64
	Packets     int64
	PacketsSent int64
	PacketsRecv int64
	Duration    int64 // seconds

	// Protocol info
	Protocol string
	Category string
	Flags    string

	// Additional context
	VirtualSystem string
	DeviceName    string
}

// ThreatLogEntry represents a threat log entry from the firewall
type ThreatLogEntry struct {
	Time        time.Time
	ReceiveTime time.Time
	Serial      string
	Type        string // threat
	Subtype     string // vulnerability, virus, spyware, url, wildfire, etc.

	// Source/Destination
	SourceIP      string
	DestIP        string
	SourcePort    int
	DestPort      int
	NATSourceIP   string
	NATDestIP     string
	NATSourcePort int
	NATDestPort   int

	// Zones
	SourceZone string
	DestZone   string

	// Identifiers
	Rule        string
	Application string
	User        string
	SessionID   int64

	// Threat details
	ThreatID       int64
	ThreatName     string
	ThreatCategory string
	Severity       string // critical, high, medium, low, informational
	Direction      string // client-to-server, server-to-client

	// Action
	Action string // alert, allow, deny, drop, reset-client, reset-server, reset-both

	// URL/File info (for url/wildfire threats)
	URL         string
	Filename    string
	FileHash    string
	ContentType string

	// Additional context
	VirtualSystem string
	DeviceName    string
	ReportID      int64
	PCAP          string // pcap ID if captured
}
