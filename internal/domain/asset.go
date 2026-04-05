package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

// AssetType identifies the kind of EDA asset.
type AssetType string

const (
	AssetTypeSymbol    AssetType = "symbol"
	AssetTypeFootprint AssetType = "footprint"
	AssetType3DModel   AssetType = "3d_model"
	AssetTypeDatasheet AssetType = "datasheet"
)

var validAssetTypes = map[AssetType]bool{
	AssetTypeSymbol:    true,
	AssetTypeFootprint: true,
	AssetType3DModel:   true,
	AssetTypeDatasheet: true,
}

func (t AssetType) Valid() bool {
	return validAssetTypes[t]
}

// AssetStatus tracks the lifecycle status of an attached asset.
type AssetStatus string

const (
	AssetStatusCandidate AssetStatus = "candidate"
	AssetStatusSelected  AssetStatus = "selected"
	AssetStatusVerified  AssetStatus = "verified"
	AssetStatusRejected  AssetStatus = "rejected"
)

var validAssetStatuses = map[AssetStatus]bool{
	AssetStatusCandidate: true,
	AssetStatusSelected:  true,
	AssetStatusVerified:  true,
	AssetStatusRejected:  true,
}

func (s AssetStatus) Valid() bool {
	return validAssetStatuses[s]
}

// ComponentAsset is a candidate EDA asset attached to a component.
type ComponentAsset struct {
	ID           string          `db:"id"`
	ComponentID  string          `db:"component_id"`
	AssetType    AssetType       `db:"asset_type"`
	Source       string          `db:"source"`
	Status       AssetStatus     `db:"status"`
	Label        string          `db:"label"`
	URLOrPath    string          `db:"url_or_path"`
	PreviewURL   *string         `db:"preview_url"`
	MetadataJSON json.RawMessage `db:"metadata_json"`
	CreatedAt    time.Time       `db:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at"`
}

// selectedColumnForType returns the components column name for the given asset type.
func SelectedColumnForType(t AssetType) (string, error) {
	switch t {
	case AssetTypeSymbol:
		return "selected_symbol_asset_id", nil
	case AssetTypeFootprint:
		return "selected_footprint_asset_id", nil
	case AssetType3DModel:
		return "selected_3d_model_asset_id", nil
	case AssetTypeDatasheet:
		return "selected_datasheet_asset_id", nil
	default:
		return "", fmt.Errorf("unknown asset type %q", t)
	}
}

// ComponentWithAssets is a read model that bundles a component with its selected
// and attached assets for easy downstream consumption.
type ComponentWithAssets struct {
	Component              Component        `json:"component"`
	SelectedSymbolAsset    *ComponentAsset  `json:"selectedSymbolAsset"`
	SelectedFootprintAsset *ComponentAsset  `json:"selectedFootprintAsset"`
	Selected3DModelAsset   *ComponentAsset  `json:"selected3dModelAsset"`
	SelectedDatasheetAsset *ComponentAsset  `json:"selectedDatasheetAsset"`
	Assets                 []ComponentAsset `json:"assets"`
}

// ErrAssetNotOwned is returned when attempting to select an asset that does not
// belong to the target component.
type ErrAssetNotOwned struct {
	AssetID     string
	ComponentID string
}

func (e ErrAssetNotOwned) Error() string {
	return fmt.Sprintf("asset %q does not belong to component %q", e.AssetID, e.ComponentID)
}

// ErrAssetTypeMismatch is returned when the asset type does not match the
// selected slot being set.
type ErrAssetTypeMismatch struct {
	AssetType AssetType
	SlotType  AssetType
}

func (e ErrAssetTypeMismatch) Error() string {
	return fmt.Sprintf("asset type %q does not match slot type %q", e.AssetType, e.SlotType)
}
