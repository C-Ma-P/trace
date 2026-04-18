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
    position: relative;
    padding: 8px 14px;
    font-size: 12px;
    font-weight: 500;
    color: var(--color-text-secondary);
    transition:
      color var(--motion-fast) var(--easing-standard),
      background var(--motion-fast) var(--easing-standard);
  }
  .tab:hover {
    color: var(--color-text-primary);
    background: rgba(255, 255, 255, 0.03);
  }
  .tab::after {
    content: '';
    position: absolute;
    left: 10px;
    right: 10px;
    bottom: -2px;
    height: 2px;
    border-radius: 999px;
    background: var(--color-accent);
    transform: scaleX(0);
    transform-origin: center;
  }
  @media (prefers-reduced-motion: no-preference) {
    .tab::after {
      transition: transform var(--motion-normal) var(--easing-standard);
    }
  }
  .tab.active {
    color: var(--color-accent);
  }
  .tab.active::after {
    transform: scaleX(1);
  }
</style>
