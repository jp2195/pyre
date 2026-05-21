package models

// AddressObject is a PAN-OS named address: IP/CIDR, range, FQDN, or wildcard.
// Type discriminates which form Value carries.
type AddressObject struct {
	Name        string
	Type        string // "ip-netmask" | "ip-range" | "fqdn" | "ip-wildcard"
	Value       string
	Description string
	Tags        []string
}

// ServiceObject is a PAN-OS named service (TCP/UDP + ports).
type ServiceObject struct {
	Name        string
	Protocol    string // "tcp" | "udp"
	DestPort    string // preserved as string — handles "443", "1024-65535", "1433,1434"
	SrcPort     string
	Description string
	Tags        []string
}
