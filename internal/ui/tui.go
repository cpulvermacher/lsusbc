package ui

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/christian/usb-c/internal/model"
	"github.com/christian/usb-c/internal/parser"
)

type UIModel struct {
	TypecDir string
	Ports    []model.Port
}

type RefreshTick time.Time

func doTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return RefreshTick(t)
	})
}

func (m UIModel) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return doTick()
}

func (m UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case RefreshTick:
		return refresh(m), doTick()
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			return refresh(m), nil
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	}

	return m, nil
}

func (m UIModel) View() string {
	if len(m.Ports) == 0 {
		return ("No USB-C ports found in snapshot")
	}

	var lines string
	for _, port := range m.Ports {
		if port.Partner == nil {
			lines += fmt.Sprintf("%s (no device connected)\n", port.Name)
		} else {
			lines += renderConnection(port)
		}
	}
	return lines
}

func refresh(m UIModel) UIModel {
	// Load ports
	ports, err := parser.LoadPorts(m.TypecDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ports: %v\n", err)
		os.Exit(1)
	}

	return UIModel{
		TypecDir: m.TypecDir,
		Ports:    ports,
	}
}

// renderConnection renders a port-partner connection
func renderConnection(port model.Port) string {
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
		deviceName := getFriendlyDeviceName(port.Partner)
		return fmt.Sprintf("%s %s %s  %s\n", port.Name, arrow, deviceName, capabilities)
	} else if len(port.Partner.USBDevices) == 1 {
		// Single USB device - show on same line
		device := port.Partner.USBDevices[0]
		deviceName := formatUSBDevice(device)
		return fmt.Sprintf("%s %s %s  %s\n", port.Name, arrow, deviceName, capabilities)
	} else {
		// Multiple USB devices - show as tree
		var tree string
		tree += fmt.Sprintf("%s %s %s\n", port.Name, arrow, capabilities)
		for i, device := range port.Partner.USBDevices {
			deviceName := formatUSBDevice(device)
			if i == len(port.Partner.USBDevices)-1 {
				tree += fmt.Sprintf("        └─ %s\n", deviceName)
			} else {
				tree += fmt.Sprintf("        ├─ %s\n", deviceName)
			}
		}
		return tree
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

// getFriendlyDeviceName generates a friendly device description when USB device info is not available
func getFriendlyDeviceName(partner *model.Partner) string {
	// Priority 1: Alternate mode description(s)
	if len(partner.AlternateModes) > 0 {
		// If there's only one alternate mode, show it directly
		if len(partner.AlternateModes) == 1 {
			return partner.AlternateModes[0].Description + " Device"
		}
		// If multiple alternate modes, concatenate them
		var modes string
		for i, mode := range partner.AlternateModes {
			if i > 0 {
				modes += ", "
			}
			modes += mode.Description
		}
		return modes + " Device"
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
