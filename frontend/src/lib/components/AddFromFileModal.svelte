<script lang="ts">
  import Modal from '../ui/Modal.svelte';
  import { createComponentAsset } from '../backend';

  let { open = false, componentId, onclose, oncreated }: {
    open?: boolean;
    componentId: string;
    onclose?: () => void;
    oncreated?: () => void;
  } = $props();

  const assetTypes = [
    { value: 'symbol', label: 'Symbol' },
    { value: 'footprint', label: 'Footprint' },
    { value: '3d_model', label: '3D Model' },
    { value: 'datasheet', label: 'Datasheet' },
  ];

  let assetType = $state('symbol');
  let label = $state('');
  let urlOrPath = $state('');
  let saving = $state(false);
  let error = $state('');

  function reset() {
    assetType = 'symbol';
    label = '';
    urlOrPath = '';
    error = '';
  }

  $effect(() => {
    if (open) reset();
  });

  async function handleSubmit() {
    if (!label.trim()) {
      error = 'Label is required';
      return;
    }
    if (!urlOrPath.trim()) {
      error = 'File path or URL is required';
      return;
    }

    saving = true;
    error = '';
    try {
      await createComponentAsset({
        componentId,
        assetType,
        source: 'local_file',
        status: 'candidate',
        label: label.trim(),
        urlOrPath: urlOrPath.trim(),
      });
      oncreated?.();
    } catch (e: any) {
      error = e?.message ?? String(e);
    } finally {
      saving = false;
    }
  }
</script>

<Modal {open} title="Import Downloaded Files" onclose={() => onclose?.()}>
  <div class="add-from-file">
    <p class="help-text">
      Import a CAD asset you already downloaded for this part — KiCad library files,
      symbols, footprints, 3D models, or datasheets. Provide the asset type, a label,
      and the file path or URL.
    </p>

    <div class="form-group">
      <label for="aft-type">Asset Type</label>
      <select id="aft-type" class="form-input" bind:value={assetType}>
        {#each assetTypes as t}
          <option value={t.value}>{t.label}</option>
        {/each}
      </select>
    </div>

    <div class="form-group">
      <label for="aft-label">Label</label>
      <input
        id="aft-label"
        class="form-input"
        type="text"
        placeholder="e.g. TSSOP-20 footprint"
        bind:value={label}
      />
    </div>

    <div class="form-group">
      <label for="aft-path">File Path or URL</label>
      <input
        id="aft-path"
        class="form-input"
        type="text"
        placeholder="e.g. /home/user/libs/symbol.kicad_sym"
        bind:value={urlOrPath}
      />
    </div>

    {#if error}
      <div class="error-text">{error}</div>
    {/if}

    <div class="modal-actions">
      <button class="btn btn-secondary" onclick={() => onclose?.()} disabled={saving}>
        Cancel
      </button>
      <button class="btn btn-primary" onclick={handleSubmit} disabled={saving}>
        {saving ? 'Importing…' : 'Import Asset'}
      </button>
    </div>
  </div>
</Modal>

<style>
  .add-from-file {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }
  .help-text {
    font-size: 12px;
    color: var(--color-text-secondary);
    line-height: 1.5;
    padding: 10px 12px;
    background: var(--color-bg-muted);
    border-radius: var(--radius-md);
  }
  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding-top: 4px;
  }
</style>
