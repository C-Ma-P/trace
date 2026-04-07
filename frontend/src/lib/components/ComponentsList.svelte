<script lang="ts">
  import {
    quantityDisplay,
    categoryDisplayName,
    type Component,
    type CategoryInfo,
    type ComponentFilter,
  } from '../backend';

  let { components = [], categories = [], selectedId = null, loading = false, onselect, oncreate, onfilter }: {
    components?: Component[];
    categories?: CategoryInfo[];
    selectedId?: string | null;
    loading?: boolean;
    onselect?: (id: string) => void;
    oncreate?: () => void;
    onfilter?: (f: Partial<ComponentFilter>) => void;
  } = $props();

  let searchText = $state('');
  let filterCategory = $state('');
  let debounceTimer: ReturnType<typeof setTimeout>;

  function handleSearch() {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => {
      emitFilter();
    }, 250);
  }

  function emitFilter() {
    const f: Partial<ComponentFilter> = {};
    if (searchText) f.text = searchText;
    if (filterCategory) f.category = filterCategory;
    onfilter?.(f);
  }

  function handleCategoryChange() {
    emitFilter();
  }
</script>

<div class="list-header">
  <div class="list-title-row">
    <h2 class="list-title">Components</h2>
    <button class="btn btn-primary btn-sm" onclick={() => oncreate?.()}>
      + New
    </button>
  </div>
  <div class="filter-row">
    <input
      class="form-input search-input"
      type="text"
      placeholder="Search MPN, manufacturer..."
      bind:value={searchText}
      oninput={handleSearch}
    />
  </div>
  <div class="filter-row">
    <select
      class="form-input filter-select"
      bind:value={filterCategory}
      onchange={handleCategoryChange}
    >
      <option value="">All categories</option>
      {#each categories as cat}
        <option value={cat.value}>{cat.displayName}</option>
      {/each}
    </select>
  </div>
</div>

<div class="list-body">
  {#if loading}
    <div class="empty-state">Loading…</div>
  {:else if components.length === 0}
    <div class="empty-state">No components found</div>
  {:else}
    {#each components as comp}
      <button
        class="list-item"
        class:selected={selectedId === comp.id}
        onclick={() => onselect?.(comp.id)}
      >
        <div class="item-thumb">
          {#if comp.imageUrl}
            <img src={comp.imageUrl} alt="" class="thumb-img" />
          {:else}
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">
              <rect x="6" y="6" width="12" height="12" rx="1"/>
              <line x1="6" y1="10" x2="3" y2="10"/>
              <line x1="6" y1="14" x2="3" y2="14"/>
              <line x1="18" y1="10" x2="21" y2="10"/>
              <line x1="18" y1="14" x2="21" y2="14"/>
              <line x1="10" y1="6" x2="10" y2="3"/>
              <line x1="14" y1="6" x2="14" y2="3"/>
              <line x1="10" y1="18" x2="10" y2="21"/>
              <line x1="14" y1="18" x2="14" y2="21"/>
            </svg>
          {/if}
        </div>
        <div class="item-content">
          <div class="item-main">
            <span class="item-mpn">{comp.mpn || '—'}</span>
            <span class="item-mfr">{comp.manufacturer || '—'}</span>
          </div>
          <div class="item-meta">
            <span class="badge">{categoryDisplayName(categories, comp.category)}</span>
            {#if comp.package}
              <span class="item-pkg">{comp.package}</span>
            {/if}
            <span class="item-qty">{quantityDisplay(comp)} pcs</span>
          </div>
        </div>
      </button>
    {/each}
  {/if}
</div>

<style>
  .list-header {
    padding: 12px;
    border-bottom: 1px solid var(--color-border);
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .list-title-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  .list-title {
    font-size: 15px;
    font-weight: 600;
  }
  .filter-row {
    display: flex;
    gap: 8px;
  }
  .search-input,
  .filter-select {
    width: 100%;
    font-size: 12px;
    padding: 5px 8px;
  }
  .list-body {
    flex: 1;
    overflow-y: auto;
  }
  .list-item {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 8px 12px;
    text-align: left;
    border-bottom: 1px solid var(--color-border);
    transition: background 0.1s;
  }
  .item-thumb {
    flex-shrink: 0;
    width: 30px;
    height: 30px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--radius-md);
    background: var(--color-bg-muted);
    border: 1px solid var(--color-border);
    color: var(--color-text-muted);
    overflow: hidden;
  }
  .thumb-img {
    width: 100%;
    height: 100%;
    object-fit: contain;
  }
  .item-content {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .list-item:hover {
    background: var(--color-bg-hover);
  }
  .list-item.selected {
    background: var(--color-bg-selected);
    border-left: 3px solid var(--color-accent);
    padding-left: 9px;
  }
  .item-main {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
  }
  .item-mpn {
    font-weight: 600;
    font-size: 13px;
    color: var(--color-text-primary);
  }
  .item-mfr {
    font-size: 12px;
    color: var(--color-text-secondary);
  }
  .item-meta {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 11px;
    color: var(--color-text-muted);
  }
  .item-pkg {
    font-family: var(--font-mono);
    font-size: 11px;
  }
  .item-qty {
    margin-left: auto;
  }
</style>
