package ui

import (
	"regexp"
	"testing"

	"github.com/cpulvermacher/lsusbc/internal/model"
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

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

func TestFormatCapabilities_NonPD(t *testing.T) {
	tests := []struct {
		mode string
		want string
	}{
		{"default", "[≤5W]"},
		{"1.5A", "[7.5W]"},
		{"3.0A", "[15W]"},
		{"", ""},
		{"unknown", ""},
	}
	for _, tt := range tests {
		got := stripANSI(formatCapabilities(nil, tt.mode))
		if got != tt.want {
			t.Errorf("formatCapabilities(nil, %q) = %q, want %q", tt.mode, got, tt.want)
		}
	}
}

func TestFormatCapabilities_PD_NilPD(t *testing.T) {
	got := stripANSI(formatCapabilities(nil, "usb_power_delivery"))
	if got != "[PD]" {
		t.Errorf("got %q, want %q", got, "[PD]")
	}
}

func TestFormatCapabilities_PD_WithRevision(t *testing.T) {
	pd := &model.PowerDelivery{Revision: "3.0"}
	got := stripANSI(formatCapabilities(pd, "usb_power_delivery"))
	if got != "[PD 3.0]" {
		t.Errorf("got %q, want %q", got, "[PD 3.0]")
	}
}

func TestFormatCapabilities_PD_WithWatts(t *testing.T) {
	pd := &model.PowerDelivery{
		Revision:           "3.0",
		SourceCapabilities: []model.PowerCapability{{Voltage: 20000, MaximumCurrent: 5000}},
	}
	got := stripANSI(formatCapabilities(pd, "usb_power_delivery"))
	if got != "[PD 3.0, 100W]" {
		t.Errorf("got %q, want %q", got, "[PD 3.0, 100W]")
	}
}

func TestFormatCapabilities_PD_ACPowered(t *testing.T) {
	pd := &model.PowerDelivery{
		Revision:           "3.0",
		ACPowered:          true,
		SourceCapabilities: []model.PowerCapability{{Voltage: 20000, MaximumCurrent: 5000}},
	}
	got := stripANSI(formatCapabilities(pd, "usb_power_delivery"))
	if got != "[PD 3.0, 100W, AC]" {
		t.Errorf("got %q, want %q", got, "[PD 3.0, 100W, AC]")
	}
}

func TestFormatCapabilities_PD_ZeroRevision(t *testing.T) {
	pd := &model.PowerDelivery{Revision: "0.0"}
	got := stripANSI(formatCapabilities(pd, "usb_power_delivery"))
	if got != "[PD]" {
		t.Errorf("got %q, want %q", got, "[PD]")
	}
}

func TestFormatAlternateMode(t *testing.T) {
	tests := []struct {
		name string
		mode model.AlternateMode
		want string
	}{
		{
			name: "non-DP mode",
			mode: model.AlternateMode{Index: 0, Description: "Thunderbolt", SVID: "8087", VDO: "0x0", Active: "no"},
			want: "    [0] Thunderbolt (SVID: 8087, VDO: 0x0)\n",
		},
		{
			name: "DP sink, native DP + tunneling",
			mode: model.AlternateMode{Index: 0, Description: "DisplayPort", SVID: "ff01", VDO: "0x001c0c05", Active: "yes"},
			want: "   *[0] DisplayPort sink, native DP + tunneling (SVID: ff01, VDO: 0x001c0c05)\n",
		},
		{
			name: "DP source+sink, native DP + tunneling",
			mode: model.AlternateMode{Index: 0, Description: "DisplayPort", SVID: "ff01", VDO: "0x001c1c43", Active: "yes"},
			want: "   *[0] DisplayPort source+sink, native DP + tunneling (SVID: ff01, VDO: 0x001c1c43)\n",
		},
		{
			name: "DP sink, native DP only",
			mode: model.AlternateMode{Index: 0, Description: "DisplayPort", SVID: "ff01", VDO: "0x00100001", Active: "no"},
			want: "    [0] DisplayPort sink, native DP (SVID: ff01, VDO: 0x00100001)\n",
		},
		{
			name: "DP source, tunneling only",
			mode: model.AlternateMode{Index: 1, Description: "DisplayPort", SVID: "ff01", VDO: "0x00000e02", Active: "no"},
			want: "    [1] DisplayPort source, tunneling (SVID: ff01, VDO: 0x00000e02)\n",
		},
		{
			name: "DP source, no pin info",
			mode: model.AlternateMode{Index: 1, Description: "DisplayPort", SVID: "ff01", VDO: "0x00000002", Active: "no"},
			want: "    [1] DisplayPort source (SVID: ff01, VDO: 0x00000002)\n",
		},
		{
			name: "DP reserved capability bits",
			mode: model.AlternateMode{Index: 0, Description: "DisplayPort", SVID: "ff01", VDO: "0x0", Active: "no"},
			want: "    [0] DisplayPort (SVID: ff01, VDO: 0x0)\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAlternateMode(tt.mode)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
