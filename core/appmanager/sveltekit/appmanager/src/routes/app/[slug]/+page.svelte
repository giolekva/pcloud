<script lang="ts">
  import { onMount } from "svelte";
  import { HSplitPane, VSplitPane } from "svelte-split-pane";
  import { TabGroup, Tab, Toast, toastStore } from "@skeletonlabs/skeleton";
  import { SubmitForm } from "@restspace/svelte-schema-form";
  import "@restspace/svelte-schema-form/css/layout.scss";
  import "@restspace/svelte-schema-form/css/basic-skin.scss";

  interface File {
    name string;
    contents string;
  }

  export let data: AppData;
  let readme: string = "";
  let files: File[] = [];

  let tabSet: number = 0;

  const submit = async (e) => {
	  const resp = await fetch(`/api/app/${data.slug}/install`, {
          method: "POST",
          headers: {
              "Accept": "application/json",
              "Content-Type": "application/json"
          },
          body: JSON.stringify(e.detail.value),
      });
      toastStore.trigger({
        message: await resp.text(),
        timeout: 1000,
      });
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
      files = app.files;
  };

  const change = (e) => render(e.detail.value);

  onMount(() => {
    if (data.config != null) {
      render(data.config);
    }
  });
</script>

{data.slug}
<HSplitPane>
    <left slot="left">
          <SubmitForm schema={data.schema} value={data.config ?? {}} on:submit={submit} on:value={change} submitText="Install" />
    </left>
    <right slot="right">
        <TabGroup>
            <Tab bind:group={tabSet} name="Readme" value={0}>Readme</Tab>
            {#each files as file, i }
                <Tab bind:group={tabSet} name={file.name} value={i + 1}>{file.name}</Tab>
            {/each}
            <svelte:fragment slot="panel">
                {#if tabSet === 0}
                    {readme}
                {:else}
                    <pre>
{files[tabSet - 1].contents}
                    </pre>
                {/if}
            </svelte:fragment>
        </TabGroup>
    </right>
</HSplitPane>
<Toast />