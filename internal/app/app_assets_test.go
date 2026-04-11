package app

import (
	"testing"

	"trace/internal/domain"
)

func TestShouldSkipEasyEDAImport(t *testing.T) {
	cases := []struct {
		name        string
		assets      []domain.ComponentAsset
		wantSkip    bool
		wantWarning string
	}{
		{
			name:     "no existing easyeda assets",
			assets:   []domain.ComponentAsset{},
			wantSkip: false,
		},
		{
			name:        "existing footprint only",
			assets:      []domain.ComponentAsset{{Source: "easyeda", AssetType: domain.AssetTypeFootprint}},
			wantSkip:    false,
			wantWarning: "Some EasyEDA assets are already imported for this component; missing asset types will still be imported.",
		},
		{
			name: "existing symbol footprint and 3d model",
			assets: []domain.ComponentAsset{
				{Source: "easyeda", AssetType: domain.AssetTypeSymbol},
				{Source: "easyeda", AssetType: domain.AssetTypeFootprint},
				{Source: "easyeda", AssetType: domain.AssetType3DModel},
			},
			wantSkip:    true,
			wantWarning: "EasyEDA assets already imported for this component",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			skip, warnings := shouldSkipEasyEDAImport(tc.assets)
			if skip != tc.wantSkip {
				t.Fatalf("expected skip=%v, got %v", tc.wantSkip, skip)
			}
			if tc.wantWarning == "" {
				if len(warnings) != 0 {
					t.Fatalf("expected no warnings, got %v", warnings)
				}
				return
			}
			if len(warnings) != 1 || warnings[0] != tc.wantWarning {
				t.Fatalf("expected warning %q, got %v", tc.wantWarning, warnings)
			}
		})
	}
}

func TestSummarizeExistingEasyEDAAssets(t *testing.T) {
	assets := []domain.ComponentAsset{
		{ID: "s1", Source: "easyeda", AssetType: domain.AssetTypeSymbol},
		{ID: "f1", Source: "easyeda", AssetType: domain.AssetTypeFootprint},
		{ID: "m1", Source: "easyeda", AssetType: domain.AssetType3DModel},
	}
	gotSymbol, gotFootprint, got3D, symbolID, footprintID, model3DID := summarizeExistingEasyEDAAssets(assets)
	if !gotSymbol || !gotFootprint || !got3D {
		t.Fatal("expected all easyeda assets to be detected")
	}
	if symbolID != "s1" || footprintID != "f1" || model3DID != "m1" {
		t.Fatalf("unexpected asset ids: %q %q %q", symbolID, footprintID, model3DID)
	}
}
