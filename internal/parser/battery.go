package parser

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/cpulvermacher/lsusbc/internal/model"
)

// LoadBatteryInfo finds the first battery in power_supply and returns its info, or nil if none found.
func LoadBatteryInfo(sysfsDir string) *model.BatteryInfo {
	powerSupplyDir := filepath.Join(sysfsDir, "class/power_supply")
	entries, err := os.ReadDir(powerSupplyDir)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		dir := filepath.Join(powerSupplyDir, entry.Name())
		capacityPath := filepath.Join(dir, "capacity")
		if _, err := os.Stat(capacityPath); err != nil {
			continue
		}
		capacity, err := strconv.Atoi(readFile(capacityPath))
		if err != nil {
			continue
		}
		return &model.BatteryInfo{
			Capacity:      capacity,
			CapacityLevel: readFile(filepath.Join(dir, "capacity_level")),
			Status:        readFile(filepath.Join(dir, "status")),
		}
	}
	return nil
}
