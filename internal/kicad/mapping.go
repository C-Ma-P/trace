package kicad

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"trace/internal/domain"
	"trace/internal/domain/registry"
)

var (
	tolerancePattern = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*%`)
	voltagePattern   = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*V`)
	packagePattern   = regexp.MustCompile(`(?i)(0201|0402|0603|0805|1206|1210|1812|2010|2512|SOT-23(?:-\d+)?|SOIC-\d+|TSSOP-\d+|QFN-\d+|QFP-\d+|LQFP-\d+|DIP-\d+|BGA-\d+|TO-\d+|SOD-\d+|SMA|SMB|SMC)`)
)

func mapBOMRows(rows []bomRow) []ImportPreviewRow {
	previewRows := make([]ImportPreviewRow, 0, len(rows))
	for index, row := range rows {
		requirement, warnings := draftRequirementFromRow(row)
		included := true
		if isTruthy(row.DNP) {
			included = false
			warnings = append(warnings, "Row is marked do not populate and is excluded by default.")
		}
		if requirement.Quantity <= 0 {
			requirement.Quantity = 1
			warnings = append(warnings, "Quantity could not be parsed cleanly; defaulted to 1.")
		}
		previewRows = append(previewRows, ImportPreviewRow{
			RowID:           fmt.Sprintf("row-%03d", index+1),
			Included:        included,
			SourceRefs:      row.Refs,
			SourceQuantity:  max(row.Quantity, 0),
			RawValue:        row.Value,
			RawFootprint:    row.Footprint,
			RawDescription:  row.Description,
			Manufacturer:    row.Manufacturer,
			MPN:             row.MPN,
			OtherFields:     copyFields(row),
			Requirement:     requirement,
			HasWarning:      len(warnings) > 0,
			WarningMessages: warnings,
		})
	}
	return previewRows
}

func draftRequirementFromRow(row bomRow) (DraftRequirement, []string) {
	warnings := make([]string, 0)
	category, confident := inferCategory(row)
	if !confident {
		warnings = append(warnings, fmt.Sprintf("Category inference is uncertain; defaulted to %s.", humanCategory(category)))
	}

	requirement := DraftRequirement{
		Category:    category,
		Quantity:    row.Quantity,
		Constraints: make([]domain.RequirementConstraint, 0, 6),
	}

	addTextConstraint(&requirement.Constraints, registry.AttrManufacturer, row.Manufacturer)
	addTextConstraint(&requirement.Constraints, registry.AttrMPN, row.MPN)
	addTextConstraint(&requirement.Constraints, registry.AttrPackage, packageFromFootprint(row.Footprint))

	tolerance := parsePatternNumber(tolerancePattern, row.Value, row.Description)
	voltage := parsePatternNumber(voltagePattern, row.Value, row.Description)

	switch category {
	case domain.CategoryResistor:
		if value, ok := parseResistorValue(row.Value); ok {
			addNumberConstraint(&requirement.Constraints, registry.AttrResistanceOhms, value, "ohm")
		} else {
			warnings = append(warnings, "Could not parse resistance from the BOM row.")
		}
		if tolerance != nil {
			addNumberConstraint(&requirement.Constraints, registry.AttrTolerancePercent, *tolerance, "percent")
		}
	case domain.CategoryCapacitor:
		if value, ok := parseCapacitanceValue(row.Value); ok {
			addNumberConstraint(&requirement.Constraints, registry.AttrCapacitanceF, value, "F")
		} else {
			warnings = append(warnings, "Could not parse capacitance from the BOM row.")
		}
		if tolerance != nil {
			addNumberConstraint(&requirement.Constraints, registry.AttrTolerancePercent, *tolerance, "percent")
		}
		if voltage != nil {
			addNumberConstraint(&requirement.Constraints, registry.AttrVoltageV, *voltage, "V")
		}
		if dielectric := findDielectric(row.Description, row.Value); dielectric != "" {
			addTextConstraint(&requirement.Constraints, registry.AttrDielectric, dielectric)
		}
	case domain.CategoryInductor:
		if value, ok := parseInductanceValue(row.Value); ok {
			addNumberConstraint(&requirement.Constraints, registry.AttrInductanceH, value, "H")
		} else {
			warnings = append(warnings, "Could not parse inductance from the BOM row.")
		}
		if tolerance != nil {
			addNumberConstraint(&requirement.Constraints, registry.AttrTolerancePercent, *tolerance, "percent")
		}
	}

	requirement.Name = draftRequirementName(row, category)
	if requirement.Name == row.Refs || requirement.Name == "Imported part" {
		warnings = append(warnings, "Name fallback is generic; review before import.")
	}
	if requirement.Quantity <= 0 {
		warnings = append(warnings, "Quantity is missing or malformed.")
	}
	if len(requirement.Constraints) == 0 {
		warnings = append(warnings, "No strong constraints were inferred from this row.")
	}

	return requirement, dedupeWarnings(warnings)
}

func inferCategory(row bomRow) (domain.Category, bool) {
	if refsSuggest(row.Refs, "R") || descriptionSuggests(row, "resistor", "ohm") {
		return domain.CategoryResistor, parseResistorValueOK(row.Value)
	}
	if refsSuggest(row.Refs, "C") || descriptionSuggests(row, "capacitor", "mlcc", "x7r", "c0g", "np0") {
		return domain.CategoryCapacitor, parseCapacitanceValueOK(row.Value)
	}
	if refsSuggest(row.Refs, "L") || descriptionSuggests(row, "inductor", "ferrite", "shielded") {
		return domain.CategoryInductor, parseInductanceValueOK(row.Value)
	}
	if parseResistorValueOK(row.Value) {
		return domain.CategoryResistor, true
	}
	if parseCapacitanceValueOK(row.Value) {
		return domain.CategoryCapacitor, true
	}
	if parseInductanceValueOK(row.Value) {
		return domain.CategoryInductor, true
	}
	if refsSuggest(row.Refs, "U", "IC", "Q", "D", "J", "P") || descriptionSuggests(row, "controller", "regulator", "driver", "sensor", "opamp", "amplifier", "mcu", "ic") {
		return domain.CategoryIntegratedCircuit, true
	}
	return domain.CategoryIntegratedCircuit, false
}

func descriptionSuggests(row bomRow, words ...string) bool {
	haystack := strings.ToLower(strings.Join([]string{row.Value, row.Description, row.Footprint, row.MPN}, " "))
	for _, word := range words {
		if strings.Contains(haystack, strings.ToLower(word)) {
			return true
		}
	}
	return false
}

func refsSuggest(refs string, prefixes ...string) bool {
	first := strings.ToUpper(strings.TrimSpace(refs))
	if first == "" {
		return false
	}
	separator := strings.IndexAny(first, ",; ")
	if separator >= 0 {
		first = first[:separator]
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(first, strings.ToUpper(prefix)) {
			return true
		}
	}
	return false
}

func draftRequirementName(row bomRow, category domain.Category) string {
	if value := strings.TrimSpace(row.Description); value != "" {
		return value
	}
	if row.Manufacturer != "" && row.MPN != "" {
		return strings.TrimSpace(row.Manufacturer + " " + row.MPN)
	}
	if row.MPN != "" {
		return row.MPN
	}
	if row.Value != "" {
		return strings.TrimSpace(row.Value + " " + genericCategoryName(category))
	}
	if row.Refs != "" {
		return row.Refs
	}
	return "Imported part"
}

func genericCategoryName(category domain.Category) string {
	switch category {
	case domain.CategoryResistor:
		return "resistor"
	case domain.CategoryCapacitor:
		return "capacitor"
	case domain.CategoryInductor:
		return "inductor"
	default:
		return "part"
	}
}

func humanCategory(category domain.Category) string {
	switch category {
	case domain.CategoryResistor:
		return "Resistor"
	case domain.CategoryCapacitor:
		return "Capacitor"
	case domain.CategoryInductor:
		return "Inductor"
	case domain.CategoryIntegratedCircuit:
		return "Integrated Circuit"
	default:
		return string(category)
	}
}

func addTextConstraint(constraints *[]domain.RequirementConstraint, key, value string) {
	value = strings.TrimSpace(value)
	if value == "" || hasConstraint(*constraints, key) {
		return
	}
	copyValue := value
	*constraints = append(*constraints, domain.RequirementConstraint{
		Key:       key,
		ValueType: domain.ValueTypeText,
		Operator:  domain.OperatorEqual,
		Text:      &copyValue,
	})
}

func addNumberConstraint(constraints *[]domain.RequirementConstraint, key string, value float64, unit string) {
	if hasConstraint(*constraints, key) {
		return
	}
	copyValue := value
	*constraints = append(*constraints, domain.RequirementConstraint{
		Key:       key,
		ValueType: domain.ValueTypeNumber,
		Operator:  domain.OperatorEqual,
		Number:    &copyValue,
		Unit:      unit,
	})
}

func hasConstraint(constraints []domain.RequirementConstraint, key string) bool {
	for _, constraint := range constraints {
		if constraint.Key == key {
			return true
		}
	}
	return false
}

func packageFromFootprint(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if match := packagePattern.FindString(trimmed); match != "" {
		return strings.ToUpper(match)
	}
	if idx := strings.LastIndex(trimmed, ":"); idx >= 0 {
		trimmed = trimmed[idx+1:]
	}
	trimmed = strings.TrimSuffix(trimmed, ".pretty")
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return ""
	}
	parts := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	if len(parts) == 0 {
		return trimmed
	}
	return parts[0]
}

func parsePatternNumber(pattern *regexp.Regexp, texts ...string) *float64 {
	for _, text := range texts {
		match := pattern.FindStringSubmatch(text)
		if len(match) < 2 {
			continue
		}
		value, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			continue
		}
		return &value
	}
	return nil
}

func findDielectric(texts ...string) string {
	for _, text := range texts {
		upper := strings.ToUpper(text)
		for _, candidate := range []string{"X7R", "X5R", "C0G", "NP0", "Y5V"} {
			if strings.Contains(upper, candidate) {
				return candidate
			}
		}
	}
	return ""
}

func parseResistorValueOK(raw string) bool {
	_, ok := parseResistorValue(raw)
	return ok
}

func parseCapacitanceValueOK(raw string) bool {
	_, ok := parseCapacitanceValue(raw)
	return ok
}

func parseInductanceValueOK(raw string) bool {
	_, ok := parseInductanceValue(raw)
	return ok
}

func parseResistorValue(raw string) (float64, bool) {
	normalized := normalizeValue(raw)
	normalized = strings.TrimSuffix(normalized, "OHMS")
	normalized = strings.TrimSuffix(normalized, "OHM")
	if normalized == "" {
		return 0, false
	}
	if strings.ContainsAny(normalized, "RKM") {
		return parseLetterDecimal(normalized, map[string]float64{"R": 1, "K": 1e3, "M": 1e6})
	}
	value, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

func parseCapacitanceValue(raw string) (float64, bool) {
	return parseSIValue(raw, []suffixMultiplier{{"PF", 1e-12}, {"NF", 1e-9}, {"UF", 1e-6}, {"MF", 1e-3}, {"F", 1}, {"P", 1e-12}, {"N", 1e-9}, {"U", 1e-6}, {"M", 1e-3}})
}

func parseInductanceValue(raw string) (float64, bool) {
	return parseSIValue(raw, []suffixMultiplier{{"NH", 1e-9}, {"UH", 1e-6}, {"MH", 1e-3}, {"H", 1}, {"N", 1e-9}, {"U", 1e-6}, {"M", 1e-3}})
}

type suffixMultiplier struct {
	suffix     string
	multiplier float64
}

func parseSIValue(raw string, suffixes []suffixMultiplier) (float64, bool) {
	normalized := normalizeValue(raw)
	if normalized == "" {
		return 0, false
	}
	sort.Slice(suffixes, func(i, j int) bool {
		return len(suffixes[i].suffix) > len(suffixes[j].suffix)
	})
	for _, item := range suffixes {
		if item.suffix == "" || !strings.HasSuffix(normalized, item.suffix) {
			continue
		}
		numberPart := strings.TrimSuffix(normalized, item.suffix)
		if numberPart == "" {
			continue
		}
		value, err := strconv.ParseFloat(numberPart, 64)
		if err != nil {
			continue
		}
		return value * item.multiplier, true
	}
	value, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

func parseLetterDecimal(raw string, multipliers map[string]float64) (float64, bool) {
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
			continue
		}
		return value * multiplier, true
	}
	return 0, false
}

func normalizeValue(raw string) string {
	replacer := strings.NewReplacer("µ", "u", "μ", "u", "Ω", "", " ", "")
	return strings.ToUpper(strings.TrimSpace(replacer.Replace(raw)))
}

func dedupeWarnings(warnings []string) []string {
	seen := make(map[string]struct{}, len(warnings))
	out := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		warning = strings.TrimSpace(warning)
		if warning == "" {
			continue
		}
		if _, ok := seen[warning]; ok {
			continue
		}
		seen[warning] = struct{}{}
		out = append(out, warning)
	}
	return out
}

func isTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y":
		return true
	default:
		return false
	}
}

func copyFields(row bomRow) map[string]string {
	fields := make(map[string]string, len(row.OtherFields)+4)
	for key, value := range row.OtherFields {
		fields[key] = value
	}
	if row.ManufacturerPart != "" {
		fields["manufacturerPartNumber"] = row.ManufacturerPart
	}
	if row.PartNumber != "" {
		fields["partNumber"] = row.PartNumber
	}
	if row.Datasheet != "" {
		fields["datasheet"] = row.Datasheet
	}
	if row.LCSC != "" {
		fields["lcsc"] = row.LCSC
	}
	return fields
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}
