import { callByName } from './wails';

const serviceName = 'main.AppService';

function call<T>(method: string, ...args: any[]): Promise<T> {
  return callByName(`${serviceName}.${method}`, ...args);
}

export interface CategoryInfo {
  value: string;
  displayName: string;
}

export interface OperatorInfo {
  value: string;
  displayName: string;
}

export interface AttributeDefinitionInfo {
  key: string;
  displayName: string;
  valueType: string;
  unit: string;
}

export interface ComponentFilter {
  category: string;
  manufacturer: string;
  mpn: string;
  package: string;
  text: string;
}

export interface Component {
  id: string;
  category: string;
  mpn: string;
  manufacturer: string;
  package: string;
  description: string;
  quantity: number | null;
  quantityMode: 'exact' | 'approximate' | 'unknown';
  location: string;
  attributes: AttributeValue[];
  createdAt: string;
  updatedAt: string;
}

export interface AttributeValue {
  key: string;
  valueType: string;
  text: string | null;
  number: number | null;
  bool: boolean | null;
  unit: string;
}

export interface Project {
  id: string;
  name: string;
  description: string;
  importSourceType: string | null;
  importSourcePath: string | null;
  importedAt: string | null;
  requirements: Requirement[];
  createdAt: string;
  updatedAt: string;
}

export interface Requirement {
  id: string;
  projectId: string;
  name: string;
  category: string;
  quantity: number;
  selectedComponentId: string | null;
  resolution: RequirementResolution | null;
  constraints: Constraint[];
}

export interface RequirementResolution {
  kind: 'internal_component' | 'supplier_part';
  componentId: string | null;
}

export interface Constraint {
  key: string;
  valueType: string;
  operator: string;
  text: string | null;
  number: number | null;
  bool: boolean | null;
  unit: string;
}

export interface ProjectPlan {
  project: Project;
  requirements: RequirementPlan[];
}

export interface SourceRequirementResult {
  offers: SupplierOffer[];
  providers: SupplierProviderStatus[];
}

export interface RequirementPlan {
  requirement: Requirement;
  matchingOnHandQuantity: number;
  shortfallQuantity: number;
  selectedPart: RequirementSelectedPart | null;
  matches: ComponentMatch[];
}

export interface RequirementSelectedPart {
  resolution: RequirementResolution;
  displayName: string;
  component: Component | null;
  onHandQuantity: number;
  shortfallQuantity: number;
}

export interface SupplierOffer {
  provider: string;
  manufacturer: string;
  mpn: string;
  supplierPartNumber: string;
  description: string;
  package: string;
  stock: number | null;
  moq: number | null;
  unitPrice: number | null;
  currency: string;
  productUrl: string;
  datasheetUrl: string;
  lifecycle: string;
  matchScore: number;
  matchReasons: string[];
  raw: Record<string, string> | null;
}

export interface SupplierProviderStatus {
  provider: string;
  status: 'success' | 'disabled' | 'error';
  error: string;
  offerCount: number;
}

export interface ComponentMatch {
  component: Component;
  onHandQuantity: number;
  score: number;
}

export interface ComponentAsset {
  id: string;
  componentId: string;
  assetType: string;
  source: string;
  status: string;
  label: string;
  urlOrPath: string;
  previewUrl: string | null;
  metadataJson: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface AssetSearchResponse {
  providerResults: AssetSearchProviderResult[];
}

export interface AssetSearchProviderResult {
  provider: string;
  candidates: AssetSearchCandidate[];
  error: string;
}

export interface AssetSearchCandidate {
  externalId: string;
  title: string;
  manufacturer: string;
  mpn: string;
  package: string;
  description: string;
  hasSymbol: boolean;
  hasFootprint: boolean;
  has3dModel: boolean;
  hasDatasheet: boolean;
  previewUrl: string | null;
  sourceUrl: string | null;
  metadata: Record<string, string> | null;
}

export interface AssetImportResponse {
  importedAssets: AssetImportedAsset[];
  warnings: string[];
}

export interface AssetImportedAsset {
  assetType: string;
  label: string;
  urlOrPath: string;
}

export interface KiCadProjectCandidate {
  name: string;
  projectPath: string;
  projectDir: string;
}

export interface KiCadImportPreviewRow {
  rowId: string;
  included: boolean;
  sourceRefs: string;
  sourceQuantity: number;
  rawValue: string;
  rawFootprint: string;
  rawDescription: string;
  manufacturer: string;
  mpn: string;
  otherFields: Record<string, string>;
  requirement: Requirement;
  hasWarning: boolean;
  warningMessages: string[];
}

export interface KiCadImportPreviewSummary {
  totalRows: number;
  includedRows: number;
  warningRows: number;
}

export interface KiCadImportPreview {
  selectedProject: KiCadProjectCandidate;
  rows: KiCadImportPreviewRow[];
  summary: KiCadImportPreviewSummary;
}

export interface ComponentDetail {
  component: Component;
  selectedSymbolAsset: ComponentAsset | null;
  selectedFootprintAsset: ComponentAsset | null;
  selected3dModelAsset: ComponentAsset | null;
  selectedDatasheetAsset: ComponentAsset | null;
  assets: ComponentAsset[];
}

export interface StartupStatus {
  ready: boolean;
  error: string;
}

export interface RecentProject {
  id: string;
  name: string;
  subtitle: string;
  openedAtUtc: string;

	pinned: boolean;
}

export interface SupplierProviderConfig {
  provider: string;
  enabled: boolean;
  complete: boolean;
  state: 'configured' | 'incomplete' | 'disabled';
  storageMode: 'keychain' | 'environment' | 'unavailable' | 'missing' | 'none';
  source: 'preferences' | 'environment' | 'mixed' | 'missing' | 'unavailable';
  message: string;
  hasSecret: boolean;
  secretStored: boolean;
}

export interface DigiKeySupplierPreferences {
  enabled: boolean;
  clientId: string;
  customerId: string;
  site: string;
  language: string;
  currency: string;
  clientSecretStored: boolean;
  status: SupplierProviderConfig;
}

export interface MouserSupplierPreferences {
  enabled: boolean;
  apiKeyStored: boolean;
  status: SupplierProviderConfig;
}

export interface LCSCSupplierPreferences {
  enabled: boolean;
  currency: string;
  status: SupplierProviderConfig;
}

export interface SupplierPreferences {
  secureStorageAvailable: boolean;
  secureStorageMessage: string;
  digikey: DigiKeySupplierPreferences;
  mouser: MouserSupplierPreferences;
  lcsc: LCSCSupplierPreferences;
}

export interface KiCadPreferences {
  projectRoots: string[];
}

export function getStartupStatus(): Promise<StartupStatus> {
  return call('GetStartupStatus');
}

export function listRecentProjects(): Promise<RecentProject[]> {
  return call('ListRecentProjects');
}

export function setRecentProjectPinned(projectId: string, pinned: boolean): Promise<void> {
  return call('SetRecentProjectPinned', projectId, pinned);
}

export function reorderRecentProjects(projectIds: string[]): Promise<void> {
  return call('ReorderRecentProjects', projectIds);
}

export function getSupplierPreferences(): Promise<SupplierPreferences> {
  return call('GetSupplierPreferences');
}

export function getKiCadPreferences(): Promise<KiCadPreferences> {
  return call('GetKiCadPreferences');
}

export function saveKiCadPreferences(input: {
  projectRoots: string[];
}): Promise<KiCadPreferences> {
  return call('SaveKiCadPreferences', input);
}

export function saveSupplierPreferences(input: {
  digikey: {
    enabled: boolean;
    clientId: string;
    customerId: string;
    site: string;
    language: string;
    currency: string;
    replaceClientSecret?: string | null;
  };
  mouser: {
    enabled: boolean;
    replaceApiKey?: string | null;
  };
  lcsc: {
    enabled: boolean;
    currency: string;
  };
}): Promise<SupplierPreferences> {
  return call('SaveSupplierPreferences', input);
}

export function clearSupplierSecret(provider: string, secret: string): Promise<SupplierPreferences> {
  return call('ClearSupplierSecret', provider, secret);
}

export function getCategories(): Promise<CategoryInfo[]> {
  return call('GetCategories');
}

export function getCategoryDefinitions(category: string): Promise<AttributeDefinitionInfo[]> {
  return call('GetCategoryDefinitions', category);
}

export function getRequirementDefinitions(category: string): Promise<AttributeDefinitionInfo[]> {
  return call('GetRequirementDefinitions', category);
}

export function getOperatorsForValueType(valueType: string): Promise<OperatorInfo[]> {
  return call('GetOperatorsForValueType', valueType);
}

export function deleteComponent(id: string): Promise<void> {
  return call('DeleteComponent', id);
}

export function listComponents(filter: Partial<ComponentFilter>): Promise<Component[]> {
  const full: ComponentFilter = {
    category: filter.category ?? '',
    manufacturer: filter.manufacturer ?? '',
    mpn: filter.mpn ?? '',
    package: filter.package ?? '',
    text: filter.text ?? '',
  };
  return call('ListComponents', full);
}

export function getComponent(id: string): Promise<Component> {
  return call('GetComponent', id);
}

export function createComponent(input: {
  category: string;
  mpn: string;
  manufacturer: string;
  package: string;
  description: string;
}): Promise<Component> {
  return call('CreateComponent', input);
}

export function updateComponentMetadata(input: {
  id: string;
  mpn: string;
  manufacturer: string;
  package: string;
  description: string;
}): Promise<Component> {
  return call('UpdateComponentMetadata', input);
}

export function replaceComponentAttributes(
  componentId: string,
  attrs: AttributeValue[]
): Promise<void> {
  return call('ReplaceComponentAttributes', componentId, attrs);
}

export function updateComponentInventory(input: {
  id: string;
  quantity: number | null;
  quantityMode: string;
  location: string;
}): Promise<Component> {
  return call('UpdateComponentInventory', input);
}

export function adjustComponentQuantity(id: string, delta: number): Promise<Component> {
  return call('AdjustComponentQuantity', id, delta);
}

export function listProjects(): Promise<Project[]> {
  return call('ListProjects');
}

export function getProject(id: string): Promise<Project> {
  return call('GetProject', id);
}

export function createProject(input: {
  name: string;
  description: string;
}): Promise<Project> {
  return call('CreateProject', input);
}

export function createBlankProject(): Promise<Project> {
  return call('CreateBlankProject');
}

export function createProjectWithDisk(input: {
  name: string;
  description: string;
}): Promise<Project> {
  return call('CreateProjectWithDisk', input);
}

export function updateProjectMetadata(input: {
  id: string;
  name: string;
  description: string;
}): Promise<Project> {
  return call('UpdateProjectMetadata', input);
}

export function deleteProject(id: string): Promise<void> {
  return call('DeleteProject', id);
}

export function getProjectDiskPath(projectId: string): Promise<string> {
  return call('GetProjectDiskPath', projectId);
}

export function revealProjectInFileBrowser(projectId: string): Promise<void> {
  return call('RevealProjectInFileBrowser', projectId);
}

export function deleteProjectAndDisk(projectId: string): Promise<void> {
  return call('DeleteProjectAndDisk', projectId);
}

export function replaceProjectRequirements(
  projectId: string,
  reqs: Requirement[]
): Promise<void> {
  return call('ReplaceProjectRequirements', projectId, reqs);
}

export function listKiCadProjects(
  roots: string[],
  query: string
): Promise<KiCadProjectCandidate[]> {
  return call('ListKiCadProjects', roots, query);
}

export function previewKiCadImport(projectPath: string): Promise<KiCadImportPreview> {
  return call('PreviewKiCadImport', projectPath);
}

export function importKiCadProject(input: {
  targetMode: 'new' | 'existing';
  newProjectName: string;
  newProjectDescription: string;
  existingProjectId: string;
  sourceProjectPath: string;
  rows: KiCadImportPreviewRow[];
}): Promise<Project> {
  return call('ImportKiCadProject', input);
}

export function planProject(projectId: string): Promise<ProjectPlan> {
  return call('PlanProject', projectId);
}

export function sourceRequirement(requirementId: string): Promise<SourceRequirementResult> {
  return call('SourceRequirement', requirementId);
}

export function selectComponentForRequirement(
  requirementId: string,
  componentId: string
): Promise<void> {
  return call('SelectComponentForRequirement', requirementId, componentId);
}

export function clearSelectedComponentForRequirement(
  requirementId: string
): Promise<void> {
  return call('ClearSelectedComponentForRequirement', requirementId);
}

export function getComponentDetail(id: string): Promise<ComponentDetail> {
  return call('GetComponentDetail', id);
}

export function listComponentAssets(componentId: string): Promise<ComponentAsset[]> {
  return call('ListComponentAssets', componentId);
}

export function createComponentAsset(input: {
  componentId: string;
  assetType: string;
  source: string;
  status: string;
  label: string;
  urlOrPath: string;
  previewUrl?: string | null;
  metadataJson?: string | null;
}): Promise<ComponentAsset> {
  return call('CreateComponentAsset', input);
}

export function selectComponentAsset(
  componentId: string,
  assetType: string,
  assetId: string
): Promise<void> {
  return call('SelectComponentAsset', componentId, assetType, assetId);
}

export function clearSelectedComponentAsset(
  componentId: string,
  assetType: string
): Promise<void> {
  return call('ClearSelectedComponentAsset', componentId, assetType);
}

export function searchComponentAssets(
  componentId: string,
  query: string
): Promise<AssetSearchResponse> {
  return call('SearchComponentAssets', componentId, query);
}

export function importComponentAssetResult(
  componentId: string,
  provider: string,
  externalId: string
): Promise<AssetImportResponse> {
  return call('ImportComponentAssetResult', componentId, provider, externalId);
}

export function emptyFilter(): ComponentFilter {
  return { category: '', manufacturer: '', mpn: '', package: '', text: '' };
}

export function quantityDisplay(comp: Component): string {
  if (comp.quantityMode === 'unknown' || comp.quantity === null) return '?';
  if (comp.quantityMode === 'approximate') return `~${comp.quantity}`;
  return String(comp.quantity);
}

export function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString();
}

export function categoryDisplayName(
  categories: CategoryInfo[],
  value: string
): string {
  return categories.find((c) => c.value === value)?.displayName ?? value;
}
