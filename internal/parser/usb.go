package parser

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/cpulvermacher/lsusbc/internal/model"
)

// LoadStandaloneUSBDevices returns USB devices from bus/usb/devices that are not
// already reachable through any typec partner.
func LoadStandaloneUSBDevices(sysfsPath string, ports []model.Port) []model.USBDevice {
	claimed := make(map[string]bool)
	for _, port := range ports {
		if port.Partner != nil {
			collectUSBDeviceIDs(claimed, port.Partner.USBDevices)
		}
	}

	usbDevicesPath := filepath.Join(sysfsPath, "bus/usb/devices")
	entries, err := os.ReadDir(usbDevicesPath)
	if err != nil {
		return nil
	}

	var devices []model.USBDevice
	for _, entry := range entries {
		name := entry.Name()
		if !isUSBDeviceID(name) || claimed[name] {
			continue
		}
		// Only root-level devices: "N-M" with no dots in the port part
		parts := strings.SplitN(name, "-", 2)
		if len(parts) == 2 && strings.Contains(parts[1], ".") {
			continue
		}

		path, err := filepath.EvalSymlinks(filepath.Join(usbDevicesPath, name))
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
			MaxPower:     readFile(filepath.Join(path, "bMaxPower")),
			Drivers:      readDrivers(path),
			USBDevices:   subDevices,
		})
	}
	return devices
}

func collectUSBDeviceIDs(claimed map[string]bool, devices []model.USBDevice) {
	for _, d := range devices {
		claimed[d.DeviceID] = true
		collectUSBDeviceIDs(claimed, d.USBDevices)
	}
}

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
			MaxPower:     readFile(filepath.Join(path, "bMaxPower")),
			Drivers:      readDrivers(path),
			USBDevices:   subDevices,
		})
	}
	return devices
}

// readDrivers returns unique driver names from interface subdirectories (e.g., "1-4:1.0/driver").
// The "usb" driver is excluded as it's the generic hub/root driver and not informative.
func readDrivers(devicePath string) []string {
	entries, err := os.ReadDir(devicePath)
	if err != nil {
		return nil
	}

	seen := make(map[string]bool)
	var drivers []string
	for _, entry := range entries {
		if !strings.Contains(entry.Name(), ":") {
			continue
		}
		link, err := os.Readlink(filepath.Join(devicePath, entry.Name(), "driver"))
		if err != nil {
			continue
		}
		name := filepath.Base(link)
		if name == "usb" || seen[name] {
			continue
		}
		seen[name] = true
		drivers = append(drivers, name)
	}
	return drivers
}

// isUSBDeviceID returns true for USB device IDs like "1-4" or "2-1.3"
func isUSBDeviceID(name string) bool {
	return len(name) > 0 &&
		name[0] >= '0' && name[0] <= '9' &&
		strings.Contains(name, "-") &&
		!strings.Contains(name, ":")
}
