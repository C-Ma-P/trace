<script lang="ts">
  import RequirementEditor from './RequirementEditor.svelte';
  import {
    replaceProjectRequirements,
    updateProjectMetadata,
    categoryDisplayName,
    type Project,
    type Requirement,
    type CategoryInfo,
  } from '../backend';

  let { project, categories = [], onupdated }: {
    project: Project;
    categories?: CategoryInfo[];
    onupdated?: () => void;
  } = $props();

  let editing = $state(false);
  let saving = $state(false);
  let error = $state('');

  // Metadata editing
  let editName = $state('');
  let editDescription = $state('');

  // Requirements editing
  let editRequirements: Requirement[] = $state([]);
  let selectedReqIndex: number | null = $state(null);

  function startEdit() {
    editName = project.name;
    editDescription = project.description;
    editRequirements = JSON.parse(JSON.stringify(project.requirements));
    selectedReqIndex = editRequirements.length > 0 ? 0 : null;
    editing = true;
    error = '';
  }

  function cancelEdit() {
    editing = false;
    error = '';
    selectedReqIndex = null;
  }

  function addRequirement() {
    const newReq: Requirement = {
      id: '',
      projectId: project.id,
      name: '',
      category: categories.length > 0 ? categories[0].value : '',
      quantity: 1,
      selectedComponentId: null,
      resolution: null,
      constraints: [],
    };
    editRequirements = [...editRequirements, newReq];
    selectedReqIndex = editRequirements.length - 1;
  }

  function removeRequirement(index: number) {
    editRequirements = editRequirements.filter((_, i) => i !== index);
    if (selectedReqIndex !== null) {
      if (selectedReqIndex === index) {
        selectedReqIndex = editRequirements.length > 0 ? Math.min(index, editRequirements.length - 1) : null;
      } else if (selectedReqIndex > index) {
        selectedReqIndex--;
      }
    }
  }

  async function save() {
    saving = true;
    error = '';
    try {
      // Update metadata if changed
      if (editName !== project.name || editDescription !== project.description) {
        await updateProjectMetadata({
          id: project.id,
          name: editName,
          description: editDescription,
        });
      }
      // Replace requirements
      await replaceProjectRequirements(project.id, editRequirements);
      editing = false;
      selectedReqIndex = null;
      onupdated?.();
    } catch (e: any) {
      // Parse Wails error JSON to extract a human-readable message
      try {
        const parsed = JSON.parse(e?.message ?? String(e));
        error = (parsed.message as string) ?? String(e);
      } catch {
        error = e?.message ?? String(e);
      }
    } finally {
      saving = false;
    }
  }
</script>

<div class="requirements-tab">
  <div class="section-header">
    <h3 class="section-title">Project Details & Requirements</h3>
    {#if !editing}
      <button class="btn btn-secondary btn-sm" onclick={startEdit}>Edit</button>
    {:else}
      <div class="edit-actions">
        <button class="btn btn-secondary btn-sm" onclick={cancelEdit} disabled={saving}>
          Cancel
        </button>
        <button class="btn btn-primary btn-sm" onclick={save} disabled={saving}>
          {saving ? 'Saving…' : 'Save All'}
        </button>
      </div>
    {/if}
  </div>

  {#if error}
    <div class="error-text" style="margin-bottom: 12px;">{error}</div>
  {/if}

  <!-- Metadata -->
  <div class="meta-section">
    {#if editing}
      <div class="fields-grid">
        <div class="form-group">
          <label>Name</label>
          <input class="form-input" bind:value={editName} />
        </div>
        <div class="form-group">
          <label>Description</label>
          <input class="form-input" bind:value={editDescription} />
        </div>
      </div>
    {/if}
  </div>

  <!-- Requirements list -->
  <div class="req-section">
    <div class="req-section-header">
      <span class="req-section-title">
        Requirements ({editing ? editRequirements.length : project.requirements.length})
      </span>
      {#if editing}
        <button class="btn btn-secondary btn-sm" onclick={addRequirement}>
          + Add Requirement
        </button>
      {/if}
    </div>

    {#if !editing}
      <!-- Read-only view -->
      {#if project.requirements.length === 0}
        <div class="empty-msg">No requirements defined</div>
      {:else}
        <div class="req-list">
          {#each project.requirements as req}
            <div class="req-card">
              <div class="req-card-header">
                <span class="req-name">{req.name || 'Unnamed'}</span>
                <div class="req-badges">
                  <span class="badge">{categoryDisplayName(categories, req.category)}</span>
                  <span class="badge">×{req.quantity}</span>
                </div>
              </div>
              {#if req.constraints.length > 0}
                <div class="req-constraints-summary">
                  {#each req.constraints as c}
                    <span class="constraint-chip">
                      {c.key} {c.operator}
                      {#if c.valueType === 'number' && c.number !== null}
                        {c.number}
                      {:else if c.valueType === 'text' && c.text}
                        {c.text}
                      {:else if c.valueType === 'bool' && c.bool !== null}
                        {c.bool}
                      {/if}
                      {#if c.unit}
                        <span class="unit-label">{c.unit}</span>
                      {/if}
                    </span>
                  {/each}
                </div>
              {/if}
              {#if req.selectedComponentId}
                <div class="req-selected">
                  Resolved part: <span class="selected-id">{req.selectedComponentId}</span>
                </div>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    {:else}
      <!-- Editing view: list + editor -->
      <div class="req-edit-layout">
        <div class="req-edit-list">
          {#if editRequirements.length === 0}
            <div class="empty-msg">Add a requirement to get started</div>
          {:else}
            {#each editRequirements as req, i}
              <div
                class="req-edit-item"
                class:selected={selectedReqIndex === i}
                onclick={() => (selectedReqIndex = i)}
                onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') selectedReqIndex = i; }}
                role="button"
                tabindex="0"
              >
                <span class="req-edit-name">{req.name || 'Unnamed'}</span>
                <div class="req-edit-meta">
                  <span class="badge">{categoryDisplayName(categories, req.category)}</span>
                  <button
                    class="btn btn-ghost btn-sm danger-text"
                    onclick={(e) => { e.stopPropagation(); removeRequirement(i); }}
                  >
                    ✕
                  </button>
                </div>
              </div>
            {/each}
          {/if}
        </div>

        <div class="req-editor-pane">
          {#if selectedReqIndex !== null && editRequirements[selectedReqIndex]}
            <RequirementEditor
              bind:requirement={editRequirements[selectedReqIndex]}
              {categories}
            />
          {:else}
            <div class="empty-state">Select a requirement to edit</div>
          {/if}
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  .requirements-tab {
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
  }
  .edit-actions {
    display: flex;
    gap: 8px;
  }
  .meta-section {
    margin-bottom: 20px;
  }
  .fields-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }
  .req-section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
  }
  .req-section-title {
    font-size: 13px;
    font-weight: 600;
    color: var(--color-text-secondary);
  }
  .empty-msg {
    color: var(--color-text-muted);
    font-size: 13px;
    padding: 16px 0;
  }

  /* Read-only requirement cards */
  .req-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .req-card {
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    padding: 12px;
    background: var(--color-bg-surface);
  }
  .req-card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  .req-name {
    font-weight: 600;
    font-size: 13px;
  }
  .req-badges {
    display: flex;
    gap: 6px;
  }
  .req-constraints-summary {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    margin-top: 8px;
  }
  .constraint-chip {
    font-size: 11px;
    padding: 2px 6px;
    background: var(--color-bg-muted);
    border-radius: 2px;
    color: var(--color-text-secondary);
    font-family: var(--font-mono);
  }
  .req-selected {
    margin-top: 6px;
    font-size: 11px;
    color: var(--color-text-muted);
  }
  .selected-id {
    font-family: var(--font-mono);
    font-size: 11px;
  }

  /* Edit layout */
  .req-edit-layout {
    display: flex;
    gap: 1px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: var(--color-border);
    min-height: 300px;
    overflow: hidden;
  }
  .req-edit-list {
    width: 220px;
    min-width: 180px;
    background: var(--color-bg-surface);
    overflow-y: auto;
  }
  .req-edit-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
    padding: 10px 12px;
    text-align: left;
    border-bottom: 1px solid var(--color-border);
    transition: background 0.1s;
  }
  .req-edit-item:hover {
    background: var(--color-bg-hover);
  }
  .req-edit-item.selected {
    background: var(--color-bg-selected);
  }
  .req-edit-name {
    font-size: 12px;
    font-weight: 500;
    color: var(--color-text-primary);
  }
  .req-edit-meta {
    display: flex;
    align-items: center;
    gap: 4px;
  }
  .danger-text {
    color: var(--color-danger);
  }
  .req-editor-pane {
    flex: 1;
    background: var(--color-bg-surface);
    overflow-y: auto;
  }
</style>
