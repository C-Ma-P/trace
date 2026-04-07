<script lang="ts">
  import AddFromFileModal from './AddFromFileModal.svelte';
  import ImportEasyEDAModal from './ImportEasyEDAModal.svelte';
  import SearchOnlineModal from './SearchOnlineModal.svelte';
  import {
    selectComponentAsset,
    clearSelectedComponentAsset,
    importEasyEDAAssets,
    type ComponentAsset,
    type Component,
    type EasyEDAImportResult,
  } from '../backend';

  let { componentId, component, assets = [], selectedSymbolAsset = null, selectedFootprintAsset = null, selected3dModelAsset = null, selectedDatasheetAsset = null, onupdated }: {
    componentId: string;
    component: Component;
    assets?: ComponentAsset[];
    selectedSymbolAsset?: ComponentAsset | null;
    selectedFootprintAsset?: ComponentAsset | null;
    selected3dModelAsset?: ComponentAsset | null;
    selectedDatasheetAsset?: ComponentAsset | null;
    onupdated?: () => void;
  } = $props();

  // --- Type definitions ---

  type SelectedSlot = {
    type: string;
    label: string;
    icon: string;
    asset: ComponentAsset | null;
    availableCount: number;
  };

  type CandidateGroup = {
    type: string;
    label: string;
    assets: ComponentAsset[];
    selectedId: string | null;
  };

  const typeLabels: Record<string, string> = {
    symbol: 'Symbol',
    footprint: 'Footprint',
    '3d_model': '3D Model',
    datasheet: 'Datasheet',
  };

  const typeIcons: Record<string, string> = {
    symbol: '⏚',
    footprint: '⬡',
    '3d_model': '◇',
    datasheet: '📄',
  };

  const typeOrder = ['symbol', 'footprint', '3d_model', 'datasheet'];

  // --- Derived data ---

  let selectedSlots = $derived(buildSelectedSlots());
  let candidateGroups = $derived(buildCandidateGroups(assets));

  function selectedAssetForType(type: string): ComponentAsset | null {
    switch (type) {
      case 'symbol': return selectedSymbolAsset;
      case 'footprint': return selectedFootprintAsset;
      case '3d_model': return selected3dModelAsset;
      case 'datasheet': return selectedDatasheetAsset;
      default: return null;
    }
  }

  function selectedIdForType(type: string): string | null {
    return selectedAssetForType(type)?.id ?? null;
  }

  function buildSelectedSlots(): SelectedSlot[] {
    const byType = new Map<string, number>();
    for (const a of assets) {
      byType.set(a.assetType, (byType.get(a.assetType) ?? 0) + 1);
    }
    return typeOrder.map((t) => ({
      type: t,
      label: typeLabels[t] ?? t,
      icon: typeIcons[t] ?? '•',
      asset: selectedAssetForType(t),
      availableCount: byType.get(t) ?? 0,
    }));
  }

  function buildCandidateGroups(all: ComponentAsset[]): CandidateGroup[] {
    const byType = new Map<string, ComponentAsset[]>();
    for (const a of all) {
      const list = byType.get(a.assetType) ?? [];
      list.push(a);
      byType.set(a.assetType, list);
    }

    return typeOrder.map((t) => ({
      type: t,
      label: typeLabels[t] ?? t,
      assets: byType.get(t) ?? [],
      selectedId: selectedIdForType(t),
    }));
  }

  // --- State ---

  let busy = $state(false);
  let error = $state('');
  let showAddFromFile = $state(false);
  let showImportEasyEDA = $state(false);
  let showSearchOnline = $state(false);

  // LCSC direct-import state
  let knownLcscId = $derived(
    component.attributes.find((a) => a.key === 'lcsc_part')?.text ?? null
  );
  let easyedaBusy = $state(false);
  let easyedaResult = $state<EasyEDAImportResult | null>(null);
  let easyedaError = $state('');

  // --- Actions ---

  async function handleSelect(assetType: string, assetId: string) {
    busy = true;
    error = '';
    try {
      await selectComponentAsset(componentId, assetType, assetId);
      onupdated?.();
    } catch (e: any) {
      error = e?.message ?? String(e);
    } finally {
      busy = false;
    }
  }

  async function handleClear(assetType: string) {
    busy = true;
    error = '';
    try {
      await clearSelectedComponentAsset(componentId, assetType);
      onupdated?.();
    } catch (e: any) {
      error = e?.message ?? String(e);
    } finally {
      busy = false;
    }
  }

  function handleAssetCreated() {
    showAddFromFile = false;
    onupdated?.();
  }

  function handleSearchImported() {
    showSearchOnline = false;
    onupdated?.();
  }

  function handleEasyEDAImported() {
    showImportEasyEDA = false;
    onupdated?.();
  }

  async function handleDirectEasyEDAImport() {
    if (easyedaBusy) return;
    easyedaBusy = true;
    easyedaResult = null;
    easyedaError = '';
    try {
      const res = await importEasyEDAAssets(componentId, knownLcscId ?? '');
      easyedaResult = res;
      if (res.symbolImported || res.footprintImported || res.model3dImported) {
        onupdated?.();
      }
    } catch (e: any) {
      easyedaError = e?.message ?? String(e);
    } finally {
      easyedaBusy = false;
    }
  }
</script>

<div class="assets-tab">
  {#if error}
    <div class="error-text" style="margin-bottom: 12px;">{error}</div>
  {/if}

  <!-- ===== Selected Assets ===== -->
  <section class="tab-section">
    <h3 class="section-title">Selected Assets</h3>
    <div class="selected-grid">
      {#each selectedSlots as slot}
        <div class="selected-row" class:has-asset={!!slot.asset}>
          <span class="selected-type-icon">{slot.icon}</span>
          <span class="selected-type-label">{slot.label}</span>
          {#if slot.asset}
            <span class="selected-asset-label">{slot.asset.label || '(unlabeled)'}</span>
            <span class="meta-tag">{slot.asset.source}</span>
            <div class="selected-actions">
              <button
                class="btn btn-ghost btn-sm"
                onclick={() => handleClear(slot.type)}
                disabled={busy}
              >Clear</button>
            </div>
          {:else}
            <span class="selected-none">None</span>
            {#if slot.availableCount > 0}
              <span class="selected-available-count">{slot.availableCount} available</span>
            {/if}
          {/if}
        </div>
      {/each}
    </div>
  </section>

  <!-- ===== Get CAD Assets ===== -->
  <section class="tab-section">
    <h3 class="section-title">Get CAD Assets</h3>
    <div class="acquire-card">
      <p class="acquire-desc">Find or import symbols, footprints, 3D models, and datasheets for this part.</p>
      <div class="acquire-actions">
        <button class="btn btn-secondary acquire-btn" onclick={() => (showSearchOnline = true)}>
          <span class="btn-icon">🔍</span>
          <span class="acquire-btn-content">
            <span class="acquire-btn-label">Lookup Online</span>
            <span class="acquire-btn-hint">Search providers for this exact part</span>
          </span>
        </button>
        <button class="btn btn-secondary acquire-btn" onclick={() => (showAddFromFile = true)}>
          <span class="btn-icon">📁</span>
          <span class="acquire-btn-content">
            <span class="acquire-btn-label">Import Downloaded Files</span>
            <span class="acquire-btn-hint">KiCad bundles, library files, datasheets</span>
          </span>
        </button>
        {#if knownLcscId}
          <button
            class="btn btn-secondary acquire-btn"
            onclick={handleDirectEasyEDAImport}
            disabled={easyedaBusy}
          >
            <span class="btn-icon">{easyedaBusy ? '…' : '⬇'}</span>
            <span class="acquire-btn-content">
              <span class="acquire-btn-label">{easyedaBusy ? 'Importing…' : 'Import from LCSC / EasyEDA'}</span>
              <span class="acquire-btn-hint acquire-btn-hint--id">{knownLcscId}</span>
            </span>
          </button>
        {:else}
          <button class="btn btn-secondary acquire-btn" onclick={() => (showImportEasyEDA = true)}>
            <span class="btn-icon">⬇</span>
            <span class="acquire-btn-content">
              <span class="acquire-btn-label">Import from LCSC / EasyEDA</span>
              <span class="acquire-btn-hint">Fetch assets by LCSC part number</span>
            </span>
          </button>
        {/if}
      </div>
      {#if easyedaResult}
        <div class="easyeda-inline-result" class:easyeda-result--ok={easyedaResult.symbolImported || easyedaResult.footprintImported || easyedaResult.model3dImported} class:easyeda-result--fail={!easyedaResult.symbolImported && !easyedaResult.footprintImported && !easyedaResult.model3dImported}>
          <span class="easyeda-result-text">
            {#if easyedaResult.symbolImported || easyedaResult.footprintImported || easyedaResult.model3dImported}
              Imported
              {[easyedaResult.symbolImported && 'symbol', easyedaResult.footprintImported && 'footprint', easyedaResult.model3dImported && '3D model'].filter(Boolean).join(', ')}
              for {easyedaResult.lcscId}
            {:else}
              Import failed for {easyedaResult.lcscId}
            {/if}
          </span>
          <button class="easyeda-result-dismiss" onclick={() => (easyedaResult = null)}>✕</button>
        </div>
      {/if}
      {#if easyedaError}
        <div class="easyeda-inline-result easyeda-result--fail">
          <span class="easyeda-result-text">{easyedaError}</span>
          <span class="easyeda-result-actions">
            {#if !knownLcscId}
              <button class="easyeda-result-link" onclick={() => { easyedaError = ''; showImportEasyEDA = true; }}>Enter ID manually</button>
            {/if}
            <button class="easyeda-result-dismiss" onclick={() => (easyedaError = '')}>✕</button>
          </span>
        </div>
      {/if}
    </div>
  </section>

  <!-- ===== Candidate Assets ===== -->
  <section class="tab-section">
    <h3 class="section-title">Candidate Assets</h3>

    {#each candidateGroups as group}
      <div class="candidate-group">
        <h4 class="group-title">{group.label}s</h4>

        {#if group.assets.length === 0}
          <div class="group-empty">No {group.label.toLowerCase()} assets attached</div>
        {:else}
          <div class="asset-list">
            {#each group.assets as asset}
              <div class="asset-row" class:selected={asset.id === group.selectedId}>
                <div class="asset-info">
                  <div class="asset-info-top">
                    <span class="asset-type-icon" title={group.label}>{typeIcons[group.type] ?? '•'}</span>
                    <span class="asset-label">{asset.label || '(unlabeled)'}</span>
                    {#if asset.id === group.selectedId}
                      <span class="selected-badge">Selected</span>
                    {/if}
                  </div>
                  <span class="asset-meta">
                    <span class="meta-tag">{asset.source}</span>
                    <span class="meta-tag status-{asset.status}">{asset.status}</span>
                  </span>
                  {#if asset.urlOrPath}
                    <span class="asset-path" title={asset.urlOrPath}>{asset.urlOrPath}</span>
                  {/if}
                </div>
                <div class="asset-actions">
                  {#if asset.id !== group.selectedId}
                    <button
                      class="btn btn-secondary btn-sm"
                      onclick={() => handleSelect(group.type, asset.id)}
                      disabled={busy}
                    >Select</button>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/each}
  </section>
</div>

<!-- Modals -->
<AddFromFileModal
  open={showAddFromFile}
  {componentId}
  oncreated={handleAssetCreated}
  onclose={() => (showAddFromFile = false)}
/>

<SearchOnlineModal
  open={showSearchOnline}
  {component}
  onimported={handleSearchImported}
  onclose={() => (showSearchOnline = false)}
/>

<ImportEasyEDAModal
  open={showImportEasyEDA}
  {componentId}
  onimported={handleEasyEDAImported}
  onclose={() => (showImportEasyEDA = false)}
/>

<style>
  .assets-tab {
    padding: 20px;
  }

  /* --- Sections --- */
  .tab-section {
    margin-bottom: 28px;
  }
  .tab-section:last-child {
    margin-bottom: 0;
  }
  .section-title {
    font-size: 12px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--color-text-secondary);
    margin-bottom: 12px;
  }

  /* --- Selected Assets --- */
  .selected-grid {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .selected-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 12px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: var(--color-bg-surface);
    min-height: 38px;
  }
  .selected-row.has-asset {
    border-color: var(--color-accent);
    background: var(--color-accent-soft);
  }
  .selected-type-icon {
    font-size: 14px;
    width: 20px;
    text-align: center;
    flex-shrink: 0;
  }
  .selected-type-label {
    font-size: 12px;
    font-weight: 600;
    color: var(--color-text-primary);
    width: 72px;
    flex-shrink: 0;
  }
  .selected-asset-label {
    font-size: 12px;
    color: var(--color-text-primary);
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .selected-none {
    font-size: 12px;
    color: var(--color-text-muted);
    flex: 1;
  }
  .selected-available-count {
    font-size: 11px;
    color: var(--color-accent-text, var(--color-text-secondary));
    background: var(--color-accent-soft);
    padding: 1px 7px;
    border-radius: 10px;
    white-space: nowrap;
  }
  .selected-actions {
    flex-shrink: 0;
  }

  /* --- Get CAD Assets --- */
  .acquire-card {
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    padding: 14px 16px;
    background: var(--color-bg-surface);
  }
  .acquire-desc {
    font-size: 12px;
    color: var(--color-text-secondary);
    margin-bottom: 12px;
  }
  .acquire-actions {
    display: flex;
    gap: 10px;
    flex-wrap: wrap;
  }
  .acquire-btn {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 14px;
    flex: 1;
    min-width: 180px;
    text-align: left;
  }
  .btn-icon {
    font-size: 16px;
    flex-shrink: 0;
  }
  .acquire-btn-content {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .acquire-btn-label {
    font-size: 13px;
    font-weight: 500;
    color: var(--color-text-primary);
  }
  .acquire-btn-hint {
    font-size: 11px;
    color: var(--color-text-secondary);
    font-weight: 400;
  }
  .acquire-btn-hint--id {
    font-family: var(--font-mono, monospace);
    font-size: 11px;
    color: var(--color-accent);
    font-weight: 500;
  }

  /* --- EasyEDA inline result --- */
  .easyeda-inline-result {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    margin-top: 10px;
    padding: 8px 12px;
    border-radius: var(--radius-md, 6px);
    font-size: 12px;
  }
  .easyeda-result--ok {
    background: rgba(72, 187, 120, 0.12);
    border: 1px solid rgba(72, 187, 120, 0.4);
    color: #276749;
  }
  .easyeda-result--fail {
    background: rgba(229, 62, 62, 0.08);
    border: 1px solid rgba(229, 62, 62, 0.35);
    color: var(--color-error, #c53030);
  }
  .easyeda-result-text {
    flex: 1;
    line-height: 1.4;
  }
  .easyeda-result-actions {
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .easyeda-result-link {
    background: none;
    border: none;
    font-size: 11px;
    cursor: pointer;
    color: inherit;
    text-decoration: underline;
    padding: 0;
  }
  .easyeda-result-dismiss {
    background: none;
    border: none;
    font-size: 13px;
    cursor: pointer;
    color: inherit;
    opacity: 0.7;
    padding: 0 2px;
    flex-shrink: 0;
  }
  .easyeda-result-dismiss:hover {
    opacity: 1;
  }

  /* --- Candidate Assets --- */
  .candidate-group {
    margin-bottom: 20px;
  }
  .candidate-group:last-child {
    margin-bottom: 0;
  }
  .group-title {
    font-size: 13px;
    font-weight: 600;
    color: var(--color-text-primary);
    margin-bottom: 8px;
  }
  .group-empty {
    font-size: 12px;
    color: var(--color-text-muted);
    padding: 12px;
    text-align: center;
    border: 1px dashed var(--color-border);
    border-radius: var(--radius-md);
  }
  .asset-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .asset-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: var(--color-bg-surface);
    gap: 12px;
  }
  .asset-row.selected {
    border-color: var(--color-accent);
    background: var(--color-accent-soft);
  }
  .asset-info {
    display: flex;
    flex-direction: column;
    gap: 3px;
    min-width: 0;
    flex: 1;
  }
  .asset-info-top {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .asset-type-icon {
    font-size: 12px;
    color: var(--color-text-muted);
    flex-shrink: 0;
    width: 14px;
    text-align: center;
  }
  .asset-label {
    font-size: 13px;
    font-weight: 500;
    color: var(--color-text-primary);
  }
  .asset-meta {
    display: flex;
    gap: 4px;
  }
  .meta-tag {
    font-size: 11px;
    padding: 1px 6px;
    border-radius: 2px;
    background: var(--color-bg-muted);
    color: var(--color-text-secondary);
  }
  .meta-tag.status-verified {
    background: var(--color-success-soft);
    color: var(--color-success-text);
  }
  .meta-tag.status-selected {
    background: var(--color-info-soft);
    color: var(--color-info-text);
  }
  .meta-tag.status-rejected {
    background: var(--color-danger-soft);
    color: var(--color-danger-text);
  }
  .asset-path {
    font-size: 11px;
    font-family: var(--font-mono);
    color: var(--color-text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .asset-actions {
    flex-shrink: 0;
  }
  .selected-badge {
    font-size: 10px;
    font-weight: 600;
    color: var(--color-accent);
    padding: 2px 6px;
    border: 1px solid var(--color-accent);
    border-radius: 2px;
    white-space: nowrap;
  }
  .error-text {
    font-size: 12px;
    color: var(--color-danger);
  }
</style>
