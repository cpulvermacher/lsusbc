package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func makeUSBDevice(t *testing.T, usbDir, id string, files map[string]string) {
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
}

func TestParseUSBDeviceInfo(t *testing.T) {
	partnerDir := t.TempDir()
	usbDir := t.TempDir()

	makeUSBDevice(t, usbDir, "1-4", map[string]string{
		"manufacturer": "Acme Corp\n",
		"product":      "Widget\n",
		"serial":       "SN123\n",
		"idVendor":     "1234\n",
		"idProduct":    "5678\n",
		"speed":        "480\n",
		"version":      "2.00\n",
	})

	if err := os.MkdirAll(filepath.Join(partnerDir, "1-4"), 0755); err != nil {
		t.Fatal(err)
	}

	devices := parseUSBDeviceInfoFrom(partnerDir, usbDir)

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

	makeUSBDevice(t, usbDir, "1-4", map[string]string{
		"idVendor":  "1234\n",
		"idProduct": "5678\n",
	})
	if err := os.MkdirAll(filepath.Join(partnerDir, "1-4"), 0755); err != nil {
		t.Fatal(err)
	}

	devices := parseUSBDeviceInfoFrom(partnerDir, usbDir)

	if len(devices) != 0 {
		t.Errorf("got %d USB devices, want 0 (no manufacturer/product)", len(devices))
	}
}

func TestParseUSBDeviceInfo_SkipsInterfaces(t *testing.T) {
	partnerDir := t.TempDir()
	usbDir := t.TempDir()

	// Interface entries (contain colon) should be ignored
	for _, name := range []string{"1-4:1.0", "1-4:1.1"} {
		if err := os.MkdirAll(filepath.Join(partnerDir, name), 0755); err != nil {
			t.Fatal(err)
		}
	}

	devices := parseUSBDeviceInfoFrom(partnerDir, usbDir)

	if len(devices) != 0 {
		t.Errorf("got %d USB devices, want 0 (interfaces should be skipped)", len(devices))
	}
}

func TestParseUSBDeviceInfo_SkipsPortDirs(t *testing.T) {
	partnerDir := t.TempDir()
	usbDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(partnerDir, "port0-partner"), 0755); err != nil {
		t.Fatal(err)
	}

	devices := parseUSBDeviceInfoFrom(partnerDir, usbDir)

	if len(devices) != 0 {
		t.Errorf("got %d USB devices, want 0 (port dirs should be skipped)", len(devices))
	}
}

func TestParseUSBDeviceInfo_DeviceNotInUSBDir(t *testing.T) {
	partnerDir := t.TempDir()
	usbDir := t.TempDir()

	// Entry exists in partner dir but not in usbDir
	if err := os.MkdirAll(filepath.Join(partnerDir, "1-4"), 0755); err != nil {
		t.Fatal(err)
	}

	devices := parseUSBDeviceInfoFrom(partnerDir, usbDir)

	if len(devices) != 0 {
		t.Errorf("got %d USB devices, want 0 (device absent from usb dir)", len(devices))
	}
}

func TestParseUSBDeviceInfo_MultipleDevices(t *testing.T) {
	partnerDir := t.TempDir()
	usbDir := t.TempDir()

	for _, id := range []string{"1-4", "2-1.3"} {
		makeUSBDevice(t, usbDir, id, map[string]string{
			"manufacturer": "Vendor",
			"product":      "Device " + id,
		})
		if err := os.MkdirAll(filepath.Join(partnerDir, id), 0755); err != nil {
			t.Fatal(err)
		}
	}

	devices := parseUSBDeviceInfoFrom(partnerDir, usbDir)

	if len(devices) != 2 {
		t.Errorf("got %d USB devices, want 2", len(devices))
	}
}
