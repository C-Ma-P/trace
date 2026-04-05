<script lang="ts">
  import { onMount } from 'svelte';
  import ProjectDetail from './ProjectDetail.svelte';
  import { createProjectsWorkspaceStore } from './projectsWorkspaceStore';

  let {
    requestedProjectId = null,
    onRequestedProjectConsumed,
  }: {
    requestedProjectId?: string | null;
    onRequestedProjectConsumed?: () => void;
  } = $props();

  const {
    categories,
    selectedProject,
    loading,
    error,
    init,
    loadProject,
  } = createProjectsWorkspaceStore();

  let lastRequestedProjectId: string | null = $state(null);

  $effect(() => {
    if (requestedProjectId && requestedProjectId !== lastRequestedProjectId) {
      lastRequestedProjectId = requestedProjectId;
      void loadProject(requestedProjectId);
      onRequestedProjectConsumed?.();
    }
  });


  onMount(async () => {
    await init(requestedProjectId);
  });

  async function handleUpdated() {
    if (requestedProjectId) {
      await loadProject(requestedProjectId);
    }
  }
</script>

<div class="home">
  {#if $error}
    <div class="error-banner">{$error}</div>
  {/if}
  {#if !requestedProjectId}
    <div class="empty-state">No project is open</div>
  {:else if $loading && !$selectedProject}
    <div class="empty-state">Loading…</div>
  {:else if $selectedProject}
    <ProjectDetail
      project={$selectedProject}
      categories={$categories}
      onupdated={handleUpdated}
    />
  {:else}
    <div class="empty-state">Project not found</div>
  {/if}
</div>

<style>
  .home {
    height: 100%;
    display: flex;
    flex-direction: column;
    background: var(--color-bg-app);
  }
  .error-banner {
    padding: 8px 16px;
    background: var(--color-danger-soft);
    color: var(--color-danger-text);
    font-size: 12px;
    border-bottom: 1px solid var(--color-danger-border);
  }
</style>
