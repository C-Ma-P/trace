<script lang="ts">
  import Modal from '../ui/Modal.svelte';
  import { createComponent, type CategoryInfo } from '../backend';

  let { open = false, categories = [], onclose, oncreated }: {
    open?: boolean;
    categories?: CategoryInfo[];
    onclose?: () => void;
    oncreated?: (comp: any) => void;
  } = $props();

  let category = $state('');
  let mpn = $state('');
  let manufacturer = $state('');
  let pkg = $state('');
  let description = $state('');
  let saving = $state(false);
  let error = $state('');

  $effect(() => {
    if (open) {
      category = categories.length > 0 ? categories[0].value : '';
      mpn = '';
      manufacturer = '';
      pkg = '';
      description = '';
      error = '';
    }
  });

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    saving = true;
    error = '';
    try {
      const comp = await createComponent({
        category,
        mpn,
        manufacturer,
        package: pkg,
        description,
      });
      oncreated?.(comp);
    } catch (e: any) {
      error = e?.message ?? String(e);
    } finally {
      saving = false;
    }
  }
</script>

<Modal {open} title="New Component" {onclose} width="480px">
  <form onsubmit={handleSubmit} class="create-form">
    <div class="form-group">
      <label>Category</label>
      <select class="form-input" bind:value={category} required>
        {#each categories as cat}
          <option value={cat.value}>{cat.displayName}</option>
        {/each}
      </select>
    </div>

    <div class="form-row">
      <div class="form-group">
        <label>MPN</label>
        <input class="form-input" type="text" bind:value={mpn} placeholder="Part number" />
      </div>
      <div class="form-group">
        <label>Manufacturer</label>
        <input class="form-input" type="text" bind:value={manufacturer} placeholder="Manufacturer" />
      </div>
    </div>

    <div class="form-group">
      <label>Package</label>
      <input class="form-input" type="text" bind:value={pkg} placeholder="e.g. 0402, SOIC-8" />
    </div>

    <div class="form-group">
      <label>Description</label>
      <textarea class="form-input" bind:value={description} rows="2" placeholder="Optional description" />
    </div>

    {#if error}
      <div class="error-text">{error}</div>
    {/if}

    <div class="form-actions">
      <button type="button" class="btn btn-secondary" onclick={() => onclose?.()}>
        Cancel
      </button>
      <button type="submit" class="btn btn-primary" disabled={saving}>
        {saving ? 'Creating…' : 'Create Component'}
      </button>
    </div>
  </form>
</Modal>

<style>
  .create-form {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .form-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }
  .form-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding-top: 8px;
  }
</style>
