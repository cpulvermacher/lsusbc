// Package model defines data structures representing USB-C ports and connected devices.
package model

type Port struct {
	Name               string // e.g. "port0"
	DataRole           string // "device" | "host"
	PowerRole          string // "source" | "sink"
	PowerOperationMode string // "default", "1.5A", "3.0A", or "usb_power_delivery"
	Partner            *Partner
	Cable              *Cable
	// SinkCapabilities are this local port's own sink PDOs (the voltages/ranges it is
	// willing to consume), read from its usb_power_delivery object. Used to determine
	// which of a partner's source capabilities can actually be negotiated. Empty if unknown.
	SinkCapabilities []PowerCapability
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
	MaxPower     string      // Max power draw (e.g., "500mA")
	Drivers      []string    // Driver names from interface subdirectories (e.g., ["usbhid", "usb-storage"])
	USBDevices   []USBDevice // USB devices connected through this hub
}

// BatteryInfo holds battery charge information
type BatteryInfo struct {
	Capacity      int    // Battery level in percent
	CapacityLevel string // Unknown, Critical, Low, Normal, High, Full
	Status        string // Charging, Discharging, Full, Not charging, Unknown
	PowerNow      int    // Current power flow in microwatts (0 if unavailable)
}

type PowerCapability struct {
	Voltage        int // in mV (fixed supply)
	MaximumCurrent int // in mA
	// Programmable supply fields (mutually exclusive with Voltage)
	Programmable   bool
	MinimumVoltage int // in mV
	MaximumVoltage int // in mV
}

// VoltageRange returns the inclusive voltage interval (mV) this capability covers.
// A fixed supply covers a single voltage; programmable/variable/battery supplies
// cover [MinimumVoltage, MaximumVoltage].
func (pc PowerCapability) VoltageRange() (minV, maxV int) {
	if pc.MaximumVoltage > 0 {
		return pc.MinimumVoltage, pc.MaximumVoltage
	}
	return pc.Voltage, pc.Voltage
}

// SourceCapUsable reports whether a sink described by sinkCaps could consume power
// from the given source capability. Per the USB Power Delivery spec a sink may only
// request a source PDO at a voltage it has advertised it can sink, so the source PDO
// is usable iff its voltage range overlaps any sink PDO's voltage range.
//
// This predicts what is negotiable; it does not reflect the PDO actually selected,
// which sysfs does not expose. Returns false when sinkCaps is empty (unknown).
func SourceCapUsable(sourceCap PowerCapability, sinkCaps []PowerCapability) bool {
	srcLo, srcHi := sourceCap.VoltageRange()
	for _, sc := range sinkCaps {
		lo, hi := sc.VoltageRange()
		if srcLo <= hi && lo <= srcHi {
			return true
		}
	}
	return false
}
