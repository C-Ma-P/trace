<script lang="ts">
  import AttributeEditor from './AttributeEditor.svelte';
  import {
    updateComponentMetadata,
    replaceComponentAttributes,
    getCategoryDefinitions,
    categoryDisplayName,
    type Component,
    type CategoryInfo,
    type AttributeDefinitionInfo,
    type AttributeValue,
  } from '../backend';

  let { component, categories = [], onupdated }: {
    component: Component;
    categories?: CategoryInfo[];
    onupdated?: () => void;
  } = $props();

  let editing = $state(false);
  let saving = $state(false);
  let error = $state('');

  // Metadata edit state
  let editMPN = $state('');
  let editManufacturer = $state('');
  let editPackage = $state('');
  let editDescription = $state('');

  // Attribute edit state
  let definitions: AttributeDefinitionInfo[] = $state([]);
  let editAttributes: AttributeValue[] = $state([]);

  $effect(() => {
    loadDefinitions(component.category);
  });

  async function loadDefinitions(cat: string) {
    definitions = (await getCategoryDefinitions(cat)) ?? [];
  }

  function startEdit() {
    editMPN = component.mpn;
    editManufacturer = component.manufacturer;
    editPackage = component.package;
    editDescription = component.description;
    editAttributes = buildEditAttributes(component.attributes, definitions);
    editing = true;
    error = '';
  }

  function cancelEdit() {
    editing = false;
    error = '';
  }

  function buildEditAttributes(
    current: AttributeValue[],
    defs: AttributeDefinitionInfo[]
  ): AttributeValue[] {
    const index = new Map(current.map((a) => [a.key, a]));
    return defs.map((def) => {
      const existing = index.get(def.key);
      if (existing) return { ...existing };
      return {
        key: def.key,
        valueType: def.valueType,
        text: def.valueType === 'text' ? '' : null,
        number: def.valueType === 'number' ? null : null,
        bool: def.valueType === 'bool' ? false : null,
        unit: def.unit,
      };
    });
  }

  async function save() {
    saving = true;
    error = '';
    try {
      await updateComponentMetadata({
        id: component.id,
        mpn: editMPN,
        manufacturer: editManufacturer,
        package: editPackage,
        description: editDescription,
      });

      // Filter out attributes with no value set
      const attrsToSave = editAttributes.filter((a) => {
        if (a.valueType === 'text') return a.text !== null && a.text !== '';
        if (a.valueType === 'number') return a.number !== null;
        if (a.valueType === 'bool') return a.bool !== null;
        return false;
      });

      await replaceComponentAttributes(component.id, attrsToSave);

      editing = false;
      onupdated?.();
    } catch (e: any) {
      error = e?.message ?? String(e);
    } finally {
      saving = false;
    }
  }
</script>

<div class="details-tab">
  <div class="section-header">
    <h3 class="section-title">Component Information</h3>
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
    <div class="error-text" style="padding: 0 0 12px 0;">{error}</div>
  {/if}

  <div class="fields-grid">
    <div class="form-group">
      <label>Category</label>
      <span class="field-value">
        {categoryDisplayName(categories, component.category)}
      </span>
    </div>

    <div class="form-group">
      <label>MPN</label>
      {#if editing}
        <input class="form-input" bind:value={editMPN} />
      {:else}
        <span class="field-value">{component.mpn || '—'}</span>
      {/if}
    </div>

    <div class="form-group">
      <label>Manufacturer</label>
      {#if editing}
        <input class="form-input" bind:value={editManufacturer} />
      {:else}
        <span class="field-value">{component.manufacturer || '—'}</span>
      {/if}
    </div>

    <div class="form-group">
      <label>Package</label>
      {#if editing}
        <input class="form-input" bind:value={editPackage} />
      {:else}
        <span class="field-value">{component.package || '—'}</span>
      {/if}
    </div>

    <div class="form-group full-width">
      <label>Description</label>
      {#if editing}
        <textarea class="form-input" bind:value={editDescription} rows="2" />
      {:else}
        <span class="field-value">{component.description || '—'}</span>
      {/if}
    </div>
  </div>

  {#if definitions.length > 0}
    <div class="section-header" style="margin-top: 24px;">
      <h3 class="section-title">Attributes</h3>
    </div>

    {#if editing}
      <AttributeEditor {definitions} bind:attributes={editAttributes} />
    {:else}
      <div class="fields-grid">
        {#each definitions as def}
          {@const attr = component.attributes.find((a) => a.key === def.key)}
          <div class="form-group">
            <label>
              {def.displayName}
              {#if def.unit}
                <span class="unit-label">({def.unit})</span>
              {/if}
            </label>
            <span class="field-value">
              {#if attr}
                {#if attr.valueType === 'number' && attr.number !== null}
                  {attr.number}
                {:else if attr.valueType === 'text' && attr.text}
                  {attr.text}
                {:else if attr.valueType === 'bool' && attr.bool !== null}
                  {attr.bool ? 'Yes' : 'No'}
                {:else}
                  —
                {/if}
              {:else}
                —
              {/if}
            </span>
          </div>
        {/each}
      </div>
    {/if}
  {/if}

</div>

<style>
  .details-tab {
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
    align-items: center;
  }
  .fields-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }
  .full-width {
    grid-column: 1 / -1;
  }
  .field-value {
    font-size: 13px;
    color: var(--color-text-primary);
    padding: 6px 0;
  }
</style>
