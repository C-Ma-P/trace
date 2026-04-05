<script lang="ts">
  import type { AttributeDefinitionInfo, AttributeValue } from '../backend';

  let { definitions = [], attributes = $bindable([]) }: {
    definitions?: AttributeDefinitionInfo[];
    attributes?: AttributeValue[];
  } = $props();

  function handleNumberInput(index: number, value: string) {
    if (value === '') {
      attributes[index].number = null;
    } else {
      const n = parseFloat(value);
      if (!isNaN(n)) {
        attributes[index].number = n;
      }
    }
  }
</script>

<div class="attr-grid">
  {#each definitions as def, i}
    {#if attributes[i]}
      <div class="form-group">
        <label for="attr-{def.key}">
          {def.displayName}
          {#if def.unit}
            <span class="unit-label">({def.unit})</span>
          {/if}
        </label>

        {#if def.valueType === 'number'}
          <input
            id="attr-{def.key}"
            class="form-input"
            type="number"
            step="any"
            value={attributes[i].number ?? ''}
            oninput={(e) => handleNumberInput(i, e.currentTarget.value)}
            placeholder={def.displayName}
          />
        {:else if def.valueType === 'text'}
          <input
            id="attr-{def.key}"
            class="form-input"
            type="text"
            bind:value={attributes[i].text}
            placeholder={def.displayName}
          />
        {:else if def.valueType === 'bool'}
          <label class="checkbox-label">
            <input id="attr-{def.key}" type="checkbox" bind:checked={attributes[i].bool} />
            {def.displayName}
          </label>
        {/if}
      </div>
    {/if}
  {/each}
</div>

<style>
  .attr-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }
  .checkbox-label {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    color: var(--color-text-primary);
    cursor: pointer;
    padding: 6px 0;
  }
  .checkbox-label input[type='checkbox'] {
    width: 16px;
    height: 16px;
  }
</style>
