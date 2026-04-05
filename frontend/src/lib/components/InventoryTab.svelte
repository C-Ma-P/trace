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
      const qty = editMode === 'unknown' ? null : editQuantity;
      await updateComponentInventory({
        id: component.id,
        quantity: qty,
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

<div class="inventory-tab">
  <div class="section-header">
    <h3 class="section-title">Inventory</h3>
    {#if !editing}
      <button class="btn btn-secondary btn-sm" onclick={startEdit}>Edit</button>
    {:else}
      <div class="edit-actions">
        <button class="btn btn-secondary btn-sm" onclick={cancelEdit} disabled={saving}>
          Cancel
        </button>
        <button class="btn btn-primary btn-sm" onclick={save} disabled={saving}>
          {saving ? 'Saving…' : 'Save'}
        </button>
      </div>
    {/if}
  </div>

  {#if error}
    <div class="error-text" style="margin-bottom: 12px;">{error}</div>
  {/if}

  {#if editing}
    <div class="fields-grid">
      <div class="form-group">
        <label>Quantity Mode</label>
        <select class="form-input" bind:value={editMode}>
          <option value="exact">Exact — counted precisely</option>
          <option value="approximate">Approximate — rough estimate</option>
          <option value="unknown">Unknown — not yet counted</option>
        </select>
      </div>

      {#if editMode !== 'unknown'}
        <div class="form-group">
          <label>Quantity</label>
          <input
            class="form-input"
            type="number"
            min="0"
            bind:value={editQuantity}
            placeholder="0"
          />
        </div>
      {/if}

      <div class="form-group full-width">
        <label>Location</label>
        <input
          class="form-input"
          type="text"
          bind:value={editLocation}
          placeholder="e.g. Drawer A1, Bin 3, Blue box"
        />
      </div>
    </div>
  {:else}
    <div class="fields-grid">
      <div class="form-group">
        <label>Quantity</label>
        <span class="field-value qty-value">
          {displayQty}
          {#if component.quantityMode === 'approximate'}
            <span class="mode-badge approx">approx</span>
          {:else if component.quantityMode === 'unknown'}
            <span class="mode-badge unknown">unknown</span>
          {/if}
        </span>
      </div>

      <div class="form-group">
        <label>Location</label>
        <span class="field-value">{component.location || '—'}</span>
      </div>
    </div>

    {#if canAdjust}
      <div class="adjust-row">
        <span class="adjust-label">Quick adjust</span>
        <div class="adjust-buttons">
          <button class="btn btn-ghost btn-sm adjust-btn" onclick={() => adjust(-10)}>−10</button>
          <button class="btn btn-ghost btn-sm adjust-btn" onclick={() => adjust(-1)}>−1</button>
          <button class="btn btn-ghost btn-sm adjust-btn" onclick={() => adjust(1)}>+1</button>
          <button class="btn btn-ghost btn-sm adjust-btn" onclick={() => adjust(10)}>+10</button>
        </div>
      </div>
    {/if}
  {/if}
</div>

<style>
  .inventory-tab {
    padding: 20px;
  }
  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 16px;
  }
  .section-title {
    font-size: 14px;
    font-weight: 600;
    color: var(--color-text-primary);
  }
  .edit-actions {
    display: flex;
    gap: 8px;
  }
  .fields-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
    margin-bottom: 20px;
  }
  .full-width {
    grid-column: 1 / -1;
  }
  .field-value {
    font-size: 13px;
    color: var(--color-text-primary);
    padding: 6px 0;
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .qty-value {
    font-size: 20px;
    font-weight: 600;
    font-variant-numeric: tabular-nums;
  }
  .mode-badge {
    font-size: 11px;
    font-weight: 500;
    padding: 2px 6px;
    border-radius: 4px;
  }
  .mode-badge.approx {
    background: var(--color-warning-soft);
    color: var(--color-warning-text);
  }
  .mode-badge.unknown {
    background: var(--color-bg-muted);
    color: var(--color-text-muted);
  }
  .adjust-row {
    display: flex;
    align-items: center;
    gap: 12px;
    border-top: 1px solid var(--color-border);
    padding-top: 16px;
  }
  .adjust-label {
    font-size: 12px;
    color: var(--color-text-secondary);
    white-space: nowrap;
  }
  .adjust-buttons {
    display: flex;
    gap: 4px;
  }
  .adjust-btn {
    font-variant-numeric: tabular-nums;
    min-width: 42px;
  }
</style>
