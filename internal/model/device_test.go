package model

import "testing"

func TestVoltageRange(t *testing.T) {
	fixed := PowerCapability{Voltage: 9000}
	if lo, hi := fixed.VoltageRange(); lo != 9000 || hi != 9000 {
		t.Errorf("fixed VoltageRange() = %d-%d, want 9000-9000", lo, hi)
	}

	rng := PowerCapability{MinimumVoltage: 5000, MaximumVoltage: 20000}
	if lo, hi := rng.VoltageRange(); lo != 5000 || hi != 20000 {
		t.Errorf("range VoltageRange() = %d-%d, want 5000-20000", lo, hi)
	}
}

func TestSourceCapUsable(t *testing.T) {
	fixedSink := []PowerCapability{{Voltage: 5000}, {Voltage: 20000}}
	batterySink := []PowerCapability{{Voltage: 5000}, {MinimumVoltage: 5000, MaximumVoltage: 20000}}

	tests := []struct {
		name   string
		source PowerCapability
		sink   []PowerCapability
		want   bool
	}{
		{"5V matches fixed sink", PowerCapability{Voltage: 5000}, fixedSink, true},
		{"20V matches fixed sink", PowerCapability{Voltage: 20000}, fixedSink, true},
		{"9V gap between fixed sinks", PowerCapability{Voltage: 9000}, fixedSink, false},
		{"28V above fixed sink max", PowerCapability{Voltage: 28000}, fixedSink, false},
		{"9V covered by battery range", PowerCapability{Voltage: 9000}, batterySink, true},
		{"28V above battery range", PowerCapability{Voltage: 28000}, batterySink, false},
		{"PPS source overlaps sink range", PowerCapability{Programmable: true, MinimumVoltage: 3300, MaximumVoltage: 11000}, batterySink, true},
		{"unknown sink (empty) is not usable", PowerCapability{Voltage: 5000}, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SourceCapUsable(tt.source, tt.sink); got != tt.want {
				t.Errorf("SourceCapUsable() = %v, want %v", got, tt.want)
			}
		})
	}
}
