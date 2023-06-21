<script lang="ts">
	import { onMount } from "svelte";

	type app = {
		 name: string;
		 slug: string;
	};

	let apps: app[] = [];

	onMount(async () => {
		const resp = await fetch("/api/app-repo");
		apps = await resp.json();
	});
</script>



<nav class="list-nav">
	<!-- (optionally you can provide a label here) -->
	<ul>
	{#each apps as app}
	<li>
		<a href="/app/{app.slug}">
		   <span class="badge bg-primary-500">💀</span>
		   <span class="flex-auto">{app.name}</span>
		</a>
	</li>
	{/each}
	</ul>
</nav>
