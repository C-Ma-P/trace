package app

type ComponentFilterInput struct {
	Category     string `json:"category"`
	Manufacturer string `json:"manufacturer"`
	MPN          string `json:"mpn"`
	Package      string `json:"package"`
	Text         string `json:"text"`
}

type CreateComponentInput struct {
	Category     string `json:"category"`
	MPN          string `json:"mpn"`
	Manufacturer string `json:"manufacturer"`
	Package      string `json:"package"`
	Description  string `json:"description"`
}

type UpdateMetadataInput struct {
	ID           string `json:"id"`
	MPN          string `json:"mpn"`
	Manufacturer string `json:"manufacturer"`
	Package      string `json:"package"`
	Description  string `json:"description"`
}

type AttributeValueInput struct {
	Key       string   `json:"key"`
	ValueType string   `json:"valueType"`
	Text      *string  `json:"text"`
	Number    *float64 `json:"number"`
	Bool      *bool    `json:"bool"`
	Unit      string   `json:"unit"`
}

type UpdateInventoryInput struct {
	ID           string `json:"id"`
	Quantity     *int   `json:"quantity"`
	QuantityMode string `json:"quantityMode"`
	Location     string `json:"location"`
}

type CreateProjectInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateProjectInput struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type RequirementConstraintInput struct {
	Key       string   `json:"key"`
	ValueType string   `json:"valueType"`
	Operator  string   `json:"operator"`
	Text      *string  `json:"text"`
	Number    *float64 `json:"number"`
	Bool      *bool    `json:"bool"`
	Unit      string   `json:"unit"`
}

type RequirementInput struct {
	ID                  string                       `json:"id"`
	Name                string                       `json:"name"`
	Category            string                       `json:"category"`
	Quantity            int                          `json:"quantity"`
	SelectedComponentID *string                      `json:"selectedComponentId"`
	Resolution          *RequirementResolutionInput  `json:"resolution"`
	Constraints         []RequirementConstraintInput `json:"constraints"`
}

type RequirementResolutionInput struct {
	Kind        string  `json:"kind"`
	ComponentID *string `json:"componentId"`
}

type CategoryInfo struct {
	Value       string `json:"value"`
	DisplayName string `json:"displayName"`
}

type OperatorInfo struct {
	Value       string `json:"value"`
	DisplayName string `json:"displayName"`
}

type AttributeDefinitionInfo struct {
	Key         string `json:"key"`
	DisplayName string `json:"displayName"`
	ValueType   string `json:"valueType"`
	Unit        string `json:"unit"`
}

type ComponentResponse struct {
	ID           string                   `json:"id"`
	Category     string                   `json:"category"`
	MPN          string                   `json:"mpn"`
	Manufacturer string                   `json:"manufacturer"`
	Package      string                   `json:"package"`
	Description  string                   `json:"description"`
	Quantity     *int                     `json:"quantity"`
	QuantityMode string                   `json:"quantityMode"`
	Location     string                   `json:"location"`
	Attributes   []AttributeValueResponse `json:"attributes"`
	CreatedAt    string                   `json:"createdAt"`
	UpdatedAt    string                   `json:"updatedAt"`
}

type AttributeValueResponse struct {
	Key       string   `json:"key"`
	ValueType string   `json:"valueType"`
	Text      *string  `json:"text"`
	Number    *float64 `json:"number"`
	Bool      *bool    `json:"bool"`
	Unit      string   `json:"unit"`
}

type ProjectResponse struct {
	ID               string                `json:"id"`
	Name             string                `json:"name"`
	Description      string                `json:"description"`
	ImportSourceType *string               `json:"importSourceType"`
	ImportSourcePath *string               `json:"importSourcePath"`
	ImportedAt       *string               `json:"importedAt"`
	Requirements     []RequirementResponse `json:"requirements"`
	CreatedAt        string                `json:"createdAt"`
	UpdatedAt        string                `json:"updatedAt"`
}

type RequirementResponse struct {
	ID                  string                         `json:"id"`
	ProjectID           string                         `json:"projectId"`
	Name                string                         `json:"name"`
	Category            string                         `json:"category"`
	Quantity            int                            `json:"quantity"`
	SelectedComponentID *string                        `json:"selectedComponentId"`
	Resolution          *RequirementResolutionResponse `json:"resolution"`
	Constraints         []ConstraintResponse           `json:"constraints"`
}

type RequirementResolutionResponse struct {
	Kind        string  `json:"kind"`
	ComponentID *string `json:"componentId"`
}

type ConstraintResponse struct {
	Key       string   `json:"key"`
	ValueType string   `json:"valueType"`
	Operator  string   `json:"operator"`
	Text      *string  `json:"text"`
	Number    *float64 `json:"number"`
	Bool      *bool    `json:"bool"`
	Unit      string   `json:"unit"`
}

type ProjectPlanResponse struct {
	Project      ProjectResponse           `json:"project"`
	Requirements []RequirementPlanResponse `json:"requirements"`
}

type SourceRequirementResponse struct {
	Offers    []SupplierOfferResponse          `json:"offers"`
	Providers []SupplierProviderStatusResponse `json:"providers"`
}

type SupplierOfferResponse struct {
	Provider           string            `json:"provider"`
	Manufacturer       string            `json:"manufacturer"`
	MPN                string            `json:"mpn"`
	SupplierPartNumber string            `json:"supplierPartNumber"`
	Description        string            `json:"description"`
	Package            string            `json:"package"`
	Stock              *int              `json:"stock"`
	MOQ                *int              `json:"moq"`
	UnitPrice          *float64          `json:"unitPrice"`
	Currency           string            `json:"currency"`
	ProductURL         string            `json:"productUrl"`
	DatasheetURL       string            `json:"datasheetUrl"`
	Lifecycle          string            `json:"lifecycle"`
	MatchScore         int               `json:"matchScore"`
	MatchReasons       []string          `json:"matchReasons"`
	Raw                map[string]string `json:"raw,omitempty"`
}

type SupplierProviderStatusResponse struct {
	Provider   string `json:"provider"`
	Status     string `json:"status"`
	Error      string `json:"error"`
	OfferCount int    `json:"offerCount"`
}

type RequirementPlanResponse struct {
	Requirement            RequirementResponse              `json:"requirement"`
	MatchingOnHandQuantity int                              `json:"matchingOnHandQuantity"`
	ShortfallQuantity      int                              `json:"shortfallQuantity"`
	SelectedPart           *RequirementSelectedPartResponse `json:"selectedPart"`
	Matches                []ComponentMatchResponse         `json:"matches"`
}

type ComponentMatchResponse struct {
	Component      ComponentResponse `json:"component"`
	OnHandQuantity int               `json:"onHandQuantity"`
	Score          int               `json:"score"`
}

type RequirementSelectedPartResponse struct {
	Resolution        RequirementResolutionResponse `json:"resolution"`
	DisplayName       string                        `json:"displayName"`
	Component         *ComponentResponse            `json:"component"`
	OnHandQuantity    int                           `json:"onHandQuantity"`
	ShortfallQuantity int                           `json:"shortfallQuantity"`
}

type CreateComponentAssetInput struct {
	ComponentID  string  `json:"componentId"`
	AssetType    string  `json:"assetType"`
	Source       string  `json:"source"`
	Status       string  `json:"status"`
	Label        string  `json:"label"`
	URLOrPath    string  `json:"urlOrPath"`
	PreviewURL   *string `json:"previewUrl"`
	MetadataJSON *string `json:"metadataJson"`
}

type UpdateComponentAssetInput struct {
	ID           string  `json:"id"`
	Source       string  `json:"source"`
	Status       string  `json:"status"`
	Label        string  `json:"label"`
	URLOrPath    string  `json:"urlOrPath"`
	PreviewURL   *string `json:"previewUrl"`
	MetadataJSON *string `json:"metadataJson"`
}

type ComponentAssetResponse struct {
	ID           string  `json:"id"`
	ComponentID  string  `json:"componentId"`
	AssetType    string  `json:"assetType"`
	Source       string  `json:"source"`
	Status       string  `json:"status"`
	Label        string  `json:"label"`
	URLOrPath    string  `json:"urlOrPath"`
	PreviewURL   *string `json:"previewUrl"`
	MetadataJSON *string `json:"metadataJson"`
	CreatedAt    string  `json:"createdAt"`
	UpdatedAt    string  `json:"updatedAt"`
}

type ComponentDetailResponse struct {
	Component              ComponentResponse        `json:"component"`
	SelectedSymbolAsset    *ComponentAssetResponse  `json:"selectedSymbolAsset"`
	SelectedFootprintAsset *ComponentAssetResponse  `json:"selectedFootprintAsset"`
	Selected3DModelAsset   *ComponentAssetResponse  `json:"selected3dModelAsset"`
	SelectedDatasheetAsset *ComponentAssetResponse  `json:"selectedDatasheetAsset"`
	Assets                 []ComponentAssetResponse `json:"assets"`
}

type StartupStatusResponse struct {
	Ready bool   `json:"ready"`
	Error string `json:"error"`
}

type RecentProjectResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Subtitle    string `json:"subtitle"`
	OpenedAtUTC string `json:"openedAtUtc"`
	Pinned      bool   `json:"pinned"`
}

type SaveKiCadPreferencesInput struct {
	ProjectRoots []string `json:"projectRoots"`
}

type KiCadPreferencesResponse struct {
	ProjectRoots []string `json:"projectRoots"`
}

type SaveSupplierPreferencesInput struct {
	DigiKey SupplierDigiKeyInput `json:"digikey"`
	Mouser  SupplierMouserInput  `json:"mouser"`
	LCSC    SupplierLCSCInput    `json:"lcsc"`
}

type SupplierDigiKeyInput struct {
	Enabled             bool    `json:"enabled"`
	ClientID            string  `json:"clientId"`
	CustomerID          string  `json:"customerId"`
	Site                string  `json:"site"`
	Language            string  `json:"language"`
	Currency            string  `json:"currency"`
	ReplaceClientSecret *string `json:"replaceClientSecret"`
}

type SupplierMouserInput struct {
	Enabled       bool    `json:"enabled"`
	ReplaceAPIKey *string `json:"replaceApiKey"`
}

type SupplierLCSCInput struct {
	Enabled  bool   `json:"enabled"`
	Currency string `json:"currency"`
}

type SupplierPreferencesResponse struct {
	SecureStorageAvailable bool                    `json:"secureStorageAvailable"`
	SecureStorageMessage   string                  `json:"secureStorageMessage"`
	DigiKey                SupplierDigiKeyResponse `json:"digikey"`
	Mouser                 SupplierMouserResponse  `json:"mouser"`
	LCSC                   SupplierLCSCResponse    `json:"lcsc"`
}

type SupplierDigiKeyResponse struct {
	Enabled            bool                           `json:"enabled"`
	ClientID           string                         `json:"clientId"`
	CustomerID         string                         `json:"customerId"`
	Site               string                         `json:"site"`
	Language           string                         `json:"language"`
	Currency           string                         `json:"currency"`
	ClientSecretStored bool                           `json:"clientSecretStored"`
	Status             SupplierProviderConfigResponse `json:"status"`
}

type SupplierMouserResponse struct {
	Enabled      bool                           `json:"enabled"`
	APIKeyStored bool                           `json:"apiKeyStored"`
	Status       SupplierProviderConfigResponse `json:"status"`
}

type SupplierLCSCResponse struct {
	Enabled  bool                           `json:"enabled"`
	Currency string                         `json:"currency"`
	Status   SupplierProviderConfigResponse `json:"status"`
}

type SupplierProviderConfigResponse struct {
	Provider     string `json:"provider"`
	Enabled      bool   `json:"enabled"`
	Complete     bool   `json:"complete"`
	State        string `json:"state"`
	StorageMode  string `json:"storageMode"`
	Source       string `json:"source"`
	Message      string `json:"message"`
	HasSecret    bool   `json:"hasSecret"`
	SecretStored bool   `json:"secretStored"`
}

type AssetSearchResponse struct {
	ProviderResults []AssetSearchProviderResult `json:"providerResults"`
}

type AssetSearchProviderResult struct {
	Provider   string                 `json:"provider"`
	Candidates []AssetSearchCandidate `json:"candidates"`
	Error      string                 `json:"error"`
}

type AssetSearchCandidate struct {
	ExternalID   string            `json:"externalId"`
	Title        string            `json:"title"`
	Manufacturer string            `json:"manufacturer"`
	MPN          string            `json:"mpn"`
	Package      string            `json:"package"`
	Description  string            `json:"description"`
	HasSymbol    bool              `json:"hasSymbol"`
	HasFootprint bool              `json:"hasFootprint"`
	Has3DModel   bool              `json:"has3dModel"`
	HasDatasheet bool              `json:"hasDatasheet"`
	PreviewURL   *string           `json:"previewUrl"`
	SourceURL    *string           `json:"sourceUrl"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type AssetImportResponse struct {
	ImportedAssets []AssetImportedAsset `json:"importedAssets"`
	Warnings       []string             `json:"warnings"`
}

type AssetImportedAsset struct {
	AssetType string `json:"assetType"`
	Label     string `json:"label"`
	URLOrPath string `json:"urlOrPath"`
}

type KiCadProjectCandidateResponse struct {
	Name        string `json:"name"`
	ProjectPath string `json:"projectPath"`
	ProjectDir  string `json:"projectDir"`
}

type KiCadImportDraftRequirement struct {
	ID                  string                       `json:"id"`
	ProjectID           string                       `json:"projectId"`
	Name                string                       `json:"name"`
	Category            string                       `json:"category"`
	Quantity            int                          `json:"quantity"`
	SelectedComponentID *string                      `json:"selectedComponentId"`
	Constraints         []RequirementConstraintInput `json:"constraints"`
}

type KiCadImportPreviewRow struct {
	RowID           string                      `json:"rowId"`
	Included        bool                        `json:"included"`
	SourceRefs      string                      `json:"sourceRefs"`
	SourceQuantity  int                         `json:"sourceQuantity"`
	RawValue        string                      `json:"rawValue"`
	RawFootprint    string                      `json:"rawFootprint"`
	RawDescription  string                      `json:"rawDescription"`
	Manufacturer    string                      `json:"manufacturer"`
	MPN             string                      `json:"mpn"`
	OtherFields     map[string]string           `json:"otherFields"`
	Requirement     KiCadImportDraftRequirement `json:"requirement"`
	HasWarning      bool                        `json:"hasWarning"`
	WarningMessages []string                    `json:"warningMessages"`
}

type KiCadImportPreviewSummary struct {
	TotalRows    int `json:"totalRows"`
	IncludedRows int `json:"includedRows"`
	WarningRows  int `json:"warningRows"`
}

type KiCadImportPreviewResponse struct {
	SelectedProject KiCadProjectCandidateResponse `json:"selectedProject"`
	Rows            []KiCadImportPreviewRow       `json:"rows"`
	Summary         KiCadImportPreviewSummary     `json:"summary"`
}

type KiCadImportCommitInput struct {
	TargetMode            string                  `json:"targetMode"`
	NewProjectName        string                  `json:"newProjectName"`
	NewProjectDescription string                  `json:"newProjectDescription"`
	ExistingProjectID     string                  `json:"existingProjectId"`
	SourceProjectPath     string                  `json:"sourceProjectPath"`
	Rows                  []KiCadImportPreviewRow `json:"rows"`
}
