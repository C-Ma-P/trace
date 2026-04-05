<script lang="ts">
  import Modal from '../ui/Modal.svelte';
  import { createProject } from '../backend';

  let { open = false, onclose, oncreated }: {
    open?: boolean;
    onclose?: () => void;
    oncreated?: (project: any) => void;
  } = $props();

  let name = $state('');
  let description = $state('');
  let saving = $state(false);
  let error = $state('');

  $effect(() => {
    if (open) {
      name = '';
      description = '';
      error = '';
    }
  });

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    saving = true;
    error = '';
    try {
      const project = await createProject({ name, description });
      oncreated?.(project);
    } catch (e: any) {
      error = e?.message ?? String(e);
    } finally {
      saving = false;
    }
  }
</script>

<Modal {open} title="New Project" {onclose} width="440px">
  <form onsubmit={handleSubmit} class="create-form">
    <div class="form-group">
      <label>Name</label>
      <input class="form-input" type="text" bind:value={name} placeholder="Project name" required />
    </div>

    <div class="form-group">
      <label>Description</label>
      <textarea
        class="form-input"
        bind:value={description}
        rows="3"
        placeholder="Optional description"
      />
    </div>

    {#if error}
      <div class="error-text">{error}</div>
    {/if}

    <div class="form-actions">
      <button type="button" class="btn btn-secondary" onclick={() => onclose?.()}>
        Cancel
      </button>
      <button type="submit" class="btn btn-primary" disabled={saving || !name.trim()}>
        {saving ? 'Creating…' : 'Create Project'}
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
  .form-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding-top: 8px;
  }
</style>
