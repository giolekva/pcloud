<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { createLabel } from '@melt-ui/svelte';

  export let name: string = "";
  export let prefix: string = "";
  export let value: string | undefined = undefined;

  const { root } = createLabel();

  const dispatch = createEventDispatcher();

  $: id = `${prefix}${name}`;
  $: dispatch("input", value);
</script>

<label use:root.action for={name} class="font-medium">
  <span>{name}</span>
</label>
<input type="text" {id} {name} bind:value />
