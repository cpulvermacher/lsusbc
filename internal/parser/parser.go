// Package parser parses /sys file-system into model
package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cpulvermacher/lsusbc/internal/model"
)

// LoadPorts loads all ports from a typec directory (live system or snapshot)
func LoadPorts(path string) ([]model.Port, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var ports []model.Port
	for _, entry := range entries {
		// Check if it's a port directory/symlink (port0, port1, etc. but not port0-partner)
		if strings.HasPrefix(entry.Name(), "port") && !strings.Contains(entry.Name(), "-") {
			portPath := filepath.Join(path, entry.Name())

			// Verify it's a directory (follow symlinks)
			info, err := os.Stat(portPath)
			if err != nil || !info.IsDir() {
				continue
			}

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
	port.PowerOperationMode = readFile(filepath.Join(portDir, "power_operation_mode"))

	// Check for cable
	cableDir := portDir + "-cable"
	if _, err := os.Stat(cableDir); err == nil {
		cable, err := parseCable(cableDir, portDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse cable for %s: %v\n", port.Name, err)
		} else {
			port.Cable = cable
		}
	}

	// Check for partner
	partnerDir := portDir + "-partner"
	if _, err := os.Stat(partnerDir); err == nil {
		partner, err := parsePartner(partnerDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse partner for %s: %v\n", port.Name, err)
		} else {
			port.Partner = partner
		}
	}

	return port, nil
}

// parsePartner parses partner information
func parsePartner(partnerDir string) (*model.Partner, error) {
	partner := &model.Partner{
		Name: filepath.Base(partnerDir),
	}

	// Parse basic partner info
	partner.AccessoryMode = readFile(filepath.Join(partnerDir, "accessory_mode"))

	// Parse all alternate modes
	partner.AlternateModes = parseAlternateModes(partnerDir)

	// Parse PD information from all pdX directories
	parsePDDirectories(partner, partnerDir)

	// Try to find and parse USB device information
	parseUSBDeviceInfo(partner, partnerDir)

	return partner, nil
}

// parseCable parses cable information from the cable directory and its plug(s)
func parseCable(cableDir string, portDir string) (*model.Cable, error) {
	cable := &model.Cable{
		Type:     readFile(filepath.Join(cableDir, "type")),
		PlugType: readFile(filepath.Join(cableDir, "plug_type")),
	}

	// Plug directories (port0-plug0, port0-plug1, ...) are siblings of the port
	portName := filepath.Base(portDir)
	typecDir := filepath.Dir(portDir)
	for i := 0; ; i++ {
		plugDir := filepath.Join(typecDir, fmt.Sprintf("%s-plug%d", portName, i))
		if _, err := os.Stat(plugDir); err != nil {
			break
		}
		cable.AlternateModes = append(cable.AlternateModes, parseAlternateModes(plugDir)...)
	}

	return cable, nil
}

// parseAlternateModes scans for and parses all alternate mode directories
func parseAlternateModes(partnerDir string) []model.AlternateMode {
	partnerName := filepath.Base(partnerDir)
	var alternateModes []model.AlternateMode

	// Scan directory for alternate mode entries (port0-partner.0, port0-partner.1, etc.)
	entries, err := os.ReadDir(partnerDir)
	if err != nil {
		return alternateModes
	}

	for _, entry := range entries {
		name := entry.Name()

		// Check if it matches the pattern: <partner-name>.<digit>
		prefix := partnerName + "."
		if strings.HasPrefix(name, prefix) && entry.IsDir() {
			// Extract the index
			indexStr := strings.TrimPrefix(name, prefix)
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				continue
			}

			// Parse the alternate mode description
			altModePath := filepath.Join(partnerDir, name)
			description := readFile(filepath.Join(altModePath, "description"))

			alternateModes = append(alternateModes, model.AlternateMode{
				Index:       index,
				Description: description,
				SVID:        readFile(filepath.Join(altModePath, "svid")),
				VDO:         readFile(filepath.Join(altModePath, "vdo")),
				Active:      readFile(filepath.Join(altModePath, "active")),
			})
		}
	}

	return alternateModes
}
