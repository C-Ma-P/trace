<script lang="ts">
  let { currentSection = $bindable() }: {
    currentSection: 'home' | 'components';
  } = $props();
  let collapsed = $state(false);
</script>

<aside class="sidebar" class:collapsed>
  <div class="sidebar-header">
    {#if !collapsed}
      <span class="sidebar-title">Trace</span>
    {/if}
  </div>
  <nav class="sidebar-nav">
    <button
      class="nav-item"
      class:active={currentSection === 'home'}
      title="Home"
      onclick={() => (currentSection = 'home')}
    >
      <svg viewBox="0 0 20 20" fill="currentColor" class="nav-icon">
        <path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
      </svg>
      {#if !collapsed}<span class="nav-label">Home</span>{/if}
    </button>
    <button
      class="nav-item"
      class:active={currentSection === 'components'}
      title="Components"
      onclick={() => (currentSection = 'components')}
    >
      <svg viewBox="0 0 20 20" fill="currentColor" class="nav-icon">
        <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
      </svg>
      {#if !collapsed}<span class="nav-label">Components</span>{/if}
    </button>
  </nav>

  <div class="sidebar-footer">
    <button
      class="collapse-btn"
      title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
      onclick={() => (collapsed = !collapsed)}
    >
      <span class="collapse-dots" class:rotated={collapsed}>⋮</span>
    </button>
  </div>
</aside>

<style>
  .sidebar {
    width: 200px;
    min-width: 200px;
    background: var(--color-bg-sidebar);
    display: flex;
    flex-direction: column;
    user-select: none;
    transition: width 0.18s ease, min-width 0.18s ease;
    overflow: hidden;
  }
  .sidebar.collapsed {
    width: 52px;
    min-width: 52px;
  }
  .sidebar.collapsed .sidebar-header {
    display: none;
  }
  .sidebar-header {
    padding: 16px;
    border-bottom: 1px solid var(--color-border);
    height: 53px;
    display: flex;
    align-items: center;
  }
  .sidebar-title {
    color: var(--color-text-primary);
    font-size: 14px;
    font-weight: 600;
    letter-spacing: -0.01em;
    white-space: nowrap;
    overflow: hidden;
  }
  .sidebar-nav {
    padding: 8px;
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1;
  }
  .nav-item {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 7px 10px;
    border-radius: var(--radius-sm);
    color: var(--color-text-secondary);
    font-size: 13px;
    font-weight: 500;
    text-align: left;
    transition: background 0.1s, color 0.1s;
    white-space: nowrap;
    overflow: hidden;
  }
  .sidebar.collapsed .nav-item {
    justify-content: center;
    padding: 8px 0;
  }
  .nav-item:hover {
    background: var(--color-bg-sidebar-hover);
    color: var(--color-text-primary);
  }
  .nav-item.active {
    background: var(--color-bg-sidebar-active);
    color: var(--color-text-primary);
    box-shadow: inset 2px 0 0 var(--color-accent);
  }
  .nav-icon {
    width: 16px;
    height: 16px;
    flex-shrink: 0;
  }
  .nav-label {
    overflow: hidden;
  }

  /* Footer / collapse button */
  .sidebar-footer {
    padding: 8px;
    border-top: 1px solid var(--color-border);
    display: flex;
    justify-content: flex-end;
  }
  .sidebar.collapsed .sidebar-footer {
    justify-content: center;
  }
  .collapse-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border-radius: var(--radius-md);
    color: var(--color-text-muted);
    transition: color 0.12s, background 0.12s;
  }
  .collapse-btn:hover {
    color: var(--color-text-primary);
    background: var(--color-bg-sidebar-hover);
  }
  .collapse-dots {
    font-size: 18px;
    line-height: 1;
    display: block;
    transform: rotate(0deg);
    transition: transform 0.18s ease;
  }
  .collapse-dots.rotated {
    transform: rotate(90deg);
  }
</style>
