<script lang="ts">
  import type { ComponentAsset } from '../backend';

  let { selectedSymbolAsset = null, selectedFootprintAsset = null, selected3dModelAsset = null }: {
    selectedSymbolAsset?: ComponentAsset | null;
    selectedFootprintAsset?: ComponentAsset | null;
    selected3dModelAsset?: ComponentAsset | null;
  } = $props();

  type PreviewSlot = {
    key: string;
    label: string;
    asset: ComponentAsset | null;
  };

  let slots = $derived([
    { key: 'symbol', label: 'Symbol', asset: selectedSymbolAsset },
    { key: 'footprint', label: 'Footprint', asset: selectedFootprintAsset },
    { key: '3d_model', label: '3D Model', asset: selected3dModelAsset },
  ] as PreviewSlot[]);

  let activeKey = $state('symbol');

  let activeSlot = $derived(slots.find((s) => s.key === activeKey) ?? slots[0]);
</script>

<div class="preview-panel">
  <div class="preview-main">
    {#if activeSlot.asset && activeSlot.asset.previewUrl}
      <img
        class="preview-image"
        src={activeSlot.asset.previewUrl}
        alt="{activeSlot.label} preview"
      />
    {:else if activeSlot.asset}
      <div class="preview-fallback">
        <div class="fallback-icon">
          {#if activeSlot.key === 'symbol'}⏚
          {:else if activeSlot.key === 'footprint'}⬡
          {:else}◇{/if}
        </div>
        <div class="fallback-label">{activeSlot.asset.label || activeSlot.label}</div>
        <div class="fallback-meta">
          {#if activeSlot.asset.source}
            <span class="meta-tag">{activeSlot.asset.source}</span>
          {/if}
          <span class="meta-tag status-{activeSlot.asset.status}">{activeSlot.asset.status}</span>
        </div>
        {#if activeSlot.asset.urlOrPath}
          <div class="fallback-path" title={activeSlot.asset.urlOrPath}>
            {activeSlot.asset.urlOrPath}
          </div>
        {/if}
        <div class="fallback-note">No preview available</div>
      </div>
    {:else}
      <div class="preview-empty">
        <div class="empty-icon">
          {#if activeSlot.key === 'symbol'}⏚
          {:else if activeSlot.key === 'footprint'}⬡
          {:else}◇{/if}
        </div>
        <div class="empty-label">No {activeSlot.label} Selected</div>
        <div class="empty-hint">Attach and select a {activeSlot.label.toLowerCase()} asset in the Assets tab</div>
      </div>
    {/if}
  </div>

  <div class="preview-selectors">
    {#each slots as slot}
      <button
        class="selector-card"
        class:active={activeKey === slot.key}
        onclick={() => (activeKey = slot.key)}
      >
        <div class="selector-icon">
          {#if slot.key === 'symbol'}⏚
          {:else if slot.key === 'footprint'}⬡
          {:else}◇{/if}
        </div>
        <div class="selector-label">{slot.label}</div>
        <div class="selector-status">
          {#if slot.asset}
            <span class="dot dot-has-asset" />
          {:else}
            <span class="dot dot-empty" />
          {/if}
        </div>
      </button>
    {/each}
  </div>
</div>

<style>
  .preview-panel {
    border-bottom: 1px solid var(--color-border);
    background: var(--color-bg-surface);
  }
  .preview-main {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 200px;
    max-height: 280px;
    padding: 20px;
    background: var(--color-bg-muted);
  }
  .preview-image {
    max-width: 100%;
    max-height: 240px;
    object-fit: contain;
    border-radius: var(--radius-md);
  }

  /* Fallback card: asset exists but no preview_url */
  .preview-fallback {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    padding: 24px;
    border: 1px dashed var(--color-border);
    border-radius: var(--radius-lg);
    background: var(--color-bg-surface);
    max-width: 320px;
    text-align: center;
  }
  .fallback-icon {
    font-size: 32px;
    color: var(--color-text-muted);
  }
  .fallback-label {
    font-size: 14px;
    font-weight: 600;
    color: var(--color-text-primary);
  }
  .fallback-meta {
    display: flex;
    gap: 6px;
    flex-wrap: wrap;
    justify-content: center;
  }
  .meta-tag {
    font-size: 11px;
    padding: 2px 6px;
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
  .fallback-path {
    font-size: 11px;
    font-family: var(--font-mono);
    color: var(--color-text-muted);
    max-width: 280px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .fallback-note {
    font-size: 11px;
    color: var(--color-text-muted);
    font-style: italic;
  }

  /* Empty state: no asset selected */
  .preview-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    color: var(--color-text-muted);
  }
  .empty-icon {
    font-size: 36px;
    opacity: 0.4;
  }
  .empty-label {
    font-size: 14px;
    font-weight: 500;
    color: var(--color-text-secondary);
  }
  .empty-hint {
    font-size: 12px;
  }

  /* Selector cards */
  .preview-selectors {
    display: flex;
    gap: 0;
    border-top: 1px solid var(--color-border);
  }
  .selector-card {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    padding: 10px 8px;
    cursor: pointer;
    border-bottom: 2px solid transparent;
    transition: background 0.12s, border-color 0.12s;
  }
  .selector-card:hover {
    background: var(--color-bg-hover);
  }
  .selector-card.active {
    border-bottom-color: var(--color-accent);
    background: var(--color-bg-hover);
  }
  .selector-card + .selector-card {
    border-left: 1px solid var(--color-border);
  }
  .selector-icon {
    font-size: 18px;
    color: var(--color-text-secondary);
  }
  .selector-label {
    font-size: 11px;
    font-weight: 500;
    color: var(--color-text-secondary);
  }
  .selector-status {
    display: flex;
    align-items: center;
  }
  .dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
  }
  .dot-has-asset {
    background: var(--color-success);
  }
  .dot-empty {
    background: var(--color-text-muted);
    opacity: 0.4;
  }
</style>
