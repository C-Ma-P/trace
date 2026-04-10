package assetsearch

import (
	"context"
	"fmt"
	"log"

	"trace/internal/domain"
	"trace/internal/ingest"
)

// Service orchestrates asset search and import across providers.
type Service struct {
	registry   *Registry
	components domain.ComponentRepository
	assets     domain.ComponentAssetRepository
	ingest     *ingest.Service
}

// NewService creates an asset search orchestration service.
// The ingest service is used to process downloaded artifacts from providers
// through the same pipeline as manual/local imports.
func NewService(registry *Registry, components domain.ComponentRepository, assets domain.ComponentAssetRepository, ingestSvc ...*ingest.Service) *Service {
	s := &Service{
		registry:   registry,
		components: components,
		assets:     assets,
	}
	if len(ingestSvc) > 0 {
		s.ingest = ingestSvc[0]
	}
	return s
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
				ProviderID:    p.Name(),
				ProviderLabel: p.DisplayName(),
				Error:         err.Error(),
			}
		} else {
			results[i] = ProviderResult{
				ProviderID:    p.Name(),
				ProviderLabel: p.DisplayName(),
				Candidates:    candidates,
			}
		}
	}

	return SearchResponse{ProviderResults: results}, nil
}

// ImportSearchResult imports a single provider candidate's assets into the
// component's asset list. When a provider returns downloaded artifacts, those
// are fed through the ingestion pipeline so that assets land in Trace-managed
// local storage. Legacy providers that return pre-built ImportedAssets are
// still supported via direct persistence.
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

	// Preferred path: provider returned downloaded artifacts — route through ingestion.
	if len(result.Artifacts) > 0 && s.ingest != nil {
		for _, artifact := range result.Artifacts {
			ingestResult, err := s.ingest.Ingest(ctx, ingest.IngestRequest{
				ComponentID: req.ComponentID,
				FilePath:    artifact.FilePath,
				SourceKind:  p.Name(),
				SourceLabel: artifact.Description,
			})
			if err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("ingestion failed for %s: %v", artifact.Description, err))
				continue
			}
			// Convert ingested assets to ImportedAsset for the response.
			for _, ia := range ingestResult.Assets {
				result.ImportedAssets = append(result.ImportedAssets, ImportedAsset{
					AssetType: ia.AssetType,
					Label:     ia.Label,
					URLOrPath: ia.StoredPath,
				})
			}
			result.Warnings = append(result.Warnings, ingestResult.Warnings...)
		}
		log.Printf("[assetsearch] provider %q: %d artifacts ingested for component %s", p.Name(), len(result.ImportedAssets), req.ComponentID)
		return result, nil
	}

	// Fallback: legacy direct-persist path for providers that only return
	// ImportedAssets without downloaded artifacts (stubs, etc.).
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
