package specparse

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	tolerancePattern   = regexp.MustCompile(`(?i)[±+\-]?\s*(\d+(?:\.\d+)?)\s*%`)
	powerPattern       = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?(?:\s*/\s*\d+(?:\.\d+)?)?)\s*(mW|W)\b`)
	tempcoPattern      = regexp.MustCompile(`(?i)[±+\-]?\s*(\d+(?:\.\d+)?)\s*ppm(?:\s*/\s*(?:°\s*)?C)?`)
	packagePattern     = regexp.MustCompile(`(?i)(0201|0402|0603|0805|1206|1210|1812|2010|2512|SOT-23(?:-\d+)?|SOIC-\d+|TSSOP-\d+|QFN-\d+|QFP-\d+|LQFP-\d+|DIP-\d+|BGA-\d+|TO-\d+|SOD-\d+|SMA|SMB|SMC)`)
	resistancePattern  = regexp.MustCompile(`(?i)(?:\b\d+[rkm]\d*\b|\b\d[\d,]*(?:\.\d+)?\s*(?:meg|[kKmM])?\s*(?:ohms?|ohm|Ω|Ω)\b|\b0r\b|\b\d+(?:\.\d+)?\s*[kKmM]\b)`)
	milliOhmPattern    = regexp.MustCompile(`(\d[\d,]*(?:\.\d+)?)\s*(m(?:[oO]hms?|[oO]hm|Ω|Ω))\b`)
	capacitancePattern = regexp.MustCompile(`(?i)(\d[\d,]*(?:\.\d+)?)\s*(pF|nF|uF|µF|μF|mF|F)\b`)
	inductancePattern  = regexp.MustCompile(`(?i)(\d[\d,]*(?:\.\d+)?)\s*(pH|nH|uH|µH|μH|mH|H)\b`)
	voltagePattern     = regexp.MustCompile(`(?i)(\d[\d,]*(?:\.\d+)?)\s*(uV|µV|μV|mV|V|kV)\b`)
	currentPattern     = regexp.MustCompile(`(?i)(\d[\d,]*(?:\.\d+)?)\s*(uA|µA|μA|mA|A)\b`)
)

type scaledSuffix struct {
	suffix     string
	multiplier float64
}

func ParseResistance(raw string) (float64, bool) {
	if value, ok := parseScaledUnit(raw, milliOhmPattern, map[string]float64{"MOHM": 1e-3, "MOHMS": 1e-3}); ok {
		return value, true
	}
	raw = extractResistanceToken(raw)
	normalized := normalizeSpecValue(raw)
	normalized = strings.TrimSuffix(normalized, "OHMS")
	normalized = strings.TrimSuffix(normalized, "OHM")
	if strings.HasSuffix(normalized, "MEG") {
		normalized = strings.TrimSuffix(normalized, "MEG") + "M"
	}
	if normalized == "" {
		return 0, false
	}
	if strings.ContainsAny(normalized, "RKM") {
		return parseLetterDecimalValue(normalized, map[string]float64{"R": 1, "K": 1e3, "M": 1e6})
	}
	return parseScaledNumber(normalized, []scaledSuffix{{suffix: "K", multiplier: 1e3}, {suffix: "M", multiplier: 1e6}})
}

func ParseCapacitance(raw string) (float64, bool) {
	return parseScaledUnit(raw, capacitancePattern, map[string]float64{
		"PF": 1e-12,
		"NF": 1e-9,
		"UF": 1e-6,
		"MF": 1e-3,
		"F":  1,
	})
}

func ParseInductance(raw string) (float64, bool) {
	return parseScaledUnit(raw, inductancePattern, map[string]float64{
		"PH": 1e-12,
		"NH": 1e-9,
		"UH": 1e-6,
		"MH": 1e-3,
		"H":  1,
	})
}

func ParseVoltage(raw string) (float64, bool) {
	return parseScaledUnit(raw, voltagePattern, map[string]float64{
		"UV": 1e-6,
		"MV": 1e-3,
		"V":  1,
		"KV": 1e3,
	})
}

func ParseCurrent(raw string) (float64, bool) {
	return parseScaledUnit(raw, currentPattern, map[string]float64{
		"UA": 1e-6,
		"MA": 1e-3,
		"A":  1,
	})
}

func ParsePower(raw string) (float64, bool) {
	match := powerPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if len(match) != 3 {
		return 0, false
	}
	value, ok := parseFractionalNumber(match[1])
	if !ok {
		return 0, false
	}
	unit := strings.ToUpper(strings.TrimSpace(match[2]))
	if unit == "MW" {
		value /= 1000
	}
	return value, true
}

func ParseTolerancePercent(raw string) (float64, bool) {
	return parsePatternNumber(raw, tolerancePattern)
}

func ParseTempcoPPMC(raw string) (float64, bool) {
	return parsePatternNumber(raw, tempcoPattern)
}

func ParsePackage(raw string) string {
	match := packagePattern.FindStringSubmatch(strings.TrimSpace(raw))
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func NormalizePackage(raw string) string {
	if parsed := ParsePackage(raw); parsed != "" {
		return parsed
	}
	return strings.TrimSpace(raw)
}

func parseScaledUnit(raw string, pattern *regexp.Regexp, multipliers map[string]float64) (float64, bool) {
	match := pattern.FindStringSubmatch(strings.TrimSpace(raw))
	if len(match) != 3 {
		return 0, false
	}
	value, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(match[1]), ",", ""), 64)
	if err != nil {
		return 0, false
	}
	unit := strings.ToUpper(strings.TrimSpace(match[2]))
	unit = strings.NewReplacer("µ", "U", "μ", "U", "Ω", "OHM", "Ω", "OHM").Replace(unit)
	multiplier, ok := multipliers[unit]
	if !ok {
		return 0, false
	}
	return value * multiplier, true
}

func parsePatternNumber(raw string, pattern *regexp.Regexp) (float64, bool) {
	match := pattern.FindStringSubmatch(strings.TrimSpace(raw))
	if len(match) < 2 {
		return 0, false
	}
	value, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

func parseScaledNumber(raw string, suffixes []scaledSuffix) (float64, bool) {
	for _, suffix := range suffixes {
		if !strings.HasSuffix(raw, suffix.suffix) {
			continue
		}
		numberPart := strings.TrimSuffix(raw, suffix.suffix)
		if numberPart == "" {
			return 0, false
		}
		value, err := strconv.ParseFloat(numberPart, 64)
		if err != nil {
			return 0, false
		}
		return value * suffix.multiplier, true
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

func parseLetterDecimalValue(raw string, multipliers map[string]float64) (float64, bool) {
	for _, letter := range []string{"R", "K", "M"} {
		multiplier, ok := multipliers[letter]
		if !ok {
			continue
		}
		idx := strings.Index(raw, letter)
		if idx < 0 {
			continue
		}
		left := raw[:idx]
		right := raw[idx+1:]
		if left == "" {
			left = "0"
		}
		numberText := left
		if right != "" {
			numberText = left + "." + right
		}
		value, err := strconv.ParseFloat(numberText, 64)
		if err != nil {
			return 0, false
		}
		return value * multiplier, true
	}
	return 0, false
}

func parseFractionalNumber(raw string) (float64, bool) {
	trimmed := strings.TrimSpace(raw)
	if strings.Contains(trimmed, "/") {
		parts := strings.SplitN(trimmed, "/", 2)
		if len(parts) != 2 {
			return 0, false
		}
		numerator, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			return 0, false
		}
		denominator, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil || denominator == 0 {
			return 0, false
		}
		return numerator / denominator, true
	}
	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

func extractResistanceToken(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if match := resistancePattern.FindString(trimmed); match != "" {
		return match
	}
	return trimmed
}

func normalizeSpecValue(raw string) string {
	replacer := strings.NewReplacer(
		"Ω", "OHM",
		"Ω", "OHM",
		"µ", "U",
		"μ", "U",
		",", "",
		" ", "",
	)
	normalized := strings.ToUpper(strings.TrimSpace(raw))
	normalized = replacer.Replace(normalized)
	normalized = strings.TrimPrefix(normalized, "±")
	return normalized
}
