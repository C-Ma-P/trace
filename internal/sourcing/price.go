package sourcing

import (
	"strconv"
	"strings"
)

// currencySymbols maps common currency symbols to their ISO 4217 codes.
var currencySymbols = map[string]string{
	"$":   "USD",
	"€":   "EUR",
	"£":   "GBP",
	"¥":   "JPY",
	"CN¥": "CNY",
	"₹":   "INR",
	"₩":   "KRW",
}

const defaultCurrency = "USD"

// normalizeCurrency converts a currency symbol (e.g. "$") or code (e.g. "usd")
// to an uppercase ISO 4217 code. Returns the trimmed, uppercased input unchanged
// when no symbol mapping exists.
func normalizeCurrency(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if iso, ok := currencySymbols[s]; ok {
		return iso
	}
	return strings.ToUpper(s)
}

// firstCurrency returns the first non-empty normalized currency from the
// candidates, falling back to defaultCurrency if all are empty.
func firstCurrency(candidates ...string) string {
	for _, c := range candidates {
		if n := normalizeCurrency(c); n != "" {
			return n
		}
	}
	return defaultCurrency
}

// parsePrice extracts a numeric price from strings like "$0.02", "USD 1.95",
// "$ 1.57", or "3.62". Commas used as thousands separators are ignored.
// Returns 0 if nothing parseable is found.
func parsePrice(s string) float64 {
	s = strings.TrimSpace(s)
	// Find the start of the numeric part (first digit).
	start := strings.IndexFunc(s, func(r rune) bool {
		return r >= '0' && r <= '9'
	})
	if start < 0 {
		return 0
	}
	s = strings.ReplaceAll(s[start:], ",", "")
	// Trim any trailing non-numeric suffix.
	end := strings.LastIndexFunc(s, func(r rune) bool {
		return r >= '0' && r <= '9'
	})
	if end < 0 {
		return 0
	}
	v, err := strconv.ParseFloat(s[:end+1], 64)
	if err != nil {
		return 0
	}
	return v
}
