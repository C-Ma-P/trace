package assetsearch

import (
	"context"
	"fmt"

	"componentmanager/internal/domain"
)

// Service orchestrates asset search and import across providers.
type Service struct {
	registry   *Registry
	components domain.ComponentRepository
	assets     domain.ComponentAssetRepository
}

// NewService creates an asset search orchestration service.
func NewService(registry *Registry, components domain.ComponentRepository, assets domain.ComponentAssetRepository) *Service {
	return &Service{
		registry:   registry,
		components: components,
		assets:     assets,
	}
}

// SearchForComponent queries all registered providers for asset candidates
// matching the given component. Partial provider failures are captured in the
// response rather than failing the entire search.
func (s *Service) SearchForComponent(ctx context.Context, req SearchRequest) (SearchResponse, error) {
	if req.ComponentID == "" {
		return SearchResponse{}, fmt.Errorf("component ID is required")
	}

	// If no explicit query or MPN, look up the component to get identity.
	if req.MPN == "" && req.Query == "" {
		c, err := s.components.GetComponent(ctx, req.ComponentID)
		if err != nil {
			return SearchResponse{}, fmt.Errorf("looking up component: %w", err)
		}
		req.MPN = c.MPN
		if req.Manufacturer == "" {
			req.Manufacturer = c.Manufacturer
		}
	}

	providers := s.registry.All()
	results := make([]ProviderResult, len(providers))

	for i, p := range providers {
		candidates, err := p.Search(ctx, req)
		if err != nil {
			results[i] = ProviderResult{
				Provider: p.DisplayName(),
				Error:    err.Error(),
			}
		} else {
			results[i] = ProviderResult{
				Provider:   p.DisplayName(),
				Candidates: candidates,
			}
		}
	}

	return SearchResponse{ProviderResults: results}, nil
}

// ImportSearchResult imports a single provider candidate's assets into the
// component's asset list. Assets land as candidates (not auto-selected).
func (s *Service) ImportSearchResult(ctx context.Context, req ImportRequest) (ImportResponse, error) {
	if req.ComponentID == "" {
		return ImportResponse{}, fmt.Errorf("component ID is required")
	}
	if req.Provider == "" {
		return ImportResponse{}, fmt.Errorf("provider is required")
	}
	if req.ExternalID == "" {
		return ImportResponse{}, fmt.Errorf("external ID is required")
	}

	p := s.registry.Get(req.Provider)
	if p == nil {
		return ImportResponse{}, fmt.Errorf("unknown provider %q", req.Provider)
	}

	result, err := p.Import(ctx, req)
	if err != nil {
		return ImportResponse{}, fmt.Errorf("provider %q import failed: %w", req.Provider, err)
	}

	// Persist each imported asset as a candidate on the component.
	for _, ia := range result.ImportedAssets {
		assetType := domain.AssetType(ia.AssetType)
		if !assetType.Valid() {
			result.Warnings = append(result.Warnings, fmt.Sprintf("skipping unknown asset type %q", ia.AssetType))
			continue
		}
		_, err := s.assets.CreateComponentAsset(ctx, domain.ComponentAsset{
			ComponentID: req.ComponentID,
			AssetType:   assetType,
			Source:      p.Name(),
			Status:      domain.AssetStatusCandidate,
			Label:       ia.Label,
			URLOrPath:   ia.URLOrPath,
		})
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to persist %s asset: %v", ia.AssetType, err))
		}
	}

	return result, nil
}
