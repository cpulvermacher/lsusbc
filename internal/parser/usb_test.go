package parser

import (
	"os"
	"path/filepath"
	"testing"
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
