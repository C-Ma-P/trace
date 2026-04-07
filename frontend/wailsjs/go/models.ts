export namespace app {
	
	export class AssetImportedAsset {
	    assetType: string;
	    label: string;
	    urlOrPath: string;
	
	    static createFrom(source: any = {}) {
	        return new AssetImportedAsset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.assetType = source["assetType"];
	        this.label = source["label"];
	        this.urlOrPath = source["urlOrPath"];
	    }
	}
	export class AssetImportResponse {
	    importedAssets: AssetImportedAsset[];
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new AssetImportResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.importedAssets = this.convertValues(source["importedAssets"], AssetImportedAsset);
	        this.warnings = source["warnings"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class AssetSearchCandidate {
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
	    previewUrl?: string;
	    sourceUrl?: string;
	    metadata?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new AssetSearchCandidate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.externalId = source["externalId"];
	        this.title = source["title"];
	        this.manufacturer = source["manufacturer"];
	        this.mpn = source["mpn"];
	        this.package = source["package"];
	        this.description = source["description"];
	        this.hasSymbol = source["hasSymbol"];
	        this.hasFootprint = source["hasFootprint"];
	        this.has3dModel = source["has3dModel"];
	        this.hasDatasheet = source["hasDatasheet"];
	        this.previewUrl = source["previewUrl"];
	        this.sourceUrl = source["sourceUrl"];
	        this.metadata = source["metadata"];
	    }
	}
	export class AssetSearchProviderResult {
	    provider: string;
	    candidates: AssetSearchCandidate[];
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new AssetSearchProviderResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.candidates = this.convertValues(source["candidates"], AssetSearchCandidate);
	        this.error = source["error"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AssetSearchResponse {
	    providerResults: AssetSearchProviderResult[];
	
	    static createFrom(source: any = {}) {
	        return new AssetSearchResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providerResults = this.convertValues(source["providerResults"], AssetSearchProviderResult);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AttributeDefinitionInfo {
	    key: string;
	    displayName: string;
	    valueType: string;
	    unit: string;
	
	    static createFrom(source: any = {}) {
	        return new AttributeDefinitionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.displayName = source["displayName"];
	        this.valueType = source["valueType"];
	        this.unit = source["unit"];
	    }
	}
	export class AttributeValueInput {
	    key: string;
	    valueType: string;
	    text?: string;
	    number?: number;
	    bool?: boolean;
	    unit: string;
	
	    static createFrom(source: any = {}) {
	        return new AttributeValueInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.valueType = source["valueType"];
	        this.text = source["text"];
	        this.number = source["number"];
	        this.bool = source["bool"];
	        this.unit = source["unit"];
	    }
	}
	export class AttributeValueResponse {
	    key: string;
	    valueType: string;
	    text?: string;
	    number?: number;
	    bool?: boolean;
	    unit: string;
	
	    static createFrom(source: any = {}) {
	        return new AttributeValueResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.valueType = source["valueType"];
	        this.text = source["text"];
	        this.number = source["number"];
	        this.bool = source["bool"];
	        this.unit = source["unit"];
	    }
	}
	export class CategoryInfo {
	    value: string;
	    displayName: string;
	
	    static createFrom(source: any = {}) {
	        return new CategoryInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.displayName = source["displayName"];
	    }
	}
	export class ComponentAssetResponse {
	    id: string;
	    componentId: string;
	    assetType: string;
	    source: string;
	    status: string;
	    label: string;
	    urlOrPath: string;
	    previewUrl?: string;
	    metadataJson?: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ComponentAssetResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.componentId = source["componentId"];
	        this.assetType = source["assetType"];
	        this.source = source["source"];
	        this.status = source["status"];
	        this.label = source["label"];
	        this.urlOrPath = source["urlOrPath"];
	        this.previewUrl = source["previewUrl"];
	        this.metadataJson = source["metadataJson"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class ComponentResponse {
	    id: string;
	    category: string;
	    mpn: string;
	    manufacturer: string;
	    package: string;
	    description: string;
	    quantity?: number;
	    quantityMode: string;
	    location: string;
	    attributes: AttributeValueResponse[];
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ComponentResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.category = source["category"];
	        this.mpn = source["mpn"];
	        this.manufacturer = source["manufacturer"];
	        this.package = source["package"];
	        this.description = source["description"];
	        this.quantity = source["quantity"];
	        this.quantityMode = source["quantityMode"];
	        this.location = source["location"];
	        this.attributes = this.convertValues(source["attributes"], AttributeValueResponse);
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BagResponse {
	    id: string;
	    label: string;
	    qrData: string;
	    componentId: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new BagResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.qrData = source["qrData"];
	        this.componentId = source["componentId"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class ComponentDetailResponse {
	    component: ComponentResponse;
	    selectedSymbolAsset?: ComponentAssetResponse;
	    selectedFootprintAsset?: ComponentAssetResponse;
	    selected3dModelAsset?: ComponentAssetResponse;
	    selectedDatasheetAsset?: ComponentAssetResponse;
	    assets: ComponentAssetResponse[];
	    imageUrl: string;
	    bags: BagResponse[];
	
	    static createFrom(source: any = {}) {
	        return new ComponentDetailResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.component = this.convertValues(source["component"], ComponentResponse);
	        this.selectedSymbolAsset = this.convertValues(source["selectedSymbolAsset"], ComponentAssetResponse);
	        this.selectedFootprintAsset = this.convertValues(source["selectedFootprintAsset"], ComponentAssetResponse);
	        this.selected3dModelAsset = this.convertValues(source["selected3dModelAsset"], ComponentAssetResponse);
	        this.selectedDatasheetAsset = this.convertValues(source["selectedDatasheetAsset"], ComponentAssetResponse);
	        this.assets = this.convertValues(source["assets"], ComponentAssetResponse);
	        this.imageUrl = source["imageUrl"];
	        this.bags = this.convertValues(source["bags"], BagResponse);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ComponentFilterInput {
	    category: string;
	    manufacturer: string;
	    mpn: string;
	    package: string;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new ComponentFilterInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.category = source["category"];
	        this.manufacturer = source["manufacturer"];
	        this.mpn = source["mpn"];
	        this.package = source["package"];
	        this.text = source["text"];
	    }
	}
	export class ComponentMatchResponse {
	    component: ComponentResponse;
	    availableQuantity: number;
	    score: number;
	
	    static createFrom(source: any = {}) {
	        return new ComponentMatchResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.component = this.convertValues(source["component"], ComponentResponse);
	        this.availableQuantity = source["availableQuantity"];
	        this.score = source["score"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ConstraintResponse {
	    key: string;
	    valueType: string;
	    operator: string;
	    text?: string;
	    number?: number;
	    bool?: boolean;
	    unit: string;
	
	    static createFrom(source: any = {}) {
	        return new ConstraintResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.valueType = source["valueType"];
	        this.operator = source["operator"];
	        this.text = source["text"];
	        this.number = source["number"];
	        this.bool = source["bool"];
	        this.unit = source["unit"];
	    }
	}
	export class CreateComponentAssetInput {
	    componentId: string;
	    assetType: string;
	    source: string;
	    status: string;
	    label: string;
	    urlOrPath: string;
	    previewUrl?: string;
	    metadataJson?: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateComponentAssetInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.componentId = source["componentId"];
	        this.assetType = source["assetType"];
	        this.source = source["source"];
	        this.status = source["status"];
	        this.label = source["label"];
	        this.urlOrPath = source["urlOrPath"];
	        this.previewUrl = source["previewUrl"];
	        this.metadataJson = source["metadataJson"];
	    }
	}
	export class CreateComponentInput {
	    category: string;
	    mpn: string;
	    manufacturer: string;
	    package: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateComponentInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.category = source["category"];
	        this.mpn = source["mpn"];
	        this.manufacturer = source["manufacturer"];
	        this.package = source["package"];
	        this.description = source["description"];
	    }
	}
	export class CreateProjectInput {
	    name: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateProjectInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	    }
	}
	export class OperatorInfo {
	    value: string;
	    displayName: string;
	
	    static createFrom(source: any = {}) {
	        return new OperatorInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.displayName = source["displayName"];
	    }
	}
	export class RequirementPlanResponse {
	    requirement: RequirementResponse;
	    availableQuantity: number;
	    missingQuantity: number;
	    matches: ComponentMatchResponse[];
	
	    static createFrom(source: any = {}) {
	        return new RequirementPlanResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.requirement = this.convertValues(source["requirement"], RequirementResponse);
	        this.availableQuantity = source["availableQuantity"];
	        this.missingQuantity = source["missingQuantity"];
	        this.matches = this.convertValues(source["matches"], ComponentMatchResponse);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RequirementResponse {
	    id: string;
	    projectId: string;
	    name: string;
	    category: string;
	    quantity: number;
	    selectedComponentId?: string;
	    constraints: ConstraintResponse[];
	
	    static createFrom(source: any = {}) {
	        return new RequirementResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.projectId = source["projectId"];
	        this.name = source["name"];
	        this.category = source["category"];
	        this.quantity = source["quantity"];
	        this.selectedComponentId = source["selectedComponentId"];
	        this.constraints = this.convertValues(source["constraints"], ConstraintResponse);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ProjectResponse {
	    id: string;
	    name: string;
	    description: string;
	    requirements: RequirementResponse[];
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ProjectResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.requirements = this.convertValues(source["requirements"], RequirementResponse);
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ProjectPlanResponse {
	    project: ProjectResponse;
	    requirements: RequirementPlanResponse[];
	
	    static createFrom(source: any = {}) {
	        return new ProjectPlanResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.project = this.convertValues(source["project"], ProjectResponse);
	        this.requirements = this.convertValues(source["requirements"], RequirementPlanResponse);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class RecentProjectResponse {
	    id: string;
	    name: string;
	    subtitle: string;
	    openedAtUtc: string;
	
	    static createFrom(source: any = {}) {
	        return new RecentProjectResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.subtitle = source["subtitle"];
	        this.openedAtUtc = source["openedAtUtc"];
	    }
	}
	export class RequirementConstraintInput {
	    key: string;
	    valueType: string;
	    operator: string;
	    text?: string;
	    number?: number;
	    bool?: boolean;
	    unit: string;
	
	    static createFrom(source: any = {}) {
	        return new RequirementConstraintInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.valueType = source["valueType"];
	        this.operator = source["operator"];
	        this.text = source["text"];
	        this.number = source["number"];
	        this.bool = source["bool"];
	        this.unit = source["unit"];
	    }
	}
	export class RequirementInput {
	    id: string;
	    name: string;
	    category: string;
	    quantity: number;
	    selectedComponentId?: string;
	    constraints: RequirementConstraintInput[];
	
	    static createFrom(source: any = {}) {
	        return new RequirementInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.category = source["category"];
	        this.quantity = source["quantity"];
	        this.selectedComponentId = source["selectedComponentId"];
	        this.constraints = this.convertValues(source["constraints"], RequirementConstraintInput);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class StartupStatusResponse {
	    ready: boolean;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new StartupStatusResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ready = source["ready"];
	        this.error = source["error"];
	    }
	}
	export class UpdateInventoryInput {
	    id: string;
	    quantity?: number;
	    quantityMode: string;
	    location: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInventoryInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.quantity = source["quantity"];
	        this.quantityMode = source["quantityMode"];
	        this.location = source["location"];
	    }
	}
	export class UpdateMetadataInput {
	    id: string;
	    mpn: string;
	    manufacturer: string;
	    package: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateMetadataInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.mpn = source["mpn"];
	        this.manufacturer = source["manufacturer"];
	        this.package = source["package"];
	        this.description = source["description"];
	    }
	}
	export class UpdateProjectInput {
	    id: string;
	    name: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateProjectInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	    }
	}

}

