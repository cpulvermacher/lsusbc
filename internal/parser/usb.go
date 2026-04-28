package parser

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/cpulvermacher/lsusbc/internal/model"
)

// parseUSBDeviceInfo scans dir for USB device entries (symlinks or subdirectories)
// matching the "1-4" / "2-1.3" pattern, reading device info and recursing into sub-devices.
func parseUSBDeviceInfo(dir string) []model.USBDevice {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var devices []model.USBDevice
	for _, entry := range entries {
		name := entry.Name()
		if !isUSBDeviceID(name) {
			continue
		}

		path, err := filepath.EvalSymlinks(filepath.Join(dir, name))
		if err != nil {
			continue
		}

		manufacturer := readFile(filepath.Join(path, "manufacturer"))
		product := readFile(filepath.Join(path, "product"))
		subDevices := parseUSBDeviceInfo(path)

		if manufacturer == "" && product == "" && len(subDevices) == 0 {
			continue
		}
		devices = append(devices, model.USBDevice{
			DeviceID:     name,
			Manufacturer: manufacturer,
			Product:      product,
			Serial:       readFile(filepath.Join(path, "serial")),
			IDVendor:     readFile(filepath.Join(path, "idVendor")),
			IDProduct:    readFile(filepath.Join(path, "idProduct")),
			Speed:        readFile(filepath.Join(path, "speed")),
			Version:      readFile(filepath.Join(path, "version")),
			USBDevices:   subDevices,
		})
	}
	return devices
}

// isUSBDeviceID returns true for USB device IDs like "1-4" or "2-1.3"
func isUSBDeviceID(name string) bool {
	return len(name) > 0 &&
		name[0] >= '0' && name[0] <= '9' &&
		strings.Contains(name, "-") &&
		!strings.Contains(name, ":")
}
