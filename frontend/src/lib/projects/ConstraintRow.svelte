<script lang="ts">
  import {
    getOperatorsForValueType,
    type Constraint,
    type AttributeDefinitionInfo,
    type OperatorInfo,
  } from '../backend';

  let { constraint = $bindable(), definitions = [], onkeychange, onremove }: {
    constraint: Constraint;
    definitions?: AttributeDefinitionInfo[];
    onkeychange?: (key: string) => void;
    onremove?: () => void;
  } = $props();

  let operators: OperatorInfo[] = $state([]);

  $effect(() => {
    loadOperators(constraint.valueType);
  });

  let currentDef = $derived(definitions.find((d) => d.key === constraint.key));

  async function loadOperators(vt: string) {
    operators = (await getOperatorsForValueType(vt)) ?? [];
  }

  function handleKeyChange(e: Event) {
    const key = (e.target as HTMLSelectElement).value;
    onkeychange?.(key);
  }

  function handleNumberInput(value: string) {
    if (value === '') {
      constraint.number = null;
    } else {
      const n = parseFloat(value);
      if (!isNaN(n)) {
        constraint.number = n;
      }
    }
  }
</script>

<div class="constraint-row">
  <select class="form-input key-select" value={constraint.key} onchange={handleKeyChange}>
    {#each definitions as def}
      <option value={def.key}>{def.displayName}</option>
    {/each}
  </select>

  <select class="form-input op-select" bind:value={constraint.operator}>
    {#each operators as op}
      <option value={op.value}>{op.displayName}</option>
    {/each}
  </select>

  <div class="value-cell">
    {#if constraint.valueType === 'number'}
      <input
        class="form-input value-input"
        type="number"
        step="any"
        value={constraint.number ?? ''}
        oninput={(e) => handleNumberInput(e.currentTarget.value)}
        placeholder="Value"
      />
    {:else if constraint.valueType === 'text'}
      <input
        class="form-input value-input"
        type="text"
        bind:value={constraint.text}
        placeholder="Value"
      />
    {:else if constraint.valueType === 'bool'}
      <select class="form-input value-input" bind:value={constraint.bool}>
        <option value={true}>Yes</option>
        <option value={false}>No</option>
      </select>
    {/if}

    {#if currentDef?.unit}
      <span class="unit-label">{currentDef.unit}</span>
    {/if}
  </div>

  <button class="btn btn-ghost btn-sm remove-btn" onclick={() => onremove?.()}>
    ✕
  </button>
</div>

<style>
  .constraint-row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px;
    background: var(--color-bg-muted);
    border-radius: var(--radius-md);
  }
  .key-select {
    width: 160px;
    font-size: 12px;
    padding: 5px 6px;
  }
  .op-select {
    width: 80px;
    font-size: 12px;
    padding: 5px 6px;
  }
  .value-cell {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 4px;
  }
  .value-input {
    flex: 1;
    font-size: 12px;
    padding: 5px 6px;
  }
  .remove-btn {
    color: var(--color-text-muted);
    flex-shrink: 0;
  }
  .remove-btn:hover {
    color: var(--color-danger);
  }
</style>
