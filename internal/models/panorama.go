package models

// ManagedDevice represents a firewall managed by Panorama.
type ManagedDevice struct {
	Serial      string // For API target= parameter
	Hostname    string // Display name
	IPAddress   string // Management IP for direct SSH
	Model       string // PA-3260, PA-5260, etc.
	SWVersion   string // PAN-OS version
	HAState     string // active, passive, suspended
	Connected   bool   // Connected to Panorama?
	DeviceGroup string // Panorama device group
}
