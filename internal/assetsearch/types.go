package assetsearch

// SearchRequest describes a part-centric asset search across providers.
type SearchRequest struct {
	ComponentID  string // required: which component we are searching for
	MPN          string // part number to search by
	Manufacturer string // optional manufacturer hint
	Query        string // optional freeform override; if empty, MPN is used
}

// SearchResponse contains grouped results from one or more providers.
// Partial failure is expected: individual providers may error while others succeed.
type SearchResponse struct {
	ProviderResults []ProviderResult `json:"providerResults"`
}

// ProviderResult holds the search outcome from a single provider.
// If Error is non-empty, the provider failed and Candidates should be ignored.
type ProviderResult struct {
	ProviderID    string            `json:"providerId"`
	ProviderLabel string            `json:"providerLabel"`
	Candidates    []SearchCandidate `json:"candidates"`
	Error         string            `json:"error"`
}

// SearchCandidate is a normalized asset candidate returned by a provider.
// Provider-specific details stay in Metadata; the rest is app-facing.
type SearchCandidate struct {
	ExternalID   string `json:"externalId"`
	Title        string `json:"title"`
	Manufacturer string `json:"manufacturer"`
	MPN          string `json:"mpn"`
	Package      string `json:"package"`
	Description  string `json:"description"`

	HasSymbol    bool `json:"hasSymbol"`
	HasFootprint bool `json:"hasFootprint"`
	Has3DModel   bool `json:"has3dModel"`
	HasDatasheet bool `json:"hasDatasheet"`

	PreviewURL *string           `json:"previewUrl"`
	SourceURL  *string           `json:"sourceUrl"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// ImportRequest identifies a single provider candidate to import for a component.
type ImportRequest struct {
	ComponentID string
	Provider    string
	ExternalID  string
}

// ImportResponse describes the assets that were imported.
type ImportResponse struct {
	ImportedAssets []ImportedAsset `json:"importedAssets"`
	Warnings       []string        `json:"warnings,omitempty"`
}

// ImportedAsset is a normalized description of one asset created during import.
type ImportedAsset struct {
	AssetType string `json:"assetType"`
	Label     string `json:"label"`
	URLOrPath string `json:"urlOrPath"`
}
