package sourcing

import (
	"strings"

	"github.com/C-Ma-P/trace/internal/domain"
)

// MapOfferCategory derives a canonical Trace category from a SupplierOffer.
// It delegates to the appropriate provider-specific mapper using the raw
// catalog/category strings preserved in offer.Raw during normalization.
// The raw supplier strings are never overwritten, so they remain available
// for debugging and future remapping.
func MapOfferCategory(offer SupplierOffer) domain.Category {
	switch offer.Provider {
	case ProviderLCSC:
		return MapLCSCCategory(offer.Raw["catalog"], offer.Raw["parentCatalog"])
	case ProviderMouser:
		return MapMouserCategory(offer.Raw["category"])
	case ProviderDigiKey:
		return MapDigiKeyCategory(offer.Raw["category"])
	default:
		return domain.CategoryIntegratedCircuit
	}
}

// MapLCSCCategory maps LCSC catalog and parentCatalog strings to a canonical
// Trace category. Both strings are checked (catalog first, then parent) to
// maximise hit-rate against LCSC's two-level taxonomy.
//
// Intentionally coarse mappings are noted with comments. When in doubt the
// mapper prefers a broader correct category over a narrow wrong one.
func MapLCSCCategory(catalog, parentCatalog string) domain.Category {
	// Check catalog (leaf) first; it is more specific than parentCatalog.
	if cat, ok := lcscCategoryLookup(catalog); ok {
		return cat
	}
	if cat, ok := lcscCategoryLookup(parentCatalog); ok {
		return cat
	}
	// Broad fallback. Not everything uncategorised is truly an IC, but it is
	// the least-wrong default for unknown digital/analog parts.
	return domain.CategoryIntegratedCircuit
}

func lcscCategoryLookup(s string) (domain.Category, bool) {
	lower := strings.ToLower(strings.TrimSpace(s))
	if lower == "" {
		return "", false
	}
	switch {
	// ---- passives ----
	case contains(lower, "resistor"):
		return domain.CategoryResistor, true
	case contains(lower, "capacitor"):
		return domain.CategoryCapacitor, true
	case contains(lower, "ferrite"):
		return domain.CategoryFerriteBead, true
	case contains(lower, "inductor") || contains(lower, "choke") || contains(lower, "coil"):
		return domain.CategoryInductor, true

	// ---- diodes / LEDs ----
	case contains(lower, "light emitting") || contains(lower, " led") || lower == "leds":
		return domain.CategoryLED, true
	case contains(lower, "diode") || contains(lower, "rectifier") || contains(lower, "zener") ||
		contains(lower, "schottky") || contains(lower, "tvs"):
		return domain.CategoryDiode, true

	// ---- transistors ----
	case contains(lower, "mosfet") || contains(lower, "jfet"):
		return domain.CategoryTransistorMOSFET, true
	case contains(lower, "bjt") || contains(lower, "bipolar") ||
		(contains(lower, "transistor") && !contains(lower, "mosfet")):
		return domain.CategoryTransistorBJT, true

	// ---- regulators ----
	// Switching regulator strings are checked before "regulator" to avoid
	// linear-regulator being matched on a switching part.
	case contains(lower, "dc-dc") || contains(lower, "dc/dc") ||
		contains(lower, "buck") || contains(lower, "boost") ||
		contains(lower, "switching regulator") || contains(lower, "switching converter"):
		return domain.CategoryRegulatorSwitching, true
	case contains(lower, "ldo") || contains(lower, "linear regulator") ||
		(contains(lower, "regulator") && !contains(lower, "switching")):
		return domain.CategoryRegulatorLinear, true

	// ---- connectors / switches ----
	case contains(lower, "connector") || contains(lower, "header") ||
		contains(lower, "terminal block") || contains(lower, "socket") ||
		contains(lower, "plug") || contains(lower, "jack") ||
		contains(lower, "usb") || contains(lower, "wire housing"):
		return domain.CategoryConnector, true
	case contains(lower, "switch") || contains(lower, "pushbutton") ||
		contains(lower, "tact") || contains(lower, "dip switch") ||
		contains(lower, "slide switch") || contains(lower, "rocker"):
		return domain.CategorySwitch, true

	// ---- crystals / oscillators ----
	case contains(lower, "crystal") || contains(lower, "oscillator") || contains(lower, "resonator"):
		return domain.CategoryCrystalOscillator, true

	// ---- fuses ----
	case contains(lower, "fuse") || contains(lower, "polyfuse") || contains(lower, "resettable"):
		return domain.CategoryFuse, true

	// ---- batteries ----
	// "battery holder" is included because LCSC often lists holders under the
	// Batteries parent; holders without an actual cell are a known edge-case
	// (intentional coarse mapping — better narrow later if needed).
	case contains(lower, "batter"):
		return domain.CategoryBattery, true

	// ---- sensors ----
	case contains(lower, "sensor") || contains(lower, "detector") || contains(lower, "transducer"):
		return domain.CategorySensor, true

	// ---- modules ----
	case contains(lower, "module") || contains(lower, "dev board") ||
		contains(lower, "breakout") || contains(lower, "evaluation"):
		return domain.CategoryModule, true
	}
	return "", false
}

// MapMouserCategory maps a Mouser category string to a canonical Trace
// category.  Mouser uses a flat category string (e.g. "Bipolar Transistors -
// BJT") populated in offer.Raw["category"] during normalization.
//
// Intentionally coarse: any ambiguous Mouser string is mapped to
// integrated_circuit rather than guessing at a sub-type.
func MapMouserCategory(category string) domain.Category {
	lower := strings.ToLower(strings.TrimSpace(category))
	if lower == "" {
		return domain.CategoryIntegratedCircuit
	}
	switch {
	// ---- passives ----
	case contains(lower, "resistor"):
		return domain.CategoryResistor
	case contains(lower, "capacitor"):
		return domain.CategoryCapacitor
	case contains(lower, "ferrite"):
		return domain.CategoryFerriteBead
	case contains(lower, "inductor") || contains(lower, "choke") || contains(lower, "coil"):
		return domain.CategoryInductor

	// ---- diodes / LEDs ----
	case contains(lower, "led") || contains(lower, "light emitting"):
		return domain.CategoryLED
	case contains(lower, "diode") || contains(lower, "rectifier") ||
		contains(lower, "zener") || contains(lower, "schottky") || contains(lower, "tvs"):
		return domain.CategoryDiode

	// ---- transistors ----
	case contains(lower, "mosfet") || contains(lower, "jfet"):
		return domain.CategoryTransistorMOSFET
	case contains(lower, "bjt") || contains(lower, "bipolar transistor"):
		return domain.CategoryTransistorBJT

	// ---- regulators ----
	case contains(lower, "dc-dc") || contains(lower, "dc/dc") ||
		contains(lower, "switching regulator") || contains(lower, "switching voltage") ||
		contains(lower, "buck") || contains(lower, "boost"):
		return domain.CategoryRegulatorSwitching
	case contains(lower, "ldo") || contains(lower, "linear regulator"):
		return domain.CategoryRegulatorLinear

	// ---- connectors / switches ----
	case contains(lower, "connector") || contains(lower, "header") ||
		contains(lower, "terminal block") || contains(lower, "socket") ||
		contains(lower, "plug") || contains(lower, "jack") || contains(lower, "usb"):
		return domain.CategoryConnector
	case contains(lower, "switch") || contains(lower, "pushbutton") ||
		contains(lower, "tactile") || contains(lower, "dip switch"):
		return domain.CategorySwitch

	// ---- crystals / oscillators ----
	case contains(lower, "crystal") || contains(lower, "oscillator") || contains(lower, "resonator"):
		return domain.CategoryCrystalOscillator

	// ---- fuses ----
	case contains(lower, "fuse") || contains(lower, "ptc resettable"):
		return domain.CategoryFuse

	// ---- batteries ----
	case contains(lower, "batter"):
		return domain.CategoryBattery

	// ---- sensors ----
	case contains(lower, "sensor") || contains(lower, "detector") || contains(lower, "transducer"):
		return domain.CategorySensor

	// ---- modules ----
	case contains(lower, "module") || contains(lower, "development kit") ||
		contains(lower, "evaluation board") || contains(lower, "breakout"):
		return domain.CategoryModule

	default:
		// Intentional broad fallback. Most uncategorised Mouser parts are ICs.
		return domain.CategoryIntegratedCircuit
	}
}

// MapDigiKeyCategory maps a DigiKey category string to a canonical Trace
// category. DigiKey category strings follow a similar pattern to Mouser and
// are stored in offer.Raw["category"].
func MapDigiKeyCategory(category string) domain.Category {
	// DigiKey uses similar naming conventions to Mouser; delegate to the same
	// logic rather than duplicating the switch. Where DigiKey differs we fall
	// through to integrated_circuit, which is an intentional coarse mapping.
	return MapMouserCategory(category)
}

// contains is a convenience wrapper so the switch cases above stay concise.
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
