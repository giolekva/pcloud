<script lang="ts">
  import { createLabel, createSelect } from '@melt-ui/svelte';
  import { createEventDispatcher } from 'svelte';
  import { derived } from 'svelte/store';

  export let name = "";
  export let availableNetworks = [];
  export let value: string | number | undefined | null = undefined;

  const { root } = createLabel();
  const { label, trigger, option, isSelected } = createSelect();

  const triggerWithoutRole = derived(trigger, ($trigger) => {
    const {role: _, ...rest} = $trigger;
    return rest;
  });

  $: (() => value = $label)();

  const dispatch = createEventDispatcher();
  $: dispatch("input", value);
</script>

<label use:root.action>
  <span>{name}</span>
</label>
<details role="list">
  <summary aria-haspopup="listbox" {...$triggerWithoutRole} use:trigger.action>{$label || "Select network"}</summary>
  <ul role="listbox">
    {#each availableNetworks as n}
      <li {...$option({ value: n.name, label: n.name })} use:option.action>
        <a>
          {n.name}: {n.domain}
          {#if $isSelected(n.name)}
            s
          {/if}
        </a>
      </li>
    {/each}
  </ul>
</details>

<style lang="postcss">
</style>
