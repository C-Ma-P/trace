package sourcing

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"componentmanager/internal/domain"
	"componentmanager/internal/domain/registry"
)

type SupplierOffer struct {
	Provider           string            `json:"provider"`
	Manufacturer       string            `json:"manufacturer"`
	MPN                string            `json:"mpn"`
	SupplierPartNumber string            `json:"supplierPartNumber"`
	Description        string            `json:"description"`
	Package            string            `json:"package"`
	Stock              *int              `json:"stock"`
	MOQ                *int              `json:"moq"`
	UnitPrice          *float64          `json:"unitPrice"`
	ProductURL         string            `json:"productUrl"`
	DatasheetURL       string            `json:"datasheetUrl"`
	Lifecycle          string            `json:"lifecycle"`
	MatchScore         int               `json:"matchScore"`
	MatchReasons       []string          `json:"matchReasons"`
	Raw                map[string]string `json:"raw,omitempty"`
}

type ProviderStatus struct {
	Provider   string `json:"provider"`
	Status     string `json:"status"`
	Error      string `json:"error"`
	OfferCount int    `json:"offerCount"`
}

type SourceResult struct {
	Offers    []SupplierOffer  `json:"offers"`
	Providers []ProviderStatus `json:"providers"`
	Currency  string           `json:"currency"`
}

type RequirementQuery struct {
	RequirementID     string
	Category          domain.Category
	RequirementName   string
	Quantity          int
	Manufacturer      string
	MPN               string
	Package           string
	TextTerms         []string
	ValueTerms        []string
	SearchTerms       []string
	SelectedComponent *domain.Component
}

func BuildRequirementQuery(requirement domain.ProjectRequirement, selected *domain.Component) RequirementQuery {
	query := RequirementQuery{
		RequirementID:     requirement.ID,
		Category:          requirement.Category,
		RequirementName:   strings.TrimSpace(requirement.Name),
		Quantity:          requirement.Quantity,
		TextTerms:         make([]string, 0, 8),
		ValueTerms:        make([]string, 0, 8),
		SearchTerms:       make([]string, 0, 8),
		SelectedComponent: selected,
	}

	constraintManufacturer := ""
	constraintMPN := ""
	constraintPackage := ""

	for _, constraint := range requirement.Constraints {
		switch constraint.Key {
		case registry.AttrManufacturer:
			if constraint.Text != nil {
				constraintManufacturer = strings.TrimSpace(*constraint.Text)
			}
		case registry.AttrMPN:
			if constraint.Text != nil {
				constraintMPN = strings.TrimSpace(*constraint.Text)
			}
		case registry.AttrPackage:
			if constraint.Text != nil {
				constraintPackage = strings.TrimSpace(*constraint.Text)
			}
		default:
			switch constraint.ValueType {
			case domain.ValueTypeText:
				if constraint.Text != nil {
					appendUniqueFold(&query.TextTerms, strings.TrimSpace(*constraint.Text))
				}
			case domain.ValueTypeNumber:
				if constraint.Number != nil {
					appendUniqueFold(&query.ValueTerms, formatConstraintValue(constraint))
				}
			}
		}
	}

	if selected != nil {
		query.Manufacturer = strings.TrimSpace(selected.Manufacturer)
		query.MPN = strings.TrimSpace(selected.MPN)
		query.Package = strings.TrimSpace(selected.Package)
		appendUniqueFold(&query.TextTerms, strings.TrimSpace(selected.Description))
	}
	if query.Manufacturer == "" {
		query.Manufacturer = constraintManufacturer
	}
	if query.MPN == "" {
		query.MPN = constraintMPN
	}
	if query.Package == "" {
		query.Package = constraintPackage
	}

	appendUniqueFold(&query.TextTerms, query.RequirementName)
	appendUniqueFold(&query.TextTerms, categoryKeyword(query.Category))
	appendUniqueFold(&query.TextTerms, query.Package)
	appendUniqueFold(&query.TextTerms, query.Manufacturer)

	if query.MPN != "" && query.Manufacturer != "" {
		appendUniqueFold(&query.SearchTerms, strings.TrimSpace(query.Manufacturer+" "+query.MPN))
	}
	appendUniqueFold(&query.SearchTerms, query.MPN)

	textSearchParts := []string{query.RequirementName, categoryKeyword(query.Category)}
	if len(query.ValueTerms) > 0 {
		textSearchParts = append(textSearchParts, query.ValueTerms[0])
	}
	if query.Package != "" {
		textSearchParts = append(textSearchParts, query.Package)
	}
	appendUniqueFold(&query.SearchTerms, strings.TrimSpace(strings.Join(filterEmpty(textSearchParts), " ")))

	for _, term := range query.ValueTerms {
		appendUniqueFold(&query.SearchTerms, strings.TrimSpace(strings.Join(filterEmpty([]string{categoryKeyword(query.Category), term, query.Package}), " ")))
	}
	for _, term := range query.TextTerms {
		appendUniqueFold(&query.SearchTerms, term)
	}

	return query
}

func RankOffers(query RequirementQuery, offers []SupplierOffer) []SupplierOffer {
	ranked := make([]SupplierOffer, len(offers))
	for i := range offers {
		ranked[i] = scoreOffer(query, offers[i])
	}

	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].MatchScore != ranked[j].MatchScore {
			return ranked[i].MatchScore > ranked[j].MatchScore
		}
		leftStock := optionalInt(ranked[i].Stock)
		rightStock := optionalInt(ranked[j].Stock)
		if leftStock != rightStock {
			return leftStock > rightStock
		}
		leftPrice := optionalFloat(ranked[i].UnitPrice)
		rightPrice := optionalFloat(ranked[j].UnitPrice)
		if leftPrice != rightPrice {
			return leftPrice < rightPrice
		}
		if ranked[i].Provider != ranked[j].Provider {
			return ranked[i].Provider < ranked[j].Provider
		}
		return ranked[i].MPN < ranked[j].MPN
	})

	return ranked
}

func scoreOffer(query RequirementQuery, offer SupplierOffer) SupplierOffer {
	score := 0
	reasons := make([]string, 0, 6)
	haystack := normalizeText(strings.Join([]string{
		offer.Manufacturer,
		offer.MPN,
		offer.SupplierPartNumber,
		offer.Description,
		offer.Package,
		rawValuesString(offer.Raw),
	}, " "))

	if query.MPN != "" {
		switch {
		case normalizePart(query.MPN) == normalizePart(offer.MPN):
			score += 120
			reasons = append(reasons, "Exact MPN match")
		case partContains(offer.MPN, query.MPN):
			score += 55
			reasons = append(reasons, "Close MPN match")
		case strings.TrimSpace(offer.MPN) != "":
			score -= 35
			reasons = append(reasons, "Different MPN")
		}
	}

	if query.Manufacturer != "" {
		switch {
		case normalizeText(query.Manufacturer) == normalizeText(offer.Manufacturer):
			score += 40
			reasons = append(reasons, "Manufacturer match")
		case containsFold(offer.Manufacturer, query.Manufacturer) || containsFold(query.Manufacturer, offer.Manufacturer):
			score += 18
			reasons = append(reasons, "Manufacturer looks related")
		case strings.TrimSpace(offer.Manufacturer) != "":
			score -= 10
		}
	}

	if query.Package != "" {
		switch {
		case normalizeText(query.Package) == normalizeText(offer.Package):
			score += 26
			reasons = append(reasons, "Package match")
		case containsFold(offer.Package, query.Package) || containsFold(offer.Description, query.Package):
			score += 14
			reasons = append(reasons, "Package hint present")
		case strings.TrimSpace(offer.Package) != "":
			score -= 8
		}
	}

	valueMatches := 0
	for _, term := range query.ValueTerms {
		if term != "" && strings.Contains(haystack, normalizeText(term)) {
			score += 12
			reasons = append(reasons, fmt.Sprintf("Matches %s", term))
			valueMatches++
		}
		if valueMatches >= 2 {
			break
		}
	}

	textMatches := 0
	for _, term := range query.TextTerms {
		normalized := normalizeText(term)
		if normalized == "" || len(normalized) < 3 {
			continue
		}
		if strings.Contains(haystack, normalized) {
			score += 8
			reasons = append(reasons, fmt.Sprintf("Matches %s", term))
			textMatches++
		}
		if textMatches >= 2 {
			break
		}
	}

	categoryTerm := normalizeText(categoryKeyword(query.Category))
	if categoryTerm != "" && strings.Contains(haystack, categoryTerm) {
		score += 10
		reasons = append(reasons, "Category aligned")
	}

	if score < 0 {
		score = 0
	}

	offer.MatchScore = score
	offer.MatchReasons = reasons
	return offer
}

func formatConstraintValue(constraint domain.RequirementConstraint) string {
	if constraint.Number == nil {
		return ""
	}
	value := *constraint.Number
	switch constraint.Key {
	case registry.AttrResistanceOhms, registry.AttrDCROhms:
		return engineeringToken(value, []engineeringPrefix{{1e6, "M"}, {1e3, "k"}, {1, ""}, {1e-3, "m"}}, "")
	case registry.AttrCapacitanceF:
		return engineeringToken(value, []engineeringPrefix{{1, "F"}, {1e-3, "mF"}, {1e-6, "uF"}, {1e-9, "nF"}, {1e-12, "pF"}}, "F")
	case registry.AttrInductanceH:
		return engineeringToken(value, []engineeringPrefix{{1, "H"}, {1e-3, "mH"}, {1e-6, "uH"}, {1e-9, "nH"}}, "H")
	case registry.AttrVoltageV:
		return trimFloat(value) + "V"
	case registry.AttrPowerW:
		return trimFloat(value) + "W"
	case registry.AttrCurrentA:
		return trimFloat(value) + "A"
	case registry.AttrTolerancePercent:
		return trimFloat(value) + "%"
	case registry.AttrTempCoPPMC:
		return trimFloat(value) + "ppm"
	default:
		if constraint.Unit != "" {
			return trimFloat(value) + constraint.Unit
		}
		return trimFloat(value)
	}
}

type engineeringPrefix struct {
	divisor float64
	suffix  string
}

func engineeringToken(value float64, prefixes []engineeringPrefix, fallback string) string {
	abs := math.Abs(value)
	for _, prefix := range prefixes {
		if abs >= prefix.divisor && prefix.divisor > 0 {
			return trimFloat(value/prefix.divisor) + prefix.suffix
		}
	}
	if len(prefixes) > 0 {
		last := prefixes[len(prefixes)-1]
		return trimFloat(value/last.divisor) + last.suffix
	}
	return trimFloat(value) + fallback
}

func trimFloat(value float64) string {
	text := strconv.FormatFloat(value, 'f', 6, 64)
	text = strings.TrimRight(text, "0")
	text = strings.TrimRight(text, ".")
	if text == "" || text == "-" {
		return "0"
	}
	return text
}

func optionalInt(value *int) int {
	if value == nil {
		return -1
	}
	return *value
}

func optionalFloat(value *float64) float64 {
	if value == nil {
		return math.MaxFloat64
	}
	return *value
}

func rawValuesString(raw map[string]string) string {
	if len(raw) == 0 {
		return ""
	}
	values := make([]string, 0, len(raw))
	for _, value := range raw {
		values = append(values, value)
	}
	sort.Strings(values)
	return strings.Join(values, " ")
}

func categoryKeyword(category domain.Category) string {
	switch category {
	case domain.CategoryIntegratedCircuit:
		return "integrated circuit"
	default:
		return strings.ReplaceAll(string(category), "_", " ")
	}
}

func normalizePart(value string) string {
	replacer := strings.NewReplacer("-", "", "_", "", " ", "", "/", "")
	return replacer.Replace(strings.ToUpper(strings.TrimSpace(value)))
}

func normalizeText(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func partContains(actual, wanted string) bool {
	normalActual := normalizePart(actual)
	normalWanted := normalizePart(wanted)
	if normalActual == "" || normalWanted == "" {
		return false
	}
	return strings.Contains(normalActual, normalWanted) || strings.Contains(normalWanted, normalActual)
}

func containsFold(actual, wanted string) bool {
	normalActual := normalizeText(actual)
	normalWanted := normalizeText(wanted)
	if normalActual == "" || normalWanted == "" {
		return false
	}
	return strings.Contains(normalActual, normalWanted)
}

func appendUniqueFold(values *[]string, candidate string) {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return
	}
	for _, value := range *values {
		if strings.EqualFold(value, candidate) {
			return
		}
	}
	*values = append(*values, candidate)
}

func filterEmpty(values []string) []string {
	out := values[:0]
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}
