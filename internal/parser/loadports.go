package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/christian/usb-c/internal/model"
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
	port.PowerOperationMode = strings.TrimSpace(readFile(filepath.Join(portDir, "power_operation_mode")))

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
	partner.AccessoryMode = strings.TrimSpace(readFile(filepath.Join(partnerDir, "accessory_mode")))

	// Parse all alternate modes
	partner.AlternateModes = parseAlternateModes(partnerDir)

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

	// Try to find and parse USB device information
	parseUSBDeviceInfo(partner, partnerDir)

	return partner, nil
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
			description := strings.TrimSpace(readFile(filepath.Join(altModePath, "description")))

			alternateModes = append(alternateModes, model.AlternateMode{
				Index:       index,
				Description: description,
				SVID:        strings.TrimSpace(readFile(filepath.Join(altModePath, "svid"))),
				VDO:         strings.TrimSpace(readFile(filepath.Join(altModePath, "vdo"))),
				Active:      strings.TrimSpace(readFile(filepath.Join(altModePath, "active"))),
			})
		}
	}

	return alternateModes
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

// parseUSBDeviceInfo attempts to find and parse USB device information
// by looking for device symlinks (like "1-4", "2-1") in the partner directory
func parseUSBDeviceInfo(partner *model.Partner, partnerDir string) {
	// Scan the partner directory for entries matching USB device patterns
	entries, err := os.ReadDir(partnerDir)
	if err != nil {
		return
	}

	var devices []model.USBDevice

	for _, entry := range entries {
		name := entry.Name()

		// Match USB device patterns like "1-4" or "2-1.3"
		// Must contain a dash, must not contain a colon (that's an interface like "1-4:1.0")
		// Must not start with "port" (that's a port directory)
		if !strings.Contains(name, "-") || strings.Contains(name, ":") || strings.HasPrefix(name, "port") {
			continue
		}

		// Check if it looks like a USB device ID (starts with a digit)
		if len(name) == 0 || name[0] < '0' || name[0] > '9' {
			continue
		}

		// Found a potential USB device link
		// Resolve the device path
		var devicePath string
		if strings.HasPrefix(partnerDir, "/sys/class/typec") {
			// Live system: resolve symlink to get actual path, then map to /sys/bus/usb/devices
			devicePath = filepath.Join("/sys/bus/usb/devices", name)
		} else {
			// Snapshot: the device directory should be captured in the snapshot
			// Look for it in the snapshot's usb/devices directory
			parts := strings.Split(partnerDir, string(filepath.Separator))
			var snapshotRoot string
			for i := len(parts) - 1; i >= 0; i-- {
				if !strings.HasPrefix(parts[i], "port") {
					snapshotRoot = filepath.Join(parts[:i+1]...)
					break
				}
			}
			if snapshotRoot != "" {
				devicePath = filepath.Join(snapshotRoot, "usb", "devices", name)
			} else {
				continue
			}
		}

		// Check if the device path exists
		if _, err := os.Stat(devicePath); err != nil {
			continue
		}

		// Parse device information
		manufacturer := strings.TrimSpace(readFile(filepath.Join(devicePath, "manufacturer")))
		product := strings.TrimSpace(readFile(filepath.Join(devicePath, "product")))

		// Only add if we have some useful information
		if manufacturer != "" || product != "" {
			devices = append(devices, model.USBDevice{
				DeviceID:     name,
				Manufacturer: manufacturer,
				Product:      product,
				Serial:       strings.TrimSpace(readFile(filepath.Join(devicePath, "serial"))),
				IDVendor:     strings.TrimSpace(readFile(filepath.Join(devicePath, "idVendor"))),
				IDProduct:    strings.TrimSpace(readFile(filepath.Join(devicePath, "idProduct"))),
			})
		}
	}

	partner.USBDevices = devices
}

// readFile reads a file and returns its content, or empty string on error
func readFile(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(content)
}
