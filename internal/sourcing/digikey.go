package sourcing

import (
	"context"
	"errors"
	"strings"

	digikey "github.com/PatrickWalther/go-digikey"
)

type DigiKeyProvider struct {
	client *digikey.Client
	config DigiKeyConfig
}

func NewDigiKeyProvider(config DigiKeyConfig) *DigiKeyProvider {
	if !config.Enabled {
		return &DigiKeyProvider{config: config}
	}
	if config.ClientID == "" || config.ClientSecret == "" {
		return &DigiKeyProvider{config: config}
	}

	locale := digikey.DefaultLocale()
	if config.Site != "" {
		locale.Site = config.Site
	}
	if config.Language != "" {
		locale.Language = config.Language
	}
	if config.Currency != "" {
		locale.Currency = config.Currency
	}

	options := []digikey.ClientOption{digikey.WithLocale(locale)}
	if config.CustomerID != "" {
		options = append(options, digikey.WithCustomerID(config.CustomerID))
	}

	return &DigiKeyProvider{
		client: digikey.NewClient(config.ClientID, config.ClientSecret, options...),
		config: config,
	}
}

func (p *DigiKeyProvider) Name() string {
	return ProviderDigiKey
}

func (p *DigiKeyProvider) Enabled() bool {
	return p.client != nil
}

func (p *DigiKeyProvider) Search(ctx context.Context, query RequirementQuery) ([]SupplierOffer, error) {
	terms := providerSearchTerms(query)
	if len(terms) == 0 {
		return nil, nil
	}

	offers := make([]SupplierOffer, 0, 12)
	seen := map[string]struct{}{}
	for _, term := range terms {
		response, err := p.client.Search.KeywordSearch(ctx, &digikey.SearchRequest{
			Keywords: term,
			Limit:    8,
		})
		if err != nil {
			return nil, err
		}

		products := append([]digikey.Product{}, response.ExactMatches...)
		products = append(products, response.Products...)
		for _, product := range products {
			offer := normalizeDigiKeyProduct(product)
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

// digikeyErrorBody is the JSON structure DigiKey returns for auth/API errors.
type digikeyErrorBody struct {
	ErrorMessage string `json:"ErrorMessage"`
	ErrorDetails string `json:"ErrorDetails"`
}

func (p *DigiKeyProvider) FriendlyError(err error) string {
	var authErr *digikey.AuthError
	if errors.As(err, &authErr) {
		if authErr.Description != "" {
			return "Invalid credentials: " + authErr.Description
		}
		return "Invalid credentials. Check your Client ID and Client Secret."
	}

	var apiErr *digikey.APIError
	if errors.As(err, &apiErr) {
		// Try to parse the raw JSON body in Details for a human-readable message.
		/*if apiErr.Details != "" {
			var body digikeyErrorBody
			if jsonErr := json.Unmarshal([]byte(apiErr.Details), &body); jsonErr == nil && body.ErrorMessage != "" {
				if body.ErrorDetails != "" {
					return body.ErrorMessage + ": " + body.ErrorDetails
				}
				return body.ErrorMessage
			}
		}*/
		if errors.Is(err, digikey.ErrUnauthorized) {
			return "Invalid credentials. Check your Client ID and Client Secret."
		}
		if errors.Is(err, digikey.ErrRateLimitExceeded) {
			return "Rate limit exceeded. Try again later."
		}
		if apiErr.Message != "" {
			return apiErr.Message
		}
	}

	if errors.Is(err, digikey.ErrUnauthorized) {
		return "Invalid credentials. Check your Client ID and Client Secret."
	}
	if errors.Is(err, digikey.ErrRateLimitExceeded) {
		return "Rate limit exceeded. Try again later."
	}

	return err.Error()
}

func normalizeDigiKeyProduct(product digikey.Product) SupplierOffer {
	stock := product.QuantityAvailable
	minOrder := product.StandardPackage
	unitPrice := product.UnitPrice
	packageName := ""
	lifecycle := product.ProductStatus.Text

	if variation := bestDigiKeyVariation(product.ProductVariations); variation != nil {
		if strings.TrimSpace(variation.PackageType.Name) != "" {
			packageName = variation.PackageType.Name
		}
		if variation.QuantityAvailable > stock {
			stock = variation.QuantityAvailable
		}
		if variation.MinimumOrderQuantity > 0 {
			minOrder = variation.MinimumOrderQuantity
		}
		if len(variation.StandardPricing) > 0 && variation.StandardPricing[0].UnitPrice > 0 {
			unitPrice = variation.StandardPricing[0].UnitPrice
		}
	}
	if packageName == "" {
		packageName = packageFromDigiKeyParameters(product.Parameters)
	}

	raw := map[string]string{
		"category": product.Category.Name,
		"status":   lifecycle,
	}

	return SupplierOffer{
		Provider:           ProviderDigiKey,
		Manufacturer:       strings.TrimSpace(product.Manufacturer.Name),
		MPN:                strings.TrimSpace(product.ManufacturerProductNumber),
		SupplierPartNumber: strings.TrimSpace(product.DigiKeyProductNumber),
		Description:        firstNonEmpty(product.DetailedDescription, product.Description.DetailedDescription, product.Description.ProductDescription),
		Package:            packageName,
		Stock:              intPointer(stock),
		MOQ:                intPointer(minOrder),
		UnitPrice:          floatPointer(unitPrice),
		ProductURL:         strings.TrimSpace(product.ProductURL),
		DatasheetURL:       strings.TrimSpace(product.DatasheetURL),
		ImageURL:           imageURLFromDigiKeyProduct(product),
		Lifecycle:          lifecycle,
		HasSymbol:          false,
		HasFootprint:       false,
		HasDatasheet:       strings.TrimSpace(product.DatasheetURL) != "",
		Raw:                raw,
	}
}

func imageURLFromDigiKeyProduct(product digikey.Product) string {
	if url := strings.TrimSpace(product.PhotoURL); url != "" {
		return url
	}
	if url := strings.TrimSpace(product.PrimaryPhoto.URL); url != "" {
		return url
	}
	for _, link := range product.MediaLinks {
		if strings.EqualFold(strings.TrimSpace(link.MediaType), "Photo") {
			if url := strings.TrimSpace(link.URL); url != "" {
				return url
			}
		}
	}
	return ""
}

func bestDigiKeyVariation(variations []digikey.ProductVariation) *digikey.ProductVariation {
	var best *digikey.ProductVariation
	for i := range variations {
		variation := &variations[i]
		if best == nil || variation.QuantityAvailable > best.QuantityAvailable {
			best = variation
		}
	}
	return best
}

func packageFromDigiKeyParameters(parameters []digikey.Parameter) string {
	for _, parameter := range parameters {
		if strings.EqualFold(strings.TrimSpace(parameter.ParameterText), "Package / Case") || strings.EqualFold(strings.TrimSpace(parameter.ParameterText), "Supplier Device Package") {
			return strings.TrimSpace(parameter.ValueText)
		}
	}
	return ""
}
