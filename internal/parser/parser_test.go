package parser

import (
	"os"
	"path/filepath"
	"testing"
)

// makeFile creates a file with the given content, creating parent dirs as needed.
func makeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// makeDir creates a directory and returns its path.
func makeDir(t *testing.T, parent, name string) string {
	t.Helper()
	dir := filepath.Join(parent, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// --- parseAlternateModes ---

func TestParseAlternateModes_Empty(t *testing.T) {
	partnerDir := makeDir(t, t.TempDir(), "port0-partner")
	modes := parseAlternateModes(partnerDir)
	if len(modes) != 0 {
		t.Errorf("got %d modes, want 0", len(modes))
	}
}

func TestParseAlternateModes_OneMode(t *testing.T) {
	partnerDir := makeDir(t, t.TempDir(), "port0-partner")
	altDir := makeDir(t, partnerDir, "port0-partner.0")
	makeFile(t, altDir, "description", "DisplayPort\n")
	makeFile(t, altDir, "svid", "ff01\n")
	makeFile(t, altDir, "vdo", "0x001c0c05\n")
	makeFile(t, altDir, "active", "yes\n")

	modes := parseAlternateModes(partnerDir)

	if len(modes) != 1 {
		t.Fatalf("got %d modes, want 1", len(modes))
	}
	m := modes[0]
	if m.Index != 0 {
		t.Errorf("Index = %d, want 0", m.Index)
	}
	if m.Description != "DisplayPort" {
		t.Errorf("Description = %q, want %q", m.Description, "DisplayPort")
	}
	if m.SVID != "ff01" {
		t.Errorf("SVID = %q, want %q", m.SVID, "ff01")
	}
	if m.Active != "yes" {
		t.Errorf("Active = %q, want %q", m.Active, "yes")
	}
}

func TestParseAlternateModes_MultipleModes(t *testing.T) {
	partnerDir := makeDir(t, t.TempDir(), "port0-partner")
	for _, name := range []string{"port0-partner.0", "port0-partner.1", "port0-partner.2"} {
		d := makeDir(t, partnerDir, name)
		makeFile(t, d, "description", "Mode "+name+"\n")
	}

	modes := parseAlternateModes(partnerDir)

	if len(modes) != 3 {
		t.Errorf("got %d modes, want 3", len(modes))
	}
}

func TestParseAlternateModes_SkipsNonMatchingDirs(t *testing.T) {
	partnerDir := makeDir(t, t.TempDir(), "port0-partner")
	makeDir(t, partnerDir, "pd0")
	makeDir(t, partnerDir, "port0-partner.0") // valid
	makeFile(t, partnerDir, "accessory_mode", "none\n")

	modes := parseAlternateModes(partnerDir)

	if len(modes) != 1 {
		t.Errorf("got %d modes, want 1 (non-matching dirs should be skipped)", len(modes))
	}
}

func TestParseAlternateModes_SkipsFiles(t *testing.T) {
	partnerDir := makeDir(t, t.TempDir(), "port0-partner")
	makeFile(t, partnerDir, "port0-partner.0", "not a dir\n") // file, not dir

	modes := parseAlternateModes(partnerDir)

	if len(modes) != 0 {
		t.Errorf("got %d modes, want 0 (files should be skipped)", len(modes))
	}
}

func TestParseAlternateModes_SkipsNonDigitSuffix(t *testing.T) {
	partnerDir := makeDir(t, t.TempDir(), "port0-partner")
	makeDir(t, partnerDir, "port0-partner.abc")

	modes := parseAlternateModes(partnerDir)

	if len(modes) != 0 {
		t.Errorf("got %d modes, want 0 (non-digit suffix should be skipped)", len(modes))
	}
}

// --- parseCable ---

func TestParseCable_Basic(t *testing.T) {
	typecDir := t.TempDir()
	portDir := makeDir(t, typecDir, "port0")
	cableDir := makeDir(t, typecDir, "port0-cable")
	makeFile(t, cableDir, "type", "passive\n")
	makeFile(t, cableDir, "plug_type", "type-c\n")

	cable, err := parseCable(cableDir, portDir)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cable.Type != "passive" {
		t.Errorf("Type = %q, want %q", cable.Type, "passive")
	}
	if cable.PlugType != "type-c" {
		t.Errorf("PlugType = %q, want %q", cable.PlugType, "type-c")
	}
	if len(cable.AlternateModes) != 0 {
		t.Errorf("got %d alternate modes, want 0", len(cable.AlternateModes))
	}
}

func TestParseCable_WithPlugAlternateModes(t *testing.T) {
	typecDir := t.TempDir()
	portDir := makeDir(t, typecDir, "port0")
	cableDir := makeDir(t, typecDir, "port0-cable")
	makeFile(t, cableDir, "type", "active\n")
	makeFile(t, cableDir, "plug_type", "type-c\n")

	plugDir := makeDir(t, typecDir, "port0-plug0")
	altDir := makeDir(t, plugDir, "port0-plug0.0")
	makeFile(t, altDir, "description", "DisplayPort\n")
	makeFile(t, altDir, "svid", "ff01\n")
	makeFile(t, altDir, "vdo", "0x0\n")
	makeFile(t, altDir, "active", "yes\n")

	cable, err := parseCable(cableDir, portDir)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cable.AlternateModes) != 1 {
		t.Fatalf("got %d alternate modes, want 1", len(cable.AlternateModes))
	}
	if cable.AlternateModes[0].Description != "DisplayPort" {
		t.Errorf("Description = %q, want %q", cable.AlternateModes[0].Description, "DisplayPort")
	}
}

func TestParseCable_MultiplePlugs(t *testing.T) {
	typecDir := t.TempDir()
	portDir := makeDir(t, typecDir, "port0")
	cableDir := makeDir(t, typecDir, "port0-cable")
	makeFile(t, cableDir, "type", "active\n")

	for _, plug := range []string{"port0-plug0", "port0-plug1"} {
		plugDir := makeDir(t, typecDir, plug)
		altDir := makeDir(t, plugDir, plug+".0")
		makeFile(t, altDir, "description", "Thunderbolt\n")
		makeFile(t, altDir, "svid", "8087\n")
		makeFile(t, altDir, "vdo", "0x0\n")
		makeFile(t, altDir, "active", "no\n")
	}

	cable, err := parseCable(cableDir, portDir)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cable.AlternateModes) != 2 {
		t.Errorf("got %d alternate modes, want 2 (one per plug)", len(cable.AlternateModes))
	}
}

// --- parsePartner ---

func TestParsePartner_Basic(t *testing.T) {
	partnerDir := makeDir(t, t.TempDir(), "port0-partner")
	makeFile(t, partnerDir, "accessory_mode", "none\n")

	partner, err := parsePartner(partnerDir)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if partner.Name != "port0-partner" {
		t.Errorf("Name = %q, want %q", partner.Name, "port0-partner")
	}
	if partner.AccessoryMode != "none" {
		t.Errorf("AccessoryMode = %q, want %q", partner.AccessoryMode, "none")
	}
	if partner.PowerDelivery != nil {
		t.Error("PowerDelivery should be nil when no pd dirs present")
	}
}

func TestParsePartner_WithAlternateModes(t *testing.T) {
	partnerDir := makeDir(t, t.TempDir(), "port0-partner")
	altDir := makeDir(t, partnerDir, "port0-partner.0")
	makeFile(t, altDir, "description", "DisplayPort\n")
	makeFile(t, altDir, "svid", "ff01\n")
	makeFile(t, altDir, "vdo", "0x0\n")
	makeFile(t, altDir, "active", "yes\n")

	partner, err := parsePartner(partnerDir)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(partner.AlternateModes) != 1 {
		t.Errorf("got %d alternate modes, want 1", len(partner.AlternateModes))
	}
}

func TestParsePartner_WithPD(t *testing.T) {
	partnerDir := makeDir(t, t.TempDir(), "port0-partner")
	pdDir := makeDir(t, partnerDir, "pd0")
	makeFile(t, pdDir, "revision", "3.0\n")
	makeFile(t, pdDir, "source-capabilities/1:fixed_supply/voltage", "20000mV\n")
	makeFile(t, pdDir, "source-capabilities/1:fixed_supply/maximum_current", "5000mA\n")

	partner, err := parsePartner(partnerDir)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if partner.PowerDelivery == nil {
		t.Fatal("PowerDelivery is nil, want non-nil")
	}
	if partner.PowerDelivery.Revision != "3.0" {
		t.Errorf("PD Revision = %q, want %q", partner.PowerDelivery.Revision, "3.0")
	}
}

// --- parsePort ---

func TestParsePort_Basic(t *testing.T) {
	typecDir := t.TempDir()
	portDir := makeDir(t, typecDir, "port0")
	makeFile(t, portDir, "data_role", "[host] device\n")
	makeFile(t, portDir, "power_role", "[source] sink\n")
	makeFile(t, portDir, "power_operation_mode", "default\n")

	port, err := parsePort(portDir)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port.Name != "port0" {
		t.Errorf("Name = %q, want %q", port.Name, "port0")
	}
	if port.DataRole != "host" {
		t.Errorf("DataRole = %q, want %q", port.DataRole, "host")
	}
	if port.PowerRole != "source" {
		t.Errorf("PowerRole = %q, want %q", port.PowerRole, "source")
	}
	if port.PowerOperationMode != "default" {
		t.Errorf("PowerOperationMode = %q, want %q", port.PowerOperationMode, "default")
	}
	if port.Partner != nil {
		t.Error("Partner should be nil when no partner dir exists")
	}
	if port.Cable != nil {
		t.Error("Cable should be nil when no cable dir exists")
	}
}

func TestParsePort_WithPartner(t *testing.T) {
	typecDir := t.TempDir()
	portDir := makeDir(t, typecDir, "port0")
	makeFile(t, portDir, "data_role", "[host] device\n")
	makeFile(t, portDir, "power_role", "[sink] source\n")
	makeFile(t, portDir, "power_operation_mode", "usb_power_delivery\n")

	partnerDir := makeDir(t, typecDir, "port0-partner")
	makeFile(t, partnerDir, "accessory_mode", "none\n")

	port, err := parsePort(portDir)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port.Partner == nil {
		t.Fatal("Partner is nil, want non-nil")
	}
	if port.Partner.Name != "port0-partner" {
		t.Errorf("Partner.Name = %q, want %q", port.Partner.Name, "port0-partner")
	}
}

func TestParsePort_WithCable(t *testing.T) {
	typecDir := t.TempDir()
	portDir := makeDir(t, typecDir, "port0")
	makeFile(t, portDir, "data_role", "[host] device\n")
	makeFile(t, portDir, "power_role", "[source] sink\n")
	makeFile(t, portDir, "power_operation_mode", "default\n")

	cableDir := makeDir(t, typecDir, "port0-cable")
	makeFile(t, cableDir, "type", "passive\n")
	makeFile(t, cableDir, "plug_type", "type-c\n")

	port, err := parsePort(portDir)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port.Cable == nil {
		t.Fatal("Cable is nil, want non-nil")
	}
	if port.Cable.Type != "passive" {
		t.Errorf("Cable.Type = %q, want %q", port.Cable.Type, "passive")
	}
}

// --- LoadPorts ---

func TestLoadPorts_Empty(t *testing.T) {
	sysRoot := t.TempDir()
	makeDir(t, sysRoot, "class/typec")

	ports, err := LoadPorts(sysRoot)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ports == nil {
		t.Error("got nil slice, want non-nil empty slice")
	}
	if len(ports) != 0 {
		t.Errorf("got %d ports, want 0", len(ports))
	}
}

func TestLoadPorts_OnePort(t *testing.T) {
	sysRoot := t.TempDir()
	typecDir := makeDir(t, sysRoot, "class/typec")
	portDir := makeDir(t, typecDir, "port0")
	makeFile(t, portDir, "data_role", "[host] device\n")
	makeFile(t, portDir, "power_role", "[source] sink\n")
	makeFile(t, portDir, "power_operation_mode", "default\n")

	ports, err := LoadPorts(sysRoot)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ports) != 1 {
		t.Fatalf("got %d ports, want 1", len(ports))
	}
	if ports[0].Name != "port0" {
		t.Errorf("Name = %q, want %q", ports[0].Name, "port0")
	}
}

func TestLoadPorts_MultiplePorts(t *testing.T) {
	sysRoot := t.TempDir()
	typecDir := makeDir(t, sysRoot, "class/typec")
	for _, name := range []string{"port0", "port1", "port2"} {
		d := makeDir(t, typecDir, name)
		makeFile(t, d, "data_role", "[host] device\n")
		makeFile(t, d, "power_role", "[source] sink\n")
		makeFile(t, d, "power_operation_mode", "default\n")
	}

	ports, err := LoadPorts(sysRoot)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ports) != 3 {
		t.Errorf("got %d ports, want 3", len(ports))
	}
}

func TestLoadPorts_SkipsSiblingDirs(t *testing.T) {
	sysRoot := t.TempDir()
	typecDir := makeDir(t, sysRoot, "class/typec")
	portDir := makeDir(t, typecDir, "port0")
	makeFile(t, portDir, "data_role", "[host] device\n")
	makeFile(t, portDir, "power_role", "[source] sink\n")
	makeFile(t, portDir, "power_operation_mode", "default\n")

	// These should all be skipped
	makeDir(t, typecDir, "port0-partner")
	makeDir(t, typecDir, "port0-cable")
	makeDir(t, typecDir, "port0-plug0")
	makeDir(t, typecDir, "usb2")

	ports, err := LoadPorts(sysRoot)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ports) != 1 {
		t.Errorf("got %d ports, want 1 (sibling dirs should be skipped)", len(ports))
	}
}

func TestLoadPorts_MissingDir(t *testing.T) {
	_, err := LoadPorts("/nonexistent/path")

	if err == nil {
		t.Error("expected error for missing directory, got nil")
	}
}
