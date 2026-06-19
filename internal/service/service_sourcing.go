package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/C-Ma-P/trace/internal/domain"
	"github.com/C-Ma-P/trace/internal/sourcing"
)

func (s *Service) LookupVendorPartID(ctx context.Context, vendor, partID string) (sourcing.SupplierOffer, error) {
	coord, err := s.resolveSourcingCoordinator(ctx)
	if err != nil {
		return sourcing.SupplierOffer{}, err
	}
	return coord.LookupByVendorPartID(ctx, vendor, partID)
}

func (s *Service) resolveSourcingCoordinator(ctx context.Context) (*sourcing.Coordinator, error) {
	if s.supplierConfig != nil {
		return s.supplierConfig.GetSourcingCoordinator(ctx)
	}
	if s.sourcing != nil {
		s.sourcingCoordinatorMu.Lock()
		defer s.sourcingCoordinatorMu.Unlock()
		if s.sourcingCoordinator == nil {
			s.sourcingCoordinator = sourcing.NewCoordinatorFromService(s.sourcing)
		}
		return s.sourcingCoordinator, nil
	}
	return nil, fmt.Errorf("sourcing not configured")
}

func (s *Service) SourcingProviders(ctx context.Context) ([]sourcing.ProviderInfo, error) {
	coord, err := s.resolveSourcingCoordinator(ctx)
	if err != nil {
		return nil, err
	}
	return coord.Providers(), nil
}

func (s *Service) SourceRequirementFromProvider(ctx context.Context, requirementID, providerName string) (sourcing.SourceResult, error) {
	requirement, err := s.projects.GetRequirement(ctx, requirementID)
	if err != nil {
		return sourcing.SourceResult{}, err
	}
	requirement.NormalizeResolution()

	var selectedDefinition *domain.Component
	if componentID := requirement.ResolvedComponentID(); componentID != nil {
		component, err := s.components.GetComponent(ctx, *componentID)
		if err != nil {
			return sourcing.SourceResult{}, err
		}
		selectedDefinition = &component
	}

	query := sourcing.BuildRequirementQuery(requirement, selectedDefinition)
	coord, err := s.resolveSourcingCoordinator(ctx)
	if err != nil {
		return sourcing.SourceResult{}, err
	}
	return coord.SourceFromProvider(ctx, query, providerName), nil
}

func (s *Service) ResolveComponentFromOffer(ctx context.Context, offer sourcing.SupplierOffer) (domain.Component, error) {
	category := sourcing.MapOfferCategory(offer)
	attrs := sourcing.ComponentAttributesFromOffer(category, offer)
	packageName := firstSupplierPackage(attrs, offer.Package)
	component, _, err := s.findOrCreateComponentWithSupplierAttrs(ctx, category, offer.Manufacturer, offer.MPN, packageName, offer.Description, attrs)
	if err != nil {
		return domain.Component{}, err
	}
	component, err = s.maybeAttachSupplierDatasheet(ctx, component, offer)
	if err != nil {
		return domain.Component{}, err
	}
	return component, nil
}

func (s *Service) findOrCreateComponentWithSupplierAttrs(ctx context.Context, category domain.Category, manufacturer, mpn, packageName, description string, attrs []domain.AttributeValue) (domain.Component, bool, error) {
	if manufacturer != "" && mpn != "" {
		cat := category
		candidates, err := s.components.FindComponents(ctx, domain.ComponentFilter{
			Category:     &cat,
			Manufacturer: manufacturer,
			MPN:          mpn,
		})
		if err == nil {
			for _, candidate := range candidates {
				if !strings.EqualFold(candidate.Manufacturer, manufacturer) || !strings.EqualFold(candidate.MPN, mpn) {
					continue
				}
				component, err := s.components.GetComponent(ctx, candidate.ID)
				if err != nil {
					return domain.Component{}, false, err
				}
				mergedAttrs, added := mergeMissingAttributes(component.Attributes, attrs)
				metadataChanged := false
				preferredPackage := firstSupplierPackage(mergedAttrs, packageName)
				if shouldAdoptSupplierPackage(component.Package, preferredPackage) {
					component.Package = preferredPackage
					metadataChanged = true
				}
				if metadataChanged {
					updated, err := s.UpdateComponentMetadata(ctx, component)
					if err != nil {
						return domain.Component{}, false, err
					}
					component = updated
				}
				if added {
					if err := s.ReplaceComponentAttributes(ctx, component.ID, mergedAttrs); err != nil {
						return domain.Component{}, false, err
					}
					component.Attributes = mergedAttrs
				}
				return component, true, nil
			}
		}
	}

	component, err := s.CreateComponent(ctx, domain.Component{
		Category:     category,
		MPN:          mpn,
		Manufacturer: manufacturer,
		Package:      packageName,
		Description:  description,
		Attributes:   attrs,
	})
	if err != nil {
		return domain.Component{}, false, err
	}
	return component, false, nil
}

func mergeMissingAttributes(existing, additions []domain.AttributeValue) ([]domain.AttributeValue, bool) {
	if len(additions) == 0 {
		merged := make([]domain.AttributeValue, len(existing))
		copy(merged, existing)
		return merged, false
	}
	merged := make([]domain.AttributeValue, len(existing))
	copy(merged, existing)
	seen := make(map[string]struct{}, len(existing))
	for _, attr := range existing {
		seen[attr.Key] = struct{}{}
	}
	added := false
	for _, attr := range additions {
		if _, ok := seen[attr.Key]; ok {
			continue
		}
		merged = append(merged, attr)
		seen[attr.Key] = struct{}{}
		added = true
	}
	return merged, added
}

func firstSupplierPackage(attrs []domain.AttributeValue, fallback string) string {
	for _, attr := range attrs {
		if attr.Key == "package" && attr.Text != nil && strings.TrimSpace(*attr.Text) != "" {
			return strings.TrimSpace(*attr.Text)
		}
	}
	return strings.TrimSpace(fallback)
}

func shouldAdoptSupplierPackage(current, supplier string) bool {
	current = strings.TrimSpace(current)
	supplier = strings.TrimSpace(supplier)
	if supplier == "" {
		return false
	}
	if current == "" {
		return true
	}
	if strings.EqualFold(current, supplier) {
		return false
	}
	return looksLikePackagingLabel(current) && !looksLikePackagingLabel(supplier)
}

func looksLikePackagingLabel(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "reel", "cut tape", "mouse reel", "tube", "tray", "bulk", "ammo pack", "bag":
		return true
	default:
		return false
	}
}

func (s *Service) maybeAttachSupplierDatasheet(ctx context.Context, component domain.Component, offer sourcing.SupplierOffer) (domain.Component, error) {
	url := strings.TrimSpace(offer.DatasheetURL)
	if url == "" || component.SelectedDatasheetAssetID != nil {
		return component, nil
	}
	assets, err := s.ListComponentAssetsByType(ctx, component.ID, domain.AssetTypeDatasheet)
	if err != nil {
		return domain.Component{}, err
	}
	for _, asset := range assets {
		if strings.EqualFold(strings.TrimSpace(asset.URLOrPath), url) {
			if err := s.SetSelectedComponentAsset(ctx, component.ID, domain.AssetTypeDatasheet, asset.ID); err != nil {
				return domain.Component{}, err
			}
			component.SelectedDatasheetAssetID = &asset.ID
			return component, nil
		}
	}
	asset, err := s.CreateComponentAsset(ctx, domain.ComponentAsset{
		ComponentID: component.ID,
		AssetType:   domain.AssetTypeDatasheet,
		Source:      "supplier:" + strings.ToLower(strings.TrimSpace(offer.Provider)),
		Status:      domain.AssetStatusSelected,
		Label:       supplierDatasheetLabel(offer),
		URLOrPath:   url,
	})
	if err != nil {
		return domain.Component{}, err
	}
	if err := s.SetSelectedComponentAsset(ctx, component.ID, domain.AssetTypeDatasheet, asset.ID); err != nil {
		return domain.Component{}, err
	}
	component.SelectedDatasheetAssetID = &asset.ID
	return component, nil
}

func supplierDatasheetLabel(offer sourcing.SupplierOffer) string {
	label := strings.TrimSpace(strings.TrimSpace(offer.Manufacturer) + " " + strings.TrimSpace(offer.MPN))
	if label != "" {
		return label
	}
	if mpn := strings.TrimSpace(offer.MPN); mpn != "" {
		return mpn
	}
	return "Supplier datasheet"
}

func (s *Service) SourceRequirement(ctx context.Context, requirementID string) (sourcing.SourceResult, error) {
	requirement, err := s.projects.GetRequirement(ctx, requirementID)
	if err != nil {
		return sourcing.SourceResult{}, err
	}
	requirement.NormalizeResolution()

	var selectedDefinition *domain.Component
	if componentID := requirement.ResolvedComponentID(); componentID != nil {
		component, err := s.components.GetComponent(ctx, *componentID)
		if err != nil {
			return sourcing.SourceResult{}, err
		}
		selectedDefinition = &component
	}

	query := sourcing.BuildRequirementQuery(requirement, selectedDefinition)
	coord, err := s.resolveSourcingCoordinator(ctx)
	if err != nil {
		return sourcing.SourceResult{}, err
	}
	return coord.Source(ctx, query), nil
}

func (s *Service) ProbeSupplierOffer(ctx context.Context, offer sourcing.SupplierOffer) (sourcing.SupplierOffer, error) {
	coord, err := s.resolveSourcingCoordinator(ctx)
	if err != nil {
		return sourcing.SupplierOffer{}, err
	}
	return coord.ProbeOffer(ctx, offer)
}
