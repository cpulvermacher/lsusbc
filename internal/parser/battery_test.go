package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func makePowerSupplyDir(t *testing.T, sysfsDir, name string, files map[string]string) {
	t.Helper()
	dir := filepath.Join(sysfsDir, "class/power_supply", name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for fname, content := range files {
		if err := os.WriteFile(filepath.Join(dir, fname), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLoadBatteryInfo_NoPowerSupply(t *testing.T) {
	if got := LoadBatteryInfo(t.TempDir()); got != nil {
		t.Errorf("want nil for empty sysfs, got %+v", got)
	}
}

func TestLoadBatteryInfo_NoCapacityFile(t *testing.T) {
	sysfsDir := t.TempDir()
	makePowerSupplyDir(t, sysfsDir, "AC", map[string]string{
		"online": "1\n",
	})
	if got := LoadBatteryInfo(sysfsDir); got != nil {
		t.Errorf("want nil for AC-only supply, got %+v", got)
	}
}

func TestLoadBatteryInfo_Discharging(t *testing.T) {
	sysfsDir := t.TempDir()
	makePowerSupplyDir(t, sysfsDir, "BAT0", map[string]string{
		"capacity":       "0\n",
		"capacity_level": "Critical\n",
		"status":         "Discharging\n",
	})

	got := LoadBatteryInfo(sysfsDir)
	if got == nil {
		t.Fatal("want BatteryInfo, got nil")
	}
	if got.Capacity != 0 {
		t.Errorf("Capacity = %d, want 0", got.Capacity)
	}
	if got.CapacityLevel != "Critical" {
		t.Errorf("CapacityLevel = %q, want %q", got.CapacityLevel, "Critical")
	}
	if got.Status != "Discharging" {
		t.Errorf("Status = %q, want %q", got.Status, "Discharging")
	}
	if got.PowerNow != 0 {
		t.Errorf("PowerNow = %d, want 0 when power_now absent", got.PowerNow)
	}
}

func TestLoadBatteryInfo_PowerNow(t *testing.T) {
	sysfsDir := t.TempDir()
	makePowerSupplyDir(t, sysfsDir, "BAT0", map[string]string{
		"capacity":       "96\n",
		"capacity_level": "Normal\n",
		"status":         "Discharging\n",
		"power_now":      "18744000\n",
	})

	got := LoadBatteryInfo(sysfsDir)
	if got == nil {
		t.Fatal("want BatteryInfo, got nil")
	}
	if got.PowerNow != 18744000 {
		t.Errorf("PowerNow = %d, want 18744000", got.PowerNow)
	}
}

func TestLoadBatteryInfo_Charging(t *testing.T) {
	sysfsDir := t.TempDir()
	makePowerSupplyDir(t, sysfsDir, "BAT0", map[string]string{
		"capacity":       "76\n",
		"capacity_level": "Normal\n",
		"status":         "Charging\n",
	})

	got := LoadBatteryInfo(sysfsDir)
	if got == nil {
		t.Fatal("want BatteryInfo, got nil")
	}
	if got.Capacity != 76 {
		t.Errorf("Capacity = %d, want 76", got.Capacity)
	}
	if got.CapacityLevel != "Normal" {
		t.Errorf("CapacityLevel = %q, want %q", got.CapacityLevel, "Normal")
	}
	if got.Status != "Charging" {
		t.Errorf("Status = %q, want %q", got.Status, "Charging")
	}
}

func TestLoadBatteryInfo_ACBeforeBattery(t *testing.T) {
	sysfsDir := t.TempDir()
	makePowerSupplyDir(t, sysfsDir, "AC", map[string]string{
		"online": "1\n",
	})
	makePowerSupplyDir(t, sysfsDir, "BAT0", map[string]string{
		"capacity":       "55\n",
		"capacity_level": "Normal\n",
		"status":         "Charging\n",
	})

	got := LoadBatteryInfo(sysfsDir)
	if got == nil {
		t.Fatal("want BatteryInfo, got nil")
	}
	if got.Capacity != 55 {
		t.Errorf("Capacity = %d, want 55", got.Capacity)
	}
	if got.Status != "Charging" {
		t.Errorf("Status = %q, want %q", got.Status, "Charging")
	}
}
