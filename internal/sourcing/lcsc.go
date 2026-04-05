package sourcing

import (
	"context"
	"strconv"
	"strings"

	lcsc "github.com/PatrickWalther/go-lcsc"
)

type LCSCProvider struct {
	client   *lcsc.Client
	enabled  bool
	currency string
}

func NewLCSCProvider(config LCSCConfig) *LCSCProvider {
	if !config.Enabled {
		return &LCSCProvider{}
	}
	options := make([]lcsc.ClientOption, 0, 1)
	if strings.TrimSpace(config.Currency) != "" {
		options = append(options, lcsc.WithCurrency(config.Currency))
	}
	return &LCSCProvider{
		client:   lcsc.NewClient(options...),
		enabled:  true,
		currency: strings.TrimSpace(config.Currency),
	}
}

func (p *LCSCProvider) Name() string {
	return "LCSC"
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

func normalizeLCSCProduct(product lcsc.Product) SupplierOffer {
	price := lowestLCSCPrice(product.ProductPriceList)
	raw := map[string]string{
		"catalog":       product.CatalogName,
		"parentCatalog": product.ParentCatalogName,
	}

	return SupplierOffer{
		Provider:           "LCSC",
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
		Raw:                raw,
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
