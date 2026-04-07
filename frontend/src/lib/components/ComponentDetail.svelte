<script lang="ts">
  import ComponentPreviewPanel from './ComponentPreviewPanel.svelte';
  import InventoryCard from './InventoryCard.svelte';
  import SpecificationsSection from './SpecificationsSection.svelte';
  import AssetSection from './AssetSection.svelte';
  import {
    deleteComponent,
    categoryDisplayName,
    type ComponentDetail,
    type CategoryInfo,
  } from '../backend';

  let { detail, categories = [], onupdated, ondeleted }: {
    detail: ComponentDetail;
    categories?: CategoryInfo[];
    onupdated?: () => void;
    ondeleted?: (id: string) => void;
  } = $props();

  let activePreviewType = $state('symbol');
  let deleting = $state(false);
  let deleteError = $state('');
  let imageError = $state(false);

  let component = $derived(detail.component);

  $effect(() => {
    imageError = false;
  });

  async function handleDelete() {
    if (!confirm(`Delete ${component.mpn || 'this component'}? This cannot be undone.`)) return;
    deleting = true;
    deleteError = '';
    try {
      await deleteComponent(component.id);
      ondeleted?.(component.id);
    } catch (e: any) {
      deleteError = e?.message ?? String(e);
      deleting = false;
    }
  }
</script>

<div class="inspector">
  <!-- Sticky identity header -->
  <div class="inspector-header">
    <div class="header-layout">
      <div class="header-image-wrap">
        {#if detail.imageUrl && !imageError}
          <img
            class="header-image"
            src={detail.imageUrl}
            alt={component.mpn}
            onerror={() => { imageError = true; }}
          />
        {:else}
          <div class="header-image-placeholder">
            <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <rect x="3" y="3" width="18" height="18" rx="2"/>
              <circle cx="8.5" cy="8.5" r="1.5"/>
              <polyline points="21 15 16 10 5 21"/>
            </svg>
          </div>
        {/if}
      </div>
      <div class="header-info">
        <h2 class="header-mpn">{component.mpn || 'Untitled Component'}</h2>
        <div class="header-meta">
          <span class="badge">{categoryDisplayName(categories, component.category)}</span>
          {#if component.manufacturer}
            <span class="meta-text">{component.manufacturer}</span>
          {/if}
          {#if component.package}
            <span class="meta-text meta-mono">{component.package}</span>
          {/if}
        </div>
      </div>
      <div class="header-actions">
        {#if deleteError}
          <span class="error-text">{deleteError}</span>
        {/if}
        <button
          class="delete-btn"
          onclick={handleDelete}
          disabled={deleting}
          title="Delete component"
        >
          {#if deleting}
            <span class="deleting-text">Deleting…</span>
          {:else}
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
              <polyline points="3 6 5 6 21 6"/>
              <path d="M19 6l-1 14a2 2 0 01-2 2H8a2 2 0 01-2-2L5 6"/>
              <path d="M10 11v6"/>
              <path d="M14 11v6"/>
              <path d="M9 6V4a1 1 0 011-1h4a1 1 0 011 1v2"/>
            </svg>
          {/if}
        </button>
      </div>
    </div>
  </div>

  <!-- Scrollable inspector body -->
  <div class="inspector-body">
    <ComponentPreviewPanel
      selectedSymbolAsset={detail.selectedSymbolAsset}
      selectedFootprintAsset={detail.selectedFootprintAsset}
      selected3dModelAsset={detail.selected3dModelAsset}
      selectedDatasheetAsset={detail.selectedDatasheetAsset}
      activeType={activePreviewType}
      onTypeChange={(t) => (activePreviewType = t)}
    />

    <InventoryCard {component} {onupdated} />

    <SpecificationsSection {component} {categories} {onupdated} />

    <AssetSection
      activeType={activePreviewType}
      componentId={component.id}
      {component}
      assets={detail.assets}
      selectedSymbolAsset={detail.selectedSymbolAsset}
      selectedFootprintAsset={detail.selectedFootprintAsset}
      selected3dModelAsset={detail.selected3dModelAsset}
      selectedDatasheetAsset={detail.selectedDatasheetAsset}
      {onupdated}
    />
  </div>
</div>

<style>
  .inspector {
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: hidden;
  }
  .inspector-header {
    padding: 14px 16px;
    background: var(--color-bg-surface);
    border-bottom: 1px solid var(--color-border);
    flex-shrink: 0;
  }
  .header-layout {
    display: flex;
    align-items: center;
    gap: 14px;
  }
  .header-image-wrap {
    flex-shrink: 0;
  }
  .header-image {
    width: 56px;
    height: 56px;
    object-fit: contain;
    border-radius: var(--radius-md);
    background: var(--color-bg-muted);
    border: 1px solid var(--color-border);
  }
  .header-image-placeholder {
    width: 56px;
    height: 56px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--radius-md);
    background: var(--color-bg-muted);
    border: 1px solid var(--color-border);
    color: var(--color-text-muted);
  }
  .header-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .header-mpn {
    font-size: 16px;
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .header-meta {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    color: var(--color-text-secondary);
  }
  .meta-text {
    color: var(--color-text-secondary);
  }
  .meta-mono {
    font-family: var(--font-mono);
  }
  .header-actions {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .delete-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border-radius: var(--radius-md);
    color: var(--color-text-muted);
    transition: color 0.15s, background 0.15s;
  }
  .delete-btn:hover:not(:disabled) {
    color: var(--color-danger);
    background: var(--color-danger-soft);
  }
  .delete-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .deleting-text {
    font-size: 11px;
    color: var(--color-danger);
    white-space: nowrap;
  }
  .inspector-body {
    flex: 1;
    overflow-y: auto;
  }
  .error-text {
    font-size: 11px;
    color: var(--color-danger);
  }
</style>
