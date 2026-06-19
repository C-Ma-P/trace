package sourcing

import "context"

type Provider interface {
	Name() string
	Enabled() bool
	Search(ctx context.Context, query RequirementQuery) ([]SupplierOffer, error)
	FriendlyError(err error) string
}

type ManufacturerPartLookupProvider interface {
	LookupByPartNumberAndManufacturer(ctx context.Context, partNumber, manufacturer string) (SupplierOffer, error)
}

type VendorPartLookupProvider interface {
	LookupByVendorPartID(ctx context.Context, vendor, partID string) (SupplierOffer, error)
}
