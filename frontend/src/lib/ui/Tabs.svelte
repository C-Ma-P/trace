<script lang="ts">
  let { tabs, activeTab = $bindable() }: {
    tabs: { key: string; label: string }[];
    activeTab: string;
  } = $props();
</script>

<div class="tab-bar">
  {#each tabs as tab}
    <button
      class="tab"
      class:active={activeTab === tab.key}
      onclick={() => (activeTab = tab.key)}
    >
      {tab.label}
    </button>
  {/each}
</div>

<style>
  .tab-bar {
    display: flex;
    gap: 0;
    border-bottom: 1px solid var(--color-border);
    padding: 0 16px;
    background: var(--color-bg-surface);
  }
  .tab {
    padding: 8px 14px;
    font-size: 12px;
    font-weight: 500;
    color: var(--color-text-secondary);
    border-bottom: 2px solid transparent;
    margin-bottom: -1px;
    transition: color 0.1s, border-color 0.1s;
  }
  .tab:hover {
    color: var(--color-text-primary);
  }
  .tab.active {
    color: var(--color-accent);
    border-bottom-color: var(--color-accent);
  }
</style>
