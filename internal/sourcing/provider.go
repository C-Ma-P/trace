package sourcing

import "context"

type Provider interface {
	Name() string
	Enabled() bool
	Search(ctx context.Context, query RequirementQuery) ([]SupplierOffer, error)
	FriendlyError(err error) string
}
