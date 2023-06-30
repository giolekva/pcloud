<script lang="ts">
  import { onMount } from "svelte";
  import Icon from '@iconify/svelte';
  import toast from "svelte-french-toast";

  import ConfigurationForm from "$lib/ConfigurationForm.svelte";
  import { writable } from "svelte/store";

  export let data;
  let config: Record<string, any> = null;
  let readme: string = null;

  const submit = async (config) => {
	const resp = await fetch(`/api/instance/${data.slug}/update`, {
      method: "POST",
      headers: {
        "Accept": "application/json",
        "Content-Type": "application/json"
      },
      body: JSON.stringify(config),
    });
    if (resp.status === 200) {
      toast.success("Installed");
    } else {
      toast.error("Installation failed");
    }
    return false;
  };

  const render = async (config) => {
    console.log(config);
	const resp = await fetch(`/api/app/${data.appSlug}/render`, {
      method: "POST",
      headers: {
        "Accept": "application/json",
        "Content-Type": "application/json"
      },
      body: JSON.stringify(config),
    });
    const app = await resp.json();
    readme = app.readme;
  };

  const extractDefaultValues = (schema) => {
    switch (schema.type) {
    case "object": {
      const ret: Record<string, any> = {};
      for (const [key, value] of Object.entries(schema.properties)) {
        ret[key] = extractDefaultValues(value);
      };
      return ret;
    }
    default: return schema.default ?? "";
    }
  };

  onMount(() => {
    config = extractDefaultValues(data.schema);
    render(config);
  });

  const formData = writable(null);
  $: render($formData);
</script>

<h1><Icon icon="{data.icon}" width="50" height="50" />{data.name}</h1>
<pre>{readme}</pre>

<form on:submit={() => submit($formData)}>
  <ConfigurationForm schema={data.schema} value={data.instances[0].config.Values} on:change={(e) => formData.set(e.detail)} />
  <input type="submit" value="Update" />
</form>

<style>
  pre {
    white-space: pre-wrap;       /* Since CSS 2.1 */
    white-space: -moz-pre-wrap;  /* Mozilla, since 1999 */
    white-space: -pre-wrap;      /* Opera 4-6 */
    white-space: -o-pre-wrap;    /* Opera 7 */
    word-wrap: break-word;       /* Internet Explorer 5.5+ */
    background-color: transparent;
  }
</style>
