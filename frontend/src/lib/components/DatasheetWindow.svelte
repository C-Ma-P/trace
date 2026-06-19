<script lang="ts">
  import { onMount } from 'svelte';
  import { getComponentAsset, type ComponentAsset } from '../backend';
  import DatasheetPreview from './DatasheetPreview.svelte';

  let { assetId = null }: { assetId?: string | null } = $props();

  let asset = $state<ComponentAsset | null>(null);
  let loading = $state(false);
  let error = $state('');

  async function loadAsset(id: string): Promise<void> {
    loading = true;
    error = '';
    try {
      asset = await getComponentAsset(id);
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
      asset = null;
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    if (!assetId) {
      loading = false;
      error = 'No datasheet asset was specified for this window.';
    }
  });

  $effect(() => {
    if (assetId) {
      void loadAsset(assetId);
    }
  });
</script>

{#if loading}
  <div class="datasheet-window-state">
    <div class="spinner"></div>
    <div>Loading datasheet…</div>
  </div>
{:else if error}
  <div class="datasheet-window-state">
    <div class="state-title">Datasheet unavailable</div>
    <div class="state-message">{error}</div>
  </div>
{:else}
  <DatasheetPreview {asset} allowPopout={false} fullWindow={true} />
{/if}

<style>
  .datasheet-window-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 12px;
    min-height: 100vh;
    padding: 24px;
    background: var(--color-bg-app);
    color: var(--color-text-secondary);
    text-align: center;
  }
  .state-title {
    font-size: 16px;
    font-weight: 600;
    color: var(--color-text-primary);
  }
  .state-message {
    max-width: 460px;
    font-size: 13px;
    line-height: 1.5;
  }
  .spinner {
    width: 24px;
    height: 24px;
    border: 2px solid var(--color-border);
    border-top-color: var(--color-accent);
    border-radius: 999px;
    animation: spin 0.8s linear infinite;
  }
  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>