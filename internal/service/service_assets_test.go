package service_test

import (
	"context"
	"errors"
	"testing"

	"trace/internal/domain"
	"trace/internal/service"
)

func TestCreateComponentAsset_Valid(t *testing.T) {
	comp := &stubComponentRepo{
		getResult: domain.Component{ID: "cid-1", Category: domain.CategoryResistor},
	}
	assets := &stubAssetRepo{}
	svc := service.New(comp, &stubProjectRepo{}, assets)

	a, err := svc.CreateComponentAsset(context.Background(), domain.ComponentAsset{
		ComponentID: "cid-1",
		AssetType:   domain.AssetTypeSymbol,
		Label:       "R_0402",
		URLOrPath:   "/symbols/R_0402.kicad_sym",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.ID == "" {
		t.Error("expected ID to be assigned")
	}
	if a.Source != "manual" {
		t.Errorf("expected default source %q, got %q", "manual", a.Source)
	}
	if a.Status != domain.AssetStatusCandidate {
		t.Errorf("expected default status %q, got %q", domain.AssetStatusCandidate, a.Status)
	}
	if assets.created == nil {
		t.Error("expected asset to be persisted via repo")
	}
}

func TestCreateComponentAsset_InvalidType_Rejected(t *testing.T) {
	comp := &stubComponentRepo{
		getResult: domain.Component{ID: "cid-1"},
	}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	_, err := svc.CreateComponentAsset(context.Background(), domain.ComponentAsset{
		ComponentID: "cid-1",
		AssetType:   "invalid_type",
	})
	if err == nil {
		t.Fatal("expected error for invalid asset type")
	}
}

func TestCreateComponentAsset_ComponentNotFound_Rejected(t *testing.T) {
	comp := &stubComponentRepo{
		getErr: domain.ErrNotFound{ID: "cid-missing"},
	}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	_, err := svc.CreateComponentAsset(context.Background(), domain.ComponentAsset{
		ComponentID: "cid-missing",
		AssetType:   domain.AssetTypeFootprint,
	})
	if err == nil {
		t.Fatal("expected error when component not found")
	}
	var target domain.ErrNotFound
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrNotFound, got %T: %v", err, err)
	}
}

func TestSetSelectedComponentAsset_InvalidType_Rejected(t *testing.T) {
	svc := service.New(&stubComponentRepo{}, &stubProjectRepo{}, &stubAssetRepo{})

	err := svc.SetSelectedComponentAsset(context.Background(), "cid-1", "bad_type", "asset-1")
	if err == nil {
		t.Fatal("expected error for invalid asset type")
	}
}

func TestSetSelectedComponentAsset_Valid_Delegated(t *testing.T) {
	assets := &stubAssetRepo{}
	svc := service.New(&stubComponentRepo{}, &stubProjectRepo{}, assets)

	err := svc.SetSelectedComponentAsset(context.Background(), "cid-1", domain.AssetTypeSymbol, "asset-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if assets.setComponentID != "cid-1" {
		t.Errorf("expected componentID %q, got %q", "cid-1", assets.setComponentID)
	}
	if assets.setAssetType != domain.AssetTypeSymbol {
		t.Errorf("expected assetType %q, got %q", domain.AssetTypeSymbol, assets.setAssetType)
	}
	if assets.setAssetID != "asset-1" {
		t.Errorf("expected assetID %q, got %q", "asset-1", assets.setAssetID)
	}
}

func TestClearSelectedComponentAsset_Valid_Delegated(t *testing.T) {
	assets := &stubAssetRepo{}
	svc := service.New(&stubComponentRepo{}, &stubProjectRepo{}, assets)

	err := svc.ClearSelectedComponentAsset(context.Background(), "cid-1", domain.AssetTypeFootprint)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if assets.clearComponentID != "cid-1" {
		t.Errorf("expected componentID %q, got %q", "cid-1", assets.clearComponentID)
	}
	if assets.clearAssetType != domain.AssetTypeFootprint {
		t.Errorf("expected assetType %q, got %q", domain.AssetTypeFootprint, assets.clearAssetType)
	}
}

func TestClearSelectedComponentAsset_InvalidType_Rejected(t *testing.T) {
	svc := service.New(&stubComponentRepo{}, &stubProjectRepo{}, &stubAssetRepo{})

	err := svc.ClearSelectedComponentAsset(context.Background(), "cid-1", "nope")
	if err == nil {
		t.Fatal("expected error for invalid asset type")
	}
}

func TestUpdateComponentAssetStatus_Valid(t *testing.T) {
	assets := &stubAssetRepo{
		getResult: domain.ComponentAsset{
			ID:        "asset-1",
			AssetType: domain.AssetTypeDatasheet,
			Status:    domain.AssetStatusCandidate,
		},
	}
	svc := service.New(&stubComponentRepo{}, &stubProjectRepo{}, assets)

	result, err := svc.UpdateComponentAssetStatus(context.Background(), "asset-1", domain.AssetStatusVerified)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != domain.AssetStatusVerified {
		t.Errorf("expected status %q, got %q", domain.AssetStatusVerified, result.Status)
	}
}

func TestUpdateComponentAssetStatus_InvalidStatus_Rejected(t *testing.T) {
	svc := service.New(&stubComponentRepo{}, &stubProjectRepo{}, &stubAssetRepo{})

	_, err := svc.UpdateComponentAssetStatus(context.Background(), "asset-1", "bogus")
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestGetComponentWithAssets_Delegated(t *testing.T) {
	assets := &stubAssetRepo{
		detail: domain.ComponentWithAssets{
			Component: domain.Component{ID: "cid-1"},
			Assets:    []domain.ComponentAsset{{ID: "a1", AssetType: domain.AssetTypeSymbol}},
		},
	}
	svc := service.New(&stubComponentRepo{}, &stubProjectRepo{}, assets)

	detail, err := svc.GetComponentWithAssets(context.Background(), "cid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Component.ID != "cid-1" {
		t.Errorf("expected component ID %q, got %q", "cid-1", detail.Component.ID)
	}
	if len(detail.Assets) != 1 {
		t.Errorf("expected 1 asset, got %d", len(detail.Assets))
	}
}
