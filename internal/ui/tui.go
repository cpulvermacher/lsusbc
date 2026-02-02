package ui

import (
	"fmt"
	"strings"

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
	deviceName := GetFriendlyDeviceName(port.Partner)
	capabilities := formatCapabilities(port.Partner)

	fmt.Printf("%s ---󱐋--> %s  %s\n", port.Name, deviceName, capabilities)
}

// GetFriendlyDeviceName generates a friendly device description
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

// formatCapabilities formats power capabilities into a condensed string
func formatCapabilities(partner *model.Partner) string {
	var parts []string

	// Extract unique current values
	currentMap := make(map[string]bool)
	for _, cap := range partner.SourceCapabilities {
		currentMap[cap.FormatCurrent()] = true
	}
	for _, cap := range partner.SinkCapabilities {
		currentMap[cap.FormatCurrent()] = true
	}

	// Convert map to sorted list
	var currents []string
	for current := range currentMap {
		currents = append(currents, current)
	}

	if len(currents) > 0 {
		parts = append(parts, currents...)
	}

	// Add PD version if available
	if partner.PDRevision != "" && partner.PDRevision != "0.0" {
		parts = append(parts, "PD "+partner.PDRevision)
	}

	if len(parts) == 0 {
		return ""
	}

	return "[" + strings.Join(parts, ", ") + "]"
}
