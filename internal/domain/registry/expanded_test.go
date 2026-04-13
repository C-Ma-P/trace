package registry_test

import (
	"testing"

	"github.com/C-Ma-P/trace/internal/domain"
	"github.com/C-Ma-P/trace/internal/domain/registry"
)

// TestCategories_ContainsAllExpanded verifies that all newly-added categories
// are present in the result from Categories().
func TestCategories_ContainsAllExpanded(t *testing.T) {
	cats := registry.Categories()
	found := make(map[domain.Category]bool, len(cats))
	for _, c := range cats {
		found[c] = true
	}
	want := []domain.Category{
		domain.CategoryResistor,
		domain.CategoryCapacitor,
		domain.CategoryInductor,
		domain.CategoryFerriteBead,
		domain.CategoryDiode,
		domain.CategoryLED,
		domain.CategoryTransistorBJT,
		domain.CategoryTransistorMOSFET,
		domain.CategoryRegulatorLinear,
		domain.CategoryRegulatorSwitching,
		domain.CategoryConnector,
		domain.CategorySwitch,
		domain.CategoryCrystalOscillator,
		domain.CategoryFuse,
		domain.CategoryBattery,
		domain.CategorySensor,
		domain.CategoryModule,
	}
	for _, w := range want {
		if !found[w] {
			t.Errorf("Categories() missing %q", w)
		}
	}
}

func TestValidateAttributes_FerriteBead(t *testing.T) {
	attrs := []domain.AttributeValue{
		numAttr(registry.AttrImpedanceOhms, 600, "ohm"),
		numAttr(registry.AttrCurrentA, 3.0, "A"),
		numAttr(registry.AttrDCROhm, 0.08, "ohm"),
		textAttr(registry.AttrPackage, "0402"),
	}
	if err := registry.ValidateAttributes(domain.CategoryFerriteBead, attrs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAttributes_Diode(t *testing.T) {
	attrs := []domain.AttributeValue{
		textAttr(registry.AttrPackage, "SOD-323"),
		textAttr(registry.AttrDiodeType, "schottky"),
		numAttr(registry.AttrReverseVoltageV, 40, "V"),
		numAttr(registry.AttrForwardCurrentA, 0.2, "A"),
		numAttr(registry.AttrForwardVoltageV, 0.3, "V"),
	}
	if err := registry.ValidateAttributes(domain.CategoryDiode, attrs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAttributes_LED(t *testing.T) {
	attrs := []domain.AttributeValue{
		textAttr(registry.AttrPackage, "0805"),
		textAttr(registry.AttrLEDColor, "red"),
		numAttr(registry.AttrForwardVoltageV, 2.0, "V"),
		numAttr(registry.AttrForwardCurrentA, 0.02, "A"),
	}
	if err := registry.ValidateAttributes(domain.CategoryLED, attrs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAttributes_TransistorBJT(t *testing.T) {
	attrs := []domain.AttributeValue{
		textAttr(registry.AttrPackage, "SOT-23"),
		textAttr(registry.AttrBJTType, "NPN"),
		numAttr(registry.AttrCollectorEmitterVoltageV, 40, "V"),
		numAttr(registry.AttrCollectorCurrentA, 0.2, "A"),
		numAttr(registry.AttrPowerW, 0.25, "W"),
	}
	if err := registry.ValidateAttributes(domain.CategoryTransistorBJT, attrs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAttributes_TransistorMOSFET(t *testing.T) {
	attrs := []domain.AttributeValue{
		textAttr(registry.AttrPackage, "SOT-23"),
		textAttr(registry.AttrMOSFETChannel, "N-channel"),
		numAttr(registry.AttrDrainSourceVoltageV, 30, "V"),
		numAttr(registry.AttrContinuousDrainCurrentA, 5.0, "A"),
		numAttr(registry.AttrRdsOnOhm, 0.012, "ohm"),
	}
	if err := registry.ValidateAttributes(domain.CategoryTransistorMOSFET, attrs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAttributes_RegulatorLinear(t *testing.T) {
	attrs := []domain.AttributeValue{
		textAttr(registry.AttrPackage, "SOT-223"),
		numAttr(registry.AttrOutputVoltageV, 3.3, "V"),
		numAttr(registry.AttrOutputCurrentA, 1.0, "A"),
		numAttr(registry.AttrDropoutV, 0.5, "V"),
		textAttr(registry.AttrPolarity, "positive"),
	}
	if err := registry.ValidateAttributes(domain.CategoryRegulatorLinear, attrs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAttributes_RegulatorSwitching(t *testing.T) {
	attrs := []domain.AttributeValue{
		textAttr(registry.AttrPackage, "SOIC-8"),
		textAttr(registry.AttrTopology, "buck"),
		numAttr(registry.AttrOutputVoltageV, 5.0, "V"),
		numAttr(registry.AttrOutputCurrentA, 2.0, "A"),
		numAttr(registry.AttrInputVoltageMinV, 4.5, "V"),
		numAttr(registry.AttrInputVoltageMaxV, 28.0, "V"),
	}
	if err := registry.ValidateAttributes(domain.CategoryRegulatorSwitching, attrs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAttributes_Connector(t *testing.T) {
	attrs := []domain.AttributeValue{
		textAttr(registry.AttrConnectorType, "pin header"),
		numAttr(registry.AttrPositions, 10, ""),
		numAttr(registry.AttrPitchMM, 2.54, "mm"),
		textAttr(registry.AttrMountingType, "through-hole"),
	}
	if err := registry.ValidateAttributes(domain.CategoryConnector, attrs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAttributes_Switch(t *testing.T) {
	attrs := []domain.AttributeValue{
		textAttr(registry.AttrSwitchType, "tact"),
		numAttr(registry.AttrPoles, 1, ""),
		numAttr(registry.AttrThrows, 1, ""),
		textAttr(registry.AttrMountingType, "SMD"),
	}
	if err := registry.ValidateAttributes(domain.CategorySwitch, attrs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAttributes_CrystalOscillator(t *testing.T) {
	attrs := []domain.AttributeValue{
		textAttr(registry.AttrPackage, "3225-4P"),
		numAttr(registry.AttrFrequencyHz, 16e6, "Hz"),
		numAttr(registry.AttrLoadCapacitancePF, 12, "pF"),
		numAttr(registry.AttrTolerancePPM, 30, "ppm"),
	}
	if err := registry.ValidateAttributes(domain.CategoryCrystalOscillator, attrs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAttributes_Fuse(t *testing.T) {
	attrs := []domain.AttributeValue{
		textAttr(registry.AttrPackage, "1206"),
		numAttr(registry.AttrCurrentA, 0.5, "A"),
		numAttr(registry.AttrVoltageV, 32, "V"),
		textAttr(registry.AttrFuseType, "slow blow"),
	}
	if err := registry.ValidateAttributes(domain.CategoryFuse, attrs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAttributes_Battery(t *testing.T) {
	attrs := []domain.AttributeValue{
		textAttr(registry.AttrBatteryChemistry, "Li-ion"),
		numAttr(registry.AttrNominalVoltageV, 3.7, "V"),
		numAttr(registry.AttrCapacityAh, 2.0, "Ah"),
		textAttr(registry.AttrPackage, "18650"),
	}
	if err := registry.ValidateAttributes(domain.CategoryBattery, attrs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAttributes_Sensor(t *testing.T) {
	attrs := []domain.AttributeValue{
		textAttr(registry.AttrSensorType, "temperature"),
		textAttr(registry.AttrPackage, "SOT-23"),
		textAttr(registry.AttrInterfaceType, "I2C"),
		numAttr(registry.AttrSupplyVoltageMinV, 1.8, "V"),
		numAttr(registry.AttrSupplyVoltageMaxV, 5.5, "V"),
	}
	if err := registry.ValidateAttributes(domain.CategorySensor, attrs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAttributes_Module(t *testing.T) {
	attrs := []domain.AttributeValue{
		textAttr(registry.AttrModuleType, "WiFi"),
		numAttr(registry.AttrSupplyVoltageMinV, 3.0, "V"),
		numAttr(registry.AttrSupplyVoltageMaxV, 3.6, "V"),
	}
	if err := registry.ValidateAttributes(domain.CategoryModule, attrs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestValidateConstraints_NewCategories spot-checks that constraint definitions
// are wired up for the new categories via the requirementCanonical map.
func TestValidateConstraints_NewCategories(t *testing.T) {
	cases := []struct {
		cat domain.Category
		key string
	}{
		{domain.CategoryFerriteBead, registry.AttrImpedanceOhms},
		{domain.CategoryDiode, registry.AttrReverseVoltageV},
		{domain.CategoryLED, registry.AttrLEDColor},
		{domain.CategoryTransistorBJT, registry.AttrCollectorCurrentA},
		{domain.CategoryTransistorMOSFET, registry.AttrRdsOnOhm},
		{domain.CategoryRegulatorLinear, registry.AttrOutputVoltageV},
		{domain.CategoryRegulatorSwitching, registry.AttrTopology},
		{domain.CategoryConnector, registry.AttrConnectorType},
		{domain.CategorySwitch, registry.AttrSwitchType},
		{domain.CategoryCrystalOscillator, registry.AttrFrequencyHz},
		{domain.CategoryFuse, registry.AttrFuseType},
		{domain.CategoryBattery, registry.AttrBatteryChemistry},
		{domain.CategorySensor, registry.AttrSensorType},
		{domain.CategoryModule, registry.AttrModuleType},
	}
	for _, tc := range cases {
		def, ok := registry.LookupConstraintDefinition(tc.cat, tc.key)
		if !ok {
			t.Errorf("LookupConstraintDefinition(%q, %q): not found", tc.cat, tc.key)
			continue
		}
		if def.Category != tc.cat {
			t.Errorf("LookupConstraintDefinition(%q, %q): wrong category %q", tc.cat, tc.key, def.Category)
		}
	}
}
