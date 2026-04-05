package sourcing

import "testing"

func TestParsePrice(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"$ 1.57", 1.57},
		{"USD 1.95", 1.95},
		{"$0.02", 0.02},
		{"3.62", 3.62},
		{"€ 2.34", 2.34},
		{"1,234.56", 1234.56},
		{"CNY 0.50", 0.50},
		{"", 0},
		{"N/A", 0},
	}
	for _, tc := range tests {
		got := parsePrice(tc.input)
		if got != tc.want {
			t.Errorf("parsePrice(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestNormalizeCurrency(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"$", "USD"},
		{"€", "EUR"},
		{"£", "GBP"},
		{"¥", "JPY"},
		{"CN¥", "CNY"},
		{"₹", "INR"},
		{"₩", "KRW"},
		{"USD", "USD"},
		{"usd", "USD"},
		{"eur", "EUR"},
		{"", ""},
		{"  USD  ", "USD"},
	}
	for _, tc := range tests {
		got := normalizeCurrency(tc.input)
		if got != tc.want {
			t.Errorf("normalizeCurrency(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
