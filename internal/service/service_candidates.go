package service

import (
	"context"
	"strings"

	"fmt"

	"componentmanager/internal/domain"
)

// --- Part Candidates ---

func (s *Service) AddPartCandidate(ctx context.Context, requirementID, componentID string) (domain.ProjectPartCandidate, error) {
	req, err := s.projects.GetRequirement(ctx, requirementID)
	if err != nil {
		return domain.ProjectPartCandidate{}, err
	}
	component, err := s.components.GetComponent(ctx, componentID)
	if err != nil {
		return domain.ProjectPartCandidate{}, err
	}
	if component.Category != req.Category {
		return domain.ProjectPartCandidate{}, domain.ErrCategoryMismatch{
			RequirementCategory: req.Category,
			ComponentCategory:   component.Category,
		}
	}

	candidate := domain.ProjectPartCandidate{
		ID:            newID(),
		ProjectID:     req.ProjectID,
		RequirementID: requirementID,
		ComponentID:   &componentID,
		Preferred:     false,
		Origin:        domain.CandidateOriginLocal,
	}
	created, err := s.projects.AddPartCandidate(ctx, candidate)
	if err != nil {
		return domain.ProjectPartCandidate{}, err
	}
	created.Component = &component
	return created, nil
}

func (s *Service) SetPreferredCandidate(ctx context.Context, requirementID, candidateID string) error {
	candidates, err := s.projects.ListPartCandidates(ctx, requirementID)
	if err != nil {
		return err
	}
	var target *domain.ProjectPartCandidate
	for i := range candidates {
		if candidates[i].ID == candidateID {
			target = &candidates[i]
			break
		}
	}
	if target == nil {
		return domain.ErrNotFound{ID: candidateID}
	}

	if err := s.projects.SetPreferredCandidate(ctx, requirementID, candidateID); err != nil {
		return err
	}

	// Only set requirement resolution for component-backed candidates.
	if target.ComponentID != nil {
		resolution := domain.NewComponentRequirementResolution(*target.ComponentID)
		return s.projects.SetRequirementResolution(ctx, requirementID, resolution)
	}
	// Provider-backed candidate: clear any existing resolution.
	return s.projects.SetRequirementResolution(ctx, requirementID, nil)
}

func (s *Service) DemotePreferredCandidate(ctx context.Context, requirementID, candidateID string) error {
	candidates, err := s.projects.ListPartCandidates(ctx, requirementID)
	if err != nil {
		return err
	}
	var target *domain.ProjectPartCandidate
	for i := range candidates {
		if candidates[i].ID == candidateID {
			target = &candidates[i]
			break
		}
	}
	if target == nil {
		return domain.ErrNotFound{ID: candidateID}
	}
	if !target.Preferred {
		return nil // already not preferred, nothing to do
	}

	// Clear preferred flag and requirement resolution together (invariant).
	if err := s.projects.ClearPreferredCandidate(ctx, requirementID); err != nil {
		return err
	}
	return s.projects.SetRequirementResolution(ctx, requirementID, nil)
}

func (s *Service) RemovePartCandidate(ctx context.Context, candidateID string) error {
	// Look up the candidate to check if it was preferred.
	candidate, err := s.projects.GetPartCandidate(ctx, candidateID)
	if err != nil {
		return err
	}

	// If removing the preferred candidate, also clear the requirement resolution (invariant).
	if candidate.Preferred {
		if err := s.projects.SetRequirementResolution(ctx, candidate.RequirementID, nil); err != nil {
			return err
		}
	}

	return s.projects.RemovePartCandidate(ctx, candidateID)
}

func (s *Service) ListPartCandidates(ctx context.Context, requirementID string) ([]domain.ProjectPartCandidate, error) {
	candidates, err := s.projects.ListPartCandidates(ctx, requirementID)
	if err != nil {
		return nil, err
	}
	return s.hydrateCandidates(ctx, candidates)
}

func (s *Service) hydrateCandidates(ctx context.Context, candidates []domain.ProjectPartCandidate) ([]domain.ProjectPartCandidate, error) {
	seenComponents := make(map[string]*domain.Component)
	seenOffers := make(map[string]*domain.SavedSupplierOffer)
	for i := range candidates {
		// Hydrate component if present.
		if cid := candidates[i].ComponentID; cid != nil && *cid != "" {
			if _, ok := seenComponents[*cid]; !ok {
				component, err := s.components.GetComponent(ctx, *cid)
				if err != nil {
					continue
				}
				seenComponents[*cid] = &component
			}
			candidates[i].Component = seenComponents[*cid]
		}
		// Hydrate source offer if present.
		if oid := candidates[i].SourceOfferID; oid != nil && *oid != "" {
			if _, ok := seenOffers[*oid]; !ok {
				offer, err := s.projects.GetSavedSupplierOffer(ctx, *oid)
				if err != nil {
					continue
				}
				seenOffers[*oid] = &offer
			}
			candidates[i].SourceOffer = seenOffers[*oid]
		}
	}
	return candidates, nil
}

// --- Saved Supplier Offers ---

func (s *Service) SaveSupplierOfferForRequirement(ctx context.Context, requirementID string, offer domain.SavedSupplierOffer) (domain.SavedSupplierOffer, error) {
	req, err := s.projects.GetRequirement(ctx, requirementID)
	if err != nil {
		return domain.SavedSupplierOffer{}, err
	}
	offer.ID = newID()
	offer.ProjectID = req.ProjectID
	offer.RequirementID = requirementID
	return s.projects.SaveSupplierOffer(ctx, offer)
}

func (s *Service) RemoveSavedSupplierOffer(ctx context.Context, offerID string) error {
	return s.projects.RemoveSavedSupplierOffer(ctx, offerID)
}

func (s *Service) ListSavedSupplierOffers(ctx context.Context, requirementID string) ([]domain.SavedSupplierOffer, error) {
	return s.projects.ListSavedSupplierOffers(ctx, requirementID)
}

func (s *Service) ImportSupplierOffer(ctx context.Context, requirementID string, offer domain.SavedSupplierOffer, setPreferred bool) (domain.ProjectPartCandidate, domain.SavedSupplierOffer, error) {
	req, err := s.projects.GetRequirement(ctx, requirementID)
	if err != nil {
		return domain.ProjectPartCandidate{}, domain.SavedSupplierOffer{}, err
	}

	// Dedupe: check for an existing component with matching manufacturer + MPN.
	component, reused, err := s.findOrCreateComponentFromOffer(ctx, req.Category, offer)
	if err != nil {
		return domain.ProjectPartCandidate{}, domain.SavedSupplierOffer{}, fmt.Errorf("create component from offer: %w", err)
	}

	offer.ID = newID()
	offer.ProjectID = req.ProjectID
	offer.RequirementID = requirementID
	offer.LinkedComponentID = &component.ID
	savedOffer, err := s.projects.SaveSupplierOffer(ctx, offer)
	if err != nil {
		return domain.ProjectPartCandidate{}, domain.SavedSupplierOffer{}, err
	}

	// If the component was reused, check whether it's already a candidate for this requirement.
	var candidateID string
	if reused {
		existing, err := s.projects.ListPartCandidates(ctx, requirementID)
		if err != nil {
			return domain.ProjectPartCandidate{}, savedOffer, err
		}
		for _, c := range existing {
			if c.ComponentID != nil && *c.ComponentID == component.ID {
				candidateID = c.ID
				break
			}
		}
	}

	var created domain.ProjectPartCandidate
	if candidateID == "" {
		candidate := domain.ProjectPartCandidate{
			ID:            newID(),
			ProjectID:     req.ProjectID,
			RequirementID: requirementID,
			ComponentID:   &component.ID,
			SourceOfferID: &savedOffer.ID,
			Preferred:     false,
			Origin:        domain.CandidateOriginImportedSupplier,
		}
		created, err = s.projects.AddPartCandidate(ctx, candidate)
		if err != nil {
			return domain.ProjectPartCandidate{}, savedOffer, err
		}
		created.Component = &component
	} else {
		created = domain.ProjectPartCandidate{
			ID:            candidateID,
			ProjectID:     req.ProjectID,
			RequirementID: requirementID,
			ComponentID:   &component.ID,
			Origin:        domain.CandidateOriginImportedSupplier,
			Component:     &component,
		}
	}

	if setPreferred {
		targetID := created.ID
		if candidateID != "" {
			targetID = candidateID
		}
		if err := s.projects.SetPreferredCandidate(ctx, requirementID, targetID); err != nil {
			return created, savedOffer, err
		}
		created.Preferred = true
		resolution := domain.NewComponentRequirementResolution(component.ID)
		if err := s.projects.SetRequirementResolution(ctx, requirementID, resolution); err != nil {
			return created, savedOffer, err
		}
	}

	return created, savedOffer, nil
}

// findOrCreateComponentFromOffer checks for an existing component that matches
// the offer's manufacturer + MPN (case-insensitive exact match). If a match is
// found in the same category, it is reused. Otherwise a new component is created.
func (s *Service) findOrCreateComponentFromOffer(ctx context.Context, category domain.Category, offer domain.SavedSupplierOffer) (domain.Component, bool, error) {
	if offer.Manufacturer != "" && offer.MPN != "" {
		cat := category
		candidates, err := s.components.FindComponents(ctx, domain.ComponentFilter{
			Category:     &cat,
			Manufacturer: offer.Manufacturer,
			MPN:          offer.MPN,
		})
		if err == nil {
			for _, c := range candidates {
				if strings.EqualFold(c.Manufacturer, offer.Manufacturer) && strings.EqualFold(c.MPN, offer.MPN) {
					return c, true, nil
				}
			}
		}
	}

	component, err := s.CreateComponent(ctx, domain.Component{
		Category:     category,
		MPN:          offer.MPN,
		Manufacturer: offer.Manufacturer,
		Package:      offer.Package,
		Description:  offer.Description,
	})
	if err != nil {
		return domain.Component{}, false, err
	}
	return component, false, nil
}

func (s *Service) AddLocalComponentAsCandidateAndSetPreferred(ctx context.Context, requirementID, componentID string) (domain.ProjectPartCandidate, error) {
	req, err := s.projects.GetRequirement(ctx, requirementID)
	if err != nil {
		return domain.ProjectPartCandidate{}, err
	}
	component, err := s.components.GetComponent(ctx, componentID)
	if err != nil {
		return domain.ProjectPartCandidate{}, err
	}
	if component.Category != req.Category {
		return domain.ProjectPartCandidate{}, domain.ErrCategoryMismatch{
			RequirementCategory: req.Category,
			ComponentCategory:   component.Category,
		}
	}

	// Check if already a candidate
	existing, err := s.projects.ListPartCandidates(ctx, requirementID)
	if err != nil {
		return domain.ProjectPartCandidate{}, err
	}
	var candidateID string
	for _, c := range existing {
		if c.ComponentID != nil && *c.ComponentID == componentID {
			candidateID = c.ID
			break
		}
	}

	if candidateID == "" {
		candidate := domain.ProjectPartCandidate{
			ID:            newID(),
			ProjectID:     req.ProjectID,
			RequirementID: requirementID,
			ComponentID:   &componentID,
			Preferred:     false,
			Origin:        domain.CandidateOriginLocal,
		}
		created, err := s.projects.AddPartCandidate(ctx, candidate)
		if err != nil {
			return domain.ProjectPartCandidate{}, err
		}
		candidateID = created.ID
	}

	if err := s.projects.SetPreferredCandidate(ctx, requirementID, candidateID); err != nil {
		return domain.ProjectPartCandidate{}, err
	}

	resolution := domain.NewComponentRequirementResolution(componentID)
	if err := s.projects.SetRequirementResolution(ctx, requirementID, resolution); err != nil {
		return domain.ProjectPartCandidate{}, err
	}

	return domain.ProjectPartCandidate{
		ID:            candidateID,
		ProjectID:     req.ProjectID,
		RequirementID: requirementID,
		ComponentID:   &componentID,
		Preferred:     true,
		Origin:        domain.CandidateOriginLocal,
		Component:     &component,
	}, nil
}

// AddProviderCandidate saves a supplier offer snapshot and creates a provider-backed
// project candidate. No component is created — the offer metadata is the candidate's
// backing data until the user imports it from Finalize.
func (s *Service) AddProviderCandidate(ctx context.Context, requirementID string, offer domain.SavedSupplierOffer, setPreferred bool) (domain.ProjectPartCandidate, error) {
	req, err := s.projects.GetRequirement(ctx, requirementID)
	if err != nil {
		return domain.ProjectPartCandidate{}, err
	}

	// Save the offer snapshot.
	offer.ID = newID()
	offer.ProjectID = req.ProjectID
	offer.RequirementID = requirementID
	savedOffer, err := s.projects.SaveSupplierOffer(ctx, offer)
	if err != nil {
		return domain.ProjectPartCandidate{}, err
	}

	// Create a provider-backed candidate (no component yet).
	candidate := domain.ProjectPartCandidate{
		ID:            newID(),
		ProjectID:     req.ProjectID,
		RequirementID: requirementID,
		SourceOfferID: &savedOffer.ID,
		Preferred:     false,
		Origin:        domain.CandidateOriginProvider,
	}
	created, err := s.projects.AddPartCandidate(ctx, candidate)
	if err != nil {
		return domain.ProjectPartCandidate{}, err
	}
	created.SourceOffer = &savedOffer

	if setPreferred {
		if err := s.projects.SetPreferredCandidate(ctx, requirementID, created.ID); err != nil {
			return created, err
		}
		created.Preferred = true
		// No requirement resolution — no component yet.
		if err := s.projects.SetRequirementResolution(ctx, requirementID, nil); err != nil {
			return created, err
		}
	}

	return created, nil
}

// ImportProviderCandidate imports a provider-backed candidate into the local
// component catalog. It finds or creates a matching component, updates the
// candidate to reference it, and links the backing offer to the component.
func (s *Service) ImportProviderCandidate(ctx context.Context, candidateID string) (domain.ProjectPartCandidate, error) {
	candidate, err := s.projects.GetPartCandidate(ctx, candidateID)
	if err != nil {
		return domain.ProjectPartCandidate{}, err
	}
	if candidate.Origin != domain.CandidateOriginProvider {
		return domain.ProjectPartCandidate{}, fmt.Errorf("candidate %s is not provider-backed (origin=%s)", candidateID, candidate.Origin)
	}
	if candidate.SourceOfferID == nil {
		return domain.ProjectPartCandidate{}, fmt.Errorf("candidate %s has no backing offer", candidateID)
	}

	offer, err := s.projects.GetSavedSupplierOffer(ctx, *candidate.SourceOfferID)
	if err != nil {
		return domain.ProjectPartCandidate{}, fmt.Errorf("load backing offer: %w", err)
	}

	req, err := s.projects.GetRequirement(ctx, candidate.RequirementID)
	if err != nil {
		return domain.ProjectPartCandidate{}, err
	}

	// Dedupe: find or create a local component from the offer metadata.
	component, _, err := s.findOrCreateComponentFromOffer(ctx, req.Category, offer)
	if err != nil {
		return domain.ProjectPartCandidate{}, fmt.Errorf("create component from offer: %w", err)
	}

	// Update the candidate to reference the component.
	if err := s.projects.UpdatePartCandidateComponent(ctx, candidateID, component.ID, domain.CandidateOriginImportedSupplier); err != nil {
		return domain.ProjectPartCandidate{}, err
	}

	// Link the offer to the component.
	if err := s.projects.LinkSupplierOfferToComponent(ctx, offer.ID, component.ID); err != nil {
		return domain.ProjectPartCandidate{}, err
	}

	// If this candidate was preferred, update the requirement resolution now that we have a component.
	if candidate.Preferred {
		resolution := domain.NewComponentRequirementResolution(component.ID)
		if err := s.projects.SetRequirementResolution(ctx, candidate.RequirementID, resolution); err != nil {
			return domain.ProjectPartCandidate{}, err
		}
	}

	// Return updated candidate.
	candidate.ComponentID = &component.ID
	candidate.Origin = domain.CandidateOriginImportedSupplier
	candidate.Component = &component
	offer.LinkedComponentID = &component.ID
	candidate.SourceOffer = &offer
	return candidate, nil
}
