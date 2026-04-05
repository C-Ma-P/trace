<script lang="ts">
  import ConstraintRow from './ConstraintRow.svelte';
  import {
    getRequirementDefinitions,
    categoryDisplayName,
    type Requirement,
    type Constraint,
    type CategoryInfo,
    type AttributeDefinitionInfo,
  } from '../backend';

  let { requirement = $bindable(), categories = [] }: {
    requirement: Requirement;
    categories?: CategoryInfo[];
  } = $props();

  let definitions: AttributeDefinitionInfo[] = $state([]);

  $effect(() => {
    loadDefinitions(requirement.category);
  });

  async function loadDefinitions(cat: string) {
    if (!cat) {
      definitions = [];
      return;
    }
    definitions = (await getRequirementDefinitions(cat)) ?? [];
  }

  function handleCategoryChange() {
    // Clear constraints when category changes since keys are category-specific
    requirement.constraints = [];
    requirement.selectedComponentId = null;
    requirement.resolution = null;
  }

  function addConstraint() {
    if (definitions.length === 0) return;
    const def = definitions[0];
    const newConstraint: Constraint = {
      key: def.key,
      valueType: def.valueType,
      operator: 'eq',
      text: def.valueType === 'text' ? '' : null,
      number: def.valueType === 'number' ? null : null,
      bool: def.valueType === 'bool' ? false : null,
      unit: def.unit,
    };
    requirement.constraints = [...requirement.constraints, newConstraint];
    requirement.selectedComponentId = null;
    requirement.resolution = null;
  }

  function removeConstraint(index: number) {
    requirement.constraints = requirement.constraints.filter((_, i) => i !== index);
    requirement.selectedComponentId = null;
    requirement.resolution = null;
  }

  function handleConstraintKeyChange(index: number, key: string) {
    const def = definitions.find((d) => d.key === key);
    if (!def) return;
    requirement.constraints[index] = {
      key: def.key,
      valueType: def.valueType,
      operator: 'eq',
      text: def.valueType === 'text' ? '' : null,
      number: def.valueType === 'number' ? null : null,
      bool: def.valueType === 'bool' ? false : null,
      unit: def.unit,
    };
    requirement.constraints = requirement.constraints;
    requirement.selectedComponentId = null;
    requirement.resolution = null;
  }
</script>

<div class="editor">
  <div class="editor-section">
    <div class="fields-row">
      <div class="form-group" style="flex: 2;">
        <label for="requirement-name">Name</label>
        <input id="requirement-name" class="form-input" bind:value={requirement.name} placeholder="Requirement name" />
      </div>
      <div class="form-group" style="flex: 1;">
        <label for="requirement-quantity">Quantity</label>
        <input id="requirement-quantity" class="form-input" type="number" min="1" bind:value={requirement.quantity} />
      </div>
    </div>

    <div class="form-group">
      <label for="requirement-category">Category</label>
      <select id="requirement-category" class="form-input" bind:value={requirement.category} onchange={handleCategoryChange}>
        {#each categories as cat}
          <option value={cat.value}>{cat.displayName}</option>
        {/each}
      </select>
    </div>
  </div>

  <div class="editor-section">
    <div class="constraints-header">
      <span class="constraints-title">Constraints</span>
      <button
        class="btn btn-secondary btn-sm"
        onclick={addConstraint}
        disabled={definitions.length === 0}
      >
        + Add
      </button>
    </div>

    {#if definitions.length === 0}
      <div class="empty-msg">No attribute definitions for this category</div>
    {:else if requirement.constraints.length === 0}
      <div class="empty-msg">No constraints — all components of this category will match</div>
    {:else}
      <div class="constraints-list">
        {#each requirement.constraints as constraint, i}
          <ConstraintRow
            bind:constraint={requirement.constraints[i]}
            {definitions}
            onkeychange={(key) => handleConstraintKeyChange(i, key)}
            onremove={() => removeConstraint(i)}
          />
        {/each}
      </div>
    {/if}
  </div>
</div>

<style>
  .editor {
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 20px;
  }
  .editor-section {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .fields-row {
    display: flex;
    gap: 12px;
  }
  .constraints-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  .constraints-title {
    font-size: 13px;
    font-weight: 600;
    color: var(--color-text-secondary);
  }
  .empty-msg {
    color: var(--color-text-muted);
    font-size: 12px;
    padding: 8px 0;
  }
  .constraints-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
</style>
