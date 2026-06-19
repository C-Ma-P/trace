package service

import (
	"context"
	"strings"

	"github.com/C-Ma-P/trace/internal/domain"
	easyedaprovider "github.com/C-Ma-P/trace/internal/providers/easyeda"
	"github.com/C-Ma-P/trace/internal/sourcing"
)

type easyEDAImporter interface {
	ImportComponentAssets(context.Context, easyedaprovider.ImportRequest) (easyedaprovider.ImportResult, error)
}

func SavedSupplierOfferFromSupplierOffer(offer sourcing.SupplierOffer) domain.SavedSupplierOffer {
	return domain.SavedSupplierOffer{
		Provider:        offer.Provider,
		ProviderPartID:  offer.SupplierPartNumber,
		ProductURL:      offer.ProductURL,
		ImageURL:        offer.ImageURL,
		DatasheetURL:    offer.DatasheetURL,
		HasSymbol:       offer.HasSymbol,
		HasFootprint:    offer.HasFootprint,
		HasDatasheet:    offer.HasDatasheet,
		Manufacturer:    offer.Manufacturer,
		MPN:             offer.MPN,
		Description:     offer.Description,
		Package:         offer.Package,
		Stock:           offer.Stock,
		MOQ:             offer.MOQ,
		UnitPrice:       offer.UnitPrice,
		Lifecycle:       offer.Lifecycle,
		Raw:             offer.Raw,
		AssetProbeState: string(offer.AssetProbeState),
		AssetProbeError: offer.AssetProbeError,
	}
}

func SupplierOfferFromSavedSupplierOffer(offer domain.SavedSupplierOffer) sourcing.SupplierOffer {
	return sourcing.SupplierOffer{
		Provider:           offer.Provider,
		Manufacturer:       offer.Manufacturer,
		MPN:                offer.MPN,
		SupplierPartNumber: offer.ProviderPartID,
		Description:        offer.Description,
		Package:            offer.Package,
		Stock:              offer.Stock,
		MOQ:                offer.MOQ,
		UnitPrice:          offer.UnitPrice,
		ProductURL:         offer.ProductURL,
		DatasheetURL:       offer.DatasheetURL,
		ImageURL:           offer.ImageURL,
		Lifecycle:          offer.Lifecycle,
		HasSymbol:          offer.HasSymbol,
		HasFootprint:       offer.HasFootprint,
		HasDatasheet:       offer.HasDatasheet,
		AssetProbeState:    sourcing.AssetProbeState(offer.AssetProbeState),
		AssetProbeError:    offer.AssetProbeError,
		Raw:                offer.Raw,
	}
}

func MergeSavedSupplierOfferFromSupplierOffer(dst *domain.SavedSupplierOffer, offer sourcing.SupplierOffer) {
	if dst == nil {
		return
	}
	merged := SavedSupplierOfferFromSupplierOffer(offer)
	dst.Provider = merged.Provider
	dst.ProviderPartID = merged.ProviderPartID
	dst.ProductURL = merged.ProductURL
	dst.ImageURL = merged.ImageURL
	dst.DatasheetURL = merged.DatasheetURL
	dst.HasSymbol = merged.HasSymbol
	dst.HasFootprint = merged.HasFootprint
	dst.HasDatasheet = merged.HasDatasheet
	dst.Manufacturer = merged.Manufacturer
	dst.MPN = merged.MPN
	dst.Description = merged.Description
	dst.Package = merged.Package
	dst.Stock = merged.Stock
	dst.MOQ = merged.MOQ
	dst.UnitPrice = merged.UnitPrice
	dst.Lifecycle = merged.Lifecycle
	dst.Raw = merged.Raw
	dst.AssetProbeState = merged.AssetProbeState
	dst.AssetProbeError = merged.AssetProbeError
}

func (s *Service) resolveComponentFromSupplierOffer(ctx context.Context, category domain.Category, offer sourcing.SupplierOffer) (domain.Component, bool, error) {
	attrs := sourcing.ComponentAttributesFromOffer(category, offer)
	packageName := firstSupplierPackage(attrs, offer.Package)
	component, reused, err := s.findOrCreateComponentWithSupplierAttrs(ctx, category, offer.Manufacturer, offer.MPN, packageName, offer.Description, attrs)
	if err != nil {
		return domain.Component{}, false, err
	}
	component, err = s.maybeImportSupplierAssets(ctx, component, offer)
	if err != nil {
		return domain.Component{}, false, err
	}
	return component, reused, nil
}

func (s *Service) maybeImportSupplierAssets(ctx context.Context, component domain.Component, offer sourcing.SupplierOffer) (domain.Component, error) {
	component, err := s.maybeAttachSupplierDatasheet(ctx, component, offer)
	if err != nil {
		return domain.Component{}, err
	}
	if err := s.maybeImportLCSCEasyEDAAssets(ctx, component.ID, offer); err != nil {
		return domain.Component{}, err
	}
	return component, nil
}

func (s *Service) maybeImportLCSCEasyEDAAssets(ctx context.Context, componentID string, offer sourcing.SupplierOffer) error {
	if s.easyeda == nil || !strings.EqualFold(strings.TrimSpace(offer.Provider), sourcing.ProviderLCSC) || strings.TrimSpace(offer.SupplierPartNumber) == "" {
		return nil
	}
	_, err := s.ImportEasyEDAAssets(ctx, componentID, offer.SupplierPartNumber)
	return err
}

func shouldSkipEasyEDAImport(existing []domain.ComponentAsset) (bool, []string) {
	haveSymbol := false
	haveFootprint := false
	have3DModel := false
	anyEasyEDA := false
	for _, asset := range existing {
		if asset.Source != "easyeda" {
			continue
		}
		anyEasyEDA = true
		switch asset.AssetType {
		case domain.AssetTypeSymbol:
			haveSymbol = true
		case domain.AssetTypeFootprint:
			haveFootprint = true
		case domain.AssetType3DModel:
			have3DModel = true
		}
	}
	if !anyEasyEDA {
		return false, nil
	}
	if haveSymbol && haveFootprint && have3DModel {
		return true, []string{"EasyEDA assets already imported for this component"}
	}
	return false, []string{"Some EasyEDA assets are already imported for this component; missing asset types will still be imported."}
}

func summarizeExistingEasyEDAAssets(existing []domain.ComponentAsset) (bool, bool, bool, string, string, string) {
	hasSymbol := false
	hasFootprint := false
	has3DModel := false
	symbolID := ""
	footprintID := ""
	model3DID := ""
	for _, asset := range existing {
		if asset.Source != "easyeda" {
			continue
		}
		switch asset.AssetType {
		case domain.AssetTypeSymbol:
			if !hasSymbol {
				hasSymbol = true
				symbolID = asset.ID
			}
		case domain.AssetTypeFootprint:
			if !hasFootprint {
				hasFootprint = true
				footprintID = asset.ID
			}
		case domain.AssetType3DModel:
			if !has3DModel {
				has3DModel = true
				model3DID = asset.ID
			}
		}
	}
	return hasSymbol, hasFootprint, has3DModel, symbolID, footprintID, model3DID
}

func firstExistingEasyEDAAssetID(existing []domain.ComponentAsset, assetType domain.AssetType) string {
	for _, asset := range existing {
		if asset.Source == "easyeda" && asset.AssetType == assetType {
			return asset.ID
		}
	}
	return ""
}

func (s *Service) autoSelectExistingEasyEDAAssets(ctx context.Context, componentID string, existing []domain.ComponentAsset) []string {
	detail, err := s.GetComponentWithAssets(ctx, componentID)
	if err != nil {
		return []string{"unable to verify selected assets: " + err.Error()}
	}

	var warnings []string
	if detail.SelectedSymbolAsset == nil {
		if assetID := firstExistingEasyEDAAssetID(existing, domain.AssetTypeSymbol); assetID != "" {
			if err := s.SetSelectedComponentAsset(ctx, componentID, domain.AssetTypeSymbol, assetID); err != nil {
				warnings = append(warnings, "auto-select symbol: "+err.Error())
			}
		}
	}
	if detail.SelectedFootprintAsset == nil {
		if assetID := firstExistingEasyEDAAssetID(existing, domain.AssetTypeFootprint); assetID != "" {
			if err := s.SetSelectedComponentAsset(ctx, componentID, domain.AssetTypeFootprint, assetID); err != nil {
				warnings = append(warnings, "auto-select footprint: "+err.Error())
			}
		}
	}
	if detail.Selected3DModelAsset == nil {
		if assetID := firstExistingEasyEDAAssetID(existing, domain.AssetType3DModel); assetID != "" {
			if err := s.SetSelectedComponentAsset(ctx, componentID, domain.AssetType3DModel, assetID); err != nil {
				warnings = append(warnings, "auto-select 3d model: "+err.Error())
			}
		}
	}
	return warnings
}

func (s *Service) ImportEasyEDAAssets(ctx context.Context, componentID, lcscID string) (easyedaprovider.ImportResult, error) {
	result := easyedaprovider.ImportResult{LCSCID: lcscID}
	if s.easyeda == nil {
		result.Warnings = []string{}
		result.Errors = []string{}
		return result, nil
	}

	existing, err := s.ListComponentAssets(ctx, componentID)
	if err != nil {
		return result, err
	}
	skip, warnings := shouldSkipEasyEDAImport(existing)
	hasSymbol, hasFootprint, has3D, symbolAssetID, footprintAssetID, model3DAssetID := summarizeExistingEasyEDAAssets(existing)
	if skip {
		result.SymbolImported = hasSymbol
		result.FootprintImported = hasFootprint
		result.Model3DImported = has3D
		result.SymbolAssetID = symbolAssetID
		result.FootprintAssetID = footprintAssetID
		result.Model3DAssetID = model3DAssetID
		result.Warnings = append(result.Warnings, warnings...)
		result.Warnings = append(result.Warnings, s.autoSelectExistingEasyEDAAssets(ctx, componentID, existing)...)
		result.Errors = []string{}
		return result, nil
	}

	result, err = s.easyeda.ImportComponentAssets(ctx, easyedaprovider.ImportRequest{ComponentID: componentID, LCSCID: lcscID})
	if err != nil {
		return result, err
	}
	if len(warnings) > 0 {
		result.Warnings = append(warnings, result.Warnings...)
	}
	if result.SymbolAssetID != "" {
		if err := s.SetSelectedComponentAsset(ctx, componentID, domain.AssetTypeSymbol, result.SymbolAssetID); err != nil {
			result.Warnings = append(result.Warnings, "auto-select symbol: "+err.Error())
		}
	}
	if result.FootprintAssetID != "" {
		if err := s.SetSelectedComponentAsset(ctx, componentID, domain.AssetTypeFootprint, result.FootprintAssetID); err != nil {
			result.Warnings = append(result.Warnings, "auto-select footprint: "+err.Error())
		}
	}
	if result.Model3DAssetID != "" {
		if err := s.SetSelectedComponentAsset(ctx, componentID, domain.AssetType3DModel, result.Model3DAssetID); err != nil {
			result.Warnings = append(result.Warnings, "auto-select 3d model: "+err.Error())
		}
	}
	if result.Warnings == nil {
		result.Warnings = []string{}
	}
	if result.Errors == nil {
		result.Errors = []string{}
	}
	return result, nil
}
