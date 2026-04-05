<script lang="ts">
  import { onMount } from 'svelte';
  import { Splitpanes, Pane } from 'svelte-splitpanes';
  import ComponentsList from './ComponentsList.svelte';
  import ComponentDetail from './ComponentDetail.svelte';
  import NewComponentModal from './NewComponentModal.svelte';
  import { createComponentsWorkspaceStore } from './componentsWorkspaceStore';
  import {
    type Component,
    type ComponentFilter,
  } from '../backend';

  const {
    categories,
    components,
    selectedId,
    selectedDetail,
    loading,
    error,
    init,
    selectComponent,
    afterCreated,
    afterUpdated,
    afterDeleted,
    setFilterAndReload,
  } = createComponentsWorkspaceStore();

  let showCreateModal = $state(false);

  onMount(async () => {
    await init();
  });

  async function handleCreated(comp: Component) {
    showCreateModal = false;
    await afterCreated(comp);
  }

  async function handleUpdated() {
    await afterUpdated();
  }

  async function handleDeleted() {
    await afterDeleted();
  }

  async function handleFilterChange(f: Partial<ComponentFilter>) {
    await setFilterAndReload(f);
  }
</script>

<Splitpanes theme="" style="height: 100%">
  <Pane size={25} minSize={15} maxSize={45}>
    <div class="list-pane">
      <ComponentsList
        components={$components}
        categories={$categories}
        selectedId={$selectedId}
        loading={$loading}
        onselect={(id) => void selectComponent(id)}
        oncreate={() => (showCreateModal = true)}
        onfilter={handleFilterChange}
      />
    </div>
  </Pane>
  <Pane>
    <div class="detail-pane">
      {#if $error}
        <div class="error-banner">{$error}</div>
      {/if}
      {#key $selectedId}
      {#if $selectedDetail}
        <ComponentDetail
          detail={$selectedDetail}
          categories={$categories}
          onupdated={handleUpdated}
          ondeleted={handleDeleted}
        />
      {:else}
        <div class="empty-state">Select a component to view details</div>
      {/if}
      {/key}
    </div>
  </Pane>
</Splitpanes>

<NewComponentModal
  open={showCreateModal}
  categories={$categories}
  onclose={() => (showCreateModal = false)}
  oncreated={handleCreated}
/>

<style>
  .list-pane {
    height: 100%;
    display: flex;
    flex-direction: column;
    background: var(--color-bg-surface);
  }
  .detail-pane {
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
