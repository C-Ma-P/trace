package sourcing

import (
	"testing"

	"github.com/C-Ma-P/trace/internal/domain"
)

// --- LCSC mapping tests ---

func TestMapLCSCCategory_Passives(t *testing.T) {
	cases := []struct {
		catalog       string
		parentCatalog string
		want          domain.Category
	}{
		{"Resistors", "", domain.CategoryResistor},
		{"Chip Resistor - Surface Mount", "Resistors", domain.CategoryResistor},
		{"Capacitors", "", domain.CategoryCapacitor},
		{"Multilayer Ceramic Capacitors MLCC - SMD/SMT", "Capacitors", domain.CategoryCapacitor},
		{"Inductors (SMD)", "Inductors, Coils, Chokes", domain.CategoryInductor},
		{"Ferrite Bead", "", domain.CategoryFerriteBead},
		{"Ferrite Beads", "Inductors, Coils, Chokes", domain.CategoryFerriteBead},
	}
	for _, tc := range cases {
		got := MapLCSCCategory(tc.catalog, tc.parentCatalog)
		if got != tc.want {
			t.Errorf("MapLCSCCategory(%q, %q) = %q, want %q", tc.catalog, tc.parentCatalog, got, tc.want)
		}
	}
}

func TestMapLCSCCategory_Diodes(t *testing.T) {
	cases := []struct {
		catalog string
		want    domain.Category
	}{
		{"Diodes - General Purpose, Power, Switching", domain.CategoryDiode},
		{"Zener Diodes", domain.CategoryDiode},
		{"Schottky Diodes & Rectifiers", domain.CategoryDiode},
		{"TVS Diodes", domain.CategoryDiode},
		{"Rectifiers", domain.CategoryDiode},
		{"Light Emitting Diodes (LED)", domain.CategoryLED},
		{"LEDs", domain.CategoryLED},
	}
	for _, tc := range cases {
		got := MapLCSCCategory(tc.catalog, "")
		if got != tc.want {
			t.Errorf("MapLCSCCategory(%q, \"\") = %q, want %q", tc.catalog, got, tc.want)
		}
	}
}

func TestMapLCSCCategory_Transistors(t *testing.T) {
	cases := []struct {
		catalog string
		want    domain.Category
	}{
		{"MOSFETs", domain.CategoryTransistorMOSFET},
		{"Transistors - FETs, MOSFETs - Single", domain.CategoryTransistorMOSFET},
		{"Transistors - BJT", domain.CategoryTransistorBJT},
		{"Bipolar Transistors - Pre-Biased", domain.CategoryTransistorBJT},
	}
	for _, tc := range cases {
		got := MapLCSCCategory(tc.catalog, "")
		if got != tc.want {
			t.Errorf("MapLCSCCategory(%q, \"\") = %q, want %q", tc.catalog, got, tc.want)
		}
	}
}

func TestMapLCSCCategory_Regulators(t *testing.T) {
	cases := []struct {
		catalog string
		want    domain.Category
	}{
		{"Linear Regulators (LDO)", domain.CategoryRegulatorLinear},
		{"LDO Voltage Regulators", domain.CategoryRegulatorLinear},
		{"DC-DC Converters", domain.CategoryRegulatorSwitching},
		{"Switching Regulators", domain.CategoryRegulatorSwitching},
		{"Buck Converters", domain.CategoryRegulatorSwitching},
		{"Boost Converters", domain.CategoryRegulatorSwitching},
	}
	for _, tc := range cases {
		got := MapLCSCCategory(tc.catalog, "")
		if got != tc.want {
			t.Errorf("MapLCSCCategory(%q, \"\") = %q, want %q", tc.catalog, got, tc.want)
		}
	}
}

func TestMapLCSCCategory_ConnectorSwitchMisc(t *testing.T) {
	cases := []struct {
		catalog string
		want    domain.Category
	}{
		{"Connectors", domain.CategoryConnector},
		{"USB Connectors", domain.CategoryConnector},
		{"Headers & Wire Housings", domain.CategoryConnector},
		{"Terminal Block", domain.CategoryConnector},
		{"Pushbutton Switches", domain.CategorySwitch},
		{"DIP Switches", domain.CategorySwitch},
		{"Tact Switch", domain.CategorySwitch},
		{"Crystals", domain.CategoryCrystalOscillator},
		{"Crystal Oscillators", domain.CategoryCrystalOscillator},
		{"Resonators", domain.CategoryCrystalOscillator},
		{"Fuses", domain.CategoryFuse},
		{"Resettable Fuses", domain.CategoryFuse},
		{"Batteries", domain.CategoryBattery},
		{"Sensors", domain.CategorySensor},
		{"Modules", domain.CategoryModule},
	}
	for _, tc := range cases {
		got := MapLCSCCategory(tc.catalog, "")
		if got != tc.want {
			t.Errorf("MapLCSCCategory(%q, \"\") = %q, want %q", tc.catalog, got, tc.want)
		}
	}
}

func TestMapLCSCCategory_FallbackToParent(t *testing.T) {
	// When the leaf catalog doesn't match but the parent does, fall back to parent.
	got := MapLCSCCategory("SMD Package", "Resistors")
	if got != domain.CategoryResistor {
		t.Errorf("expected parent-level fallback to resistor, got %q", got)
	}
}

func TestMapLCSCCategory_UnknownFallsToIC(t *testing.T) {
	got := MapLCSCCategory("Some Weird Category", "Another Unknown")
	if got != domain.CategoryIntegratedCircuit {
		t.Errorf("expected integrated_circuit fallback, got %q", got)
	}
}

// --- Mouser mapping tests ---

func TestMapMouserCategory_Common(t *testing.T) {
	cases := []struct {
		category string
		want     domain.Category
	}{
		{"Resistors", domain.CategoryResistor},
		{"Capacitors", domain.CategoryCapacitor},
		{"Inductors / Chokes & Coils", domain.CategoryInductor},
		{"Ferrite Beads & Chips", domain.CategoryFerriteBead},
		{"Diodes & Rectifiers", domain.CategoryDiode},
		{"Zener Diodes", domain.CategoryDiode},
		{"Schottky Diodes & Rectifiers", domain.CategoryDiode},
		{"LEDs", domain.CategoryLED},
		{"Bipolar Transistors - BJT", domain.CategoryTransistorBJT},
		{"MOSFETs", domain.CategoryTransistorMOSFET},
		{"LDO Voltage Regulators", domain.CategoryRegulatorLinear},
		{"DC-DC Switching Regulators", domain.CategoryRegulatorSwitching},
		{"Connectors", domain.CategoryConnector},
		{"Headers & Wire Housings", domain.CategoryConnector},
		{"Pushbutton Switches", domain.CategorySwitch},
		{"Tactile Switches", domain.CategorySwitch},
		{"Crystals & Oscillators", domain.CategoryCrystalOscillator},
		{"PTC Resettable Fuses", domain.CategoryFuse},
		{"Batteries", domain.CategoryBattery},
		{"Sensors", domain.CategorySensor},
		{"Development Kits", domain.CategoryModule},
		{"Multi-layer Ceramic Capacitors (MLCC)", domain.CategoryCapacitor},
	}
	for _, tc := range cases {
		got := MapMouserCategory(tc.category)
		if got != tc.want {
			t.Errorf("MapMouserCategory(%q) = %q, want %q", tc.category, got, tc.want)
		}
	}
}

func TestMapMouserCategory_EmptyFallsToIC(t *testing.T) {
	if got := MapMouserCategory(""); got != domain.CategoryIntegratedCircuit {
		t.Errorf("expected integrated_circuit for empty string, got %q", got)
	}
}

func TestMapMouserCategory_UnknownFallsToIC(t *testing.T) {
	if got := MapMouserCategory("Optoelectronics"); got != domain.CategoryIntegratedCircuit {
		t.Errorf("expected integrated_circuit fallback, got %q", got)
	}
}

// --- MapOfferCategory dispatch tests ---

func TestMapOfferCategory_DispatchByProvider(t *testing.T) {
	lcscOffer := SupplierOffer{
		Provider: ProviderLCSC,
		Raw:      map[string]string{"catalog": "Capacitors", "parentCatalog": "Passive Components"},
	}
	if got := MapOfferCategory(lcscOffer); got != domain.CategoryCapacitor {
		t.Errorf("LCSC dispatch: expected capacitor, got %q", got)
	}

	mouserOffer := SupplierOffer{
		Provider: ProviderMouser,
		Raw:      map[string]string{"category": "MOSFETs"},
	}
	if got := MapOfferCategory(mouserOffer); got != domain.CategoryTransistorMOSFET {
		t.Errorf("Mouser dispatch: expected transistor_mosfet, got %q", got)
	}

	digikeyOffer := SupplierOffer{
		Provider: ProviderDigiKey,
		Raw:      map[string]string{"category": "Ferrite Beads & Chips"},
	}
	if got := MapOfferCategory(digikeyOffer); got != domain.CategoryFerriteBead {
		t.Errorf("DigiKey dispatch: expected ferrite_bead, got %q", got)
	}
}

func TestMapOfferCategory_NilRawFallsToIC(t *testing.T) {
	offer := SupplierOffer{Provider: ProviderLCSC, Raw: nil}
	if got := MapOfferCategory(offer); got != domain.CategoryIntegratedCircuit {
		t.Errorf("nil Raw: expected integrated_circuit, got %q", got)
	}
}
