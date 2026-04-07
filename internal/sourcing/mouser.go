package sourcing

import (
	"context"
	"errors"
	"fmt"
	"strings"

	mouser "github.com/PatrickWalther/go-mouser"
)

type MouserProvider struct {
	client *mouser.Client
}

func NewMouserProvider(config MouserConfig) *MouserProvider {
	if !config.Enabled {
		return &MouserProvider{}
	}
	if strings.TrimSpace(config.APIKey) == "" {
		return &MouserProvider{}
	}
	client, err := mouser.NewClient(config.APIKey)
	if err != nil {
		return &MouserProvider{}
	}
	return &MouserProvider{client: client}
}

func (p *MouserProvider) Name() string {
	return "Mouser"
}

func (p *MouserProvider) Enabled() bool {
	return p.client != nil
}

func (p *MouserProvider) Search(ctx context.Context, query RequirementQuery) ([]SupplierOffer, error) {
	var result *mouser.SearchResult
	var err error

	if query.MPN != "" && query.Manufacturer != "" {
		result, err = p.client.Search.PartNumberAndManufacturerSearch(ctx, mouser.PartNumberAndManufacturerSearchOptions{
			PartNumber:       query.MPN,
			ManufacturerName: query.Manufacturer,
			PartSearchOption: mouser.PartSearchOptionExact,
		})
	} else if query.MPN != "" {
		result, err = p.client.Search.PartNumberSearch(ctx, mouser.PartNumberSearchOptions{
			PartNumber:       query.MPN,
			PartSearchOption: mouser.PartSearchOptionExact,
		})
	} else if query.Manufacturer != "" && len(query.SearchTerms) > 0 {
		result, err = p.client.Search.KeywordAndManufacturerSearch(ctx, mouser.KeywordAndManufacturerSearchOptions{
			Keyword:          query.SearchTerms[0],
			ManufacturerName: query.Manufacturer,
			Records:          10,
			PageNumber:       1,
			SearchOption:     mouser.SearchOptionInStock,
		})
	} else if len(query.SearchTerms) > 0 {
		result, err = p.client.Search.KeywordSearch(ctx, mouser.SearchOptions{
			Keyword:      query.SearchTerms[0],
			Records:      10,
			SearchOption: mouser.SearchOptionInStock,
		})
	}
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}

	offers := make([]SupplierOffer, 0, len(result.Parts))
	for _, part := range result.Parts {
		offers = append(offers, normalizeMouserPart(part))
	}
	return offers, nil
}

func (p *MouserProvider) LookupByPartNumber(ctx context.Context, partNumber string) (SupplierOffer, error) {
	if !p.Enabled() {
		return SupplierOffer{}, fmt.Errorf("Mouser provider not configured")
	}
	result, err := p.client.Search.PartNumberSearch(ctx, mouser.PartNumberSearchOptions{
		PartNumber:       partNumber,
		PartSearchOption: mouser.PartSearchOptionExact,
	})
	if err != nil {
		return SupplierOffer{}, err
	}
	if result == nil || len(result.Parts) == 0 {
		return SupplierOffer{}, fmt.Errorf("part %q not found on Mouser", partNumber)
	}
	return normalizeMouserPart(result.Parts[0]), nil
}

func (p *MouserProvider) FriendlyError(err error) string {
	var apiErrs mouser.APIErrors
	if errors.As(err, &apiErrs) && len(apiErrs) > 0 {
		msg := apiErrs[0].Message
		if len(apiErrs) > 1 {
			return msg + " (and more errors)"
		}
		return msg
	}

	var mouserErr *mouser.MouserError
	if errors.As(err, &mouserErr) {
		if len(mouserErr.Errors) > 0 {
			msg := mouserErr.Errors[0].Message
			if len(mouserErr.Errors) > 1 {
				return msg + " (and more errors)"
			}
			return msg
		}
		if errors.Is(err, mouser.ErrUnauthorized) {
			return "Invalid API key. Check your Mouser API key."
		}
		if errors.Is(err, mouser.ErrRateLimitExceeded) || errors.Is(err, mouser.ErrDailyLimitExceeded) {
			return "Rate limit exceeded. Try again later."
		}
		if mouserErr.Message != "" {
			return mouserErr.Message
		}
	}

	if errors.Is(err, mouser.ErrUnauthorized) {
		return "Invalid API key. Check your Mouser API key."
	}
	if errors.Is(err, mouser.ErrRateLimitExceeded) || errors.Is(err, mouser.ErrDailyLimitExceeded) {
		return "Rate limit exceeded. Try again later."
	}

	return err.Error()
}

func normalizeMouserPart(part mouser.Part) SupplierOffer {
	packageName := packageFromMouserAttributes(part.ProductAttributes)
	price := lowestMouserPrice(part.PriceBreaks)
	raw := map[string]string{
		"availability": part.Availability,
		"leadTime":     part.LeadTime,
		"category":     part.Category,
	}

	return SupplierOffer{
		Provider:           "Mouser",
		Manufacturer:       strings.TrimSpace(firstNonEmpty(part.ActualMfrName, part.Manufacturer)),
		MPN:                strings.TrimSpace(part.ManufacturerPartNumber),
		SupplierPartNumber: strings.TrimSpace(part.MouserPartNumber),
		Description:        strings.TrimSpace(part.Description),
		Package:            packageName,
		Stock:              intPointer(parseLooseInt(firstNonEmpty(part.AvailabilityInStock, part.FactoryStock))),
		MOQ:                intPointer(parseLooseInt(part.Min)),
		UnitPrice:          floatPointer(price),
		ProductURL:         strings.TrimSpace(part.ProductDetailUrl),
		DatasheetURL:       strings.TrimSpace(part.DataSheetUrl),
		ImageURL:           strings.TrimSpace(part.ImagePath),
		Lifecycle:          strings.TrimSpace(part.LifecycleStatus),
		Raw:                raw,
	}
}

func packageFromMouserAttributes(attributes []mouser.ProductAttribute) string {
	for _, attribute := range attributes {
		name := strings.TrimSpace(attribute.AttributeName)
		if strings.EqualFold(name, "Package / Case") || strings.EqualFold(name, "Packaging") || strings.EqualFold(name, "Mounting Style") {
			return strings.TrimSpace(attribute.AttributeValue)
		}
	}
	return ""
}

func lowestMouserPrice(breaks []mouser.PriceBreak) float64 {
	best := 0.0
	for _, priceBreak := range breaks {
		price := parsePrice(priceBreak.Price)
		if price <= 0 {
			continue
		}
		if best == 0 || price < best {
			best = price
		}
	}
	return best
}
