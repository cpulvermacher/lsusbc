package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cpulvermacher/lsusbc/internal/model"
)

func makeUSBDevice(t *testing.T, usbDir, id string, files map[string]string) string {
	t.Helper()
	dir := filepath.Join(usbDir, id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestParseUSBDeviceInfo(t *testing.T) {
	partnerDir := t.TempDir()
	usbDir := t.TempDir()

	devicePath := makeUSBDevice(t, usbDir, "1-4", map[string]string{
		"manufacturer": "Acme Corp\n",
		"product":      "Widget\n",
		"serial":       "SN123\n",
		"idVendor":     "1234\n",
		"idProduct":    "5678\n",
		"speed":        "480\n",
		"version":      "2.00\n",
		"bMaxPower":    "500mA\n",
	})
	if err := os.Symlink(devicePath, filepath.Join(partnerDir, "1-4")); err != nil {
		t.Fatal(err)
	}

	devices := parseUSBDeviceInfo(partnerDir)

	if len(devices) != 1 {
		t.Fatalf("got %d USB devices, want 1", len(devices))
	}
	dev := devices[0]
	if dev.DeviceID != "1-4" {
		t.Errorf("DeviceID = %q, want %q", dev.DeviceID, "1-4")
	}
	if dev.Manufacturer != "Acme Corp" {
		t.Errorf("Manufacturer = %q, want %q", dev.Manufacturer, "Acme Corp")
	}
	if dev.Product != "Widget" {
		t.Errorf("Product = %q, want %q", dev.Product, "Widget")
	}
	if dev.Serial != "SN123" {
		t.Errorf("Serial = %q, want %q", dev.Serial, "SN123")
	}
	if dev.IDVendor != "1234" {
		t.Errorf("IDVendor = %q, want %q", dev.IDVendor, "1234")
	}
	if dev.Speed != "480" {
		t.Errorf("Speed = %q, want %q", dev.Speed, "480")
	}
	if dev.MaxPower != "500mA" {
		t.Errorf("MaxPower = %q, want %q", dev.MaxPower, "500mA")
	}
}

func TestParseUSBDeviceInfo_NoManufacturerOrProduct(t *testing.T) {
	partnerDir := t.TempDir()
	usbDir := t.TempDir()

	devicePath := makeUSBDevice(t, usbDir, "1-4", map[string]string{
		"idVendor":  "1234\n",
		"idProduct": "5678\n",
	})
	if err := os.Symlink(devicePath, filepath.Join(partnerDir, "1-4")); err != nil {
		t.Fatal(err)
	}

	devices := parseUSBDeviceInfo(partnerDir)

	if len(devices) != 0 {
		t.Errorf("got %d USB devices, want 0 (no manufacturer/product)", len(devices))
	}
}

func TestParseUSBDeviceInfo_SkipsInterfaces(t *testing.T) {
	partnerDir := t.TempDir()

	// Interface entries (contain colon) should be ignored
	for _, name := range []string{"1-4:1.0", "1-4:1.1"} {
		if err := os.MkdirAll(filepath.Join(partnerDir, name), 0755); err != nil {
			t.Fatal(err)
		}
	}

	devices := parseUSBDeviceInfo(partnerDir)

	if len(devices) != 0 {
		t.Errorf("got %d USB devices, want 0 (interfaces should be skipped)", len(devices))
	}
}

func TestParseUSBDeviceInfo_SkipsPortDirs(t *testing.T) {
	partnerDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(partnerDir, "port0-partner"), 0755); err != nil {
		t.Fatal(err)
	}

	devices := parseUSBDeviceInfo(partnerDir)

	if len(devices) != 0 {
		t.Errorf("got %d USB devices, want 0 (port dirs should be skipped)", len(devices))
	}
}

func TestParseUSBDeviceInfo_BrokenSymlink(t *testing.T) {
	partnerDir := t.TempDir()

	// Entry exists in partner dir but not in usbDir
	if err := os.Symlink("/nonexistent/path", filepath.Join(partnerDir, "1-4")); err != nil {
		t.Fatal(err)
	}

	devices := parseUSBDeviceInfo(partnerDir)

	if len(devices) != 0 {
		t.Errorf("got %d USB devices, want 0 (broken symlink should be skipped)", len(devices))
	}
}

func TestParseUSBDeviceInfo_MultipleDevices(t *testing.T) {
	partnerDir := t.TempDir()
	usbDir := t.TempDir()

	for _, id := range []string{"1-4", "2-1.3"} {
		devicePath := makeUSBDevice(t, usbDir, id, map[string]string{
			"manufacturer": "Vendor",
			"product":      "Device " + id,
		})
		if err := os.Symlink(devicePath, filepath.Join(partnerDir, id)); err != nil {
			t.Fatal(err)
		}
	}

	devices := parseUSBDeviceInfo(partnerDir)

	if len(devices) != 2 {
		t.Errorf("got %d USB devices, want 2", len(devices))
	}
}

func TestParseUSBDeviceInfo_DeepNesting(t *testing.T) {
	partnerDir := t.TempDir()
	usbDir := t.TempDir()

	hubPath := makeUSBDevice(t, usbDir, "1-7", map[string]string{})
	subPath := makeUSBDevice(t, hubPath, "1-7.1", map[string]string{})
	makeUSBDevice(t, subPath, "1-7.1.1", map[string]string{
		"manufacturer": "Acme\n",
		"product":      "Gadget\n",
	})

	if err := os.Symlink(hubPath, filepath.Join(partnerDir, "1-7")); err != nil {
		t.Fatal(err)
	}

	devices := parseUSBDeviceInfo(partnerDir)

	if len(devices) != 1 {
		t.Fatalf("got %d top-level devices, want 1", len(devices))
	}
	if len(devices[0].USBDevices) != 1 {
		t.Fatalf("got %d sub-devices, want 1", len(devices[0].USBDevices))
	}
	sub := devices[0].USBDevices[0]
	if len(sub.USBDevices) != 1 {
		t.Fatalf("got %d sub-sub-devices, want 1", len(sub.USBDevices))
	}
	if sub.USBDevices[0].DeviceID != "1-7.1.1" {
		t.Errorf("DeviceID = %q, want %q", sub.USBDevices[0].DeviceID, "1-7.1.1")
	}
}

func TestParseUSBDeviceInfo_WithSubDevices(t *testing.T) {
	partnerDir := t.TempDir()
	usbDir := t.TempDir()

	// Hub with no manufacturer/product of its own
	hubPath := makeUSBDevice(t, usbDir, "1-7", map[string]string{})
	// Sub-device 1-7.1 with product info
	makeUSBDevice(t, hubPath, "1-7.1", map[string]string{
		"manufacturer": "Jabra\n",
		"product":      "Headset\n",
	})
	// Sub-device 1-7.2 with no product info — should be excluded
	makeUSBDevice(t, hubPath, "1-7.2", map[string]string{})
	// Interface entry — should be ignored
	makeUSBDevice(t, hubPath, "1-7:1.0", map[string]string{
		"manufacturer": "ignored\n",
	})

	if err := os.Symlink(hubPath, filepath.Join(partnerDir, "1-7")); err != nil {
		t.Fatal(err)
	}

	devices := parseUSBDeviceInfo(partnerDir)

	if len(devices) != 1 {
		t.Fatalf("got %d top-level devices, want 1", len(devices))
	}
	hub := devices[0]
	if hub.DeviceID != "1-7" {
		t.Errorf("DeviceID = %q, want %q", hub.DeviceID, "1-7")
	}
	if len(hub.USBDevices) != 1 {
		t.Fatalf("got %d sub-devices, want 1", len(hub.USBDevices))
	}
	sub := hub.USBDevices[0]
	if sub.DeviceID != "1-7.1" {
		t.Errorf("sub DeviceID = %q, want %q", sub.DeviceID, "1-7.1")
	}
	if sub.Manufacturer != "Jabra" {
		t.Errorf("sub Manufacturer = %q, want %q", sub.Manufacturer, "Jabra")
	}
	if sub.Product != "Headset" {
		t.Errorf("sub Product = %q, want %q", sub.Product, "Headset")
	}
}

func TestReadDrivers(t *testing.T) {
	deviceDir := t.TempDir()

	for _, iface := range []struct{ name, driver string }{
		{"1-4:1.0", "usbhid"},
		{"1-4:1.1", "usbhid"}, // duplicate — should appear once
		{"1-4:1.2", "usb"},    // should be filtered out
	} {
		ifaceDir := filepath.Join(deviceDir, iface.name)
		if err := os.MkdirAll(ifaceDir, 0755); err != nil {
			t.Fatal(err)
		}
		target := "../../../../../../bus/usb/drivers/" + iface.driver
		if err := os.Symlink(target, filepath.Join(ifaceDir, "driver")); err != nil {
			t.Fatal(err)
		}
	}

	drivers := readDrivers(deviceDir)

	if len(drivers) != 1 || drivers[0] != "usbhid" {
		t.Errorf("got %v, want [usbhid]", drivers)
	}
}

func TestReadDrivers_NoInterfaces(t *testing.T) {
	deviceDir := t.TempDir()
	drivers := readDrivers(deviceDir)
	if len(drivers) != 0 {
		t.Errorf("got %v, want empty", drivers)
	}
}

func TestParseUSBDeviceInfo_Driver(t *testing.T) {
	partnerDir := t.TempDir()
	usbDir := t.TempDir()

	devicePath := makeUSBDevice(t, usbDir, "1-4", map[string]string{
		"manufacturer": "Acme\n",
		"product":      "Widget\n",
	})
	ifaceDir := filepath.Join(devicePath, "1-4:1.0")
	if err := os.MkdirAll(ifaceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../../../../../bus/usb/drivers/usb-storage", filepath.Join(ifaceDir, "driver")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(devicePath, filepath.Join(partnerDir, "1-4")); err != nil {
		t.Fatal(err)
	}

	devices := parseUSBDeviceInfo(partnerDir)

	if len(devices) != 1 {
		t.Fatalf("got %d devices, want 1", len(devices))
	}
	if len(devices[0].Drivers) != 1 || devices[0].Drivers[0] != "usb-storage" {
		t.Errorf("Drivers = %v, want [usb-storage]", devices[0].Drivers)
	}
}

func makeUSBBusDir(t *testing.T, sysfsRoot string) string {
	t.Helper()
	dir := filepath.Join(sysfsRoot, "bus/usb/devices")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLoadStandaloneUSBDevices_Basic(t *testing.T) {
	sysfsRoot := t.TempDir()
	busDir := makeUSBBusDir(t, sysfsRoot)
	usbDir := t.TempDir()

	devicePath := makeUSBDevice(t, usbDir, "1-1", map[string]string{
		"manufacturer": "Acme\n",
		"product":      "Widget\n",
	})
	if err := os.Symlink(devicePath, filepath.Join(busDir, "1-1")); err != nil {
		t.Fatal(err)
	}

	devices := LoadStandaloneUSBDevices(sysfsRoot, nil)

	if len(devices) != 1 {
		t.Fatalf("got %d devices, want 1", len(devices))
	}
	if devices[0].DeviceID != "1-1" {
		t.Errorf("DeviceID = %q, want 1-1", devices[0].DeviceID)
	}
	if devices[0].Product != "Widget" {
		t.Errorf("Product = %q, want Widget", devices[0].Product)
	}
}

func TestLoadStandaloneUSBDevices_ExcludesClaimed(t *testing.T) {
	sysfsRoot := t.TempDir()
	busDir := makeUSBBusDir(t, sysfsRoot)
	usbDir := t.TempDir()

	for _, id := range []string{"1-1", "1-2"} {
		devicePath := makeUSBDevice(t, usbDir, id, map[string]string{
			"manufacturer": "Vendor\n",
			"product":      "Device " + id + "\n",
		})
		if err := os.Symlink(devicePath, filepath.Join(busDir, id)); err != nil {
			t.Fatal(err)
		}
	}

	// Mark 1-1 as claimed by a typec partner
	ports := []model.Port{
		{Partner: &model.Partner{
			USBDevices: []model.USBDevice{{DeviceID: "1-1"}},
		}},
	}

	devices := LoadStandaloneUSBDevices(sysfsRoot, ports)

	if len(devices) != 1 {
		t.Fatalf("got %d devices, want 1 (1-1 should be excluded)", len(devices))
	}
	if devices[0].DeviceID != "1-2" {
		t.Errorf("DeviceID = %q, want 1-2", devices[0].DeviceID)
	}
}

func TestLoadStandaloneUSBDevices_SkipsSubDevices(t *testing.T) {
	sysfsRoot := t.TempDir()
	busDir := makeUSBBusDir(t, sysfsRoot)
	usbDir := t.TempDir()

	// 1-1 is a hub with a sub-device 1-1.2
	hubPath := makeUSBDevice(t, usbDir, "1-1", map[string]string{"product": "Hub\n"})
	makeUSBDevice(t, hubPath, "1-1.2", map[string]string{"product": "Sub\n"})

	if err := os.Symlink(hubPath, filepath.Join(busDir, "1-1")); err != nil {
		t.Fatal(err)
	}
	// Also add 1-1.2 as a flat entry in bus/usb/devices (as Linux does)
	subPath := filepath.Join(hubPath, "1-1.2")
	if err := os.Symlink(subPath, filepath.Join(busDir, "1-1.2")); err != nil {
		t.Fatal(err)
	}

	devices := LoadStandaloneUSBDevices(sysfsRoot, nil)

	if len(devices) != 1 {
		t.Fatalf("got %d top-level devices, want 1 (1-1.2 should not appear as top-level)", len(devices))
	}
	if devices[0].DeviceID != "1-1" {
		t.Errorf("DeviceID = %q, want 1-1", devices[0].DeviceID)
	}
	if len(devices[0].USBDevices) != 1 {
		t.Errorf("got %d sub-devices, want 1", len(devices[0].USBDevices))
	}
}

func TestLoadStandaloneUSBDevices_NoBusDir(t *testing.T) {
	sysfsRoot := t.TempDir()
	// No bus/usb/devices directory — should return nil gracefully
	devices := LoadStandaloneUSBDevices(sysfsRoot, nil)
	if devices != nil {
		t.Errorf("expected nil, got %v", devices)
	}
}
