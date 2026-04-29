// Package model defines data structures representing USB-C ports and connected devices.
package model

import "fmt"

type Port struct {
	Name               string // e.g. "port0"
	DataRole           string // "device" | "host"
	PowerRole          string // "source" | "sink"
	PowerOperationMode string // "default", "1.5A", "3.0A", or "usb_power_delivery"
	Partner            *Partner
	Cable              *Cable
}

type Cable struct {
	Type           string          // "passive", "active", "undefined"
	PlugType       string          // "type-a", "type-b", "type-c", "captive"
	AlternateModes []AlternateMode // from plug device(s)
}

type Partner struct {
	Name           string // e.g. "port0-partner"
	PowerDelivery  *PowerDelivery
	AlternateModes []AlternateMode
	AccessoryMode  string
	USBDevices     []USBDevice // USB devices connected through this partner (from symlinks in partner directory)
}

type PowerDelivery struct {
	Revision           string
	ACPowered          bool // unconstrained_power bit: source is externally (AC) powered (false: battery-powered or not set)
	SourceCapabilities []PowerCapability
	SinkCapabilities   []PowerCapability
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
	Speed        string      // Speed in Mb/s (e.g., "480", "5000")
	Version      string      // USB version (e.g., "2.10", "3.20")
	USBDevices   []USBDevice // USB devices connected through this hub
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

// Watts returns the maximum power in watts for this capability.
func (pc PowerCapability) Watts() int {
	if pc.Programmable {
		return pc.MaximumVoltage * pc.MaximumCurrent / 1_000_000
	}
	return pc.Voltage * pc.MaximumCurrent / 1_000_000
}

// MaxWatts returns the maximum wattage across a slice of capabilities.
func MaxWatts(caps []PowerCapability) int {
	max := 0
	for _, c := range caps {
		if w := c.Watts(); w > max {
			max = w
		}
	}
	return max
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
