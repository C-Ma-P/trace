<script lang="ts">
  import Tabs from '../ui/Tabs.svelte';
  import DetailsTab from './DetailsTab.svelte';
  import InventoryTab from './InventoryTab.svelte';
  import AssetsTab from './AssetsTab.svelte';
  import ComponentPreviewPanel from './ComponentPreviewPanel.svelte';
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

  let activeTab = $state('details');
  let deleting = $state(false);
  let deleteError = $state('');

  const tabs = [
    { key: 'details', label: 'Details' },
    { key: 'inventory', label: 'Inventory' },
    { key: 'assets', label: 'Assets' },
  ];

  let component = $derived(detail.component);

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

<div class="detail-container">
  <div class="detail-header">
    <div class="header-info">
      <h2 class="header-mpn">{component.mpn || 'Untitled Component'}</h2>
      <div class="header-meta">
        <span class="badge">{categoryDisplayName(categories, component.category)}</span>
        {#if component.manufacturer}
          <span class="meta-text">{component.manufacturer}</span>
        {/if}
        {#if component.package}
          <span class="meta-text">{component.package}</span>
        {/if}
      </div>
    </div>
  </div>

  <ComponentPreviewPanel
    selectedSymbolAsset={detail.selectedSymbolAsset}
    selectedFootprintAsset={detail.selectedFootprintAsset}
    selected3dModelAsset={detail.selected3dModelAsset}
  />

  <Tabs {tabs} bind:activeTab />

  <div class="tab-content">
    {#if activeTab === 'details'}
      <DetailsTab {component} {categories} {onupdated} />
    {:else if activeTab === 'inventory'}
      <InventoryTab {component} {onupdated} />
    {:else if activeTab === 'assets'}
      <AssetsTab
        componentId={component.id}
        {component}
        assets={detail.assets}
        selectedSymbolAsset={detail.selectedSymbolAsset}
        selectedFootprintAsset={detail.selectedFootprintAsset}
        selected3dModelAsset={detail.selected3dModelAsset}
        selectedDatasheetAsset={detail.selectedDatasheetAsset}
        {onupdated}
      />
    {/if}
  </div>

  <div class="detail-footer">
    {#if deleteError}
      <span class="error-text">{deleteError}</span>
    {/if}
    <button class="btn btn-danger btn-sm" onclick={handleDelete} disabled={deleting}>
      {deleting ? 'Deleting…' : 'Delete Component'}
    </button>
  </div>
</div>

<style>
  .detail-container {
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: hidden;
  }
  .detail-header {
    padding: 16px 20px;
    background: var(--color-bg-surface);
    border-bottom: 1px solid var(--color-border);
  }
  .header-info {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .header-mpn {
    font-size: 17px;
    font-weight: 600;
  }
  .header-meta {
    display: flex;
    align-items: center;
    gap: 10px;
    font-size: 12px;
    color: var(--color-text-secondary);
  }
  .meta-text {
    color: var(--color-text-secondary);
  }
  .tab-content {
    flex: 1;
    overflow-y: auto;
  }
  .detail-footer {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 12px;
    padding: 12px 20px;
    border-top: 1px solid var(--color-border);
    background: var(--color-bg-surface);
  }
  .error-text {
    font-size: 12px;
    color: var(--color-danger);
  }
  .btn-danger {
    border: 1px solid var(--color-danger);
    background: transparent;
    color: var(--color-danger);
    font-size: 12px;
    padding: 4px 12px;
    border-radius: 5px;
    cursor: pointer;
  }
  .btn-danger:hover:not(:disabled) {
    background: var(--color-danger-soft);
    color: var(--color-danger-text);
  }
  .btn-danger:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
