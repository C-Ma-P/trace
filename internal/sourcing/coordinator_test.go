package sourcing_test

import (
	"context"
	"testing"

	"trace/internal/sourcing"
)

type countingProvider struct {
	callCount int
	query     sourcing.RequirementQuery
}

type countingProbeProvider struct {
	countingProvider
	probeCount int
}

func (p *countingProvider) Name() string {
	return "count"
}

func (p *countingProvider) Enabled() bool {
	return true
}

func (p *countingProvider) Search(_ context.Context, query sourcing.RequirementQuery) ([]sourcing.SupplierOffer, error) {
	p.callCount++
	p.query = query
	return []sourcing.SupplierOffer{{Provider: "count", MPN: "XYZ"}}, nil
}

func (p *countingProvider) FriendlyError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (p *countingProbeProvider) ProbeOffer(_ context.Context, offer sourcing.SupplierOffer) (sourcing.SupplierOffer, error) {
	p.probeCount++
	return offer, nil
}

func TestCoordinator_CachesSourceResult(t *testing.T) {
	provider := &countingProvider{}
	coord := sourcing.NewCoordinatorFromService(sourcing.NewService(provider))

	query := sourcing.RequirementQuery{Category: "resistor", RequirementName: "1k pull-up", Quantity: 4}
	first := coord.Source(context.Background(), query)
	second := coord.Source(context.Background(), query)

	if provider.callCount != 1 {
		t.Fatalf("expected exactly one provider call, got %d", provider.callCount)
	}
	if len(first.Offers) != 1 || len(second.Offers) != 1 {
		t.Fatalf("expected one offer in both results")
	}
}

func TestCoordinator_CacheKeyIgnoresRequirementID(t *testing.T) {
	provider := &countingProvider{}
	coord := sourcing.NewCoordinatorFromService(sourcing.NewService(provider))

	queryA := sourcing.RequirementQuery{RequirementID: "req-1", RequirementName: "generic part", Category: "resistor", Quantity: 2}
	queryB := sourcing.RequirementQuery{RequirementID: "req-2", RequirementName: "generic part", Category: "resistor", Quantity: 2}

	coord.Source(context.Background(), queryA)
	coord.Source(context.Background(), queryB)

	if provider.callCount != 1 {
		t.Fatalf("expected single cache entry for identical query content, got %d calls", provider.callCount)
	}
}

func TestCoordinator_LookupVendorPartIDCacheKey_IsStable(t *testing.T) {
	keyA := sourcing.LookupVendorPartIDCacheKey("Mouser", "abc-123", "fingerprint")
	keyB := sourcing.LookupVendorPartIDCacheKey("mouser", "ABC-123", "fingerprint")
	if keyA != keyB {
		t.Fatalf("expected equivalent lookup cache keys, got %q and %q", keyA, keyB)
	}
}

func TestCoordinator_ConfigFingerprintChangesService(t *testing.T) {
	configA := sourcing.Config{Mouser: sourcing.MouserConfig{Enabled: true, APIKey: "A"}}
	configB := sourcing.Config{Mouser: sourcing.MouserConfig{Enabled: true, APIKey: "B"}}

	if sourcing.NormalizedConfigFingerprint(configA) == sourcing.NormalizedConfigFingerprint(configB) {
		t.Fatal("expected different config fingerprints for different API keys")
	}
}

func TestCoordinator_ProbeOfferCachesResult(t *testing.T) {
	provider := &countingProbeProvider{}
	coord := sourcing.NewCoordinatorFromService(sourcing.NewService(provider))
	offer := sourcing.SupplierOffer{Provider: "count", SupplierPartNumber: "ABC123", MPN: "XYZ"}

	first, err := coord.ProbeOffer(context.Background(), offer)
	if err != nil {
		t.Fatalf("probe failed: %v", err)
	}
	second, err := coord.ProbeOffer(context.Background(), offer)
	if err != nil {
		t.Fatalf("probe failed: %v", err)
	}

	if provider.probeCount != 1 {
		t.Fatalf("expected exactly one probe call, got %d", provider.probeCount)
	}
	if first.Provider != second.Provider || first.SupplierPartNumber != second.SupplierPartNumber {
		t.Fatal("expected same offer returned from cache")
	}
}
