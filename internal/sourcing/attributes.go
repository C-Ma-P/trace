package sourcing

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/C-Ma-P/trace/internal/domain"
	"github.com/C-Ma-P/trace/internal/domain/registry"
)

var (
	attributeTolerancePattern  = regexp.MustCompile(`(?i)[±+\-]?\s*(\d+(?:\.\d+)?)\s*%`)
	attributePowerPattern      = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?(?:\s*/\s*\d+(?:\.\d+)?)?)\s*(mW|W)\b`)
	attributeTempcoPattern     = regexp.MustCompile(`(?i)[±+\-]?\s*(\d+(?:\.\d+)?)\s*ppm(?:\s*/\s*(?:°\s*)?C)?`)
	attributePackagePattern    = regexp.MustCompile(`(?i)(0201|0402|0603|0805|1206|1210|1812|2010|2512|SOT-23(?:-\d+)?|SOIC-\d+|TSSOP-\d+|QFN-\d+|QFP-\d+|LQFP-\d+|DIP-\d+|BGA-\d+|TO-\d+|SOD-\d+|SMA|SMB|SMC)`)
	attributeResistancePattern = regexp.MustCompile(`(?i)(?:\b\d+[rkm]\d*\b|\b\d+(?:\.\d+)?\s*(?:meg|[kKmM])?\s*(?:ohms?|ohm|Ω|Ω)\b|\b0r\b|\b\d+(?:\.\d+)?\s*[kKmM]\b)`)
)

func ComponentAttributesFromOffer(category domain.Category, offer SupplierOffer) []domain.AttributeValue {
	out := make([]domain.AttributeValue, 0, 6)
	seen := make(map[string]struct{}, 6)
	specs := newOfferSpecIndex(offer.Raw)

	addAttribute := func(attr domain.AttributeValue) {
		if attr.Key == "" {
			return
		}
		if _, ok := seen[attr.Key]; ok {
			return
		}
		if err := registry.ValidateAttributes(category, []domain.AttributeValue{attr}); err != nil {
			return
		}
		out = append(out, attr)
		seen[attr.Key] = struct{}{}
	}

	switch category {
	case domain.CategoryResistor:
		if value, ok := parseResistanceValue(firstNonEmpty(
			specs.firstResistance(),
			offer.Description,
		)); ok {
			addAttribute(numberAttribute(registry.AttrResistanceOhms, value, "ohm"))
		}
		if value, ok := parsePatternNumber(firstNonEmpty(
			specs.firstFuzzy(
				[]string{"Tolerance", "Resistance Tolerance"},
				[]string{"tolerance"},
			),
			offer.Description,
		), attributeTolerancePattern); ok {
			addAttribute(numberAttribute(registry.AttrTolerancePercent, value, "percent"))
		}
		if value, ok := parsePowerValue(firstNonEmpty(
			specs.firstFuzzy(
				[]string{"Power Rating", "Power (Watts)", "Power", "Wattage"},
				[]string{"power", "watt"},
			),
			offer.Description,
		)); ok {
			addAttribute(numberAttribute(registry.AttrPowerW, value, "W"))
		}
		if value, ok := parsePatternNumber(firstNonEmpty(
			specs.firstFuzzy(
				[]string{"Temperature Coefficient", "Temp Coefficient", "TC", "TCR"},
				[]string{"temperaturecoefficient", "tempcoefficient", "tempco", "tcr", "ppm"},
			),
			offer.Description,
		), attributeTempcoPattern); ok {
			addAttribute(numberAttribute(registry.AttrTempCoPPMC, value, "ppm/C"))
		}
		if resistorType := firstNonEmpty(
			specs.firstFuzzy(
				[]string{"Composition", "Technology", "Element Type", "Resistor Type"},
				[]string{"composition", "technology", "elementtype", "resistortype"},
			),
			parseResistorTypeFromDescription(offer.Description),
		); resistorType != "" {
			addAttribute(textAttribute(registry.AttrResistorType, resistorType))
		}
	}

	if packageName := firstNonEmpty(
		specs.firstFuzzy(
			[]string{"Package / Case", "Supplier Device Package", "Package", "EncapStandard", "Case Code - in", "Case Code - mm"},
			[]string{"packagecase", "supplierdevicepackage", "package", "encapstandard", "casecode"},
		),
		strings.TrimSpace(offer.Package),
		parsePackageFromText(offer.Description),
	); packageName != "" {
		addAttribute(textAttribute(registry.AttrPackage, normalizePackageValue(packageName)))
	}

	return out
}

func numberAttribute(key string, value float64, unit string) domain.AttributeValue {
	copy := value
	return domain.AttributeValue{Key: key, ValueType: domain.ValueTypeNumber, Number: &copy, Unit: unit}
}

func textAttribute(key, value string) domain.AttributeValue {
	copy := strings.TrimSpace(value)
	return domain.AttributeValue{Key: key, ValueType: domain.ValueTypeText, Text: &copy}
}

func parseResistanceValue(raw string) (float64, bool) {
	raw = extractResistanceToken(raw)
	normalized := normalizeSpecValue(raw)
	normalized = strings.TrimSuffix(normalized, "OHMS")
	normalized = strings.TrimSuffix(normalized, "OHM")
	if normalized == "" {
		return 0, false
	}
	if strings.ContainsAny(normalized, "RKM") {
		return parseLetterDecimalValue(normalized, map[string]float64{"R": 1, "K": 1e3, "M": 1e6})
	}
	return parseScaledNumber(normalized, []scaledSuffix{{suffix: "K", multiplier: 1e3}, {suffix: "M", multiplier: 1e6}})
}

func parsePowerValue(raw string) (float64, bool) {
	match := attributePowerPattern.FindStringSubmatch(strings.TrimSpace(raw))
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

type scaledSuffix struct {
	suffix     string
	multiplier float64
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

func parseResistorTypeFromDescription(raw string) string {
	upper := strings.ToUpper(raw)
	for _, candidate := range []string{"THICK FILM", "THIN FILM", "METAL FILM", "CARBON FILM", "WIREWOUND"} {
		if strings.Contains(upper, candidate) {
			return strings.Title(strings.ToLower(candidate))
		}
	}
	return ""
}

func parsePackageFromText(raw string) string {
	match := attributePackagePattern.FindStringSubmatch(strings.TrimSpace(raw))
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func extractResistanceToken(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if match := attributeResistancePattern.FindString(trimmed); match != "" {
		return match
	}
	return trimmed
}

func normalizePackageValue(raw string) string {
	if parsed := parsePackageFromText(raw); parsed != "" {
		return parsed
	}
	return strings.TrimSpace(raw)
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

type offerSpecIndex struct {
	values map[string]string
}

func newOfferSpecIndex(raw map[string]string) offerSpecIndex {
	values := make(map[string]string, len(raw))
	for key, value := range raw {
		normalizedKey := normalizeSpecKey(key)
		trimmedValue := strings.TrimSpace(value)
		if normalizedKey == "" || trimmedValue == "" {
			continue
		}
		if _, exists := values[normalizedKey]; exists {
			continue
		}
		values[normalizedKey] = trimmedValue
	}
	return offerSpecIndex{values: values}
}

func (i offerSpecIndex) first(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(i.values[normalizeSpecKey(key)]); value != "" {
			return value
		}
	}
	return ""
}

func (i offerSpecIndex) firstFuzzy(exactKeys []string, fuzzyNeedles []string) string {
	if value := i.first(exactKeys...); value != "" {
		return value
	}
	for key, value := range i.values {
		for _, needle := range fuzzyNeedles {
			normalizedNeedle := normalizeSpecKey(needle)
			if normalizedNeedle == "" {
				continue
			}
			if strings.Contains(key, normalizedNeedle) {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

func (i offerSpecIndex) firstResistance() string {
	if value := i.first("Resistance", "Resistance Value", "Value", "Resistance (Ohms)"); value != "" {
		return value
	}
	for key, value := range i.values {
		if strings.Contains(key, "tolerance") {
			continue
		}
		if strings.Contains(key, "resistance") || strings.Contains(key, "ohm") {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func normalizeSpecKey(raw string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(raw)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func putRawValue(raw map[string]string, key, value string) {
	trimmedKey := strings.TrimSpace(key)
	trimmedValue := strings.TrimSpace(value)
	if trimmedKey == "" || trimmedValue == "" {
		return
	}
	raw[trimmedKey] = trimmedValue
}
