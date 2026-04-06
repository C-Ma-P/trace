<script lang="ts">
  import { importEasyEDAAssets, type EasyEDAImportResult } from '../backend';

  let { open = false, componentId, onimported, onclose }: {
    open: boolean;
    componentId: string;
    onimported?: () => void;
    onclose?: () => void;
  } = $props();

  let lcscId = $state('');
  let busy = $state(false);
  let error = $state('');
  let result = $state<EasyEDAImportResult | null>(null);

  function isValidLCSCID(id: string): boolean {
    return /^C\d+$/.test(id.trim());
  }

  let canImport = $derived(!busy && isValidLCSCID(lcscId));

  async function handleImport() {
    if (!canImport) return;
    busy = true;
    error = '';
    result = null;
    try {
      const res = await importEasyEDAAssets(componentId, lcscId.trim());
      result = res;
      if (res.symbolImported || res.footprintImported || res.model3dImported) {
        onimported?.();
      }
    } catch (e: any) {
      error = e?.message ?? String(e);
    } finally {
      busy = false;
    }
  }

  function handleClose() {
    if (busy) return;
    lcscId = '';
    error = '';
    result = null;
    onclose?.();
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') handleClose();
    if (e.key === 'Enter' && canImport) handleImport();
  }
</script>

{#if open}
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div class="modal-overlay" role="dialog" aria-modal="true" onkeydown={handleKeydown}>
    <div class="modal-panel">
      <div class="modal-header">
        <h2 class="modal-title">Import from EasyEDA / LCSC</h2>
        <button class="modal-close" onclick={handleClose} disabled={busy}>✕</button>
      </div>

      <div class="modal-body">
        {#if !result}
          <p class="modal-desc">
            Enter an LCSC part number to import KiCad-compatible symbol, footprint, and 3D model assets.
          </p>

          <div class="input-group">
            <label for="lcsc-input" class="input-label">LCSC Part Number</label>
            <input
              id="lcsc-input"
              type="text"
              class="input-field"
              bind:value={lcscId}
              placeholder="e.g. C2040"
              disabled={busy}
            />
            {#if lcscId && !isValidLCSCID(lcscId)}
              <span class="input-hint error-text">Must be "C" followed by digits (e.g. C2040)</span>
            {/if}
          </div>

          {#if error}
            <div class="error-banner">{error}</div>
          {/if}

          <div class="modal-actions">
            <button class="btn btn-ghost" onclick={handleClose} disabled={busy}>Cancel</button>
            <button class="btn btn-primary" onclick={handleImport} disabled={!canImport}>
              {#if busy}
                Importing…
              {:else}
                Import Assets
              {/if}
            </button>
          </div>
        {:else}
          <div class="result-panel">
            <h3 class="result-title">
              {#if result.symbolImported || result.footprintImported || result.model3dImported}
                Import Complete
              {:else}
                Import Failed
              {/if}
            </h3>
            <p class="result-lcsc">LCSC {result.lcscId}</p>

            <div class="result-items">
              <div class="result-item" class:success={result.symbolImported} class:failure={!result.symbolImported}>
                <span class="result-icon">{result.symbolImported ? '✓' : '✗'}</span>
                <span>Symbol</span>
              </div>
              <div class="result-item" class:success={result.footprintImported} class:failure={!result.footprintImported}>
                <span class="result-icon">{result.footprintImported ? '✓' : '✗'}</span>
                <span>Footprint</span>
              </div>
              <div class="result-item" class:success={result.model3dImported} class:failure={!result.model3dImported}>
                <span class="result-icon">{result.model3dImported ? '✓' : '✗'}</span>
                <span>3D Model</span>
              </div>
            </div>

            {#if result.warnings.length > 0}
              <div class="result-warnings">
                <h4 class="warnings-title">Warnings</h4>
                <ul class="warnings-list">
                  {#each result.warnings as warning}
                    <li>{warning}</li>
                  {/each}
                </ul>
              </div>
            {/if}

            {#if error}
              <div class="error-banner">{error}</div>
            {/if}

            <div class="modal-actions">
              <button class="btn btn-primary" onclick={handleClose}>Done</button>
            </div>
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-overlay {
    position: fixed;
    inset: 0;
    z-index: 1000;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.5);
  }
  .modal-panel {
    background: var(--color-bg-primary);
    border-radius: var(--radius-lg, 12px);
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
    width: 420px;
    max-width: 90vw;
    max-height: 80vh;
    overflow-y: auto;
  }
  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px 12px;
    border-bottom: 1px solid var(--color-border);
  }
  .modal-title {
    font-size: 15px;
    font-weight: 600;
    color: var(--color-text-primary);
  }
  .modal-close {
    background: none;
    border: none;
    font-size: 16px;
    cursor: pointer;
    color: var(--color-text-secondary);
    padding: 4px;
    border-radius: var(--radius-sm, 4px);
  }
  .modal-close:hover {
    color: var(--color-text-primary);
    background: var(--color-bg-hover);
  }
  .modal-body {
    padding: 16px 20px 20px;
  }
  .modal-desc {
    font-size: 12px;
    color: var(--color-text-secondary);
    margin-bottom: 16px;
    line-height: 1.5;
  }
  .input-group {
    margin-bottom: 16px;
  }
  .input-label {
    display: block;
    font-size: 12px;
    font-weight: 500;
    color: var(--color-text-secondary);
    margin-bottom: 6px;
  }
  .input-field {
    width: 100%;
    padding: 8px 12px;
    font-size: 14px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md, 6px);
    background: var(--color-bg-surface);
    color: var(--color-text-primary);
    box-sizing: border-box;
  }
  .input-field:focus {
    outline: none;
    border-color: var(--color-accent);
  }
  .input-hint {
    display: block;
    font-size: 11px;
    margin-top: 4px;
  }
  .error-text {
    color: var(--color-error, #e53e3e);
  }
  .error-banner {
    background: rgba(229, 62, 62, 0.1);
    border: 1px solid var(--color-error, #e53e3e);
    border-radius: var(--radius-md, 6px);
    padding: 10px 12px;
    font-size: 12px;
    color: var(--color-error, #e53e3e);
    margin-bottom: 16px;
  }
  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 16px;
  }

  /* Result panel */
  .result-panel {
    text-align: center;
  }
  .result-title {
    font-size: 15px;
    font-weight: 600;
    margin-bottom: 4px;
    color: var(--color-text-primary);
  }
  .result-lcsc {
    font-size: 12px;
    color: var(--color-text-secondary);
    margin-bottom: 16px;
  }
  .result-items {
    display: flex;
    gap: 12px;
    justify-content: center;
    margin-bottom: 16px;
  }
  .result-item {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 12px;
    border-radius: var(--radius-md, 6px);
    font-size: 13px;
    font-weight: 500;
  }
  .result-item.success {
    background: rgba(72, 187, 120, 0.15);
    color: #38a169;
  }
  .result-item.failure {
    background: rgba(160, 160, 160, 0.1);
    color: var(--color-text-muted);
  }
  .result-icon {
    font-size: 14px;
    font-weight: 700;
  }
  .result-warnings {
    text-align: left;
    margin-bottom: 12px;
  }
  .warnings-title {
    font-size: 12px;
    font-weight: 600;
    color: var(--color-text-secondary);
    margin-bottom: 6px;
  }
  .warnings-list {
    font-size: 11px;
    color: var(--color-text-secondary);
    padding-left: 16px;
    margin: 0;
  }
  .warnings-list li {
    margin-bottom: 2px;
  }
</style>
