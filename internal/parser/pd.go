package parser

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cpulvermacher/lsusbc/internal/model"
)

// Power-delivery related bits

// parsePDDirectories scans for and parses all PD directories (pd0, pd1, pd2, etc.)
// Returns nil if no PD directories are found.
func parsePDDirectories(partnerDir string) *model.PowerDelivery {
	entries, err := os.ReadDir(partnerDir)
	if err != nil {
		return nil
	}

	var pd *model.PowerDelivery
	for _, entry := range entries {
		name := entry.Name()

		// Check if it matches the pattern: pd<digit>
		if !strings.HasPrefix(name, "pd") || !entry.IsDir() {
			continue
		}

		// Verify it's pdX format (pd followed by digits)
		suffix := strings.TrimPrefix(name, "pd")
		if _, err := strconv.Atoi(suffix); err != nil {
			continue
		}

		if pd == nil {
			pd = &model.PowerDelivery{}
		}

		pdDir := filepath.Join(partnerDir, name)

		// Parse PD revision (use first one found if not already set)
		if pd.Revision == "" {
			pd.Revision = readFile(filepath.Join(pdDir, "revision"))
		}

		// Parse source capabilities
		sourceCapsDir := filepath.Join(pdDir, "source-capabilities")
		if caps, err := parseCapabilities(sourceCapsDir); err == nil {
			pd.SourceCapabilities = append(pd.SourceCapabilities, caps...)
		}
		if readFile(filepath.Join(sourceCapsDir, "1:fixed_supply", "unconstrained_power")) == "1" {
			pd.ACPowered = true
		}

		// Parse sink capabilities
		sinkCapsDir := filepath.Join(pdDir, "sink-capabilities")
		if caps, err := parseCapabilities(sinkCapsDir); err == nil {
			pd.SinkCapabilities = append(pd.SinkCapabilities, caps...)
		}
	}
	return pd
}

// parseSinkCapabilities parses sink PDOs, capturing the voltage (fixed supply) or
// voltage range (battery/variable/programmable supply) each one covers.
//
// Unlike parseCapabilities (source PDOs), sink PDOs advertise operational current/power
// rather than maximum_current, so this only requires voltage information. Returns nil
// if the directory is missing. Used for the local port's own sink capabilities.
func parseSinkCapabilities(capsDir string) []model.PowerCapability {
	entries, err := os.ReadDir(capsDir)
	if err != nil {
		return nil
	}

	var capabilities []model.PowerCapability
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		capDir := filepath.Join(capsDir, name)

		switch {
		case strings.HasSuffix(name, ":fixed_supply"):
			voltage, err := parseMilliValue(readFile(filepath.Join(capDir, "voltage")))
			if err != nil {
				continue
			}
			capabilities = append(capabilities, model.PowerCapability{Voltage: voltage})
		case strings.HasSuffix(name, ":battery"),
			strings.HasSuffix(name, ":variable_supply"),
			strings.HasSuffix(name, ":programmable_supply"):
			minVoltage, err1 := parseMilliValue(readFile(filepath.Join(capDir, "minimum_voltage")))
			maxVoltage, err2 := parseMilliValue(readFile(filepath.Join(capDir, "maximum_voltage")))
			if err1 != nil || err2 != nil {
				continue
			}
			capabilities = append(capabilities, model.PowerCapability{
				Programmable:   strings.HasSuffix(name, ":programmable_supply"),
				MinimumVoltage: minVoltage,
				MaximumVoltage: maxVoltage,
			})
		}
	}

	return capabilities
}

// parseCapabilities parses PD capabilities from a directory
func parseCapabilities(capsDir string) ([]model.PowerCapability, error) {
	entries, err := os.ReadDir(capsDir)
	if err != nil {
		return nil, err
	}

	var capabilities []model.PowerCapability
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		capDir := filepath.Join(capsDir, entry.Name())
		name := entry.Name()

		currentStr := readFile(filepath.Join(capDir, "maximum_current"))
		current, err := parseMilliValue(currentStr)
		if err != nil {
			continue
		}

		if strings.HasSuffix(name, ":programmable_supply") || strings.HasSuffix(name, ":variable_supply") {
			minVoltageStr := readFile(filepath.Join(capDir, "minimum_voltage"))
			maxVoltageStr := readFile(filepath.Join(capDir, "maximum_voltage"))
			minVoltage, err := parseMilliValue(minVoltageStr)
			if err != nil {
				continue
			}
			maxVoltage, err := parseMilliValue(maxVoltageStr)
			if err != nil {
				continue
			}
			capabilities = append(capabilities, model.PowerCapability{
				Programmable:   true,
				MinimumVoltage: minVoltage,
				MaximumVoltage: maxVoltage,
				MaximumCurrent: current,
			})
		} else {
			voltageStr := readFile(filepath.Join(capDir, "voltage"))
			voltage, err := parseMilliValue(voltageStr)
			if err != nil {
				continue
			}
			capabilities = append(capabilities, model.PowerCapability{
				Voltage:        voltage,
				MaximumCurrent: current,
			})
		}
	}

	return capabilities, nil
}
