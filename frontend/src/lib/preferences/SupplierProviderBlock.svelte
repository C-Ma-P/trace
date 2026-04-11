<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { SupplierProviderConfig } from '../backend';

  let {
    title,
    description,
    status,
    enabled = $bindable<boolean>(),
    storageText,
    sourceText,
    secretText,
    message,
    children,
  }: {
    title: string;
    description: string;
    status: SupplierProviderConfig | null;
    enabled: boolean;
    storageText: string;
    sourceText: string;
    secretText: string;
    message: string;
    children?: Snippet;
  } = $props();

  function stateLabel(state: SupplierProviderConfig | null): string {
    if (!state) return 'Loading';
    if (state.state === 'configured') return 'Configured';
    if (state.state === 'incomplete') return 'Incomplete';
    return 'Disabled';
  }

  function stateClass(state: SupplierProviderConfig | null): string {
    if (!state) return 'badge';
    if (state.state === 'configured') return 'badge badge-success';
    if (state.state === 'incomplete') return 'badge badge-warning';
    return 'badge';
  }

  function messageClass(state: SupplierProviderConfig | null): string {
    if (!state) return 'provider-message';
    if (state.state === 'configured') return 'provider-message';
    return 'provider-message provider-message-warning';
  }
</script>

<section class="provider-block">
  <header class="provider-header">
    <div class="provider-header-main">
      <div class="provider-title-row">
        <h2>{title}</h2>
        <span class={stateClass(status)}>{stateLabel(status)}</span>
      </div>
      <p>{description}</p>
    </div>

    <label class="provider-toggle" for={`toggle-${title}`}>
      <input id={`toggle-${title}`} type="checkbox" bind:checked={enabled} />
      <span>Enabled</span>
    </label>
  </header>

  <div class="provider-meta">
    <div class="provider-meta-row">
      <span class="meta-label">Storage</span>
      <span>{storageText}</span>
    </div>
    <div class="provider-meta-row">
      <span class="meta-label">Source</span>
      <span>{sourceText}</span>
    </div>
    <div class="provider-meta-row">
      <span class="meta-label">Secret</span>
      <span>{secretText}</span>
    </div>
  </div>

  {#if message}
    <p class={messageClass(status)}>{message}</p>
  {/if}

  <div class="provider-fields">
    {@render children?.()}
  </div>
</section>

<style>
  .provider-block {
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    background: var(--color-bg-surface);
    display: flex;
    flex-direction: column;
  }

  .provider-header {
    display: flex;
    justify-content: space-between;
    gap: 16px;
    align-items: flex-start;
    padding: 14px 16px;
    border-bottom: 1px solid var(--color-border);
  }

  .provider-header-main {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .provider-title-row {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  h2 {
    font-size: 14px;
    font-weight: 600;
  }

  .provider-header p {
    max-width: 620px;
    color: var(--color-text-secondary);
    line-height: 1.45;
  }

  .provider-toggle {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    color: var(--color-text-secondary);
    white-space: nowrap;
  }

  .provider-toggle input {
    accent-color: var(--color-accent);
  }

  .provider-meta {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    border-bottom: 1px solid var(--color-border);
    background: rgba(255, 255, 255, 0.015);
  }

  .provider-meta-row {
    padding: 10px 16px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .provider-meta-row + .provider-meta-row {
    border-left: 1px solid var(--color-border);
  }

  .meta-label {
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.12em;
    color: var(--color-text-muted);
  }

  .provider-message {
    padding: 10px 16px 0;
    color: var(--color-text-secondary);
    line-height: 1.45;
  }
  .provider-message-warning {
    color: var(--color-warning-text);
  }

  .provider-fields {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 14px 16px 16px;
  }

  @media (max-width: 720px) {
    .provider-header {
      flex-direction: column;
    }

    .provider-meta {
      grid-template-columns: 1fr;
    }

    .provider-meta-row + .provider-meta-row {
      border-left: none;
      border-top: 1px solid var(--color-border);
    }
  }
</style>