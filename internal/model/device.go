package model

import "fmt"

type Port struct {
	Name               string // e.g. "port0"
	DataRole           string // "device" | "host"
	PowerRole          string // "source" | "sink"
	PowerOperationMode string // "default", "1.5A", "3.0A", or "usb_power_delivery"
	Partner            *Partner
}

type Partner struct {
	Name               string // e.g. "port0-partner"
	PDRevision         string
	SourceCapabilities []PowerCapability
	SinkCapabilities   []PowerCapability
	AlternateModes     []AlternateMode
	AccessoryMode      string
	// USB devices connected through this partner (from symlinks in partner directory)
	USBDevices []USBDevice
}

type AlternateMode struct {
	Index       int    // The index (0, 1, 2, etc.)
	Description string // Description from the alternate mode
	SVID        string // Standard or Vendor ID
	VDO         string // Vendor Defined Object
	Active      string // "yes" | "no""
}

type USBDevice struct {
	DeviceID     string // e.g., "1-4", "2-1.3"
	Manufacturer string
	Product      string
	Serial       string
	IDVendor     string
	IDProduct    string
	Speed        string // Speed in Mb/s (e.g., "480", "5000")
	Version      string // USB version (e.g., "2.10", "3.20")
}

type PowerCapability struct {
	Voltage        int // in mV (fixed supply)
	MaximumCurrent int // in mA
	// Programmable supply fields (mutually exclusive with Voltage)
	Programmable   bool
	MinimumVoltage int // in mV
	MaximumVoltage int // in mV
}

// FormatVoltage converts mV to human-readable format (e.g., "5V", "20V", "3.3-21V")
func (pc PowerCapability) FormatVoltage() string {
	if pc.Programmable {
		return fmt.Sprintf("%s-%s", formatMilliVolts(pc.MinimumVoltage), formatMilliVolts(pc.MaximumVoltage))
	}
	return formatMilliVolts(pc.Voltage)
}

func formatMilliVolts(mv int) string {
	if mv%1000 == 0 {
		return fmt.Sprintf("%dV", mv/1000)
	}
	return fmt.Sprintf("%.2gV", float64(mv)/1000.0)
}

// FormatCurrent converts mA to human-readable format (e.g., "1.5A", "3A")
func (pc PowerCapability) FormatCurrent() string {
	if pc.MaximumCurrent%1000 == 0 {
		return fmt.Sprintf("%dA", pc.MaximumCurrent/1000)
	}
	return fmt.Sprintf("%.1fA", float64(pc.MaximumCurrent)/1000.0)
}
