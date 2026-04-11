package sourcing

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	ProviderDigiKey = "DigiKey"
	ProviderMouser  = "Mouser"
	ProviderLCSC    = "LCSC"
)

type Config struct {
	DigiKey DigiKeyConfig
	Mouser  MouserConfig
	LCSC    LCSCConfig
}

type DigiKeyConfig struct {
	Enabled      bool
	ClientID     string
	ClientSecret string
	CustomerID   string
	Site         string
	Language     string
	Currency     string
}

type MouserConfig struct {
	Enabled bool
	APIKey  string
}

type LCSCConfig struct {
	Enabled  bool
	Currency string
}

type Service struct {
	providers []Provider
	currency  string
}

func NewService(providers ...Provider) *Service {
	return &Service{providers: providers}
}

func NewFromConfig(config Config) *Service {
	svc := NewService(
		NewDigiKeyProvider(config.DigiKey),
		NewMouserProvider(config.Mouser),
		NewLCSCProvider(config.LCSC),
	)
	svc.currency = firstCurrency(config.DigiKey.Currency, config.LCSC.Currency)
	return svc
}

func NewFromEnv() *Service {
	config := LoadConfigFromEnv()
	return NewFromConfig(config)
}

func LoadConfigFromEnv() Config {
	digiKeyEnabled := true
	if raw := strings.TrimSpace(os.Getenv("DIGIKEY_ENABLED")); raw != "" {
		if parsed, err := strconv.ParseBool(raw); err == nil {
			digiKeyEnabled = parsed
		}
	}

	mouserEnabled := true
	if raw := strings.TrimSpace(os.Getenv("MOUSER_ENABLED")); raw != "" {
		if parsed, err := strconv.ParseBool(raw); err == nil {
			mouserEnabled = parsed
		}
	}

	lcscEnabled := true
	if raw := strings.TrimSpace(os.Getenv("LCSC_ENABLED")); raw != "" {
		if parsed, err := strconv.ParseBool(raw); err == nil {
			lcscEnabled = parsed
		}
	}

	return Config{
		DigiKey: DigiKeyConfig{
			Enabled:      digiKeyEnabled,
			ClientID:     strings.TrimSpace(os.Getenv("DIGIKEY_CLIENT_ID")),
			ClientSecret: strings.TrimSpace(os.Getenv("DIGIKEY_CLIENT_SECRET")),
			CustomerID:   strings.TrimSpace(os.Getenv("DIGIKEY_CUSTOMER_ID")),
			Site:         strings.TrimSpace(os.Getenv("DIGIKEY_SITE")),
			Language:     strings.TrimSpace(os.Getenv("DIGIKEY_LANGUAGE")),
			Currency:     strings.TrimSpace(os.Getenv("DIGIKEY_CURRENCY")),
		},
		Mouser: MouserConfig{
			Enabled: mouserEnabled,
			APIKey:  strings.TrimSpace(os.Getenv("MOUSER_API_KEY")),
		},
		LCSC: LCSCConfig{
			Enabled:  lcscEnabled,
			Currency: strings.TrimSpace(os.Getenv("LCSC_CURRENCY")),
		},
	}
}

// ProviderInfo describes a sourcing provider for informational purposes.
type ProviderInfo struct {
	Name    string
	Enabled bool
}

type AssetProbeProvider interface {
	Provider
	ProbeOffer(ctx context.Context, offer SupplierOffer) (SupplierOffer, error)
}

// Providers returns metadata about each configured provider.
func (s *Service) Providers() []ProviderInfo {
	out := make([]ProviderInfo, len(s.providers))
	for i, p := range s.providers {
		out[i] = ProviderInfo{Name: p.Name(), Enabled: p.Enabled()}
	}
	return out
}

// SourceFromProvider searches a single named provider and returns its result.
// If the provider is not found or not enabled, an error status is returned.
func (s *Service) SourceFromProvider(ctx context.Context, query RequirementQuery, providerName string) SourceResult {
	result := SourceResult{
		Offers:    make([]SupplierOffer, 0, 8),
		Providers: make([]ProviderStatus, 0, 1),
		Currency:  s.currency,
	}
	for _, provider := range s.providers {
		if !strings.EqualFold(provider.Name(), providerName) {
			continue
		}
		status := ProviderStatus{Provider: provider.Name()}
		if !provider.Enabled() {
			status.Status = "disabled"
			status.Error = "Provider is not configured"
			result.Providers = append(result.Providers, status)
			return result
		}
		offers, err := provider.Search(ctx, query)
		if err != nil {
			status.Status = "error"
			status.Error = provider.FriendlyError(err)
			result.Providers = append(result.Providers, status)
			return result
		}
		status.Status = "success"
		status.OfferCount = len(offers)
		result.Providers = append(result.Providers, status)
		for i := range offers {
			if offers[i].AssetProbeState == "" {
				offers[i].AssetProbeState = AssetProbeStateUnknown
			}
		}
		result.Offers = RankOffers(query, offers)
		return result
	}
	result.Providers = append(result.Providers, ProviderStatus{
		Provider: providerName,
		Status:   "error",
		Error:    "provider not found",
	})
	return result
}

func (s *Service) Source(ctx context.Context, query RequirementQuery) SourceResult {
	result := SourceResult{
		Offers:    make([]SupplierOffer, 0, 16),
		Providers: make([]ProviderStatus, 0, len(s.providers)),
		Currency:  s.currency,
	}

	for _, provider := range s.providers {
		status := ProviderStatus{Provider: provider.Name()}
		if !provider.Enabled() {
			status.Status = "disabled"
			status.Error = "Provider is not configured"
			result.Providers = append(result.Providers, status)
			continue
		}

		offers, err := provider.Search(ctx, query)
		if err != nil {
			status.Status = "error"
			status.Error = provider.FriendlyError(err)
			result.Providers = append(result.Providers, status)
			continue
		}

		status.Status = "success"
		status.OfferCount = len(offers)
		result.Providers = append(result.Providers, status)
		for i := range offers {
			if offers[i].AssetProbeState == "" {
				offers[i].AssetProbeState = AssetProbeStateUnknown
			}
		}
		result.Offers = append(result.Offers, offers...)
	}

	result.Offers = RankOffers(query, dedupeOffers(result.Offers))
	return result
}

func dedupeOffers(offers []SupplierOffer) []SupplierOffer {
	seen := make(map[string]int, len(offers))
	result := make([]SupplierOffer, 0, len(offers))
	for _, offer := range offers {
		key := normalizeText(strings.Join([]string{offer.Provider, normalizePart(offer.SupplierPartNumber), normalizePart(offer.MPN)}, "|"))
		if existing, ok := seen[key]; ok {
			if optionalInt(offer.Stock) > optionalInt(result[existing].Stock) {
				result[existing] = offer
			}
			continue
		}
		seen[key] = len(result)
		result = append(result, offer)
	}
	return result
}

func (s *Service) LookupByVendorPartID(ctx context.Context, vendor, partID string) (SupplierOffer, error) {
	for _, p := range s.providers {
		if !strings.EqualFold(p.Name(), vendor) || !p.Enabled() {
			continue
		}
		switch v := p.(type) {
		case *LCSCProvider:
			return v.LookupByPartCode(ctx, partID)
		case *MouserProvider:
			return v.LookupByPartNumber(ctx, partID)
		default:
			return SupplierOffer{}, fmt.Errorf("provider %q does not support barcode lookup", vendor)
		}
	}
	return SupplierOffer{}, fmt.Errorf("provider %q not found or not enabled", vendor)
}

func (s *Service) ProbeOffer(ctx context.Context, offer SupplierOffer) (SupplierOffer, error) {
	for _, p := range s.providers {
		if !strings.EqualFold(p.Name(), offer.Provider) || !p.Enabled() {
			continue
		}
		if probeProvider, ok := p.(AssetProbeProvider); ok {
			result, err := probeProvider.ProbeOffer(ctx, offer)
			if err != nil {
				result.AssetProbeState = AssetProbeStateError
				result.AssetProbeError = err.Error()
			}
			if result.AssetProbeState == "" {
				result.AssetProbeState = AssetProbeStateProbed
			}
			return result, err
		}
		if offer.AssetProbeState == "" {
			offer.AssetProbeState = AssetProbeStateUnknown
		}
		return offer, nil
	}
	return SupplierOffer{}, fmt.Errorf("provider %q not found or not enabled", offer.Provider)
}
