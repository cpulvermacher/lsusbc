package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cpulvermacher/lsusbc/internal/model"
)

func makeCapDir(t *testing.T, capsDir, name string, files map[string]string) {
	t.Helper()
	dir := filepath.Join(capsDir, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for fname, content := range files {
		if err := os.WriteFile(filepath.Join(dir, fname), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestParseCapabilities_FixedSupply(t *testing.T) {
	capsDir := t.TempDir()
	makeCapDir(t, capsDir, "1:fixed_supply", map[string]string{
		"voltage":         "5000mV\n",
		"maximum_current": "3000mA\n",
	})

	caps, err := parseCapabilities(capsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(caps) != 1 {
		t.Fatalf("got %d caps, want 1", len(caps))
	}
	c := caps[0]
	if c.Programmable {
		t.Error("Programmable = true, want false")
	}
	if c.Voltage != 5000 {
		t.Errorf("Voltage = %d, want 5000", c.Voltage)
	}
	if c.MaximumCurrent != 3000 {
		t.Errorf("MaximumCurrent = %d, want 3000", c.MaximumCurrent)
	}
}

func TestParseCapabilities_ProgrammableSupply(t *testing.T) {
	capsDir := t.TempDir()
	makeCapDir(t, capsDir, "2:programmable_supply", map[string]string{
		"minimum_voltage": "3300mV\n",
		"maximum_voltage": "21000mV\n",
		"maximum_current": "5000mA\n",
	})

	caps, err := parseCapabilities(capsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(caps) != 1 {
		t.Fatalf("got %d caps, want 1", len(caps))
	}
	c := caps[0]
	if !c.Programmable {
		t.Error("Programmable = false, want true")
	}
	if c.MinimumVoltage != 3300 {
		t.Errorf("MinimumVoltage = %d, want 3300", c.MinimumVoltage)
	}
	if c.MaximumVoltage != 21000 {
		t.Errorf("MaximumVoltage = %d, want 21000", c.MaximumVoltage)
	}
	if c.MaximumCurrent != 5000 {
		t.Errorf("MaximumCurrent = %d, want 5000", c.MaximumCurrent)
	}
}

func TestParseCapabilities_VariableSupply(t *testing.T) {
	capsDir := t.TempDir()
	makeCapDir(t, capsDir, "3:variable_supply", map[string]string{
		"minimum_voltage": "4750mV\n",
		"maximum_voltage": "5000mV\n",
		"maximum_current": "1500mA\n",
	})

	caps, err := parseCapabilities(capsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(caps) != 1 {
		t.Fatalf("got %d caps, want 1", len(caps))
	}
	c := caps[0]
	if !c.Programmable {
		t.Error("Programmable = false, want true for variable_supply")
	}
}

func TestParseCapabilities_MultipleCaps(t *testing.T) {
	capsDir := t.TempDir()
	makeCapDir(t, capsDir, "1:fixed_supply", map[string]string{
		"voltage":         "5000mV\n",
		"maximum_current": "3000mA\n",
	})
	makeCapDir(t, capsDir, "2:fixed_supply", map[string]string{
		"voltage":         "9000mV\n",
		"maximum_current": "3000mA\n",
	})
	makeCapDir(t, capsDir, "3:fixed_supply", map[string]string{
		"voltage":         "20000mV\n",
		"maximum_current": "5000mA\n",
	})

	caps, err := parseCapabilities(capsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(caps) != 3 {
		t.Errorf("got %d caps, want 3", len(caps))
	}
}

func TestParseCapabilities_SkipsInvalidCurrent(t *testing.T) {
	capsDir := t.TempDir()
	makeCapDir(t, capsDir, "1:fixed_supply", map[string]string{
		"voltage":         "5000mV\n",
		"maximum_current": "bad\n",
	})

	caps, err := parseCapabilities(capsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(caps) != 0 {
		t.Errorf("got %d caps, want 0 (invalid current should be skipped)", len(caps))
	}
}

func TestParseCapabilities_SkipsInvalidVoltage(t *testing.T) {
	capsDir := t.TempDir()
	makeCapDir(t, capsDir, "1:fixed_supply", map[string]string{
		"voltage":         "bad\n",
		"maximum_current": "3000mA\n",
	})

	caps, err := parseCapabilities(capsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(caps) != 0 {
		t.Errorf("got %d caps, want 0 (invalid voltage should be skipped)", len(caps))
	}
}

func TestParseCapabilities_SkipsFiles(t *testing.T) {
	capsDir := t.TempDir()
	// Write a plain file (not a directory) — should be skipped
	if err := os.WriteFile(filepath.Join(capsDir, "not_a_dir"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	makeCapDir(t, capsDir, "1:fixed_supply", map[string]string{
		"voltage":         "5000mV\n",
		"maximum_current": "3000mA\n",
	})

	caps, err := parseCapabilities(capsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(caps) != 1 {
		t.Errorf("got %d caps, want 1 (files should be skipped)", len(caps))
	}
}

func TestParseCapabilities_MissingDir(t *testing.T) {
	_, err := parseCapabilities("/nonexistent/path")
	if err == nil {
		t.Error("expected error for missing directory, got nil")
	}
}

func makePDDir(t *testing.T, partnerDir, pdName string, files map[string]string) string {
	t.Helper()
	pdDir := filepath.Join(partnerDir, pdName)
	if err := os.MkdirAll(pdDir, 0755); err != nil {
		t.Fatal(err)
	}
	for fname, content := range files {
		fullPath := filepath.Join(pdDir, fname)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return pdDir
}

func TestParsePDDirectories_Basic(t *testing.T) {
	partnerDir := t.TempDir()
	makePDDir(t, partnerDir, "pd0", map[string]string{
		"revision": "3.0\n",
		"source-capabilities/1:fixed_supply/voltage":         "20000mV\n",
		"source-capabilities/1:fixed_supply/maximum_current": "5000mA\n",
		"sink-capabilities/1:fixed_supply/voltage":           "5000mV\n",
		"sink-capabilities/1:fixed_supply/maximum_current":   "3000mA\n",
	})

	partner := &model.Partner{}
	parsePDDirectories(partner, partnerDir)

	if partner.PDRevision != "3.0" {
		t.Errorf("PDRevision = %q, want %q", partner.PDRevision, "3.0")
	}
	if len(partner.SourceCapabilities) != 1 {
		t.Errorf("got %d source caps, want 1", len(partner.SourceCapabilities))
	}
	if len(partner.SinkCapabilities) != 1 {
		t.Errorf("got %d sink caps, want 1", len(partner.SinkCapabilities))
	}
}

func TestParsePDDirectories_ACPowered(t *testing.T) {
	partnerDir := t.TempDir()
	makePDDir(t, partnerDir, "pd0", map[string]string{
		"source-capabilities/1:fixed_supply/voltage":            "20000mV\n",
		"source-capabilities/1:fixed_supply/maximum_current":    "5000mA\n",
		"source-capabilities/1:fixed_supply/unconstrained_power": "1\n",
	})

	partner := &model.Partner{}
	parsePDDirectories(partner, partnerDir)

	if !partner.ACPowered {
		t.Error("ACPowered = false, want true")
	}
}

func TestParsePDDirectories_NotACPowered(t *testing.T) {
	partnerDir := t.TempDir()
	makePDDir(t, partnerDir, "pd0", map[string]string{
		"source-capabilities/1:fixed_supply/voltage":            "5000mV\n",
		"source-capabilities/1:fixed_supply/maximum_current":    "3000mA\n",
		"source-capabilities/1:fixed_supply/unconstrained_power": "0\n",
	})

	partner := &model.Partner{}
	parsePDDirectories(partner, partnerDir)

	if partner.ACPowered {
		t.Error("ACPowered = true, want false")
	}
}

func TestParsePDDirectories_MultiplePDDirs(t *testing.T) {
	partnerDir := t.TempDir()
	// pd0 sets revision; pd1 adds more capabilities
	makePDDir(t, partnerDir, "pd0", map[string]string{
		"revision": "2.0\n",
		"source-capabilities/1:fixed_supply/voltage":         "5000mV\n",
		"source-capabilities/1:fixed_supply/maximum_current": "3000mA\n",
	})
	makePDDir(t, partnerDir, "pd1", map[string]string{
		"revision": "3.0\n",
		"source-capabilities/1:fixed_supply/voltage":         "20000mV\n",
		"source-capabilities/1:fixed_supply/maximum_current": "5000mA\n",
	})

	partner := &model.Partner{}
	parsePDDirectories(partner, partnerDir)

	// PDRevision should be from the first pd dir found
	if partner.PDRevision == "" {
		t.Error("PDRevision should be set")
	}
	if len(partner.SourceCapabilities) != 2 {
		t.Errorf("got %d source caps, want 2 (one per pd dir)", len(partner.SourceCapabilities))
	}
}

func TestParsePDDirectories_SkipsNonPDDirs(t *testing.T) {
	partnerDir := t.TempDir()
	// These should not be parsed
	for _, name := range []string{"mode0", "port0-partner", "pdX"} {
		if err := os.MkdirAll(filepath.Join(partnerDir, name), 0755); err != nil {
			t.Fatal(err)
		}
	}

	partner := &model.Partner{}
	parsePDDirectories(partner, partnerDir)

	if partner.PDRevision != "" || len(partner.SourceCapabilities) != 0 || len(partner.SinkCapabilities) != 0 {
		t.Error("non-pd directories should not be parsed")
	}
}

func TestParsePDDirectories_MissingDir(t *testing.T) {
	partner := &model.Partner{}
	// Should not panic on missing directory
	parsePDDirectories(partner, "/nonexistent/path")

	if partner.PDRevision != "" {
		t.Errorf("PDRevision = %q, want empty", partner.PDRevision)
	}
}

func TestParsePDDirectories_PDRevisionUsesFirst(t *testing.T) {
	partnerDir := t.TempDir()
	makePDDir(t, partnerDir, "pd0", map[string]string{
		"revision": "2.0\n",
	})
	makePDDir(t, partnerDir, "pd1", map[string]string{
		"revision": "3.0\n",
	})

	partner := &model.Partner{}
	parsePDDirectories(partner, partnerDir)

	// Once set from pd0, should not be overwritten by pd1
	if partner.PDRevision == "" {
		t.Error("PDRevision should be set")
	}
}
