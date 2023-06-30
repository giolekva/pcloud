<script lang="ts">
import { createEventDispatcher } from 'svelte';

  import { derived, writable, type Writable } from 'svelte/store'

  import NetworkSelector from "./NetworkSelector.svelte";
  import TextInput from "./TextInput.svelte";

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

  const isNetwork = (schema): boolean => {
    return "$ref" in schema &&
      typeof schema["$ref"] === "string" &&
      schema["$ref"] === "#/definitions/network";
  };

  type Data = Record<string, Writable<any>>;

  const children = Object.fromEntries(Object.entries(schema.properties).map(([field, fieldSchema]) => {
    switch (fieldSchema.type) {
    case "object":
      return [field, writable<Data | undefined>(undefined)];
    default:
      return [field, writable<string | undefined>(field)];
    }
  }));
  const data = derived(Object.values(children), ($values) => {
    return Object.fromEntries(Object.keys(children).map((field, index) => ([
      field,
      $values[index],
    ])));
  });

  const dispatch = createEventDispatcher();
  $: dispatch("change", $data);
</script>

{#each Object.entries(schema.properties) as [name, schema]}
  {#if schema.type === "object"}
    <svelte:self schema />
  {:else if isNetwork(schema)}
    <NetworkSelector {name} {availableNetworks} on:input={(v) => children[name].set(v.detail)} />
  {:else if schema.type === "string"}
    <TextInput {name} on:input={(v) => children[name].set(v.detail)} />
  {/if}
{/each}

<style>
</style>
