<script lang="ts">
  import { formatDate, type Project } from '../backend';

  let { projects = [], selectedId = null, loading = false, onselect }: {
    projects?: Project[];
    selectedId?: string | null;
    loading?: boolean;
    onselect?: (id: string) => void;
  } = $props();
</script>

<div class="list-header">
  <div class="list-title-row">
    <h2 class="list-title">Projects</h2>
  </div>
</div>

<div class="list-body">
  {#if loading}
    <div class="empty-state">Loading…</div>
  {:else if projects.length === 0}
    <div class="empty-state">No projects yet</div>
  {:else}
    {#each projects as project}
      <button
        class="list-item"
        class:selected={selectedId === project.id}
        onclick={() => onselect?.(project.id)}
      >
        <div class="item-name">{project.name}</div>
        <div class="item-meta">
          <span>{project.requirements.length} requirement{project.requirements.length !== 1 ? 's' : ''}</span>
          <span class="item-date">{formatDate(project.createdAt)}</span>
        </div>
      </button>
    {/each}
  {/if}
</div>

<style>
  .list-header {
    padding: 12px;
    border-bottom: 1px solid var(--color-border);
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
  .list-body {
    flex: 1;
    overflow-y: auto;
  }
  .list-item {
    display: flex;
    flex-direction: column;
    gap: 4px;
    width: 100%;
    padding: 10px 9px;
    text-align: left;
    border-left: 3px solid transparent;
    border-bottom: 1px solid var(--color-border);
    transition:
      background var(--motion-fast) var(--easing-standard),
      border-color var(--motion-fast) var(--easing-standard),
      transform var(--motion-fast) var(--easing-standard);
    transform: translateX(0);
  }
  .list-item:hover {
    background: var(--color-bg-hover);
    transform: translateX(1px);
  }
  .list-item.selected {
    background: var(--color-bg-selected);
    border-left-color: var(--color-accent);
    transform: translateX(0);
  }
  .item-name {
    font-weight: 600;
    font-size: 13px;
    color: var(--color-text-primary);
  }
  .item-meta {
    display: flex;
    justify-content: space-between;
    font-size: 11px;
    color: var(--color-text-muted);
  }
  .item-date {
    color: var(--color-text-muted);
  }
</style>
