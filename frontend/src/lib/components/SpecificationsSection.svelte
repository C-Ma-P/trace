<script lang="ts">
  import AttributeEditor from './AttributeEditor.svelte';
  import {
    updateComponentMetadata,
    replaceComponentAttributes,
    getCategoryDefinitions,
    formatAttributeValue,
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

  let editMPN = $state('');
  let editManufacturer = $state('');
  let editPackage = $state('');
  let editDescription = $state('');

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

<section class="specs-section">
  <div class="section-header">
    <h3 class="section-title">Specifications</h3>
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
    <!-- Edit mode: identity metadata + description + attributes -->
    <div class="fields-grid">
      <div class="form-group">
        <label for="spec-mpn">MPN</label>
        <input id="spec-mpn" class="form-input" bind:value={editMPN} />
      </div>
      <div class="form-group">
        <label for="spec-mfr">Manufacturer</label>
        <input id="spec-mfr" class="form-input" bind:value={editManufacturer} />
      </div>
      <div class="form-group">
        <label for="spec-pkg">Package</label>
        <input id="spec-pkg" class="form-input" bind:value={editPackage} />
      </div>
      <div class="form-group full-width">
        <label for="spec-desc">Description</label>
        <textarea id="spec-desc" class="form-input" bind:value={editDescription} rows="2"></textarea>
      </div>
    </div>

    {#if definitions.length > 0}
      <h4 class="subsection-title">Attributes</h4>
      <AttributeEditor {definitions} bind:attributes={editAttributes} />
    {/if}
  {:else}
    <!-- View mode: description + category-defined attributes -->
    {#if component.description}
      <p class="spec-description">{component.description}</p>
    {/if}

    {#if definitions.length > 0}
      <div class="spec-grid">
        {#each definitions as def}
          {@const attr = component.attributes.find((a) => a.key === def.key)}
          <div class="spec-item">
            <span class="spec-label">
              {def.displayName}
            </span>
            <span class="spec-value">
              {#if attr}
                {#if attr.valueType === 'number' && attr.number !== null}
                  {formatAttributeValue(attr.number, def.unit)}
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

    {#if !component.description && definitions.length === 0}
      <p class="empty-hint">No specifications defined.</p>
    {/if}
  {/if}
</section>

<style>
  .specs-section {
    padding: 20px;
    border-bottom: 1px solid var(--color-border);
  }
  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 14px;
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
  .fields-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 14px;
  }
  .full-width {
    grid-column: 1 / -1;
  }
  .subsection-title {
    font-size: 12px;
    font-weight: 600;
    color: var(--color-text-secondary);
    margin: 18px 0 12px;
  }
  .spec-description {
    font-size: 13px;
    color: var(--color-text-primary);
    line-height: 1.5;
    margin-bottom: 14px;
  }
  .spec-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px 16px;
  }
  .spec-item {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .spec-label {
    font-size: 11px;
    font-weight: 500;
    color: var(--color-text-muted);
  }
  .spec-value {
    font-size: 13px;
    color: var(--color-text-primary);
  }
  .empty-hint {
    font-size: 12px;
    color: var(--color-text-muted);
  }
  .error-text {
    font-size: 12px;
    color: var(--color-danger);
    margin-bottom: 10px;
  }
</style>
