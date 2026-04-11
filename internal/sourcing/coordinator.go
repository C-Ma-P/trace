package sourcing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/C-Ma-P/trace/internal/activity"
	"github.com/C-Ma-P/trace/internal/cache"

	"golang.org/x/sync/singleflight"
)

const defaultSourcingCacheTTL = 5 * time.Minute

// Coordinator wraps a sourcing.Service and adds request caching and dedupe behavior.
type Coordinator struct {
	service           *Service
	configFingerprint string
	reporter          *activity.Reporter
	group             singleflight.Group
	allCache          *cache.TTLCache[string, SourceResult]
	byProviderCache   *cache.TTLCache[string, SourceResult]
	lookupCache       *cache.TTLCache[string, SupplierOffer]
	probeCache        *cache.TTLCache[string, SupplierOffer]
}

func NewCoordinator(config Config) *Coordinator {
	return NewCoordinatorWithEmitter(config, activity.NopEmitter)
}

func NewCoordinatorWithEmitter(config Config, emitter activity.Emitter) *Coordinator {
	return newCoordinatorFromService(NewFromConfig(config), NormalizedConfigFingerprint(config), emitter)
}

func NewCoordinatorFromService(service *Service) *Coordinator {
	return NewCoordinatorFromServiceWithEmitter(service, activity.NopEmitter)
}

func NewCoordinatorFromServiceWithEmitter(service *Service, emitter activity.Emitter) *Coordinator {
	return newCoordinatorFromService(service, "", emitter)
}

func newCoordinatorFromService(service *Service, fingerprint string, emitter activity.Emitter) *Coordinator {
	return &Coordinator{
		service:           service,
		reporter:          activity.NewReporter(nil, emitter),
		configFingerprint: fingerprint,
		allCache:          cache.NewTTLCache[string, SourceResult](defaultSourcingCacheTTL),
		byProviderCache:   cache.NewTTLCache[string, SourceResult](defaultSourcingCacheTTL),
		lookupCache:       cache.NewTTLCache[string, SupplierOffer](defaultSourcingCacheTTL),
		probeCache:        cache.NewTTLCache[string, SupplierOffer](defaultSourcingCacheTTL),
	}
}

func (c *Coordinator) ConfigFingerprint() string {
	return c.configFingerprint
}

func (c *Coordinator) Service() *Service {
	return c.service
}

func (c *Coordinator) Providers() []ProviderInfo {
	return c.service.Providers()
}

func (c *Coordinator) Source(ctx context.Context, query RequirementQuery) SourceResult {
	metadata := eventMetadata(query, "", -1)
	c.reporter.Emit(activity.NewSourcingEvent(activity.SeverityInfo, "request-started", "Sourcing request started", metadata))

	key := BuildRequirementCacheKey(query, c.configFingerprint)
	if result, ok := c.allCache.Get(key); ok {
		c.reporter.Sourcing(activity.Sourcing{
			Severity: activity.SeverityInfo,
			Kind:     "cache-hit",
			Message:  "Sourcing cache hit",
			Metadata: metadata,
		})
		return result
	}
	c.reporter.Sourcing(activity.Sourcing{
		Severity: activity.SeverityInfo,
		Kind:     "cache-miss",
		Message:  "Sourcing cache miss",
		Metadata: metadata,
	})

	value, err, shared := c.group.Do("source:"+key, func() (any, error) {
		result := c.service.Source(ctx, query)
		c.allCache.Set(key, result)
		return result, nil
	})
	if shared {
		c.reporter.Sourcing(activity.Sourcing{
			Severity: activity.SeverityInfo,
			Kind:     "deduped",
			Message:  "Sourcing request deduped",
			Metadata: metadata,
		})
	}
	if err != nil {
		c.reporter.Sourcing(activity.Sourcing{
			Severity: activity.SeverityError,
			Kind:     "request-failed",
			Message:  "Sourcing request failed",
			Metadata: mergeMetadata(metadata, map[string]any{"error": err.Error()}),
		})
		return SourceResult{Providers: []ProviderStatus{{Provider: "", Status: "error", Error: err.Error()}}}
	}
	result := value.(SourceResult)
	c.reporter.Emit(activity.NewSourcingEvent(activity.SeveritySuccess, "request-completed", "Sourcing request completed", mergeMetadata(metadata, map[string]any{"offers": len(result.Offers)})))
	return result
}

func (c *Coordinator) SourceFromProvider(ctx context.Context, query RequirementQuery, providerName string) SourceResult {
	metadata := eventMetadata(query, providerName, -1)
	c.reporter.Emit(activity.NewSourcingEvent(activity.SeverityInfo, "request-started", "Provider sourcing started", metadata))

	key := BuildRequirementProviderCacheKey(query, providerName, c.configFingerprint)
	if result, ok := c.byProviderCache.Get(key); ok {
		c.reporter.Sourcing(activity.Sourcing{
			Severity: activity.SeverityInfo,
			Kind:     "cache-hit",
			Message:  "Provider cache hit",
			Metadata: metadata,
		})
		return result
	}
	c.reporter.Sourcing(activity.Sourcing{
		Severity: activity.SeverityInfo,
		Kind:     "cache-miss",
		Message:  "Provider cache miss",
		Metadata: metadata,
	})

	value, err, shared := c.group.Do("provider:"+key, func() (any, error) {
		result := c.service.SourceFromProvider(ctx, query, providerName)
		c.byProviderCache.Set(key, result)
		return result, nil
	})
	if shared {
		c.reporter.Sourcing(activity.Sourcing{
			Severity: activity.SeverityInfo,
			Kind:     "deduped",
			Message:  "Provider request deduped",
			Metadata: metadata,
		})
	}
	if err != nil {
		c.reporter.Sourcing(activity.Sourcing{
			Severity: activity.SeverityError,
			Kind:     "request-failed",
			Message:  "Provider request failed",
			Metadata: mergeMetadata(metadata, map[string]any{"error": err.Error()}),
		})
		return SourceResult{Providers: []ProviderStatus{{Provider: providerName, Status: "error", Error: err.Error()}}}
	}
	result := value.(SourceResult)
	c.reporter.Emit(activity.NewSourcingEvent(activity.SeveritySuccess, "provider-offers", "Provider returned offers", mergeMetadata(metadata, map[string]any{"offers": len(result.Offers)})))
	return result
}

func (c *Coordinator) LookupByVendorPartID(ctx context.Context, vendor, partID string) (SupplierOffer, error) {
	metadata := map[string]any{"vendor": vendor, "partId": partID}
	c.reporter.Emit(activity.NewSourcingEvent(activity.SeverityInfo, "lookup-started", "Vendor lookup started", metadata))

	key := LookupVendorPartIDCacheKey(vendor, partID, c.configFingerprint)
	if offer, ok := c.lookupCache.Get(key); ok {
		c.reporter.Sourcing(activity.Sourcing{
			Severity: activity.SeverityInfo,
			Kind:     "cache-hit",
			Message:  "Vendor lookup cache hit",
			Metadata: metadata,
		})
		return offer, nil
	}
	c.reporter.Sourcing(activity.Sourcing{
		Severity: activity.SeverityInfo,
		Kind:     "cache-miss",
		Message:  "Vendor lookup cache miss",
		Metadata: metadata,
	})

	value, err, shared := c.group.Do("lookup:"+key, func() (any, error) {
		offer, lookupErr := c.service.LookupByVendorPartID(ctx, vendor, partID)
		if lookupErr != nil {
			return SupplierOffer{}, lookupErr
		}
		c.lookupCache.Set(key, offer)
		return offer, nil
	})
	if shared {
		c.reporter.Sourcing(activity.Sourcing{
			Severity: activity.SeverityInfo,
			Kind:     "deduped",
			Message:  "Vendor lookup deduped",
			Metadata: metadata,
		})
	}
	if err != nil {
		c.reporter.Emit(activity.NewSourcingEvent(activity.SeverityError, "lookup-failed", "Vendor lookup failed", mergeMetadata(metadata, map[string]any{"error": err.Error()})))
		return SupplierOffer{}, err
	}
	result := value.(SupplierOffer)
	c.reporter.Emit(activity.NewSourcingEvent(activity.SeveritySuccess, "lookup-completed", "Vendor lookup completed", mergeMetadata(metadata, map[string]any{"mpn": result.MPN, "provider": result.Provider})))
	return result, nil
}

func (c *Coordinator) ProbeOffer(ctx context.Context, offer SupplierOffer) (SupplierOffer, error) {
	metadata := map[string]any{"provider": offer.Provider, "mpn": offer.MPN, "supplierPartNumber": offer.SupplierPartNumber}
	c.reporter.Emit(activity.NewAssetProbeEvent(activity.SeverityInfo, "probe-started", "Asset probe started", metadata))

	key := ProbeOfferCacheKey(offer, c.configFingerprint)
	if result, ok := c.probeCache.Get(key); ok {
		c.reporter.AssetProbe(activity.AssetProbe{
			Severity: activity.SeverityInfo,
			Kind:     "cache-hit",
			Message:  "Asset probe cache hit",
			Metadata: metadata,
		})
		return result, nil
	}
	c.reporter.AssetProbe(activity.AssetProbe{
		Severity: activity.SeverityInfo,
		Kind:     "cache-miss",
		Message:  "Asset probe cache miss",
		Metadata: metadata,
	})

	value, err, shared := c.group.Do("probe:"+key, func() (any, error) {
		result, probeErr := c.service.ProbeOffer(ctx, offer)
		c.probeCache.Set(key, result)
		return result, probeErr
	})
	if shared {
		c.reporter.AssetProbe(activity.AssetProbe{
			Severity: activity.SeverityInfo,
			Kind:     "deduped",
			Message:  "Asset probe deduped",
			Metadata: metadata,
		})
	}
	result := value.(SupplierOffer)
	if err != nil {
		c.reporter.Emit(activity.NewAssetProbeEvent(activity.SeverityError, "probe-failed", "Asset probe failed", mergeMetadata(metadata, map[string]any{"error": err.Error()})))
		return result, err
	}
	c.reporter.Emit(activity.NewAssetProbeEvent(activity.SeveritySuccess, "probe-completed", "Asset probe succeeded", mergeMetadata(metadata, map[string]any{"assetProbeState": result.AssetProbeState, "assetProbeError": result.AssetProbeError})))
	return result, nil
}

func mergeMetadata(base, extra map[string]any) map[string]any {
	if len(base) == 0 {
		return copyMetadata(extra)
	}
	out := copyMetadata(base)
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func copyMetadata(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	out := make(map[string]any, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}

func eventMetadata(query RequirementQuery, provider string, offerCount int) map[string]any {
	meta := map[string]any{"requirementId": query.RequirementID}
	if query.MPN != "" {
		meta["mpn"] = query.MPN
	}
	if query.Manufacturer != "" {
		meta["manufacturer"] = query.Manufacturer
	}
	if provider != "" {
		meta["provider"] = provider
	}
	if offerCount >= 0 {
		meta["offerCount"] = offerCount
	}
	return meta
}

func ProbeOfferCacheKey(offer SupplierOffer, configFingerprint string) string {
	return combineFingerprint(configFingerprint, fmt.Sprintf("probe:%s|%s|%s", normalizeText(offer.Provider), normalizePart(offer.SupplierPartNumber), normalizePart(offer.MPN)))
}

func NormalizedConfigFingerprint(config Config) string {
	parts := []string{
		fmt.Sprintf("dk:%t|%s|%s|%s|%s|%s|%s", config.DigiKey.Enabled, normalizeText(config.DigiKey.ClientID), normalizeText(config.DigiKey.ClientSecret), normalizeText(config.DigiKey.CustomerID), normalizeText(config.DigiKey.Site), normalizeText(config.DigiKey.Language), normalizeText(config.DigiKey.Currency)),
		fmt.Sprintf("mouser:%t|%s", config.Mouser.Enabled, normalizeText(config.Mouser.APIKey)),
		fmt.Sprintf("lcsc:%t|%s", config.LCSC.Enabled, normalizeText(config.LCSC.Currency)),
	}
	hash := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(hash[:])
}

func BuildRequirementCacheKey(query RequirementQuery, configFingerprint string) string {
	return combineFingerprint(configFingerprint, buildQueryFingerprint(query))
}

func BuildRequirementProviderCacheKey(query RequirementQuery, providerName, configFingerprint string) string {
	provider := normalizeText(providerName)
	return combineFingerprint(configFingerprint, fmt.Sprintf("provider:%s|%s", provider, buildQueryFingerprint(query)))
}

func LookupVendorPartIDCacheKey(vendor, partID, configFingerprint string) string {
	return combineFingerprint(configFingerprint, fmt.Sprintf("lookup:%s|%s", normalizeText(vendor), normalizePart(partID)))
}

func combineFingerprint(fingerprint, body string) string {
	if fingerprint == "" {
		return body
	}
	return fingerprint + ":" + body
}

func buildQueryFingerprint(query RequirementQuery) string {
	pieces := []string{
		normalizeText(string(query.Category)),
		normalizeText(query.RequirementName),
		fmt.Sprintf("quantity:%d", query.Quantity),
		normalizeText(query.Manufacturer),
		normalizeText(query.MPN),
		normalizeText(query.Package),
	}

	pieces = append(pieces, normalizedTerms(query.TextTerms)...)
	pieces = append(pieces, normalizedTerms(query.ValueTerms)...)
	pieces = append(pieces, normalizedTerms(query.SearchTerms)...)

	if query.SelectedComponent != nil {
		pieces = append(pieces,
			normalizeText(query.SelectedComponent.Manufacturer),
			normalizeText(query.SelectedComponent.MPN),
			normalizeText(query.SelectedComponent.Package),
			normalizeText(query.SelectedComponent.Description),
		)
	}

	sort.Strings(pieces)
	return strings.Join(filterEmpty(pieces), "|")
}

func normalizedTerms(values []string) []string {
	set := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized := normalizeText(value)
		if normalized == "" {
			continue
		}
		if _, ok := set[normalized]; ok {
			continue
		}
		set[normalized] = struct{}{}
		out = append(out, normalized)
	}
	sort.Strings(out)
	return out
}
