package service

import (
	"context"
	"fmt"

	"github.com/C-Ma-P/trace/internal/domain"
)

func (s *Service) CreateComponentAsset(ctx context.Context, asset domain.ComponentAsset) (domain.ComponentAsset, error) {
	if asset.ID == "" {
		asset.ID = newID()
	}
	if !asset.AssetType.Valid() {
		return domain.ComponentAsset{}, fmt.Errorf("invalid asset type %q", asset.AssetType)
	}
	if !asset.Status.Valid() {
		asset.Status = domain.AssetStatusCandidate
	}
	if asset.Source == "" {
		asset.Source = "manual"
	}
	// Verify the component exists.
	if _, err := s.components.GetComponent(ctx, asset.ComponentID); err != nil {
		return domain.ComponentAsset{}, err
	}
	return s.assets.CreateComponentAsset(ctx, asset)
}

func (s *Service) ListComponentAssets(ctx context.Context, componentID string) ([]domain.ComponentAsset, error) {
	return s.assets.ListComponentAssets(ctx, componentID)
}

func (s *Service) ListComponentAssetsByType(ctx context.Context, componentID string, assetType domain.AssetType) ([]domain.ComponentAsset, error) {
	if !assetType.Valid() {
		return nil, fmt.Errorf("invalid asset type %q", assetType)
	}
	return s.assets.ListComponentAssetsByType(ctx, componentID, assetType)
}

func (s *Service) GetComponentAsset(ctx context.Context, id string) (domain.ComponentAsset, error) {
	return s.assets.GetComponentAsset(ctx, id)
}

func (s *Service) UpdateComponentAsset(ctx context.Context, asset domain.ComponentAsset) (domain.ComponentAsset, error) {
	if asset.Status != "" && !asset.Status.Valid() {
		return domain.ComponentAsset{}, fmt.Errorf("invalid asset status %q", asset.Status)
	}
	return s.assets.UpdateComponentAsset(ctx, asset)
}

func (s *Service) DeleteComponentAsset(ctx context.Context, id string) error {
	return s.assets.DeleteComponentAsset(ctx, id)
}

func (s *Service) SetSelectedComponentAsset(ctx context.Context, componentID string, assetType domain.AssetType, assetID string) error {
	if !assetType.Valid() {
		return fmt.Errorf("invalid asset type %q", assetType)
	}
	return s.assets.SetSelectedComponentAsset(ctx, componentID, assetType, assetID)
}

func (s *Service) ClearSelectedComponentAsset(ctx context.Context, componentID string, assetType domain.AssetType) error {
	if !assetType.Valid() {
		return fmt.Errorf("invalid asset type %q", assetType)
	}
	return s.assets.ClearSelectedComponentAsset(ctx, componentID, assetType)
}

func (s *Service) GetComponentWithAssets(ctx context.Context, componentID string) (domain.ComponentWithAssets, error) {
	return s.assets.GetComponentWithAssets(ctx, componentID)
}

func (s *Service) UpdateComponentAssetStatus(ctx context.Context, assetID string, status domain.AssetStatus) (domain.ComponentAsset, error) {
	if !status.Valid() {
		return domain.ComponentAsset{}, fmt.Errorf("invalid asset status %q", status)
	}
	asset, err := s.assets.GetComponentAsset(ctx, assetID)
	if err != nil {
		return domain.ComponentAsset{}, err
	}
	asset.Status = status
	return s.assets.UpdateComponentAsset(ctx, asset)
}
