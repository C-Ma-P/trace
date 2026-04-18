<script lang="ts">
  import type { Snippet } from 'svelte';
  import { fade, scale } from 'svelte/transition';

  let { open = false, title = '', width = '480px', onclose, children }: {
    open?: boolean;
    title?: string;
    width?: string;
    onclose?: () => void;
    children?: Snippet;
  } = $props();

  function handleBackdrop() {
    onclose?.();
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') onclose?.();
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="modal-backdrop" onclick={handleBackdrop} transition:fade={{ duration: 120 }}>
    <div
      class="modal-content"
      style="max-width: {width}"
      onclick={(e) => e.stopPropagation()}
      transition:scale={{ duration: 180, start: 0.97 }}
    >
      <div class="modal-header">
        <h3 class="modal-title">{title}</h3>
        <button class="modal-close" onclick={() => onclose?.()}>✕</button>
      </div>
      <div class="modal-body">
        {#if children}{@render children()}{/if}
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: var(--color-backdrop);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }
  .modal-content {
    background: var(--color-bg-surface-elevated);
    border: 1px solid var(--color-border-strong);
    border-radius: var(--radius-sm);
    box-shadow: var(--shadow-lg);
    width: 100%;
    max-height: 85vh;
    display: flex;
    flex-direction: column;
  }
  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 16px;
    border-bottom: 1px solid var(--color-border);
    background: var(--color-bg-surface);
  }
  .modal-title {
    font-size: 13px;
    font-weight: 600;
    letter-spacing: 0.01em;
  }
  .modal-close {
    color: var(--color-text-muted);
    font-size: 14px;
    padding: 3px 6px;
    border-radius: var(--radius-sm);
    transition:
      background var(--motion-fast) var(--easing-standard),
      color var(--motion-fast) var(--easing-standard);
  }
  .modal-close:hover {
    background: var(--color-bg-hover);
    color: var(--color-text-primary);
  }
  .modal-body {
    padding: 16px;
    overflow-y: auto;
  }
</style>
