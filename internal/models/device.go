package models

import "time"

type SystemInfo struct {
	// Basic Info
	Hostname   string
	Model      string
	Serial     string
	Version    string
	Uptime     string

	// Network
	IPAddress    string
	Netmask      string
	Gateway      string
	IPv6Address  string
	MACAddress   string

	// Settings
	Domain       string
	TimeZone     string
	CurrentTime  time.Time

	// Content Versions
	AppVersion         string // e.g., "8039-9750"
	AppReleaseDate     string // e.g., "11/12/25"
	ThreatVersion      string
	ThreatReleaseDate  string
	AntivirusVersion   string
	AntivirusDate      string
	WildFireVersion    string
	WildFireDate       string
	URLFilteringVersion string

	// Features
	MultiVsys       bool
	OperationalMode string // "normal", "fips-cc"
}

type Resources struct {
	CPUPercent       float64
	MemoryPercent    float64
	Load1            float64
	Load5            float64
	Load15           float64

	// Separate CPU metrics
	ManagementCPU    float64
	DataPlaneCPU     float64
}

type SessionInfo struct {
	ActiveCount    int
	MaxCount       int
	ThroughputKbps int64
	CPS            int
	// Per-protocol breakdown (optional)
	TCPSessions    int
	UDPSessions    int
	ICMPSessions   int
}

type HAStatus struct {
	Enabled   bool
	State     string // active, passive, suspended, initial
	PeerState string
	PeerIP    string
	SyncState string
	Running   bool
	Mode      string // active-passive, active-active
}

type Interface struct {
	Name   string
	State  string // up, down
	Speed  string
	Duplex string // full, half, auto
	Zone   string
	IP     string
	MAC    string
	Vsys   string
	Type   string // ethernet, vlan, loopback, tunnel
	Tag    int    // VLAN tag

	// Counters
	BytesIn    int64
	BytesOut   int64
	PacketsIn  int64
	PacketsOut int64
	ErrorsIn   int64
	ErrorsOut  int64
	DropsIn    int64
	DropsOut   int64

	// Additional info
	MTU           int
	VirtualRouter string
	Mode          string // layer3, layer2, virtual-wire, tap
	Comment       string
}

type ThreatSummary struct {
	TotalThreats   int64
	CriticalCount  int64
	HighCount      int64
	MediumCount    int64
	LowCount       int64
	BlockedCount   int64
	AlertedCount   int64
}

type GlobalProtectInfo struct {
	TotalUsers     int
	ActiveUsers    int
	ActiveGateways int
	TotalGateways  int
	Portals        int
}

type LicenseInfo struct {
	Feature     string
	Description string
	Expires     string
	Expired     bool
	DaysLeft    int
	Authcode    string
}

// LoggedInAdmin represents an admin currently logged in
type LoggedInAdmin struct {
	Username     string
	From         string // IP address
	Type         string // Web, CLI, API
	SessionStart time.Time
	IdleTime     string
}

// SystemLogEntry represents a recent system log entry
type SystemLogEntry struct {
	Time        time.Time
	Type        string // SYSTEM, CONFIG, etc.
	Severity    string // informational, low, medium, high, critical
	Description string
}

// Job represents a system job (commit, download, install, etc.)
type Job struct {
	ID        int
	Type      string    // Commit, Download, Install, etc.
	Status    string    // ACT (active), FIN (finished), PEND (pending), FAIL (failed)
	Result    string    // OK, FAIL, etc.
	Progress  int       // Progress percentage (0-100)
	StartTime time.Time // When job started
	EndTime   time.Time // When job ended
	Message   string    // Status/result message
	User      string    // User who initiated the job
}

// DiskUsage represents filesystem disk usage information
type DiskUsage struct {
	Filesystem string  // Filesystem name/path
	Size       string  // Total size (e.g., "100G")
	Used       string  // Used space (e.g., "45G")
	Available  string  // Available space (e.g., "55G")
	Percent    float64 // Usage percentage
	MountPoint string  // Mount point path
}

// Environmental represents hardware environmental sensor data
type Environmental struct {
	Component string // Component name (Fan, PSU, Temp sensor)
	Status    string // Status (normal, warning, critical)
	Value     string // Current value (e.g., "45C", "5000 RPM")
	Alarm     bool   // Whether alarm is active
}

// Certificate represents a certificate on the device
type Certificate struct {
	Name         string    // Certificate name
	Subject      string    // Certificate subject (CN)
	Issuer       string    // Certificate issuer
	NotBefore    time.Time // Valid from date
	NotAfter     time.Time // Expiration date
	DaysLeft     int       // Days until expiration
	Status       string    // Status (valid, expiring, expired)
	SerialNumber string    // Certificate serial number
	Algorithm    string    // Signature algorithm
}

// ARPEntry represents an entry in the ARP table
type ARPEntry struct {
	IP        string // IP address
	MAC       string // MAC address
	Interface string // Interface name
	Status    string // Status (complete, incomplete)
	TTL       int    // Time to live (seconds)
	Port      string // Physical port
}

// RouteEntry represents an entry in the routing table
type RouteEntry struct {
	Destination   string // Destination network/host
	Nexthop       string // Next hop address
	Metric        int    // Route metric
	Interface     string // Outgoing interface
	Protocol      string // Routing protocol (static, bgp, ospf, connected)
	VirtualRouter string // Virtual router name
	Flags         string // Route flags
	Age           int    // Route age in seconds
}

// IPSecTunnel represents an IPSec VPN tunnel
type IPSecTunnel struct {
	Name       string // Tunnel name
	Gateway    string // Remote gateway IP
	State      string // Tunnel state (up, down, init)
	LocalIP    string // Local tunnel IP
	RemoteIP   string // Remote tunnel IP
	LocalSPI   string // Local Security Parameter Index
	RemoteSPI  string // Remote SPI
	Protocol   string // Protocol (ESP, AH)
	Encryption string // Encryption algorithm
	Auth       string // Authentication algorithm
	BytesIn    int64  // Bytes received
	BytesOut   int64  // Bytes sent
	PacketsIn  int64  // Packets received
	PacketsOut int64  // Packets sent
	Uptime     string // Tunnel uptime
	Errors     int    // Error count
}

// GlobalProtectUser represents a GlobalProtect VPN user
type GlobalProtectUser struct {
	Username     string    // Username
	Domain       string    // User domain
	Computer     string    // Computer name
	ClientIP     string    // Client public IP
	VirtualIP    string    // Assigned virtual IP
	Gateway      string    // Connected gateway
	Client       string    // Client version
	LoginTime    time.Time // Login timestamp
	Duration     string    // Session duration
	BytesIn      int64     // Bytes received
	BytesOut     int64     // Bytes sent
	SourceRegion string    // Geographic region
}

// ObjectCounts represents configuration object counts
type ObjectCounts struct {
	AddressObjects  int // Address objects count
	AddressGroups   int // Address groups count
	ServiceObjects  int // Service objects count
	ServiceGroups   int // Service groups count
	ApplicationGroups int // Application groups count
	SecurityRules   int // Security rules count
	NATRules        int // NAT rules count
	Zones           int // Security zones count
	Interfaces      int // Interfaces count
	VirtualRouters  int // Virtual routers count
}

// PendingChange represents a pending configuration change
type PendingChange struct {
	User        string    // User who made the change
	Location    string    // Config location/path
	Type        string    // Type of change (add, edit, delete)
	Description string    // Change description
	Time        time.Time // When change was made
}
