<script lang="ts">
  type PageKey =
    | 'global-suppliers'
    | 'global-supplier-digikey'
    | 'global-supplier-mouser'
    | 'global-supplier-lcsc'
    | 'global-integration-kicad'
    | 'project-general'
    | 'project-sourcing';

  type NavigationNode = {
    id: string;
    label: string;
    hint?: string;
    key?: PageKey;
    children?: NavigationNode[];
    defaultExpanded?: boolean;
  };

  type NavigationGroup = {
    label: string;
    nodes: NavigationNode[];
  };

  let {
    groups,
    selectedPage = $bindable<PageKey>(),
  }: {
    groups: Array<{
      label: string;
      nodes: Array<unknown>;
    }>;
    selectedPage: PageKey;
  } = $props();

  const navigationGroups = $derived(groups as NavigationGroup[]);

  let expanded = $state<Record<string, boolean>>({});

  function hasSelectedDescendant(node: NavigationNode): boolean {
    if (node.key === selectedPage) {
      return true;
    }

    return node.children?.some((child) => hasSelectedDescendant(child)) ?? false;
  }

  function isExpanded(node: NavigationNode): boolean {
    return expanded[node.id] ?? node.defaultExpanded ?? hasSelectedDescendant(node);
  }

  function toggleNode(id: string) {
    expanded = {
      ...expanded,
      [id]: !(expanded[id] ?? true),
    };
  }
</script>

<aside class="preferences-sidebar">
  <div class="sidebar-header">
    <span class="sidebar-title">Trace</span>
    <span class="sidebar-subtitle">Preferences</span>
  </div>

  <nav class="sidebar-groups" aria-label="Preferences sections">
    {#each navigationGroups as group}
      <section class="sidebar-group">
        <h3>{group.label}</h3>
        <div class="sidebar-tree">
          {#snippet renderNode(node: NavigationNode, depth: number)}
            <div class="tree-node">
              <div class="tree-row" style={`--depth:${depth};`}>
                {#if node.children?.length}
                  <button
                    type="button"
                    class="tree-toggle"
                    aria-label={isExpanded(node) ? `Collapse ${node.label}` : `Expand ${node.label}`}
                    aria-expanded={isExpanded(node)}
                    onclick={() => toggleNode(node.id)}
                  >
                    <span class:expanded={isExpanded(node)}>▸</span>
                  </button>
                {:else}
                  <span class="tree-toggle-spacer"></span>
                {/if}

                {#if node.key}
                  <button
                    type="button"
                    class:selected={selectedPage === node.key}
                    class="tree-item"
                    onclick={() => (selectedPage = node.key!)}
                  >
                    <span class="tree-item-label">{node.label}</span>
                    {#if node.hint}
                      <span class="tree-item-hint">{node.hint}</span>
                    {/if}
                  </button>
                {:else}
                  <button
                    type="button"
                    class="tree-item tree-item-group"
                    onclick={() => toggleNode(node.id)}
                  >
                    <span class="tree-item-label">{node.label}</span>
                    {#if node.hint}
                      <span class="tree-item-hint">{node.hint}</span>
                    {/if}
                  </button>
                {/if}
              </div>

              {#if node.children?.length && isExpanded(node)}
                <div class="tree-children">
                  {#each node.children as child (child.id)}
                    {@render renderNode(child, depth + 1)}
                  {/each}
                </div>
              {/if}
            </div>
          {/snippet}

          {#each group.nodes as node (node.id)}
            {@render renderNode(node, 0)}
          {/each}
        </div>
      </section>
    {/each}
  </nav>
</aside>

<style>
  .preferences-sidebar {
    display: flex;
    flex-direction: column;
    min-width: 0;
    background: var(--color-bg-sidebar);
    border-right: 1px solid var(--color-border);
  }

  .sidebar-header {
    padding: 16px;
    border-bottom: 1px solid var(--color-border);
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .sidebar-title {
    color: var(--color-text-primary);
    font-size: 14px;
    font-weight: 600;
    letter-spacing: -0.01em;
  }

  .sidebar-subtitle {
    color: var(--color-text-secondary);
    font-size: 12px;
  }

  .sidebar-groups {
    display: flex;
    flex-direction: column;
    gap: 14px;
    padding: 10px 8px 12px;
    overflow: auto;
  }

  .sidebar-group {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .sidebar-group h3 {
    padding: 0 8px;
    font-size: 11px;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--color-text-muted);
  }

  .sidebar-tree {
    display: flex;
    flex-direction: column;
  }

  .tree-node,
  .tree-children {
    display: flex;
    flex-direction: column;
  }

  .tree-row {
    --depth: 0;
    display: grid;
    grid-template-columns: 14px minmax(0, 1fr);
    gap: 6px;
    align-items: start;
    padding-left: calc(var(--depth) * 14px);
  }

  .tree-toggle,
  .tree-toggle-spacer {
    width: 14px;
    min-width: 14px;
    height: 28px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    color: var(--color-text-muted);
    font-size: 10px;
  }

  .tree-toggle span {
    transition: transform 0.14s ease;
  }

  .tree-toggle span.expanded {
    transform: rotate(90deg);
  }

  .tree-item {
    width: 100%;
    text-align: left;
    border-left: 3px solid transparent;
    padding: 6px 10px 6px 9px;
    display: flex;
    flex-direction: column;
    gap: 2px;
    color: var(--color-text-secondary);
    transition: background 0.12s ease, color 0.12s ease;
  }

  .tree-item:hover,
  .tree-toggle:hover + .tree-item,
  .tree-item-group:hover {
    background: var(--color-bg-sidebar-hover);
    color: var(--color-text-primary);
  }

  .tree-item.selected {
    background: var(--color-bg-selected);
    border-left-color: var(--color-accent);
    color: var(--color-text-primary);
  }

  .tree-item-group {
    color: var(--color-text-primary);
  }

  .tree-item-label {
    font-size: 13px;
    font-weight: 500;
  }

  .tree-item-hint {
    font-size: 11px;
    line-height: 1.35;
    color: inherit;
    opacity: 0.8;
  }

  @media (max-width: 860px) {
    .preferences-sidebar {
      border-right: none;
      border-bottom: 1px solid var(--color-border);
      max-height: 280px;
    }
  }
</style>