package model

import "fmt"

type Port struct {
	Name               string
	DataRole           string
	PowerRole          string
	PowerOperationMode string // "default", "1.5A", "3.0A", or "usb_power_delivery"
	Partner            *Partner
}

type Partner struct {
	Name               string
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
}

type USBDevice struct {
	DeviceID     string // e.g., "1-4", "2-1.3"
	Manufacturer string
	Product      string
	Serial       string
	IDVendor     string
	IDProduct    string
}

type PowerCapability struct {
	Voltage        int // in mV
	MaximumCurrent int // in mA
}

// FormatVoltage converts mV to human-readable format (e.g., "5V", "20V")
func (pc PowerCapability) FormatVoltage() string {
	if pc.Voltage%1000 == 0 {
		return fmt.Sprintf("%dV", pc.Voltage/1000)
	}
	return fmt.Sprintf("%.1fV", float64(pc.Voltage)/1000.0)
}

// FormatCurrent converts mA to human-readable format (e.g., "1.5A", "3A")
func (pc PowerCapability) FormatCurrent() string {
	if pc.MaximumCurrent%1000 == 0 {
		return fmt.Sprintf("%dA", pc.MaximumCurrent/1000)
	}
	return fmt.Sprintf("%.1fA", float64(pc.MaximumCurrent)/1000.0)
}
