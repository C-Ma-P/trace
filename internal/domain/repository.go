package domain

import (
	"context"
	"time"
)

type ComponentRepository interface {
	UpsertAttributeDefinition(context.Context, AttributeDefinition) error
	CreateComponent(context.Context, Component) (Component, error)
	UpdateComponentMetadata(context.Context, Component) (Component, error)
	UpdateComponentInventory(context.Context, Component) (Component, error)
	ReplaceComponentAttributes(context.Context, string, []AttributeValue) error
	GetComponent(context.Context, string) (Component, error)
	DeleteComponent(context.Context, string) error
	ListComponentsByCategory(context.Context, Category) ([]Component, error)
	FindComponents(context.Context, ComponentFilter) ([]Component, error)
}

type ComponentAssetRepository interface {
	CreateComponentAsset(context.Context, ComponentAsset) (ComponentAsset, error)
	GetComponentAsset(context.Context, string) (ComponentAsset, error)
	ListComponentAssets(context.Context, string) ([]ComponentAsset, error)
	ListComponentAssetsByType(context.Context, string, AssetType) ([]ComponentAsset, error)
	UpdateComponentAsset(context.Context, ComponentAsset) (ComponentAsset, error)
	DeleteComponentAsset(context.Context, string) error
	SetSelectedComponentAsset(ctx context.Context, componentID string, assetType AssetType, assetID string) error
	ClearSelectedComponentAsset(ctx context.Context, componentID string, assetType AssetType) error
	GetComponentWithAssets(context.Context, string) (ComponentWithAssets, error)
}

type ProjectRepository interface {
	CreateProject(context.Context, Project) (Project, error)
	GetProject(context.Context, string) (Project, error)
	ListProjects(context.Context) ([]Project, error)
	UpdateProject(context.Context, Project) (Project, error)
	DeleteProject(context.Context, string) error
	ReplaceProjectRequirements(context.Context, string, []ProjectRequirement) error
	AddProjectRequirements(context.Context, string, []ProjectRequirement) error
	SetProjectImportMetadata(context.Context, string, *string, *string, *time.Time) error
	GetRequirement(context.Context, string) (ProjectRequirement, error)
	SetRequirementResolution(context.Context, string, *RequirementResolution) error

	// Part candidates
	AddPartCandidate(context.Context, ProjectPartCandidate) (ProjectPartCandidate, error)
	SetPreferredCandidate(ctx context.Context, requirementID, candidateID string) error
	ClearPreferredCandidate(ctx context.Context, requirementID string) error
	GetPartCandidate(ctx context.Context, candidateID string) (ProjectPartCandidate, error)
	RemovePartCandidate(context.Context, string) error
	ListPartCandidates(ctx context.Context, requirementID string) ([]ProjectPartCandidate, error)
	ListPartCandidatesByProject(ctx context.Context, projectID string) ([]ProjectPartCandidate, error)
	UpdatePartCandidateComponent(ctx context.Context, candidateID string, componentID string, origin CandidateOrigin) error

	// Saved supplier offers
	SaveSupplierOffer(context.Context, SavedSupplierOffer) (SavedSupplierOffer, error)
	GetSavedSupplierOffer(ctx context.Context, offerID string) (SavedSupplierOffer, error)
	RemoveSavedSupplierOffer(context.Context, string) error
	ListSavedSupplierOffers(ctx context.Context, requirementID string) ([]SavedSupplierOffer, error)
	ListSavedSupplierOffersByProject(ctx context.Context, projectID string) ([]SavedSupplierOffer, error)
	LinkSupplierOfferToComponent(ctx context.Context, offerID, componentID string) error
}

type InventoryBagRepository interface {
	CreateBag(context.Context, InventoryBag) (InventoryBag, error)
	GetBagByQRData(context.Context, string) (InventoryBag, error)
	ListBagsByComponent(context.Context, string) ([]InventoryBag, error)
	DeleteBag(context.Context, string) error
	FindComponentImageURL(context.Context, string) string
}

type PreferenceRepository interface {
	List(context.Context, string) (map[string]string, error)
	SetMany(context.Context, map[string]string) error
}
