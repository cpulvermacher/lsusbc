package parser

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/cpulvermacher/lsusbc/internal/model"
)

// parseUSBDeviceInfo attempts to find and parse USB device information
// by looking for device symlinks (like "1-4", "2-1") in the partner directory
// Note that depending on hardware, these might just not exist
func parseUSBDeviceInfo(partnerDir string) []model.USBDevice {
	return parseUSBDeviceInfoFrom(partnerDir, "/sys/bus/usb/devices")
}

func parseUSBDeviceInfoFrom(partnerDir string, usbDevicesDir string) []model.USBDevice {
	entries, err := os.ReadDir(partnerDir)
	if err != nil {
		return nil
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

		devicePath := filepath.Join(usbDevicesDir, name)
		if _, err := os.Stat(devicePath); err != nil {
			continue
		}

		manufacturer := readFile(filepath.Join(devicePath, "manufacturer"))
		product := readFile(filepath.Join(devicePath, "product"))

		if manufacturer != "" || product != "" {
			devices = append(devices, model.USBDevice{
				DeviceID:     name,
				Manufacturer: manufacturer,
				Product:      product,
				Serial:       readFile(filepath.Join(devicePath, "serial")),
				IDVendor:     readFile(filepath.Join(devicePath, "idVendor")),
				IDProduct:    readFile(filepath.Join(devicePath, "idProduct")),
				Speed:        readFile(filepath.Join(devicePath, "speed")),
				Version:      readFile(filepath.Join(devicePath, "version")),
			})
		}
	}

	return devices
}
