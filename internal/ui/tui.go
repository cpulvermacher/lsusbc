package ui

import (
	"fmt"

	"github.com/christian/usb-c/internal/model"
)

// RenderPorts renders the ports to stdout
func RenderPorts(ports []model.Port) {
	if len(ports) == 0 {
		fmt.Println("No USB-C ports found in snapshot")
		return
	}

	for _, port := range ports {
		if port.Partner == nil {
			fmt.Printf("%s (no device connected)\n", port.Name)
		} else {
			renderConnection(port)
		}
	}
}

// renderConnection renders a port-partner connection
func renderConnection(port model.Port) {
	capabilities := formatCapabilities(port.Partner, port.PowerOperationMode)

	// Arrow direction based on power flow
	// If port is sink, it receives power (arrow points toward port)
	// If port is source, it provides power (arrow points toward device)
	var arrow string
	if port.PowerRole == "sink" {
		arrow = "<--󱐋---"
	} else {
		arrow = "---󱐋-->"
	}

	// Handle single device vs multiple devices
	if len(port.Partner.USBDevices) == 0 {
		// No USB device info available - use generic name
		deviceName := GetFriendlyDeviceName(port.Partner)
		fmt.Printf("%s %s %s  %s\n", port.Name, arrow, deviceName, capabilities)
	} else if len(port.Partner.USBDevices) == 1 {
		// Single USB device - show on same line
		device := port.Partner.USBDevices[0]
		deviceName := formatUSBDevice(device)
		fmt.Printf("%s %s %s  %s\n", port.Name, arrow, deviceName, capabilities)
	} else {
		// Multiple USB devices - show as tree
		fmt.Printf("%s %s %s\n", port.Name, arrow, capabilities)
		for i, device := range port.Partner.USBDevices {
			deviceName := formatUSBDevice(device)
			if i == len(port.Partner.USBDevices)-1 {
				fmt.Printf("        └─ %s\n", deviceName)
			} else {
				fmt.Printf("        ├─ %s\n", deviceName)
			}
		}
	}
}

// formatUSBDevice formats a USB device name
func formatUSBDevice(device model.USBDevice) string {
	if device.Manufacturer != "" && device.Product != "" {
		return device.Manufacturer + " " + device.Product
	}
	if device.Product != "" {
		return device.Product
	}
	if device.Manufacturer != "" {
		return device.Manufacturer + " Device"
	}
	return "USB Device"
}

// GetFriendlyDeviceName generates a friendly device description when USB device info is not available
func GetFriendlyDeviceName(partner *model.Partner) string {
	// Priority 1: Alternate mode description
	if partner.AlternateMode != "" {
		return partner.AlternateMode + " Device"
	}

	// Priority 2: Charger (device role + sink power role)
	if partner.DataRole == "device" && partner.PowerRole == "sink" {
		return "Charger"
	}

	// Priority 3: Phone/Device (device role + source power role)
	if partner.DataRole == "device" && partner.PowerRole == "source" {
		return "Phone/Device"
	}

	// Priority 4: Audio accessory
	if partner.AccessoryMode == "audio" {
		return "Audio Accessory"
	}

	// Fallback
	return "USB Device"
}

// formatCapabilities formats power capabilities based on power operation mode
func formatCapabilities(partner *model.Partner, powerOperationMode string) string {
	// Use power_operation_mode to decide what to show
	switch powerOperationMode {
	case "default":
		return "[USB]"
	case "1.5A":
		return "[1.5A]"
	case "3.0A":
		return "[3A]"
	case "usb_power_delivery":
		// Show PD version only
		if partner.PDRevision != "" && partner.PDRevision != "0.0" {
			return "[PD " + partner.PDRevision + "]"
		}
		return "[PD]"
	default:
		return ""
	}
}
