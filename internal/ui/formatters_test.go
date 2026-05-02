package ui

import (
	"testing"

	"github.com/cpulvermacher/lsusbc/internal/model"
)

func TestFormatVoltage_Fixed(t *testing.T) {
	tests := []struct {
		voltage int
		want    string
	}{
		{5000, "5V"},
		{20000, "20V"},
		{9000, "9V"},
		{3300, "3.3V"},
	}
	for _, tt := range tests {
		pc := model.PowerCapability{Voltage: tt.voltage}
		if got := FormatVoltage(pc); got != tt.want {
			t.Errorf("FormatVoltage(%d mV) = %q, want %q", tt.voltage, got, tt.want)
		}
	}
}

func TestFormatVoltage_Programmable(t *testing.T) {
	pc := model.PowerCapability{
		Programmable:   true,
		MinimumVoltage: 3300,
		MaximumVoltage: 21000,
	}
	want := "3.3V-21V"
	if got := FormatVoltage(pc); got != want {
		t.Errorf("FormatVoltage(programmable 3.3-21V) = %q, want %q", got, want)
	}
}

func TestFormatCurrent(t *testing.T) {
	tests := []struct {
		current int
		want    string
	}{
		{3000, "3A"},
		{1000, "1A"},
		{1500, "1.5A"},
		{5000, "5A"},
		{900, "0.9A"},
	}
	for _, tt := range tests {
		pc := model.PowerCapability{MaximumCurrent: tt.current}
		if got := FormatCurrent(pc); got != tt.want {
			t.Errorf("FormatCurrent(%d mA) = %q, want %q", tt.current, got, tt.want)
		}
	}
}

func TestWatts_Fixed(t *testing.T) {
	tests := []struct {
		voltage int
		current int
		want    int
	}{
		{5000, 3000, 15},
		{20000, 5000, 100},
		{9000, 3000, 27},
	}
	for _, tt := range tests {
		pc := model.PowerCapability{Voltage: tt.voltage, MaximumCurrent: tt.current}
		if got := Watts(pc); got != tt.want {
			t.Errorf("Watts(%dV, %dmA) = %d, want %d", tt.voltage/1000, tt.current, got, tt.want)
		}
	}
}

func TestWatts_Programmable(t *testing.T) {
	pc := model.PowerCapability{
		Programmable:   true,
		MaximumVoltage: 21000,
		MaximumCurrent: 5000,
	}
	want := 105
	if got := Watts(pc); got != want {
		t.Errorf("Watts(programmable 21V, 5A) = %d, want %d", got, want)
	}
}

func TestMaxWatts(t *testing.T) {
	caps := []model.PowerCapability{
		{Voltage: 5000, MaximumCurrent: 3000},
		{Voltage: 20000, MaximumCurrent: 5000},
		{Voltage: 9000, MaximumCurrent: 3000},
	}
	want := 100
	if got := MaxWatts(caps); got != want {
		t.Errorf("MaxWatts() = %d, want %d", got, want)
	}
}

func TestMaxWatts_Empty(t *testing.T) {
	if got := MaxWatts(nil); got != 0 {
		t.Errorf("MaxWatts(nil) = %d, want 0", got)
	}
}
