<script lang="ts">
  import type { Snippet } from 'svelte';
  import PreferencesSidebar from './PreferencesSidebar.svelte';

  type PageKey =
    | 'global-suppliers'
    | 'global-supplier-digikey'
    | 'global-supplier-mouser'
    | 'global-supplier-lcsc'
    | 'global-integrations'
    | 'global-integration-kicad'
    | 'project-general'
    | 'project-sourcing';

  let {
    groups,
    selectedPage = $bindable<PageKey>(),
    pageTitle,
    pageDescription,
    pageScope,
    pagePath,
    children,
  }: {
    groups: Array<{
      label: string;
      nodes: Array<{
        id: string;
        label: string;
        hint?: string;
        key?: PageKey;
        children?: Array<unknown>;
        defaultExpanded?: boolean;
      }>;
    }>;
    selectedPage: PageKey;
    pageTitle: string;
    pageDescription: string;
    pageScope: string;
    pagePath: string[];
    children?: Snippet;
  } = $props();
</script>

<div class="preferences-shell">
  <PreferencesSidebar {groups} bind:selectedPage />

  <main class="preferences-detail">
    <header class="detail-header">
      <div class="detail-heading">
        <p class="detail-scope">{pageScope}</p>
        <h1>{pageTitle}</h1>
        <p>{pageDescription}</p>
      </div>
      <div class="detail-path" aria-label="Selected preferences path">
        {#each pagePath as segment, index (segment)}
          <span>{segment}</span>
          {#if index < pagePath.length - 1}
            <span class="detail-path-separator">/</span>
          {/if}
        {/each}
      </div>
    </header>

    <section class="detail-body">
      <div class="detail-content">
        {@render children?.()}
      </div>
    </section>
  </main>
</div>

<style>
  .preferences-shell {
    display: grid;
    grid-template-columns: 260px minmax(0, 1fr);
    height: 100%;
    background: var(--color-bg-app);
  }

  .preferences-detail {
    min-width: 0;
    display: flex;
    flex-direction: column;
    background: var(--color-bg-app);
  }

  .detail-header {
    display: flex;
    justify-content: space-between;
    gap: 20px;
    align-items: flex-start;
    padding: 16px 20px;
    border-bottom: 1px solid var(--color-border);
    background: var(--color-bg-surface);
  }

  .detail-scope {
    margin-bottom: 6px;
    font-size: 11px;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--color-text-muted);
  }

  h1 {
    font-size: 18px;
    font-weight: 600;
  }

  .detail-heading p:last-child {
    margin-top: 6px;
    max-width: 640px;
    color: var(--color-text-secondary);
    line-height: 1.45;
  }

  .detail-path {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    align-items: center;
    justify-content: flex-end;
    min-height: 20px;
    color: var(--color-text-muted);
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.08em;
  }

  .detail-path-separator {
    color: var(--color-text-secondary);
  }

  .detail-body {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: 20px;
  }

  .detail-content {
    max-width: 880px;
  }

  @media (max-width: 860px) {
    .preferences-shell {
      grid-template-columns: 1fr;
      grid-template-rows: auto 1fr;
    }

    .detail-header {
      flex-direction: column;
      align-items: stretch;
    }

    .detail-path {
      justify-content: flex-start;
    }
  }
</style>