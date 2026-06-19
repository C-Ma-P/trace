package specparse

import (
	"math"
	"testing"
)

func TestParseValues(t *testing.T) {
	tests := []struct {
		name  string
		input string
		parse func(string) (float64, bool)
		want  float64
	}{
		{name: "resistance kilo-ohms", input: "10 kOhms", parse: ParseResistance, want: 10000},
		{name: "resistance letter decimal", input: "4K7", parse: ParseResistance, want: 4700},
		{name: "resistance milli-ohms", input: "90 mOhms Max", parse: ParseResistance, want: 0.09},
		{name: "capacitance microfarads", input: "0.1 uF", parse: ParseCapacitance, want: 100e-9},
		{name: "capacitance nanofarads", input: "100nF", parse: ParseCapacitance, want: 100e-9},
		{name: "inductance microhenry", input: "10 uH", parse: ParseInductance, want: 10e-6},
		{name: "voltage volts", input: "25V", parse: ParseVoltage, want: 25},
		{name: "current milliamps", input: "600mA", parse: ParseCurrent, want: 0.6},
		{name: "power fractional watts", input: "1/10W", parse: ParsePower, want: 0.1},
		{name: "tolerance percent", input: "±5%", parse: ParseTolerancePercent, want: 5},
		{name: "tempco ppm", input: "100ppm/°C", parse: ParseTempcoPPMC, want: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, ok := tt.parse(tt.input)
			if got := mustParseFloat(t, value, ok); !almostEqual(got, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestParsePackage(t *testing.T) {
	if got := ParsePackage("Cut Tape 0402"); got != "0402" {
		t.Fatalf("expected 0402, got %q", got)
	}
	if got := NormalizePackage("Tape & Reel (TR) SOT-23-5"); got != "SOT-23-5" {
		t.Fatalf("expected SOT-23-5, got %q", got)
	}
}

func mustParseFloat(t *testing.T, value float64, ok bool) float64 {
	t.Helper()
	if !ok {
		t.Fatal("expected parse to succeed")
	}
	return value
}

func almostEqual(got, want float64) bool {
	const epsilon = 1e-12
	return math.Abs(got-want) <= epsilon
}
