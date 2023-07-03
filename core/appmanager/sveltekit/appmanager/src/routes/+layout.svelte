<script lang="ts">
  import "@picocss/pico";
  import "../app.postcss";

  import { Toaster } from "svelte-french-toast";
</script>

<div id="main">
  <slot />
</div>

<Toaster />

<style>
@media screen and (min-width:801px) {
#main {
  width: 600px;
}
}
</style>
