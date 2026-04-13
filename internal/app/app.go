package app

import (
	"fmt"

	"github.com/C-Ma-P/trace/internal/activity"
	"github.com/C-Ma-P/trace/internal/assetsearch"
	"github.com/C-Ma-P/trace/internal/domain"
	"github.com/C-Ma-P/trace/internal/domain/registry"
	"github.com/C-Ma-P/trace/internal/ingest"
	"github.com/C-Ma-P/trace/internal/launcher"
	"github.com/C-Ma-P/trace/internal/phoneintake"
	easyedaprovider "github.com/C-Ma-P/trace/internal/providers/easyeda"
	"github.com/C-Ma-P/trace/internal/service"
)

type App struct {
	svc         *service.Service
	assetSearch *assetsearch.Service
	ingest      *ingest.Service
	easyeda     *easyedaprovider.Service
	launcher    *launcher.Store
	intake      *phoneintake.Server
	activityHub *activity.Hub
	bagRepo     domain.InventoryBagRepository
	initErr     string
}

func New(svc *service.Service, assetSearch *assetsearch.Service, ingestSvc *ingest.Service, easyedaSvc *easyedaprovider.Service) *App {
	return &App{svc: svc, assetSearch: assetSearch, ingest: ingestSvc, easyeda: easyedaSvc, launcher: launcher.NewStore()}
}

func NewFailed(errMsg string) *App {
	return &App{initErr: errMsg, launcher: launcher.NewStore()}
}

func (a *App) checkReady() error {
	if a.svc == nil {
		return fmt.Errorf("database not available: %s", a.initErr)
	}
	return nil
}

func (a *App) GetStartupStatus() StartupStatusResponse {
	if a.svc == nil {
		return StartupStatusResponse{Ready: false, Error: a.initErr}
	}
	return StartupStatusResponse{Ready: true}
}

func (a *App) GetCategories() []CategoryInfo {
	return []CategoryInfo{
		{Value: string(domain.CategoryResistor), DisplayName: "Resistor"},
		{Value: string(domain.CategoryCapacitor), DisplayName: "Capacitor"},
		{Value: string(domain.CategoryInductor), DisplayName: "Inductor"},
		{Value: string(domain.CategoryFerriteBead), DisplayName: "Ferrite Bead"},
		{Value: string(domain.CategoryDiode), DisplayName: "Diode"},
		{Value: string(domain.CategoryLED), DisplayName: "LED"},
		{Value: string(domain.CategoryTransistorBJT), DisplayName: "BJT"},
		{Value: string(domain.CategoryTransistorMOSFET), DisplayName: "MOSFET"},
		{Value: string(domain.CategoryRegulatorLinear), DisplayName: "Linear Regulator"},
		{Value: string(domain.CategoryRegulatorSwitching), DisplayName: "Switching Regulator"},
		{Value: string(domain.CategoryIntegratedCircuit), DisplayName: "IC"},
		{Value: string(domain.CategoryConnector), DisplayName: "Connector"},
		{Value: string(domain.CategorySwitch), DisplayName: "Switch"},
		{Value: string(domain.CategoryCrystalOscillator), DisplayName: "Crystal / Osc."},
		{Value: string(domain.CategoryFuse), DisplayName: "Fuse"},
		{Value: string(domain.CategoryBattery), DisplayName: "Battery"},
		{Value: string(domain.CategorySensor), DisplayName: "Sensor"},
		{Value: string(domain.CategoryModule), DisplayName: "Module"},
	}
}

func (a *App) GetCategoryDefinitions(category string) []AttributeDefinitionInfo {
	defs := registry.DefinitionsForCategory(domain.Category(category))
	out := make([]AttributeDefinitionInfo, len(defs))
	for i, d := range defs {
		unit := ""
		if d.Unit != nil {
			unit = *d.Unit
		}
		out[i] = AttributeDefinitionInfo{
			Key:         d.Key,
			DisplayName: d.DisplayName,
			ValueType:   string(d.ValueType),
			Unit:        unit,
		}
	}
	return out
}

func (a *App) GetRequirementDefinitions(category string) []AttributeDefinitionInfo {
	defs := registry.ConstraintDefinitionsForCategory(domain.Category(category))
	out := make([]AttributeDefinitionInfo, len(defs))
	for i, d := range defs {
		unit := ""
		if d.Unit != nil {
			unit = *d.Unit
		}
		out[i] = AttributeDefinitionInfo{
			Key:         d.Key,
			DisplayName: d.DisplayName,
			ValueType:   string(d.ValueType),
			Unit:        unit,
		}
	}
	return out
}

func (a *App) GetOperatorsForValueType(valueType string) []OperatorInfo {
	switch domain.ValueType(valueType) {
	case domain.ValueTypeNumber:
		return []OperatorInfo{
			{Value: string(domain.OperatorEqual), DisplayName: "equals"},
			{Value: string(domain.OperatorGTE), DisplayName: "≥"},
			{Value: string(domain.OperatorLTE), DisplayName: "≤"},
		}
	case domain.ValueTypeText:
		return []OperatorInfo{
			{Value: string(domain.OperatorEqual), DisplayName: "equals"},
		}
	case domain.ValueTypeBool:
		return []OperatorInfo{
			{Value: string(domain.OperatorEqual), DisplayName: "equals"},
		}
	default:
		return []OperatorInfo{}
	}
}
