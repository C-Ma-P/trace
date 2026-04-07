<script lang="ts">
  import type { ComponentAsset } from '../backend';
  import ModelPreview from './ModelPreview.svelte';

  let {
    selectedSymbolAsset = null,
    selectedFootprintAsset = null,
    selected3dModelAsset = null,
    selectedDatasheetAsset = null,
    activeType = 'symbol',
    onTypeChange,
  }: {
    selectedSymbolAsset?: ComponentAsset | null;
    selectedFootprintAsset?: ComponentAsset | null;
    selected3dModelAsset?: ComponentAsset | null;
    selectedDatasheetAsset?: ComponentAsset | null;
    activeType?: string;
    onTypeChange?: (type: string) => void;
  } = $props();

  type PreviewSlot = {
    key: string;
    label: string;
    icon: string;
    asset: ComponentAsset | null;
  };

  let slots = $derived<PreviewSlot[]>([
    { key: 'symbol', label: 'Symbol', icon: '⏚', asset: selectedSymbolAsset },
    { key: 'footprint', label: 'Footprint', icon: '⬡', asset: selectedFootprintAsset },
    { key: '3d_model', label: '3D Model', icon: '◇', asset: selected3dModelAsset },
    { key: 'datasheet', label: 'Datasheet', icon: '📄', asset: selectedDatasheetAsset },
  ]);

  let activeSlot = $derived(slots.find((s) => s.key === activeType) ?? slots[0]);
  let show3dViewer = $derived(activeSlot.key === '3d_model' && activeSlot.asset != null);
  let showDatasheet = $derived(activeSlot.key === 'datasheet');
</script>

<div class="preview-panel">
  <div class="preview-main" class:preview-3d={show3dViewer}>
    {#if show3dViewer}
      <ModelPreview asset={activeSlot.asset!} />
    {:else if showDatasheet}
      <!-- Datasheet: show info card or empty state -->
      {#if activeSlot.asset}
        <div class="preview-fallback">
          <div class="fallback-icon">📄</div>
          <div class="fallback-label">{activeSlot.asset.label || 'Datasheet'}</div>
          <div class="fallback-meta">
            {#if activeSlot.asset.source}
              <span class="meta-tag">{activeSlot.asset.source}</span>
            {/if}
          </div>
          {#if activeSlot.asset.urlOrPath}
            <div class="fallback-path" title={activeSlot.asset.urlOrPath}>
              {activeSlot.asset.urlOrPath}
            </div>
          {/if}
          <div class="fallback-note">Datasheet viewer coming soon</div>
        </div>
      {:else}
        <div class="preview-empty">
          <div class="empty-icon">📄</div>
          <div class="empty-label">No Datasheet Selected</div>
          <div class="empty-hint">Attach and select a datasheet asset below</div>
        </div>
      {/if}
    {:else if activeSlot.asset && activeSlot.asset.previewUrl}
      <img
        class="preview-image"
        src={activeSlot.asset.previewUrl}
        alt="{activeSlot.label} preview"
      />
    {:else if activeSlot.asset}
      <div class="preview-fallback">
        <div class="fallback-icon">{activeSlot.icon}</div>
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
        <div class="empty-icon">{activeSlot.icon}</div>
        <div class="empty-label">No {activeSlot.label} Selected</div>
        <div class="empty-hint">Attach and select a {activeSlot.label.toLowerCase()} asset below</div>
      </div>
    {/if}
  </div>

  <!-- Representation type selector -->
  <div class="preview-selectors">
    {#each slots as slot}
      <button
        class="selector-btn"
        class:active={activeType === slot.key}
        onclick={() => onTypeChange?.(slot.key)}
      >
        <span class="selector-icon">{slot.icon}</span>
        <span class="selector-label">{slot.label}</span>
        {#if slot.asset}
          <span class="selector-dot has-asset"></span>
        {:else}
          <span class="selector-dot"></span>
        {/if}
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
  .preview-main.preview-3d {
    padding: 0;
    height: 280px;
  }
  .preview-image {
    max-width: 100%;
    max-height: 240px;
    object-fit: contain;
    border-radius: var(--radius-md);
  }

  /* Fallback card: asset exists but no renderable preview */
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

  /* Type selector bar */
  .preview-selectors {
    display: flex;
    gap: 0;
    border-top: 1px solid var(--color-border);
  }
  .selector-btn {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 5px;
    padding: 8px 4px;
    cursor: pointer;
    border-bottom: 2px solid transparent;
    transition: background 0.12s, border-color 0.12s;
  }
  .selector-btn:hover {
    background: var(--color-bg-hover);
  }
  .selector-btn.active {
    border-bottom-color: var(--color-accent);
    background: var(--color-bg-hover);
  }
  .selector-btn + .selector-btn {
    border-left: 1px solid var(--color-border);
  }
  .selector-icon {
    font-size: 14px;
    color: var(--color-text-secondary);
  }
  .selector-label {
    font-size: 11px;
    font-weight: 500;
    color: var(--color-text-secondary);
  }
  .selector-dot {
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background: var(--color-text-muted);
    opacity: 0.3;
  }
  .selector-dot.has-asset {
    background: var(--color-success);
    opacity: 1;
  }
</style>
