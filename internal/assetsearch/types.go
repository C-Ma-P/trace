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
// Providers should populate Artifacts with paths to downloaded files/archives/
// directories. The orchestrating service feeds those artifacts through the
// ingestion pipeline so that assets end up in Trace-managed local storage.
// ImportedAssets is kept for backward-compatibility with any provider that
// cannot download artifacts locally (fully-stubbed providers fall here).
type ImportResponse struct {
	ImportedAssets []ImportedAsset      `json:"importedAssets"`
	Artifacts      []DownloadedArtifact `json:"artifacts"`
	Warnings       []string             `json:"warnings,omitempty"`
}

// ImportedAsset is a normalized description of one asset created during import.
// Deprecated: new providers should return DownloadedArtifact entries instead,
// allowing the ingestion pipeline to handle storage normalisation.
type ImportedAsset struct {
	AssetType string `json:"assetType"`
	Label     string `json:"label"`
	URLOrPath string `json:"urlOrPath"`
}

// DownloadedArtifact represents a file, archive, or directory that a provider
// downloaded to a local temporary path. The ingestion service will classify,
// store, and persist these as managed component assets.
type DownloadedArtifact struct {
	FilePath    string `json:"filePath"`    // local path to downloaded artifact
	Description string `json:"description"` // human-readable description
}
