<script lang="ts">
  import { tick } from 'svelte';
  import Modal from '../ui/Modal.svelte';
  import {
    searchComponentAssets,
    importComponentAssetResult,
    type AssetSearchResponse,
    type AssetSearchProviderResult,
    type AssetSearchCandidate,
    type Component,
  } from '../backend';
  import { Browser } from '@wailsio/runtime';

  let { open = false, component = null, onclose, onimported }: {
    open?: boolean;
    component?: Component | null;
    onclose?: () => void;
    onimported?: () => void;
  } = $props();

  let refineQuery = $state('');
  let searching = $state(false);
  let searchError = $state('');
  let results: AssetSearchResponse | null = $state(null);
  let importingId: string | null = $state(null);

  let hasMpn = $derived(!!(component?.mpn?.trim()));

  $effect(() => {
    if (open) {
      refineQuery = '';
      results = null;
      searchError = '';
      importingId = null;
      if (component?.mpn) {
        tick().then(() => runSearch(component!.mpn!));
      }
    }
  });

  function handleClose() {
    onclose?.();
  }

  async function runSearch(q: string) {
    if (!component || !q.trim()) return;
    searching = true;
    searchError = '';
    results = null;
    try {
      results = await searchComponentAssets(component.id, q.trim());
    } catch (e: any) {
      searchError = e?.message ?? String(e);
    } finally {
      searching = false;
    }
  }

  async function handleRefineSearch() {
    const q = refineQuery.trim() || component?.mpn || '';
    await runSearch(q);
  }

  async function handleImport(providerResult: AssetSearchProviderResult, candidate: AssetSearchCandidate) {
    if (!component) return;
    importingId = `${providerResult.provider}:${candidate.externalId}`;
    try {
      await importComponentAssetResult(component.id, providerResult.provider, candidate.externalId);
      onimported?.();
    } catch (e: any) {
      searchError = e?.message ?? String(e);
    } finally {
      importingId = null;
    }
  }

  function openUrl(url: string) {
    if (url) {
      Browser.OpenURL(url);
    }
  }

  function hasAnyResults(resp: AssetSearchResponse): boolean {
    return resp.providerResults.some(
      (pr) => pr.candidates && pr.candidates.length > 0
    );
  }
</script>

<Modal {open} title="Find Assets for This Part" width="620px" onclose={handleClose}>
  <div class="lookup-online">

    <!-- Part identity header -->
    {#if component}
      <div class="part-header">
        <div class="part-identity">
          <span class="part-mpn">{component.mpn || '—'}</span>
          {#if component.manufacturer}
            <span class="part-mfr">{component.manufacturer}</span>
          {/if}
        </div>
        <div class="part-label">Looking up CAD assets for this part</div>
      </div>
    {/if}

    <!-- No MPN guard -->
    {#if !hasMpn}
      <div class="no-mpn-notice">
        <div class="no-mpn-icon">⚠</div>
        <div class="no-mpn-body">
          <p class="no-mpn-title">MPN required for online lookup</p>
          <p class="no-mpn-detail">
            Online asset lookup uses the component's manufacturer part number (MPN) to find exact matches.
            Add an MPN to this component to use this feature.
          </p>
        </div>
      </div>
      <div class="modal-actions">
        <button class="btn btn-secondary" onclick={handleClose}>Close</button>
      </div>
    {:else}

      {#if searchError}
        <div class="error-text">{searchError}</div>
      {/if}

      {#if searching}
        <div class="searching-state">
          <span class="searching-spinner">⟳</span>
          <span>Looking up assets…</span>
        </div>
      {:else if results}
        {#if !hasAnyResults(results) && results.providerResults.every((pr) => !pr.error)}
          <div class="placeholder-area">
            <p class="placeholder-title">No results found</p>
            <p class="placeholder-detail">Try refining the part number below, or check the manufacturer's site.</p>
          </div>
        {/if}

        {#each results.providerResults as pr}
          <div class="provider-section">
            <h4 class="provider-name">{pr.provider}</h4>
            {#if pr.error}
              <div class="provider-error">{pr.error}</div>
            {:else if !pr.candidates || pr.candidates.length === 0}
              <div class="provider-empty">No results from this provider</div>
            {:else}
              <div class="candidate-list">
                {#each pr.candidates as candidate}
                  <div class="candidate-row">
                    <div class="candidate-info">
                      <div class="candidate-title">{candidate.title || candidate.mpn}</div>
                      <div class="candidate-meta">
                        {#if candidate.manufacturer}
                          <span class="meta-tag">{candidate.manufacturer}</span>
                        {/if}
                        {#if candidate.mpn}
                          <span class="meta-tag">{candidate.mpn}</span>
                        {/if}
                        {#if candidate.package}
                          <span class="meta-tag">{candidate.package}</span>
                        {/if}
                      </div>
                      {#if candidate.description}
                        <div class="candidate-desc">{candidate.description}</div>
                      {/if}
                      <div class="candidate-badges">
                        {#if candidate.hasSymbol}<span class="asset-badge">Symbol</span>{/if}
                        {#if candidate.hasFootprint}<span class="asset-badge">Footprint</span>{/if}
                        {#if candidate.has3dModel}<span class="asset-badge">3D</span>{/if}
                        {#if candidate.hasDatasheet}<span class="asset-badge">Datasheet</span>{/if}
                      </div>
                    </div>
                    <div class="candidate-actions">
                      {#if candidate.sourceUrl}
                        <button
                          class="btn btn-ghost btn-sm"
                          onclick={() => openUrl(candidate.sourceUrl || '')}
                        >View</button>
                      {/if}
                      <button
                        class="btn btn-secondary btn-sm"
                        onclick={() => handleImport(pr, candidate)}
                        disabled={importingId === `${pr.provider}:${candidate.externalId}`}
                      >
                        {importingId === `${pr.provider}:${candidate.externalId}` ? 'Importing…' : 'Import'}
                      </button>
                    </div>
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        {/each}
      {/if}

      <!-- Refinement (secondary) -->
      <details class="refine-section">
        <summary class="refine-summary">Refine search</summary>
        <div class="refine-body">
          <div class="refine-bar">
            <input
              class="form-input refine-input"
              type="text"
              placeholder="Override part number…"
              bind:value={refineQuery}
              onkeydown={(e) => e.key === 'Enter' && handleRefineSearch()}
            />
            <button
              class="btn btn-secondary"
              onclick={handleRefineSearch}
              disabled={searching}
            >
              {searching ? 'Searching…' : 'Re-search'}
            </button>
          </div>
          <p class="refine-hint">Overrides the MPN used for this lookup.</p>
        </div>
      </details>

      <div class="modal-actions">
        <button class="btn btn-secondary" onclick={handleClose}>Close</button>
      </div>
    {/if}
  </div>
</Modal>

<style>
  .lookup-online {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  /* Part identity header */
  .part-header {
    display: flex;
    flex-direction: column;
    gap: 3px;
    padding: 10px 14px;
    background: var(--color-bg-muted);
    border-radius: var(--radius-md);
    border: 1px solid var(--color-border);
  }
  .part-identity {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .part-mpn {
    font-size: 15px;
    font-weight: 600;
    color: var(--color-text-primary);
    font-family: var(--font-mono);
  }
  .part-mfr {
    font-size: 12px;
    color: var(--color-text-secondary);
  }
  .part-label {
    font-size: 11px;
    color: var(--color-text-muted);
  }

  /* No MPN guard */
  .no-mpn-notice {
    display: flex;
    gap: 12px;
    align-items: flex-start;
    padding: 14px;
    background: var(--color-warning-soft);
    border: 1px solid var(--color-warning-border);
    border-radius: var(--radius-md);
  }
  .no-mpn-icon {
    font-size: 18px;
    flex-shrink: 0;
    color: var(--color-warning-text);
  }
  .no-mpn-body {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .no-mpn-title {
    font-size: 13px;
    font-weight: 600;
    color: var(--color-text-primary);
  }
  .no-mpn-detail {
    font-size: 12px;
    color: var(--color-text-secondary);
    line-height: 1.5;
  }

  /* Searching state */
  .searching-state {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 20px;
    justify-content: center;
    font-size: 13px;
    color: var(--color-text-secondary);
  }
  .searching-spinner {
    font-size: 18px;
    animation: spin 1s linear infinite;
    display: inline-block;
  }
  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  /* Placeholder */
  .placeholder-area {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    padding: 28px 16px;
    border: 1px dashed var(--color-border);
    border-radius: var(--radius-md);
    background: var(--color-bg-muted);
  }
  .placeholder-title {
    font-size: 14px;
    font-weight: 600;
    color: var(--color-text-primary);
    margin-bottom: 8px;
  }
  .placeholder-detail {
    font-size: 12px;
    color: var(--color-text-secondary);
    line-height: 1.5;
    max-width: 340px;
  }

  /* Provider results */
  .provider-section {
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    padding: 12px;
  }
  .provider-name {
    font-size: 13px;
    font-weight: 600;
    margin-bottom: 8px;
    color: var(--color-text-primary);
  }
  .provider-error {
    font-size: 12px;
    color: var(--color-warning-text);
    background: var(--color-warning-soft);
    border-radius: var(--radius-sm);
    padding: 8px 10px;
  }
  .provider-empty {
    font-size: 12px;
    color: var(--color-text-secondary);
    padding: 8px 0;
  }
  .candidate-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .candidate-row {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 12px;
    padding: 8px;
    border-radius: var(--radius-sm);
    background: var(--color-bg-muted);
  }
  .candidate-info {
    flex: 1;
    min-width: 0;
  }
  .candidate-title {
    font-size: 13px;
    font-weight: 500;
    color: var(--color-text-primary);
  }
  .candidate-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    margin-top: 4px;
  }
  .candidate-desc {
    font-size: 11px;
    color: var(--color-text-secondary);
    margin-top: 4px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .candidate-badges {
    display: flex;
    gap: 4px;
    margin-top: 6px;
  }
  .asset-badge {
    font-size: 10px;
    padding: 1px 6px;
    border-radius: 3px;
    background: var(--color-accent-soft);
    color: var(--color-accent-text);
    font-weight: 500;
  }
  .candidate-actions {
    display: flex;
    gap: 4px;
    flex-shrink: 0;
  }

  /* Refinement (secondary) */
  .refine-section {
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
  }
  .refine-summary {
    font-size: 12px;
    color: var(--color-text-secondary);
    cursor: pointer;
    padding: 8px 12px;
    user-select: none;
    list-style: none;
  }
  .refine-summary::-webkit-details-marker { display: none; }
  .refine-summary::before {
    content: '▸ ';
    font-size: 10px;
  }
  details[open] .refine-summary::before {
    content: '▾ ';
  }
  .refine-body {
    padding: 0 12px 12px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .refine-bar {
    display: flex;
    gap: 8px;
  }
  .refine-input {
    flex: 1;
  }
  .refine-hint {
    font-size: 11px;
    color: var(--color-text-muted);
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }
  .error-text {
    color: var(--color-danger);
    font-size: 12px;
  }
  .meta-tag {
    font-size: 11px;
    padding: 1px 6px;
    border-radius: 2px;
    background: var(--color-bg-muted);
    color: var(--color-text-secondary);
  }
</style>
