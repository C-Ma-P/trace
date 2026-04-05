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
    flex-direction: column;
    gap: 4px;
    width: 100%;
    padding: 10px 12px;
    text-align: left;
    border-bottom: 1px solid var(--color-border);
    transition: background 0.1s;
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
