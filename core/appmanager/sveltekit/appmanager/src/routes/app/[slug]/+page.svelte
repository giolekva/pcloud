<script lang="ts">
  import { onMount } from "svelte";
  import { SubmitForm } from "@restspace/svelte-schema-form";
  import "@restspace/svelte-schema-form/css/layout.scss";
  // import "@restspace/svelte-schema-form/css/basic-skin.scss";
  import Icon from '@iconify/svelte';
  import toast from "svelte-french-toast";

  import ConfigurationForm from "$lib/ConfigurationForm.svelte";
import { writable } from "svelte/store";

  export let data: AppData;
  let config: Record<string, any> = null;
  let readme: string = null;

  const submit = async (config) => {
	const resp = await fetch(`/api/app/${data.slug}/install`, {
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
	const resp = await fetch(`/api/app/${data.slug}/render`, {
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
    case "string": return schema.default ?? "";
    case "object": {
      const ret: Record<string, any> = {};
      for (const [key, value] of Object.entries(schema.properties)) {
        ret[key] = extractDefaultValues(value);
      };
      return ret;
    }
    }
  };

  onMount(() => {
    data.config = null; // TODO(giolekva): remove
    if (data.config != null) {
      config = data.config;
    } else {
      config = extractDefaultValues(data.schema);
      console.log(config);
    }
    render(config);
  });

  const formData = writable(null);
  $: render($formData);
</script>

<h1><Icon icon="{data.icon}" width="50" height="50" />{data.name}</h1>
<pre>{readme}</pre>

<form on:submit={() => submit($formData)}>
  <ConfigurationForm schema={data.schema} on:change={(e) => formData.set(e.detail)} />
  <input type="submit" value="Install" />
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
