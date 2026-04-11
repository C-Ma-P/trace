package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"trace/internal/assetsearch"
	"trace/internal/domain"
	"trace/internal/domain/registry"
	"trace/internal/ingest"
	easyedaprovider "trace/internal/providers/easyeda"
)

func (a *App) CreateComponentAsset(req CreateComponentAssetInput) (ComponentAssetResponse, error) {
	if err := a.checkReady(); err != nil {
		return ComponentAssetResponse{}, err
	}
	var metaJSON json.RawMessage
	if req.MetadataJSON != nil {
		metaJSON = json.RawMessage(*req.MetadataJSON)
	}
	asset, err := a.svc.CreateComponentAsset(context.Background(), domain.ComponentAsset{
		ComponentID:  req.ComponentID,
		AssetType:    domain.AssetType(req.AssetType),
		Source:       req.Source,
		Status:       domain.AssetStatus(req.Status),
		Label:        req.Label,
		URLOrPath:    req.URLOrPath,
		PreviewURL:   req.PreviewURL,
		MetadataJSON: metaJSON,
	})
	if err != nil {
		return ComponentAssetResponse{}, err
	}
	return assetToResponse(asset), nil
}

func (a *App) ListComponentAssets(componentID string) ([]ComponentAssetResponse, error) {
	if err := a.checkReady(); err != nil {
		return nil, err
	}
	assets, err := a.svc.ListComponentAssets(context.Background(), componentID)
	if err != nil {
		return nil, err
	}
	out := make([]ComponentAssetResponse, len(assets))
	for i, asset := range assets {
		out[i] = assetToResponse(asset)
	}
	return out, nil
}

func (a *App) SelectComponentAsset(componentID, assetType, assetID string) error {
	if err := a.checkReady(); err != nil {
		return err
	}
	return a.svc.SetSelectedComponentAsset(context.Background(), componentID, domain.AssetType(assetType), assetID)
}

func (a *App) ClearSelectedComponentAsset(componentID, assetType string) error {
	if err := a.checkReady(); err != nil {
		return err
	}
	return a.svc.ClearSelectedComponentAsset(context.Background(), componentID, domain.AssetType(assetType))
}

func (a *App) GetComponentDetail(componentID string) (ComponentDetailResponse, error) {
	if err := a.checkReady(); err != nil {
		return ComponentDetailResponse{}, err
	}
	detail, err := a.svc.GetComponentWithAssets(context.Background(), componentID)
	if err != nil {
		return ComponentDetailResponse{}, err
	}
	r := componentDetailToResponse(detail)

	if a.bagRepo != nil {
		r.ImageURL = a.bagRepo.FindComponentImageURL(context.Background(), componentID)
		bagList, bagErr := a.bagRepo.ListBagsByComponent(context.Background(), componentID)
		if bagErr == nil {
			bags := make([]BagResponse, len(bagList))
			for i, b := range bagList {
				bags[i] = bagToResponse(b)
			}
			r.Bags = bags
		}
	}
	if r.Bags == nil {
		r.Bags = []BagResponse{}
	}

	return r, nil
}

func assetToResponse(a domain.ComponentAsset) ComponentAssetResponse {
	var meta *string
	if a.MetadataJSON != nil {
		s := string(a.MetadataJSON)
		meta = &s
	}
	return ComponentAssetResponse{
		ID:           a.ID,
		ComponentID:  a.ComponentID,
		AssetType:    string(a.AssetType),
		Source:       a.Source,
		Status:       string(a.Status),
		Label:        a.Label,
		URLOrPath:    a.URLOrPath,
		PreviewURL:   a.PreviewURL,
		MetadataJSON: meta,
		CreatedAt:    a.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    a.UpdatedAt.Format(time.RFC3339),
	}
}

func assetPtrToResponse(a *domain.ComponentAsset) *ComponentAssetResponse {
	if a == nil {
		return nil
	}
	r := assetToResponse(*a)
	return &r
}

func componentDetailToResponse(d domain.ComponentWithAssets) ComponentDetailResponse {
	assets := make([]ComponentAssetResponse, len(d.Assets))
	for i, a := range d.Assets {
		assets[i] = assetToResponse(a)
	}
	return ComponentDetailResponse{
		Component:              componentToResponse(d.Component),
		SelectedSymbolAsset:    assetPtrToResponse(d.SelectedSymbolAsset),
		SelectedFootprintAsset: assetPtrToResponse(d.SelectedFootprintAsset),
		Selected3DModelAsset:   assetPtrToResponse(d.Selected3DModelAsset),
		SelectedDatasheetAsset: assetPtrToResponse(d.SelectedDatasheetAsset),
		Assets:                 assets,
	}
}

func (a *App) SearchComponentAssets(componentID string, query string) (AssetSearchResponse, error) {
	if err := a.checkReady(); err != nil {
		return AssetSearchResponse{}, err
	}
	resp, err := a.assetSearch.SearchForComponent(context.Background(), assetsearch.SearchRequest{
		ComponentID: componentID,
		Query:       query,
	})
	if err != nil {
		return AssetSearchResponse{}, err
	}
	return searchResponseToApp(resp), nil
}

func (a *App) ImportComponentAssetResult(componentID string, provider string, externalID string) (AssetImportResponse, error) {
	if err := a.checkReady(); err != nil {
		return AssetImportResponse{}, err
	}
	resp, err := a.assetSearch.ImportSearchResult(context.Background(), assetsearch.ImportRequest{
		ComponentID: componentID,
		Provider:    provider,
		ExternalID:  externalID,
	})
	if err != nil {
		return AssetImportResponse{}, err
	}
	return importResponseToApp(resp), nil
}

func searchResponseToApp(r assetsearch.SearchResponse) AssetSearchResponse {
	results := make([]AssetSearchProviderResult, len(r.ProviderResults))
	for i, pr := range r.ProviderResults {
		candidates := make([]AssetSearchCandidate, len(pr.Candidates))
		for j, c := range pr.Candidates {
			candidates[j] = AssetSearchCandidate{
				ExternalID:   c.ExternalID,
				Title:        c.Title,
				Manufacturer: c.Manufacturer,
				MPN:          c.MPN,
				Package:      c.Package,
				Description:  c.Description,
				HasSymbol:    c.HasSymbol,
				HasFootprint: c.HasFootprint,
				Has3DModel:   c.Has3DModel,
				HasDatasheet: c.HasDatasheet,
				PreviewURL:   c.PreviewURL,
				SourceURL:    c.SourceURL,
				Metadata:     c.Metadata,
			}
		}
		results[i] = AssetSearchProviderResult{
			ProviderId:    pr.ProviderID,
			ProviderLabel: pr.ProviderLabel,
			Candidates:    candidates,
			Error:         pr.Error,
		}
	}
	return AssetSearchResponse{ProviderResults: results}
}

func importResponseToApp(r assetsearch.ImportResponse) AssetImportResponse {
	assets := make([]AssetImportedAsset, len(r.ImportedAssets))
	for i, a := range r.ImportedAssets {
		assets[i] = AssetImportedAsset{
			AssetType: a.AssetType,
			Label:     a.Label,
			URLOrPath: a.URLOrPath,
		}
	}
	warnings := r.Warnings
	if warnings == nil {
		warnings = []string{}
	}
	return AssetImportResponse{
		ImportedAssets: assets,
		Warnings:       warnings,
	}
}

func (a *App) ValidateAssetPath(path string) ValidateAssetPathResponse {
	v := ingest.ValidatePath(path)
	return ValidateAssetPathResponse{
		Valid:    v.Valid,
		Reason:   v.Reason,
		PathKind: string(v.PathKind),
	}
}

func (a *App) IngestComponentAssets(componentID string, filePath string) (IngestComponentAssetsResponse, error) {
	if err := a.checkReady(); err != nil {
		return IngestComponentAssetsResponse{}, err
	}
	if a.ingest == nil {
		return IngestComponentAssetsResponse{}, fmt.Errorf("ingestion service not available")
	}

	result, err := a.ingest.Ingest(context.Background(), ingest.IngestRequest{
		ComponentID: componentID,
		FilePath:    filePath,
		SourceKind:  "local",
	})
	if err != nil {
		return IngestComponentAssetsResponse{}, err
	}

	return ingestResultToResponse(result), nil
}

func ingestResultToResponse(r ingest.IngestResult) IngestComponentAssetsResponse {
	assets := make([]IngestedAssetResponse, len(r.Assets))
	for i, a := range r.Assets {
		assets[i] = IngestedAssetResponse{
			AssetID:          a.AssetID,
			AssetType:        a.AssetType,
			Label:            a.Label,
			StoredPath:       a.StoredPath,
			OriginalFilename: a.OriginalFilename,
		}
	}
	warnings := r.Warnings
	if warnings == nil {
		warnings = []string{}
	}
	unsupported := r.Unsupported
	if unsupported == nil {
		unsupported = []string{}
	}
	countByType := r.CountByType
	if countByType == nil {
		countByType = map[string]int{}
	}
	return IngestComponentAssetsResponse{
		Assets:      assets,
		Warnings:    warnings,
		Unsupported: unsupported,
		CountByType: countByType,
	}
}

func (a *App) ImportEasyEDAAssets(input ImportEasyEDAInput) (ImportEasyEDAResponse, error) {
	if err := a.checkReady(); err != nil {
		return ImportEasyEDAResponse{}, err
	}
	if a.easyeda == nil {
		return ImportEasyEDAResponse{}, fmt.Errorf("EasyEDA provider not available")
	}

	lcscID := input.LCSCID
	if lcscID == "" {
		comp, err := a.svc.GetComponent(context.Background(), input.ComponentID)
		if err != nil {
			return ImportEasyEDAResponse{}, fmt.Errorf("loading component: %w", err)
		}
		attr, ok := comp.GetAttribute(registry.AttrLCSCPart)
		if !ok || attr.Text == nil || *attr.Text == "" {
			return ImportEasyEDAResponse{}, fmt.Errorf("no LCSC part number stored on this component; add a %q attribute or provide an LCSC ID", registry.AttrLCSCPart)
		}
		lcscID = *attr.Text
	}

	ctx := context.Background()
	existing, err := a.svc.ListComponentAssets(ctx, input.ComponentID)
	if err != nil {
		return ImportEasyEDAResponse{}, fmt.Errorf("listing existing assets: %w", err)
	}
	skip, warnings := shouldSkipEasyEDAImport(existing)
	hasSymbol, hasFootprint, has3D, symbolAssetID, footprintAssetID, model3DAssetID := summarizeExistingEasyEDAAssets(existing)
	if skip {
		selectWarnings := a.autoSelectExistingEasyEDAAssets(ctx, input.ComponentID, existing)
		warnings = append(warnings, selectWarnings...)
		return ImportEasyEDAResponse{
			LCSCID:            lcscID,
			SymbolImported:    hasSymbol,
			FootprintImported: hasFootprint,
			Model3DImported:   has3D,
			SymbolAssetID:     symbolAssetID,
			FootprintAssetID:  footprintAssetID,
			Model3DAssetID:    model3DAssetID,
			Warnings:          warnings,
			Errors:            []string{},
		}, nil
	}

	result, err := a.easyeda.ImportComponentAssets(ctx, easyedaprovider.ImportRequest{
		ComponentID: input.ComponentID,
		LCSCID:      lcscID,
	})
	if err != nil {
		return ImportEasyEDAResponse{}, err
	}
	if len(warnings) > 0 {
		result.Warnings = append(warnings, result.Warnings...)
	}
	if result.SymbolAssetID != "" {
		if selErr := a.svc.SetSelectedComponentAsset(ctx, input.ComponentID, domain.AssetTypeSymbol, result.SymbolAssetID); selErr != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("auto-select symbol: %v", selErr))
		}
	}
	if result.FootprintAssetID != "" {
		if selErr := a.svc.SetSelectedComponentAsset(ctx, input.ComponentID, domain.AssetTypeFootprint, result.FootprintAssetID); selErr != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("auto-select footprint: %v", selErr))
		}
	}
	if result.Model3DAssetID != "" {
		if selErr := a.svc.SetSelectedComponentAsset(ctx, input.ComponentID, domain.AssetType3DModel, result.Model3DAssetID); selErr != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("auto-select 3d model: %v", selErr))
		}
	}

	warnings = result.Warnings
	if warnings == nil {
		warnings = []string{}
	}
	errors := result.Errors
	if errors == nil {
		errors = []string{}
	}

	return ImportEasyEDAResponse{
		LCSCID:            result.LCSCID,
		SymbolImported:    result.SymbolImported,
		FootprintImported: result.FootprintImported,
		Model3DImported:   result.Model3DImported,
		SymbolAssetID:     result.SymbolAssetID,
		FootprintAssetID:  result.FootprintAssetID,
		Model3DAssetID:    result.Model3DAssetID,
		Warnings:          warnings,
		Errors:            errors,
	}, nil
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

func (a *App) autoSelectExistingEasyEDAAssets(ctx context.Context, componentID string, existing []domain.ComponentAsset) []string {
	detail, err := a.svc.GetComponentWithAssets(ctx, componentID)
	if err != nil {
		return []string{fmt.Sprintf("unable to verify selected assets: %v", err)}
	}

	var warnings []string
	if detail.SelectedSymbolAsset == nil {
		if assetID := firstExistingAssetID(existing, domain.AssetTypeSymbol); assetID != "" {
			if err := a.svc.SetSelectedComponentAsset(ctx, componentID, domain.AssetTypeSymbol, assetID); err != nil {
				warnings = append(warnings, fmt.Sprintf("auto-select symbol: %v", err))
			}
		}
	}
	if detail.SelectedFootprintAsset == nil {
		if assetID := firstExistingAssetID(existing, domain.AssetTypeFootprint); assetID != "" {
			if err := a.svc.SetSelectedComponentAsset(ctx, componentID, domain.AssetTypeFootprint, assetID); err != nil {
				warnings = append(warnings, fmt.Sprintf("auto-select footprint: %v", err))
			}
		}
	}
	if detail.Selected3DModelAsset == nil {
		if assetID := firstExistingAssetID(existing, domain.AssetType3DModel); assetID != "" {
			if err := a.svc.SetSelectedComponentAsset(ctx, componentID, domain.AssetType3DModel, assetID); err != nil {
				warnings = append(warnings, fmt.Sprintf("auto-select 3d model: %v", err))
			}
		}
	}
	return warnings
}

func firstExistingAssetID(existing []domain.ComponentAsset, assetType domain.AssetType) string {
	for _, asset := range existing {
		if asset.Source == "easyeda" && asset.AssetType == assetType {
			return asset.ID
		}
	}
	return ""
}

// ReadAssetFile reads the file contents of a component asset and returns them
// as base64-encoded data. This is used by the frontend to load asset files
// (e.g. STEP models) that live in managed storage.
func (a *App) ReadAssetFile(assetID string) (ReadAssetFileResponse, error) {
	if err := a.checkReady(); err != nil {
		return ReadAssetFileResponse{}, err
	}

	asset, err := a.svc.GetComponentAsset(context.Background(), assetID)
	if err != nil {
		return ReadAssetFileResponse{}, fmt.Errorf("asset lookup: %w", err)
	}

	filePath := asset.URLOrPath
	if filePath == "" {
		return ReadAssetFileResponse{}, fmt.Errorf("asset has no file path")
	}

	// Security: ensure the path is within thr managed assets directory.
	// Reject absolute paths that escape the expected storage root.
	cleanPath := filepath.Clean(filePath)
	if !filepath.IsAbs(cleanPath) {
		return ReadAssetFileResponse{}, fmt.Errorf("asset path is not absolute")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ReadAssetFileResponse{}, fmt.Errorf("resolve home dir: %w", err)
	}
	assetsRoot := filepath.Join(home, ".trace", "assets")
	if !strings.HasPrefix(cleanPath, assetsRoot+string(filepath.Separator)) {
		return ReadAssetFileResponse{}, fmt.Errorf("asset path is outside managed storage")
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return ReadAssetFileResponse{}, fmt.Errorf("read asset file: %w", err)
	}

	return ReadAssetFileResponse{
		Data:     base64.StdEncoding.EncodeToString(data),
		Filename: filepath.Base(cleanPath),
	}, nil
}
