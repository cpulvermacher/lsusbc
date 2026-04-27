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
func parsePDDirectories(partner *model.Partner, partnerDir string) {
	entries, err := os.ReadDir(partnerDir)
	if err != nil {
		return
	}

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

		pdDir := filepath.Join(partnerDir, name)

		// Parse PD revision (use first one found if not already set)
		if partner.PDRevision == "" {
			partner.PDRevision = readFile(filepath.Join(pdDir, "revision"))
		}

		// Parse source capabilities
		sourceCapsDir := filepath.Join(pdDir, "source-capabilities")
		if caps, err := parseCapabilities(sourceCapsDir); err == nil {
			partner.SourceCapabilities = append(partner.SourceCapabilities, caps...)
		}
		if readFile(filepath.Join(sourceCapsDir, "1:fixed_supply", "unconstrained_power")) == "1" {
			partner.ACPowered = true
		}

		// Parse sink capabilities
		sinkCapsDir := filepath.Join(pdDir, "sink-capabilities")
		if caps, err := parseCapabilities(sinkCapsDir); err == nil {
			partner.SinkCapabilities = append(partner.SinkCapabilities, caps...)
		}
	}
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
