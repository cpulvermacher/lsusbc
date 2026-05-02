package ui

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/cpulvermacher/lsusbc/internal/model"
)

var (
	powerModePd          = lipgloss.NewStyle().Foreground(lipgloss.Color("#91e500"))
	powerModeCurrent3A   = lipgloss.NewStyle().Foreground(lipgloss.Color("#d0e440"))
	powerModeCurrent1_5A = lipgloss.NewStyle().Foreground(lipgloss.Color("#fae470"))
	powerModeUsb         = lipgloss.NewStyle().Foreground(lipgloss.Color("#6f453d"))
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

// formatCapabilities formats the power mode label for the port list overview.
func formatCapabilities(pd *model.PowerDelivery, powerOperationMode string) string {
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

// FormatCurrent converts mA to human-readable format (e.g., "1.5A", "3A")
func FormatCurrent(pc model.PowerCapability) string {
	if pc.MaximumCurrent%1000 == 0 {
		return fmt.Sprintf("%dA", pc.MaximumCurrent/1000)
	}
	return fmt.Sprintf("%.1fA", float64(pc.MaximumCurrent)/1000.0)
}
