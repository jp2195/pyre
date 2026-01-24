package api

import (
	"fmt"
	"time"
)

// protoToName converts IP protocol number to name
func protoToName(proto string) string {
	switch proto {
	case "6":
		return "tcp"
	case "17":
		return "udp"
	case "1":
		return "icmp"
	case "47":
		return "gre"
	case "50":
		return "esp"
	case "51":
		return "ah"
	case "58":
		return "icmp6"
	case "89":
		return "ospf"
	default:
		if proto == "" {
			return "tcp"
		}
		return proto
	}
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 24 {
		days := hours / 24
		hours = hours % 24
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
