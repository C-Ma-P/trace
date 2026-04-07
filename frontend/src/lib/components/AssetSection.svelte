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

  let {
    activeType = 'symbol',
    componentId,
    component,
    assets = [],
    selectedSymbolAsset = null,
    selectedFootprintAsset = null,
    selected3dModelAsset = null,
    selectedDatasheetAsset = null,
    onupdated,
  }: {
    activeType?: string;
    componentId: string;
    component: Component;
    assets?: ComponentAsset[];
    selectedSymbolAsset?: ComponentAsset | null;
    selectedFootprintAsset?: ComponentAsset | null;
    selected3dModelAsset?: ComponentAsset | null;
    selectedDatasheetAsset?: ComponentAsset | null;
    onupdated?: () => void;
  } = $props();

  const typeLabels: Record<string, string> = {
    symbol: 'Symbol',
    footprint: 'Footprint',
    '3d_model': '3D Model',
    datasheet: 'Datasheet',
  };

  let typeLabel = $derived(typeLabels[activeType] ?? activeType);

  let selectedAsset = $derived(
    activeType === 'symbol' ? selectedSymbolAsset :
    activeType === 'footprint' ? selectedFootprintAsset :
    activeType === '3d_model' ? selected3dModelAsset :
    activeType === 'datasheet' ? selectedDatasheetAsset :
    null
  );

  let candidates = $derived(assets.filter((a) => a.assetType === activeType));
  let selectedId = $derived(selectedAsset?.id ?? null);

  let busy = $state(false);
  let error = $state('');
  let showAddFromFile = $state(false);
  let showImportEasyEDA = $state(false);
  let showSearchOnline = $state(false);

  let knownLcscId = $derived(
    component.attributes.find((a) => a.key === 'lcsc_part')?.text ?? null
  );
  let easyedaBusy = $state(false);
  let easyedaResult = $state<EasyEDAImportResult | null>(null);
  let easyedaError = $state('');

  async function handleSelect(assetId: string) {
    busy = true;
    error = '';
    try {
      await selectComponentAsset(componentId, activeType, assetId);
      onupdated?.();
    } catch (e: any) {
      error = e?.message ?? String(e);
    } finally {
      busy = false;
    }
  }

  async function handleClear() {
    busy = true;
    error = '';
    try {
      await clearSelectedComponentAsset(componentId, activeType);
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

<section class="asset-section">
  <div class="section-header">
    <h3 class="section-title">{typeLabel} Assets</h3>
  </div>

  {#if error}
    <div class="error-text">{error}</div>
  {/if}

  <!-- Current selection for active type -->
  {#if selectedAsset}
    <div class="current-asset">
      <div class="current-asset-info">
        <span class="current-asset-name">{selectedAsset.label || '(unlabeled)'}</span>
        <div class="current-asset-meta">
          <span class="meta-tag">{selectedAsset.source}</span>
          <span class="meta-tag status-{selectedAsset.status}">{selectedAsset.status}</span>
        </div>
      </div>
      <button class="btn btn-ghost btn-sm" onclick={handleClear} disabled={busy}>Clear</button>
    </div>
  {:else}
    <div class="no-selection">No {typeLabel.toLowerCase()} selected</div>
  {/if}

  <!-- Candidates for active type -->
  {#if candidates.length > 0}
    <div class="candidates">
      <h4 class="subsection-title">Available</h4>
      <div class="candidate-list">
        {#each candidates as asset}
          <div class="candidate-row" class:active={asset.id === selectedId}>
            <div class="candidate-info">
              <div class="candidate-top">
                <span class="candidate-label">{asset.label || '(unlabeled)'}</span>
                {#if asset.id === selectedId}
                  <span class="selected-indicator">Selected</span>
                {/if}
              </div>
              <div class="candidate-meta">
                <span class="meta-tag">{asset.source}</span>
                <span class="meta-tag status-{asset.status}">{asset.status}</span>
              </div>
              {#if asset.urlOrPath}
                <span class="candidate-path" title={asset.urlOrPath}>{asset.urlOrPath}</span>
              {/if}
            </div>
            {#if asset.id !== selectedId}
              <button
                class="btn btn-secondary btn-sm"
                onclick={() => handleSelect(asset.id)}
                disabled={busy}
              >Select</button>
            {/if}
          </div>
        {/each}
      </div>
    </div>
  {:else}
    <div class="no-candidates">No {typeLabel.toLowerCase()} assets attached</div>
  {/if}

  <!-- Asset acquisition -->
  <div class="acquire">
    <h4 class="subsection-title">Get Assets</h4>
    <div class="acquire-buttons">
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
      <div class="result-banner" class:result-ok={easyedaResult.symbolImported || easyedaResult.footprintImported || easyedaResult.model3dImported}>
        <span class="result-text">
          {#if easyedaResult.symbolImported || easyedaResult.footprintImported || easyedaResult.model3dImported}
            Imported {[easyedaResult.symbolImported && 'symbol', easyedaResult.footprintImported && 'footprint', easyedaResult.model3dImported && '3D model'].filter(Boolean).join(', ')}
          {:else}
            Import failed for {easyedaResult.lcscId}
          {/if}
        </span>
        <button class="dismiss-btn" onclick={() => (easyedaResult = null)}>✕</button>
      </div>
    {/if}

    {#if easyedaError}
      <div class="result-banner result-error">
        <span class="result-text">{easyedaError}</span>
        <span class="result-actions">
          {#if !knownLcscId}
            <button class="result-link" onclick={() => { easyedaError = ''; showImportEasyEDA = true; }}>Enter ID manually</button>
          {/if}
          <button class="dismiss-btn" onclick={() => (easyedaError = '')}>✕</button>
        </span>
      </div>
    {/if}
  </div>
</section>

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
  .asset-section {
    padding: 20px;
  }
  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 14px;
  }
  .section-title {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--color-text-secondary);
  }
  .subsection-title {
    font-size: 12px;
    font-weight: 600;
    color: var(--color-text-secondary);
    margin-bottom: 8px;
  }

  /* Current selection */
  .current-asset {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 12px;
    border: 1px solid var(--color-accent);
    border-radius: var(--radius-md);
    background: var(--color-accent-soft);
    margin-bottom: 16px;
    gap: 12px;
  }
  .current-asset-info {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
    flex: 1;
  }
  .current-asset-name {
    font-size: 13px;
    font-weight: 500;
    color: var(--color-text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .current-asset-meta {
    display: flex;
    gap: 4px;
  }
  .no-selection {
    font-size: 12px;
    color: var(--color-text-muted);
    padding: 10px 12px;
    border: 1px dashed var(--color-border);
    border-radius: var(--radius-md);
    text-align: center;
    margin-bottom: 16px;
  }

  /* Candidates */
  .candidates {
    margin-bottom: 16px;
  }
  .candidate-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .candidate-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: var(--color-bg-surface);
    gap: 12px;
  }
  .candidate-row.active {
    border-color: var(--color-accent);
    background: var(--color-accent-soft);
  }
  .candidate-info {
    display: flex;
    flex-direction: column;
    gap: 3px;
    min-width: 0;
    flex: 1;
  }
  .candidate-top {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .candidate-label {
    font-size: 13px;
    font-weight: 500;
    color: var(--color-text-primary);
  }
  .selected-indicator {
    font-size: 10px;
    font-weight: 600;
    color: var(--color-accent);
    padding: 1px 5px;
    border: 1px solid var(--color-accent);
    border-radius: 2px;
  }
  .candidate-meta {
    display: flex;
    gap: 4px;
  }
  .candidate-path {
    font-size: 11px;
    font-family: var(--font-mono);
    color: var(--color-text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .no-candidates {
    font-size: 12px;
    color: var(--color-text-muted);
    padding: 10px 12px;
    border: 1px dashed var(--color-border);
    border-radius: var(--radius-md);
    text-align: center;
    margin-bottom: 16px;
  }

  /* Acquire */
  .acquire {
    margin-top: 4px;
  }
  .acquire-buttons {
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
    font-size: 13px;
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

  /* Result banners */
  .result-banner {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    margin-top: 10px;
    padding: 8px 12px;
    border-radius: var(--radius-md);
    font-size: 12px;
    background: rgba(72, 187, 120, 0.12);
    border: 1px solid rgba(72, 187, 120, 0.4);
    color: #6ee7b7;
  }
  .result-banner:not(.result-ok) {
    background: rgba(229, 62, 62, 0.08);
    border: 1px solid rgba(229, 62, 62, 0.35);
    color: var(--color-danger-text);
  }
  .result-banner.result-error {
    background: rgba(229, 62, 62, 0.08);
    border: 1px solid rgba(229, 62, 62, 0.35);
    color: var(--color-danger-text);
  }
  .result-text {
    flex: 1;
    line-height: 1.4;
  }
  .result-actions {
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .result-link {
    background: none;
    border: none;
    font-size: 11px;
    cursor: pointer;
    color: inherit;
    text-decoration: underline;
    padding: 0;
  }
  .dismiss-btn {
    background: none;
    border: none;
    font-size: 13px;
    cursor: pointer;
    color: inherit;
    opacity: 0.7;
    padding: 0 2px;
    flex-shrink: 0;
  }
  .dismiss-btn:hover {
    opacity: 1;
  }

  /* Shared */
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
  .error-text {
    font-size: 12px;
    color: var(--color-danger);
    margin-bottom: 10px;
  }
</style>
