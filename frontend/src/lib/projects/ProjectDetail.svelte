<script lang="ts">
  import Tabs from '../ui/Tabs.svelte';
  import RequirementsTab from './RequirementsTab.svelte';
  import PlanTab from './PlanTab.svelte';
  import FinalizeTab from './FinalizeTab.svelte';
  import { type Project, type CategoryInfo } from '../backend';

  let { project, categories = [], onupdated }: {
    project: Project;
    categories?: CategoryInfo[];
    onupdated?: () => void;
  } = $props();

  let activeTab = $state('requirements');

  const tabs = [
    { key: 'requirements', label: 'Requirements' },
    { key: 'plan', label: 'Plan' },
    { key: 'finalize', label: 'Finalize' },
  ];
</script>

<div class="detail-container">
  <div class="detail-header">
    <div class="header-info">
      <h2 class="header-name">{project.name}</h2>
      {#if project.description}
        <p class="header-desc">{project.description}</p>
      {/if}
    </div>
  </div>

  <Tabs {tabs} bind:activeTab />

  <div class="tab-content">
    {#if activeTab === 'requirements'}
      <RequirementsTab {project} {categories} {onupdated} />
    {:else if activeTab === 'plan'}
      <PlanTab {project} {categories} {onupdated} />
    {:else if activeTab === 'finalize'}
      <FinalizeTab {project} {categories} {onupdated} />
    {/if}
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
    gap: 4px;
  }
  .header-name {
    font-size: 17px;
    font-weight: 600;
  }
  .header-desc {
    font-size: 13px;
    color: var(--color-text-secondary);
  }
  .tab-content {
    flex: 1;
    overflow-y: auto;
  }
</style>
