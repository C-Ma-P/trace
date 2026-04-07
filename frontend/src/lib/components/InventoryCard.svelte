<script lang="ts">
  import {
    updateComponentInventory,
    adjustComponentQuantity,
    type Component,
  } from '../backend';

  let { component, onupdated }: {
    component: Component;
    onupdated?: () => void;
  } = $props();

  let editing = $state(false);
  let saving = $state(false);
  let error = $state('');

  let editMode = $state('unknown');
  let editQuantity: number | null = $state(null);
  let editLocation = $state('');

  function startEdit() {
    editMode = component.quantityMode ?? 'unknown';
    editQuantity = component.quantity;
    editLocation = component.location ?? '';
    error = '';
    editing = true;
  }

  function cancelEdit() {
    editing = false;
    error = '';
  }

  async function save() {
    saving = true;
    error = '';
    try {
      await updateComponentInventory({
        id: component.id,
        quantity: editMode === 'unknown' ? null : editQuantity,
        quantityMode: editMode,
        location: editLocation,
      });
      editing = false;
      onupdated?.();
    } catch (e: any) {
      error = e?.message ?? String(e);
    } finally {
      saving = false;
    }
  }

  async function adjust(delta: number) {
    error = '';
    try {
      await adjustComponentQuantity(component.id, delta);
      onupdated?.();
    } catch (e: any) {
      error = e?.message ?? String(e);
    }
  }

  let canAdjust = $derived(!editing && component.quantityMode !== 'unknown');
  let displayQty = $derived(
    component.quantityMode === 'unknown' || component.quantity === null
      ? '?'
      : component.quantityMode === 'approximate'
        ? `~${component.quantity}`
        : String(component.quantity)
  );
</script>

<section class="inv-section">
  <div class="section-header">
    <h3 class="section-title">Inventory</h3>
    {#if !editing}
      <button class="btn btn-ghost btn-sm" onclick={startEdit}>Edit</button>
    {:else}
      <div class="edit-actions">
        <button class="btn btn-secondary btn-sm" onclick={cancelEdit} disabled={saving}>Cancel</button>
        <button class="btn btn-primary btn-sm" onclick={save} disabled={saving}>
          {saving ? 'Saving…' : 'Save'}
        </button>
      </div>
    {/if}
  </div>

  {#if error}
    <div class="error-text">{error}</div>
  {/if}

  {#if editing}
    <div class="inv-edit-grid">
      <div class="form-group">
        <label for="inv-mode">Quantity Mode</label>
        <select id="inv-mode" class="form-input" bind:value={editMode}>
          <option value="exact">Exact</option>
          <option value="approximate">Approximate</option>
          <option value="unknown">Unknown</option>
        </select>
      </div>
      {#if editMode !== 'unknown'}
        <div class="form-group">
          <label for="inv-qty">Quantity</label>
          <input id="inv-qty" class="form-input" type="number" min="0" bind:value={editQuantity} placeholder="0" />
        </div>
      {/if}
      <div class="form-group" class:full-width={editMode === 'unknown'}>
        <label for="inv-loc">Location</label>
        <input id="inv-loc" class="form-input" type="text" bind:value={editLocation} placeholder="e.g. Drawer A1, Bin 3" />
      </div>
    </div>
  {:else}
    <div class="inv-display">
      <div class="inv-stat">
        <span class="inv-qty">{displayQty}</span>
        <span class="inv-unit">pcs</span>
        {#if component.quantityMode === 'approximate'}
          <span class="mode-badge approx">approx</span>
        {:else if component.quantityMode === 'unknown'}
          <span class="mode-badge unknown">unknown</span>
        {/if}
      </div>
      {#if component.location}
        <div class="inv-location">
          <span class="inv-location-label">Location:</span>
          <span>{component.location}</span>
        </div>
      {/if}
      {#if canAdjust}
        <div class="inv-adjust">
          <button class="btn btn-ghost btn-sm adj-btn" onclick={() => adjust(-10)}>−10</button>
          <button class="btn btn-ghost btn-sm adj-btn" onclick={() => adjust(-1)}>−1</button>
          <button class="btn btn-ghost btn-sm adj-btn" onclick={() => adjust(1)}>+1</button>
          <button class="btn btn-ghost btn-sm adj-btn" onclick={() => adjust(10)}>+10</button>
        </div>
      {/if}
    </div>
  {/if}
</section>

<style>
  .inv-section {
    padding: 14px 20px;
    border-bottom: 1px solid var(--color-border);
    background: var(--color-bg-surface);
  }
  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 10px;
  }
  .section-title {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--color-text-secondary);
  }
  .edit-actions {
    display: flex;
    gap: 6px;
  }
  .inv-display {
    display: flex;
    align-items: center;
    gap: 16px;
    flex-wrap: wrap;
  }
  .inv-stat {
    display: flex;
    align-items: baseline;
    gap: 4px;
  }
  .inv-qty {
    font-size: 20px;
    font-weight: 600;
    font-variant-numeric: tabular-nums;
    color: var(--color-text-primary);
  }
  .inv-unit {
    font-size: 12px;
    color: var(--color-text-secondary);
  }
  .mode-badge {
    font-size: 10px;
    font-weight: 500;
    padding: 1px 5px;
    border-radius: 3px;
    margin-left: 2px;
  }
  .mode-badge.approx {
    background: var(--color-warning-soft);
    color: var(--color-warning-text);
  }
  .mode-badge.unknown {
    background: var(--color-bg-muted);
    color: var(--color-text-muted);
  }
  .inv-location {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 12px;
    color: var(--color-text-secondary);
  }
  .inv-location-label {
    color: var(--color-text-muted);
  }
  .inv-adjust {
    display: flex;
    gap: 3px;
    margin-left: auto;
  }
  .adj-btn {
    font-variant-numeric: tabular-nums;
    min-width: 36px;
    font-size: 12px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
  }
  .inv-edit-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }
  .full-width {
    grid-column: 1 / -1;
  }
  .error-text {
    font-size: 12px;
    color: var(--color-danger);
    margin-bottom: 8px;
  }
</style>
