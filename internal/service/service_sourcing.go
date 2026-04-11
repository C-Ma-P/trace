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

// SourcingProviders returns metadata about each configured sourcing provider.
func (s *Service) SourcingProviders(ctx context.Context) ([]sourcing.ProviderInfo, error) {
	coord, err := s.resolveSourcingCoordinator(ctx)
	if err != nil {
		return nil, err
	}
	return coord.Providers(), nil
}

// SourceRequirementFromProvider searches a single named provider for the given requirement.
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
	if offer.Manufacturer != "" && offer.MPN != "" {
		candidates, err := s.components.FindComponents(ctx, domain.ComponentFilter{
			Manufacturer: offer.Manufacturer,
			MPN:          offer.MPN,
		})
		if err == nil {
			for _, c := range candidates {
				if strings.EqualFold(c.Manufacturer, offer.Manufacturer) && strings.EqualFold(c.MPN, offer.MPN) {
					return c, nil
				}
			}
		}
	}
	return s.CreateComponent(ctx, domain.Component{
		Category:     categoryFromCatalog(offer.Raw["parentCatalog"]),
		MPN:          offer.MPN,
		Manufacturer: offer.Manufacturer,
		Package:      offer.Package,
		Description:  offer.Description,
	})
}

func categoryFromCatalog(parentCatalog string) domain.Category {
	lower := strings.ToLower(parentCatalog)
	switch {
	case strings.Contains(lower, "resistor"):
		return domain.CategoryResistor
	case strings.Contains(lower, "capacitor"):
		return domain.CategoryCapacitor
	case strings.Contains(lower, "inductor") || strings.Contains(lower, "coil"):
		return domain.CategoryInductor
	default:
		return domain.CategoryIntegratedCircuit
	}
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
