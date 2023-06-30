<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  import NetworkSelector from "./NetworkSelector.svelte";
  import TextInput from "./TextInput.svelte";

  const dispatch = createEventDispatcher();

  export let availableNetworks = [
    {
      name: "Public",
      domain: "qwe.lekva.me",
    },
    {
      name: "Private",
      domain: "p.qwe.lekva.me",
    },
  ];
  export let schema = null;
  export let value: Record<string, unknown> = {};
  export let readonly: boolean = false;

  function update(k: string, v: unknown) {
    value[k] = v;
    dispatch("change", value);
  }

  function updater(key: string) {
    return (v) => update(key, v.detail);
  }

  const isNetwork = (schema): boolean => {
    return "$ref" in schema &&
      typeof schema["$ref"] === "string" &&
      schema["$ref"] === "#/definitions/network";
  };
</script>

{#each Object.entries(schema.properties) as [name, schema]}
  {#if schema.type === "object"}
    <svelte:self {readonly} {schema} on:change={updater(name)} />
  {:else if isNetwork(schema)}
    <NetworkSelector {readonly} {name} value={value[name]} {availableNetworks} on:input={updater(name)} />
  {:else if schema.type === "string"}
    <TextInput {readonly} {name} value={value[name]} on:input={updater(name)} />
  {/if}
{/each}

<style>
</style>
