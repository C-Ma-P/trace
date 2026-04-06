<script lang="ts">
  import Modal from '../ui/Modal.svelte';
  import { ingestComponentAssets, validateAssetPath, type IngestResult, type ValidateAssetPathResult } from '../backend';
  import { pickAssetFile, pickAssetDir } from '../windowService';
  import FileArrowDown from '../icons/FileArrowDown.svelte';
  import FolderOpen from '../icons/FolderOpen.svelte';

  let { open = false, componentId, onclose, oncreated }: {
    open?: boolean;
    componentId: string;
    onclose?: () => void;
    oncreated?: () => void;
  } = $props();

  let filePath = $state('');
  let importing = $state(false);
  let error = $state('');
  let result: IngestResult | null = $state(null);
  let pathValidation: ValidateAssetPathResult | null = $state(null);
  let validating = $state(false);

  function reset() {
    filePath = '';
    error = '';
    result = null;
    pathValidation = null;
    validating = false;
  }

  $effect(() => {
    if (open) reset();
  });

  let debounceTimer: ReturnType<typeof setTimeout> | null = null;

  $effect(() => {
    const path = filePath.trim();
    pathValidation = null;
    if (debounceTimer !== null) clearTimeout(debounceTimer);
    if (!path) {
      validating = false;
      return;
    }
    validating = true;
    debounceTimer = setTimeout(async () => {
      pathValidation = await validateAssetPath(path);
      validating = false;
    }, 300);
  });

  const isPathInvalid = $derived(
    filePath.trim() !== '' && pathValidation !== null && !pathValidation.valid
  );

  const canImport = $derived(
    filePath.trim() !== '' && !isPathInvalid && !validating && !importing
  );

  async function handleBrowse() {
    try {
      const selected = await pickAssetFile();
      if (selected) filePath = selected;
    } catch (e: any) {
      error = e?.message ?? String(e);
    }
  }

  async function handleBrowseDir() {
    try {
      const selected = await pickAssetDir();
      if (selected) filePath = selected;
    } catch (e: any) {
      error = e?.message ?? String(e);
    }
  }

  async function handleImport() {
    if (!canImport) return;
    importing = true;
    error = '';
    result = null;
    try {
      result = await ingestComponentAssets(componentId, filePath.trim());
      if (result.assets.length > 0) oncreated?.();
    } catch (e: any) {
      error = e?.message ?? String(e);
    } finally {
      importing = false;
    }
  }

  function handleClose() {
    onclose?.();
  }

  const typeLabels: Record<string, string> = {
    symbol: 'Symbol',
    footprint: 'Footprint',
    '3d_model': '3D Model',
    datasheet: 'Datasheet',
  };
</script>

<Modal {open} title="Import Component Assets" onclose={handleClose}>
  <div class="import-assets">
    <p class="help-text">
      Import KiCad library files, 3D models, datasheets, or zip archives containing these.
      Supported: .kicad_sym, .kicad_mod, .pretty dirs, .step, .stp, .wrl, .pdf, .zip
    </p>

    <div class="form-group">
      <label for="aft-path">File or Directory</label>
      <div class="file-picker">
        <input
          id="aft-path"
          class="form-input file-input"
          class:invalid={isPathInvalid}
          type="text"
          placeholder="Select a file or .pretty directory…"
          bind:value={filePath}
        />
        <button class="btn btn-secondary icon-btn" onclick={handleBrowse} disabled={importing} title="Browse for a file (.kicad_sym, .zip, .step, .pdf, …)">
          <FileArrowDown size={18} />
        </button>
        <button class="btn btn-secondary icon-btn" onclick={handleBrowseDir} disabled={importing} title="Browse for a .pretty footprint library directory">
          <FolderOpen size={18} />
        </button>
      </div>
      {#if isPathInvalid && pathValidation}
        <div class="path-error">{pathValidation.reason}</div>
      {/if}
    </div>

    {#if error}
      <div class="error-text">{error}</div>
    {/if}

    {#if result}
      <div class="result-section">
        {#if result.assets.length > 0}
          <div class="result-success">
            <span class="result-icon">✓</span>
            <span>Imported {result.assets.length} asset{result.assets.length !== 1 ? 's' : ''}</span>
          </div>
          <div class="result-list">
            {#each result.assets as asset}
              <div class="result-row">
                <span class="meta-tag">{typeLabels[asset.assetType] ?? asset.assetType}</span>
                <span class="result-label">{asset.label}</span>
                <span class="result-filename" title={asset.originalFilename}>{asset.originalFilename}</span>
              </div>
            {/each}
          </div>
        {:else}
          <div class="result-empty">No supported assets found in this file.</div>
        {/if}

        {#if result.warnings.length > 0}
          <div class="result-warnings">
            {#each result.warnings as w}
              <div class="result-warning">⚠ {w}</div>
            {/each}
          </div>
        {/if}

        {#if result.unsupported.length > 0}
          <details class="unsupported-section">
            <summary class="unsupported-summary">{result.unsupported.length} unsupported file{result.unsupported.length !== 1 ? 's' : ''} skipped</summary>
            <div class="unsupported-list">
              {#each result.unsupported as f}
                <div class="unsupported-item">{f}</div>
              {/each}
            </div>
          </details>
        {/if}
      </div>
    {/if}

    <div class="modal-actions">
      {#if result && result.assets.length > 0}
        <button class="btn btn-primary" onclick={handleClose}>
          Done
        </button>
      {:else}
        <button class="btn btn-secondary" onclick={handleClose} disabled={importing}>
          Cancel
        </button>
        <button class="btn btn-primary" onclick={handleImport} disabled={!canImport}>
          {importing ? 'Importing…' : 'Import'}
        </button>
      {/if}
    </div>
  </div>
</Modal>

<style>
  .import-assets {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }
  .help-text {
    font-size: 12px;
    color: var(--color-text-secondary);
    line-height: 1.5;
    padding: 10px 12px;
    background: var(--color-bg-muted);
    border-radius: var(--radius-md);
  }
  .file-picker {
    display: flex;
    gap: 8px;
    align-items: center;
  }
  .file-input {
    flex: 1;
  }
  .file-input.invalid {
    border-color: var(--color-error, #ef4444);
    outline-color: var(--color-error, #ef4444);
  }
  .path-error {
    margin-top: 4px;
    font-size: 11px;
    color: var(--color-error, #ef4444);
  }
  .icon-btn {
    min-width: 40px;
    padding: 6px 8px;
    font-size: 18px;
    line-height: 1;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding-top: 4px;
  }

  /* Result styles */
  .result-section {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .result-success {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 13px;
    font-weight: 600;
    color: var(--color-success, #22c55e);
  }
  .result-icon {
    font-size: 16px;
  }
  .result-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .result-row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 10px;
    background: var(--color-bg-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    font-size: 12px;
  }
  .result-label {
    font-weight: 500;
    color: var(--color-text-primary);
  }
  .result-filename {
    color: var(--color-text-muted);
    font-family: var(--font-mono);
    font-size: 11px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .result-empty {
    font-size: 12px;
    color: var(--color-text-secondary);
    padding: 8px 0;
  }
  .result-warnings {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .result-warning {
    font-size: 11px;
    color: var(--color-warning-text, #f59e0b);
  }
  .unsupported-section {
    font-size: 11px;
    color: var(--color-text-muted);
  }
  .unsupported-summary {
    cursor: pointer;
  }
  .unsupported-list {
    padding: 4px 0 0 16px;
  }
  .unsupported-item {
    font-family: var(--font-mono);
    font-size: 10px;
  }
</style>
