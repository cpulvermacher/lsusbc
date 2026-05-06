package ui

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/cpulvermacher/lsusbc/internal/model"
	"github.com/cpulvermacher/lsusbc/internal/svid"
)

var (
	powerModePd          = lipgloss.NewStyle().Foreground(lipgloss.Color("#91e500"))
	powerModeCurrent3A   = lipgloss.NewStyle().Foreground(lipgloss.Color("#d0e440"))
	powerModeCurrent1_5A = lipgloss.NewStyle().Foreground(lipgloss.Color("#fae470"))
	powerModeUsb         = lipgloss.NewStyle().Foreground(lipgloss.Color("#6f453d"))

	// USB 1.0
	usbSpeed12 = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("#ffffff")).Foreground(lipgloss.Black)
	// USB 2.0
	usbSpeed480 = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("#000000")).Foreground(lipgloss.White)
	// USB 3.0 / 3.1 gen1
	usbSpeed5000 = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("#005eb8")).Foreground(lipgloss.White)
	// USB 3.1 gen2 / 3.2
	usbSpeed10000 = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("#0baadf")).Foreground(lipgloss.Black)
	// USB 3.2
	usbSpeed20000 = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("#00d6e6")).Foreground(lipgloss.Black)
	// USB 4.0 / Thunderbolt 3
	usbSpeed40000 = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("#54ffd6")).Foreground(lipgloss.Black)
)

// FormatVoltage converts mV to human-readable format (e.g., "5V", "20V", "3.3-21V")
func FormatVoltage(pc model.PowerCapability) string {
	if pc.Programmable {
		return fmt.Sprintf("%s-%s", formatMilliVolts(pc.MinimumVoltage), formatMilliVolts(pc.MaximumVoltage))
	}
	return formatMilliVolts(pc.Voltage)
}

// Watts returns the maximum power in watts for this capability.
func Watts(pc model.PowerCapability) int {
	if pc.Programmable {
		return pc.MaximumVoltage * pc.MaximumCurrent / 1_000_000
	}
	return pc.Voltage * pc.MaximumCurrent / 1_000_000
}

// MaxWatts returns the maximum wattage across a slice of capabilities.
func MaxWatts(caps []model.PowerCapability) int {
	max := 0
	for _, c := range caps {
		if w := Watts(c); w > max {
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

// formats the power mode label for the port list overview.
func formatPowerModeInline(pd *model.PowerDelivery, powerOperationMode string) string {
	switch powerOperationMode {
	case "default":
		return powerModeUsb.Render("[≤5W]")
	case "1.5A":
		return powerModeCurrent1_5A.Render("[7.5W]")
	case "3.0A":
		return powerModeCurrent3A.Render("[15W]")
	case "usb_power_delivery":
		label := "PD"
		if pd != nil {
			if pd.Revision != "" && pd.Revision != "0.0" {
				label = "PD " + pd.Revision
			}
			if watts := MaxWatts(pd.SourceCapabilities); watts > 0 {
				label = fmt.Sprintf("%s, %dW", label, watts)
			}
			if pd.ACPowered {
				label += ", AC"
			}
		}
		return powerModePd.Render("[" + label + "]")
	default:
		return ""
	}
}

func formatUsbSpeedInline(device model.USBDevice) string {
	label := formatUsbSpeed(device)
	if label != "" {
		return fmt.Sprintf(" [%s]", label)
	}
	return label
}

func formatUsbSpeed(device model.USBDevice) string {
	if device.Speed == "" {
		return ""
	}
	text := fmt.Sprintf("%s Mb/s", device.Speed)
	switch device.Speed {
	case "12":
		return usbSpeed12.Render(text)
	case "480":
		return usbSpeed480.Render(text)
	case "5000":
		return usbSpeed5000.Render(text)
	case "10000":
		return usbSpeed10000.Render(text)
	case "20000":
		return usbSpeed20000.Render(text)
	case "40000":
		return usbSpeed40000.Render(text)
	default:
		return text
	}
}

// formatAlternateMode formats a single alternate mode entry for the details panel.
func formatAlternateMode(mode model.AlternateMode) string {
	marker := " "
	if mode.Active == "yes" {
		marker = "*"
	}
	description := mode.Description
	if vendor := svid.VendorName(mode.SVID); vendor != "" && vendor != description {
		if description == "" {
			description = vendor
		} else {
			description += " (" + vendor + ")"
		}
	}
	extra := dpPortCapability(mode)
	if extra != "" {
		return fmt.Sprintf("   %s[%d] %s %s (SVID: %s, VDO: %s)\n", marker, mode.Index, description, extra, mode.SVID, mode.VDO)
	}
	return fmt.Sprintf("   %s[%d] %s (SVID: %s, VDO: %s)\n", marker, mode.Index, description, mode.SVID, mode.VDO)
}

// dpPortCapability parses a DisplayPort VDO to return a string like "sink, native DP" or "source+sink, tunneling".
func dpPortCapability(mode model.AlternateMode) string {
	if mode.SVID != "ff01" {
		return ""
	}
	vdo, err := strconv.ParseUint(strings.TrimPrefix(mode.VDO, "0x"), 16, 32)
	if err != nil {
		return ""
	}

	// Bits [1:0]: port capability
	var portCap string
	switch vdo & 0x3 {
	case 0x1:
		portCap = "sink"
	case 0x2:
		portCap = "source"
	case 0x3:
		portCap = "source+sink"
	default:
		return ""
	}

	// Bits [15:8]: DFP_D pin assignments; bits [23:16]: UFP_D pin assignments.
	// Union the sets relevant to this device's capability.
	var pins uint64
	if vdo&0x1 != 0 { // UFP_D capable
		pins |= (vdo >> 16) & 0xFF
	}
	if vdo&0x2 != 0 { // DFP_D capable
		pins |= (vdo >> 8) & 0xFF
	}

	// Bits 4-5 (E, F) = native DisplayPort; bits 0-3 (A-D) = tunneling/protocol converter.
	hasNative := pins&0x30 != 0
	hasTunnel := pins&0x0F != 0

	switch {
	case hasNative && hasTunnel:
		return portCap + ", native DP + tunneling"
	case hasNative:
		return portCap + ", native DP"
	case hasTunnel:
		return portCap + ", tunneling"
	default:
		return portCap
	}
}

// FormatCurrent converts mA to human-readable format (e.g., "1.5A", "3A")
func FormatCurrent(pc model.PowerCapability) string {
	if pc.MaximumCurrent%1000 == 0 {
		return fmt.Sprintf("%dA", pc.MaximumCurrent/1000)
	}
	return fmt.Sprintf("%.1fA", float64(pc.MaximumCurrent)/1000.0)
}
