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
