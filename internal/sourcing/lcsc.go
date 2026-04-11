package sourcing

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	easyeda "github.com/C-Ma-P/go-easyeda"
	lcsc "github.com/PatrickWalther/go-lcsc"
)

type LCSCProvider struct {
	client        *lcsc.Client
	bundleFetcher func(ctx context.Context, lcscID string, opts easyeda.FetchOptions) (*easyeda.ComponentBundle, error)
	enabled       bool
	currency      string
}

func NewLCSCProvider(config LCSCConfig) *LCSCProvider {
	if !config.Enabled {
		return &LCSCProvider{}
	}
	options := make([]lcsc.ClientOption, 0, 1)
	if strings.TrimSpace(config.Currency) != "" {
		options = append(options, lcsc.WithCurrency(config.Currency))
	}
	eClient := easyeda.NewClient()
	return &LCSCProvider{
		client: lcsc.NewClient(options...),
		bundleFetcher: func(ctx context.Context, lcscID string, opts easyeda.FetchOptions) (*easyeda.ComponentBundle, error) {
			return eClient.FetchComponentBundle(ctx, lcscID, opts)
		},
		enabled:  true,
		currency: strings.TrimSpace(config.Currency),
	}
}

func (p *LCSCProvider) Name() string {
	return ProviderLCSC
}

func (p *LCSCProvider) Enabled() bool {
	return p.enabled && p.client != nil
}

func (p *LCSCProvider) Search(ctx context.Context, query RequirementQuery) ([]SupplierOffer, error) {
	terms := providerSearchTerms(query)
	if len(terms) == 0 {
		return nil, nil
	}

	offers := make([]SupplierOffer, 0, 10)
	seen := map[string]struct{}{}
	for _, term := range terms {
		response, err := p.client.Search.Keyword(ctx, &lcsc.SearchRequest{Keyword: term})
		if err != nil {
			return nil, err
		}
		for _, product := range response.Products {
			offer := normalizeLCSCProduct(product)
			key := normalizePart(offer.SupplierPartNumber + offer.MPN)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			offers = append(offers, offer)
		}
		if len(offers) >= 10 {
			break
		}
	}

	return offers, nil
}

func (p *LCSCProvider) FriendlyError(err error) string {
	return err.Error()
}

func (p *LCSCProvider) ProbeOffer(ctx context.Context, offer SupplierOffer) (SupplierOffer, error) {
	if offer.SupplierPartNumber == "" {
		return offer, fmt.Errorf("missing supplier part number")
	}
	p.enrichOfferAssets(ctx, &offer)
	return offer, nil
}

func (p *LCSCProvider) LookupByPartCode(ctx context.Context, partCode string) (SupplierOffer, error) {
	if !p.Enabled() {
		return SupplierOffer{}, fmt.Errorf("LCSC provider not configured")
	}
	partCode = strings.TrimSpace(partCode)
	if partCode == "" {
		return SupplierOffer{}, fmt.Errorf("part code is required")
	}

	product, err := p.client.Product.Details(ctx, partCode)
	if err == nil {
		offer := normalizeLCSCProduct(*product)
		p.enrichOfferAssets(ctx, &offer)
		return offer, nil
	}
	if !errors.Is(err, lcsc.ErrNotFound) {
		return SupplierOffer{}, err
	}

	response, err := p.client.Search.Keyword(ctx, &lcsc.SearchRequest{Keyword: partCode})
	if err != nil {
		return SupplierOffer{}, err
	}
	for _, product := range response.Products {
		if strings.EqualFold(strings.TrimSpace(product.ProductCode), partCode) {
			offer := normalizeLCSCProduct(product)
			p.enrichOfferAssets(ctx, &offer)
			return offer, nil
		}
	}
	if len(response.Products) > 0 {
		offer := normalizeLCSCProduct(response.Products[0])
		p.enrichOfferAssets(ctx, &offer)
		return offer, nil
	}
	return SupplierOffer{}, fmt.Errorf("part %q not found on LCSC", partCode)
}

func normalizeLCSCProduct(product lcsc.Product) SupplierOffer {
	price := lowestLCSCPrice(product.ProductPriceList)
	raw := map[string]string{
		"catalog":       product.CatalogName,
		"parentCatalog": product.ParentCatalogName,
	}

	return SupplierOffer{
		Provider:           ProviderLCSC,
		Manufacturer:       strings.TrimSpace(product.BrandNameEn),
		MPN:                strings.TrimSpace(product.ProductModel),
		SupplierPartNumber: strings.TrimSpace(product.ProductCode),
		Description:        strings.TrimSpace(product.ProductIntroEn),
		Package:            strings.TrimSpace(product.EncapStandard),
		Stock:              intPointer(product.StockNumber),
		MOQ:                intPointer(product.MinPacketNumber),
		UnitPrice:          floatPointer(price),
		ProductURL:         product.GetProductURL(),
		DatasheetURL:       strings.TrimSpace(product.PdfURL),
		ImageURL:           strings.TrimSpace(product.ProductImageURL),
		HasSymbol:          false,
		HasFootprint:       false,
		HasDatasheet:       strings.TrimSpace(product.PdfURL) != "",
		Raw:                raw,
	}
}

func (p *LCSCProvider) enrichOfferAssets(ctx context.Context, offer *SupplierOffer) {
	if p == nil || p.bundleFetcher == nil || offer == nil || offer.SupplierPartNumber == "" {
		return
	}
	bundle, err := p.bundleFetcher(ctx, offer.SupplierPartNumber, easyeda.FetchOptions{})
	if err != nil || bundle == nil || bundle.Extracted == nil {
		return
	}
	meta := bundle.Extracted
	if !offer.HasSymbol && len(meta.SymbolRaw) > 0 && string(meta.SymbolRaw) != "null" {
		offer.HasSymbol = true
	}
	if !offer.HasFootprint && len(meta.FootprintRaw) > 0 && string(meta.FootprintRaw) != "null" {
		offer.HasFootprint = true
	}
	if !offer.HasDatasheet && strings.TrimSpace(meta.DatasheetURL) != "" {
		offer.HasDatasheet = true
		if strings.TrimSpace(offer.DatasheetURL) == "" {
			offer.DatasheetURL = strings.TrimSpace(meta.DatasheetURL)
		}
	}
}

func lowestLCSCPrice(prices []lcsc.PriceBreak) float64 {
	best := 0.0
	for _, priceBreak := range prices {
		price := float64(priceBreak.ProductPrice)
		if price <= 0 {
			continue
		}
		if best == 0 || price < best {
			best = price
		}
	}
	return best
}

func providerSearchTerms(query RequirementQuery) []string {
	terms := make([]string, 0, 4)
	for _, term := range query.SearchTerms {
		appendUniqueFold(&terms, term)
		if len(terms) >= 3 {
			break
		}
	}
	return terms
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func intPointer(value int) *int {
	if value <= 0 {
		return nil
	}
	copy := value
	return &copy
}

func floatPointer(value float64) *float64 {
	if value <= 0 {
		return nil
	}
	copy := value
	return &copy
}

func parseLooseInt(value string) int {
	builder := strings.Builder{}
	for _, r := range value {
		if r >= '0' && r <= '9' {
			builder.WriteRune(r)
		}
	}
	if builder.Len() == 0 {
		return 0
	}
	parsed, err := strconv.Atoi(builder.String())
	if err != nil {
		return 0
	}
	return parsed
}
