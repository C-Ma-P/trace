package sourcing

import (
	"strings"

	"github.com/C-Ma-P/trace/internal/domain"
	"github.com/C-Ma-P/trace/internal/domain/registry"
	"github.com/C-Ma-P/trace/internal/electronics/specparse"
)

func ComponentAttributesFromOffer(category domain.Category, offer SupplierOffer) []domain.AttributeValue {
	out := make([]domain.AttributeValue, 0, 8)
	seen := make(map[string]struct{}, 8)
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
		if value, ok := specparse.ParseResistance(firstNonEmpty(
			specs.firstResistance(),
			offer.Description,
		)); ok {
			addAttribute(numberAttribute(registry.AttrResistanceOhms, value, "ohm"))
		}
		if value, ok := specparse.ParseTolerancePercent(firstNonEmpty(
			specs.firstFuzzy(
				[]string{"Tolerance", "Resistance Tolerance"},
				[]string{"tolerance"},
			),
			offer.Description,
		)); ok {
			addAttribute(numberAttribute(registry.AttrTolerancePercent, value, "percent"))
		}
		if value, ok := specparse.ParsePower(firstNonEmpty(
			specs.firstFuzzy(
				[]string{"Power Rating", "Power (Watts)", "Power", "Wattage"},
				[]string{"power", "watt"},
			),
			offer.Description,
		)); ok {
			addAttribute(numberAttribute(registry.AttrPowerW, value, "W"))
		}
		if value, ok := specparse.ParseTempcoPPMC(firstNonEmpty(
			specs.firstFuzzy(
				[]string{"Temperature Coefficient", "Temp Coefficient", "TC", "TCR"},
				[]string{"temperaturecoefficient", "tempcoefficient", "tempco", "tcr", "ppm"},
			),
			offer.Description,
		)); ok {
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
	case domain.CategoryCapacitor:
		if value, ok := specparse.ParseCapacitance(firstNonEmpty(
			specs.firstFuzzy(
				[]string{"Capacitance", "Value", "Capacitance Value"},
				[]string{"capacitance", "capvalue"},
			),
			offer.Description,
		)); ok {
			addAttribute(numberAttribute(registry.AttrCapacitanceF, value, "F"))
		}
		if value, ok := specparse.ParseTolerancePercent(firstNonEmpty(
			specs.firstFuzzy(
				[]string{"Tolerance", "Capacitance Tolerance"},
				[]string{"tolerance"},
			),
			offer.Description,
		)); ok {
			addAttribute(numberAttribute(registry.AttrTolerancePercent, value, "percent"))
		}
		if value, ok := specparse.ParseVoltage(firstNonEmpty(
			specs.firstFuzzy(
				[]string{"Voltage - Rated", "Voltage Rated", "Voltage Rating", "Voltage Rating (DC)", "Rated Voltage", "Voltage"},
				[]string{"voltagerated", "voltagerating", "ratedvoltage", "voltage"},
			),
			offer.Description,
		)); ok {
			addAttribute(numberAttribute(registry.AttrVoltageV, value, "V"))
		}
		if dielectric := firstNonEmpty(
			specs.firstFuzzy(
				[]string{"Temperature Coefficient", "Dielectric", "Dielectric Material", "Class"},
				[]string{"temperaturecoefficient", "dielectric", "class"},
			),
			parseCapacitorDielectricFromDescription(offer.Description),
		); dielectric != "" {
			addAttribute(textAttribute(registry.AttrDielectric, dielectric))
		}
		if capacitorType := firstNonEmpty(
			specs.firstFuzzy(
				[]string{"Capacitor Type", "Type"},
				[]string{"capacitortype"},
			),
			parseCapacitorTypeFromDescription(offer.Description),
		); capacitorType != "" {
			addAttribute(textAttribute(registry.AttrCapacitorType, capacitorType))
		}
	case domain.CategoryInductor:
		if value, ok := specparse.ParseInductance(firstNonEmpty(
			specs.firstFuzzy(
				[]string{"Inductance", "Value"},
				[]string{"inductance"},
			),
			offer.Description,
		)); ok {
			addAttribute(numberAttribute(registry.AttrInductanceH, value, "H"))
		}
		if value, ok := specparse.ParseTolerancePercent(firstNonEmpty(
			specs.firstFuzzy(
				[]string{"Tolerance"},
				[]string{"tolerance"},
			),
			offer.Description,
		)); ok {
			addAttribute(numberAttribute(registry.AttrTolerancePercent, value, "percent"))
		}
		if value, ok := specparse.ParseCurrent(firstNonEmpty(
			specs.firstFuzzy(
				[]string{"Current Rating (Amps)", "Current Rating", "Rated Current", "Current - Rated", "Current - Saturation (Isat)", "Saturation Current (Isat)"},
				[]string{"currentrating", "ratedcurrent", "currentrated", "currentsaturation", "saturationcurrent", "isat"},
			),
			offer.Description,
		)); ok {
			addAttribute(numberAttribute(registry.AttrCurrentA, value, "A"))
		}
		if value, ok := specparse.ParseResistance(firstNonEmpty(
			specs.firstFuzzy(
				[]string{"DC Resistance (DCR)", "DC Resistance", "DCR"},
				[]string{"dcresistance", "dcr"},
			),
		)); ok {
			addAttribute(numberAttribute(registry.AttrDCROhms, value, "ohm"))
		}
		if inductorType := firstNonEmpty(
			specs.firstFuzzy(
				[]string{"Inductor Type", "Type"},
				[]string{"inductortype"},
			),
			parseInductorTypeFromDescription(offer.Description),
		); inductorType != "" {
			addAttribute(textAttribute(registry.AttrInductorType, inductorType))
		}
	case domain.CategoryFerriteBead:
		if value, ok := specparse.ParseResistance(firstNonEmpty(
			specs.firstFuzzy(
				[]string{"Impedance @ 100MHz", "Impedance @ 100 MHz", "Impedance @ Frequency", "Impedance"},
				[]string{"impedance"},
			),
			offer.Description,
		)); ok {
			addAttribute(numberAttribute(registry.AttrImpedanceOhms, value, "ohm"))
		}
		if value, ok := specparse.ParseCurrent(firstNonEmpty(
			specs.firstFuzzy(
				[]string{"Current Rating (Max)", "Current Rating", "Rated Current", "Current"},
				[]string{"currentrating", "ratedcurrent", "current"},
			),
			offer.Description,
		)); ok {
			addAttribute(numberAttribute(registry.AttrCurrentA, value, "A"))
		}
		if value, ok := specparse.ParseResistance(firstNonEmpty(
			specs.firstFuzzy(
				[]string{"DC Resistance (DCR)", "DC Resistance", "DCR", "Resistance"},
				[]string{"dcresistance", "dcr"},
			),
		)); ok {
			addAttribute(numberAttribute(registry.AttrDCROhm, value, "ohm"))
		}
	}

	if packageName := firstNonEmpty(
		specs.firstFuzzy(
			[]string{"Package / Case", "Supplier Device Package", "Package", "EncapStandard", "Case Code - in", "Case Code - mm"},
			[]string{"packagecase", "supplierdevicepackage", "package", "encapstandard", "casecode"},
		),
		strings.TrimSpace(offer.Package),
		specparse.ParsePackage(offer.Description),
	); packageName != "" {
		addAttribute(textAttribute(registry.AttrPackage, specparse.NormalizePackage(packageName)))
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

func parseResistorTypeFromDescription(raw string) string {
	upper := strings.ToUpper(raw)
	for _, candidate := range []string{"THICK FILM", "THIN FILM", "METAL FILM", "CARBON FILM", "WIREWOUND"} {
		if strings.Contains(upper, candidate) {
			return titleWords(candidate)
		}
	}
	return ""
}

func parseCapacitorDielectricFromDescription(raw string) string {
	upper := strings.ToUpper(raw)
	for _, candidate := range []string{"C0G", "NP0", "X5R", "X7R", "Y5V"} {
		if strings.Contains(upper, candidate) {
			return candidate
		}
	}
	return ""
}

func parseCapacitorTypeFromDescription(raw string) string {
	upper := strings.ToUpper(raw)
	for _, candidate := range []struct {
		needle string
		value  string
	}{
		{needle: "MLCC", value: "MLCC"},
		{needle: "CERAMIC", value: "Ceramic"},
		{needle: "TANTALUM", value: "Tantalum"},
		{needle: "ELECTROLYTIC", value: "Electrolytic"},
	} {
		if strings.Contains(upper, candidate.needle) {
			return candidate.value
		}
	}
	return ""
}

func parseInductorTypeFromDescription(raw string) string {
	upper := strings.ToUpper(raw)
	for _, candidate := range []struct {
		needle string
		value  string
	}{
		{needle: "SHIELDED", value: "Shielded"},
		{needle: "UNSHIELDED", value: "Unshielded"},
		{needle: "WIREWOUND", value: "Wirewound"},
		{needle: "POWER INDUCTOR", value: "Power Inductor"},
	} {
		if strings.Contains(upper, candidate.needle) {
			return candidate.value
		}
	}
	return ""
}

func titleWords(raw string) string {
	parts := strings.Fields(strings.ToLower(raw))
	for i := range parts {
		if len(parts[i]) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, " ")
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
