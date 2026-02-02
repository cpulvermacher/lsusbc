package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/christian/usb-c/internal/model"
)

// LoadSnapshot loads all ports from a snapshot directory
func LoadSnapshot(path string) ([]model.Port, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot directory: %w", err)
	}

	var ports []model.Port
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "port") && !strings.Contains(entry.Name(), "-") {
			portPath := filepath.Join(path, entry.Name())
			port, err := parsePort(portPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to parse port %s: %v\n", entry.Name(), err)
				continue
			}
			ports = append(ports, *port)
		}
	}

	return ports, nil
}

// parsePort parses an individual port directory
func parsePort(portDir string) (*model.Port, error) {
	port := &model.Port{
		Name: filepath.Base(portDir),
	}

	// Parse port files
	port.DataRole = extractActiveRole(readFile(filepath.Join(portDir, "data_role")))
	port.PowerRole = extractActiveRole(readFile(filepath.Join(portDir, "power_role")))
	port.PowerOperationMode = strings.TrimSpace(readFile(filepath.Join(portDir, "power_operation_mode")))

	// Check for partner
	partnerDir := portDir + "-partner"
	if _, err := os.Stat(partnerDir); err == nil {
		partner, err := parsePartner(partnerDir, port.DataRole, port.PowerRole)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse partner for %s: %v\n", port.Name, err)
		} else {
			port.Partner = partner
		}
	}

	return port, nil
}

// parsePartner parses partner information
func parsePartner(partnerDir string, dataRole, powerRole string) (*model.Partner, error) {
	partner := &model.Partner{
		Name:      filepath.Base(partnerDir),
		DataRole:  dataRole,
		PowerRole: powerRole,
	}

	// Parse basic partner info
	partner.AccessoryMode = strings.TrimSpace(readFile(filepath.Join(partnerDir, "accessory_mode")))

	// Parse alternate mode
	altModeDir := partnerDir + ".0"
	if _, err := os.Stat(altModeDir); err == nil {
		description := strings.TrimSpace(readFile(filepath.Join(altModeDir, "description")))
		if description != "" {
			partner.AlternateMode = description
		}
	}

	// Parse PD information
	pd1Dir := filepath.Join(partnerDir, "pd1")
	if _, err := os.Stat(pd1Dir); err == nil {
		partner.PDRevision = strings.TrimSpace(readFile(filepath.Join(pd1Dir, "revision")))

		// Parse source capabilities
		sourceCapsDir := filepath.Join(pd1Dir, "source-capabilities")
		if caps, err := parseCapabilities(sourceCapsDir); err == nil {
			partner.SourceCapabilities = caps
		}

		// Parse sink capabilities
		sinkCapsDir := filepath.Join(pd1Dir, "sink-capabilities")
		if caps, err := parseCapabilities(sinkCapsDir); err == nil {
			partner.SinkCapabilities = caps
		}
	}

	return partner, nil
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

		voltageStr := strings.TrimSpace(readFile(filepath.Join(capDir, "voltage")))
		currentStr := strings.TrimSpace(readFile(filepath.Join(capDir, "maximum_current")))

		voltage, err := parseMilliValue(voltageStr)
		if err != nil {
			continue
		}

		current, err := parseMilliValue(currentStr)
		if err != nil {
			continue
		}

		capabilities = append(capabilities, model.PowerCapability{
			Voltage:        voltage,
			MaximumCurrent: current,
		})
	}

	return capabilities, nil
}

// extractActiveRole extracts the active role from bracketed format (e.g., "[host] device" -> "host")
func extractActiveRole(content string) string {
	content = strings.TrimSpace(content)
	start := strings.Index(content, "[")
	end := strings.Index(content, "]")
	if start != -1 && end != -1 && end > start {
		return content[start+1 : end]
	}
	return content
}

// parseMilliValue parses values like "5000mV" or "3000mA" and returns the integer value
func parseMilliValue(s string) (int, error) {
	s = strings.TrimSpace(s)
	// Remove the unit suffix (mV, mA, etc.)
	s = strings.TrimSuffix(s, "mV")
	s = strings.TrimSuffix(s, "mA")
	s = strings.TrimSpace(s)

	return strconv.Atoi(s)
}

// readFile reads a file and returns its content, or empty string on error
func readFile(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(content)
}
