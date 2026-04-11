package app

import (
	"fmt"

	"trace/internal/activity"
	"trace/internal/assetsearch"
	"trace/internal/domain"
	"trace/internal/domain/registry"
	"trace/internal/ingest"
	"trace/internal/launcher"
	"trace/internal/phoneintake"
	easyedaprovider "trace/internal/providers/easyeda"
	"trace/internal/service"
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
		{Value: string(domain.CategoryIntegratedCircuit), DisplayName: "Integrated Circuit"},
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
