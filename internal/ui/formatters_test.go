package ui

import (
	"regexp"
	"strings"
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

func TestFormatSourceCapabilities_UnknownSink(t *testing.T) {
	// No local sink caps: plain rows, no glyph, no "usable" summary.
	source := []model.PowerCapability{
		{Voltage: 5000, MaximumCurrent: 3000},
		{Voltage: 20000, MaximumCurrent: 5000},
	}
	got := stripANSI(formatSourceCapabilities(source, nil))
	want := "  Charger: 100W max\n" +
		"    5V @ 3A      15W\n" +
		"    20V @ 5A     100W\n"
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestFormatSourceCapabilities_MixedUsable(t *testing.T) {
	// Charger offers 5/9/20/28V; local sink accepts only 5V and 20V fixed.
	source := []model.PowerCapability{
		{Voltage: 5000, MaximumCurrent: 3000},
		{Voltage: 9000, MaximumCurrent: 3000},
		{Voltage: 20000, MaximumCurrent: 3400},
		{Voltage: 28000, MaximumCurrent: 3500},
	}
	sink := []model.PowerCapability{{Voltage: 5000}, {Voltage: 20000}}

	raw := formatSourceCapabilities(source, sink)
	got := stripANSI(raw)
	want := "  Charger: 98W max · 68W usable\n" +
		"    5V @ 3A      15W\n" +
		"    9V @ 3A      27W\n" + // gap between sink voltages: no reason
		"    20V @ 3.4A   68W\n" +
		"    28V @ 3.5A   98W  (sink max 20V)\n" // above sink max: reason shown
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}

	// Usable rows are shown normally; only unusable rows are dimmed.
	if !strings.Contains(raw, inactiveStyle.Render("9V @ 3A      27W")) {
		t.Error("unusable 9V row should be rendered dimmed")
	}
	if strings.Contains(raw, inactiveStyle.Render("5V @ 3A      15W")) {
		t.Error("usable 5V row should not be dimmed")
	}
}

func TestFormatSourceCapabilities_AllUsable(t *testing.T) {
	// Local sink has a battery range covering everything: no "usable" summary, all ✓.
	source := []model.PowerCapability{
		{Voltage: 5000, MaximumCurrent: 3000},
		{Voltage: 9000, MaximumCurrent: 3000},
		{Voltage: 20000, MaximumCurrent: 3400},
	}
	sink := []model.PowerCapability{{Voltage: 5000}, {MinimumVoltage: 5000, MaximumVoltage: 20000}}

	got := stripANSI(formatSourceCapabilities(source, sink))
	if strings.Contains(got, "usable") {
		t.Errorf("did not expect a usable summary when all caps are usable, got:\n%s", got)
	}
	if strings.Contains(got, "sink max") {
		t.Errorf("did not expect any unusable rows, got:\n%s", got)
	}
}

func TestFormatPowerModeInline_NonPD(t *testing.T) {
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
		got := stripANSI(formatPowerModeInline(nil, tt.mode))
		if got != tt.want {
			t.Errorf("formatPowerModeInline(nil, %q) = %q, want %q", tt.mode, got, tt.want)
		}
	}
}

func TestFormatPowerModeInline_PD_NilPD(t *testing.T) {
	got := stripANSI(formatPowerModeInline(nil, "usb_power_delivery"))
	if got != "[PD]" {
		t.Errorf("got %q, want %q", got, "[PD]")
	}
}

func TestFormatPowerModeInline_PD_WithRevision(t *testing.T) {
	pd := &model.PowerDelivery{Revision: "3.0"}
	got := stripANSI(formatPowerModeInline(pd, "usb_power_delivery"))
	if got != "[PD 3.0]" {
		t.Errorf("got %q, want %q", got, "[PD 3.0]")
	}
}

func TestFormatPowerModeInline_PD_WithWatts(t *testing.T) {
	pd := &model.PowerDelivery{
		Revision:           "3.0",
		SourceCapabilities: []model.PowerCapability{{Voltage: 20000, MaximumCurrent: 5000}},
	}
	got := stripANSI(formatPowerModeInline(pd, "usb_power_delivery"))
	if got != "[PD 3.0, 100W]" {
		t.Errorf("got %q, want %q", got, "[PD 3.0, 100W]")
	}
}

func TestFormatPowerModeInline_PD_ACPowered(t *testing.T) {
	pd := &model.PowerDelivery{
		Revision:           "3.0",
		ACPowered:          true,
		SourceCapabilities: []model.PowerCapability{{Voltage: 20000, MaximumCurrent: 5000}},
	}
	got := stripANSI(formatPowerModeInline(pd, "usb_power_delivery"))
	if got != "[PD 3.0, 100W, AC]" {
		t.Errorf("got %q, want %q", got, "[PD 3.0, 100W, AC]")
	}
}

func TestFormatPowerModeInline_PD_ZeroRevision(t *testing.T) {
	pd := &model.PowerDelivery{Revision: "0.0"}
	got := stripANSI(formatPowerModeInline(pd, "usb_power_delivery"))
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
			want: "    [0] Thunderbolt (Thunderbolt 3) (SVID: 8087, VDO: 0x0)\n",
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
