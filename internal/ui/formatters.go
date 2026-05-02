package ui

import (
	"fmt"

	"github.com/cpulvermacher/lsusbc/internal/model"
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

// FormatCurrent converts mA to human-readable format (e.g., "1.5A", "3A")
func FormatCurrent(pc model.PowerCapability) string {
	if pc.MaximumCurrent%1000 == 0 {
		return fmt.Sprintf("%dA", pc.MaximumCurrent/1000)
	}
	return fmt.Sprintf("%.1fA", float64(pc.MaximumCurrent)/1000.0)
}
