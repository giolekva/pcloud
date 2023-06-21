<script lang="ts">
	import "@skeletonlabs/skeleton/themes/theme-skeleton.css";
	import "@skeletonlabs/skeleton/styles/skeleton.css";
  import "../app.postcss";
import { onMount } from "svelte";

  onMount(() => {
    window.addEventListener('message', (e) => {
      console.log(e);
      if ("go_back" in e.data) {
        window.history.go(-1);
      }
    });
  });
</script>

<slot />

