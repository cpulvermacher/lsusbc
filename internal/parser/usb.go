package parser

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/cpulvermacher/lsusbc/internal/model"
)

// USB Devices
//
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
				Speed:        strings.TrimSpace(readFile(filepath.Join(devicePath, "speed"))),
				Version:      strings.TrimSpace(readFile(filepath.Join(devicePath, "version"))),
			})
		}
	}

	partner.USBDevices = devices
}
