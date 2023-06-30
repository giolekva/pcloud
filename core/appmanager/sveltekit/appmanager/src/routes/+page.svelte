<script lang="ts">
  import { onMount } from "svelte";
  import Icon from '@iconify/svelte';

	type app = {
		 name: string;
	  slug: string;
      icon: string;
      shortDescription: string;
	};

	let apps: app[] = [];

	onMount(async () => {
		const resp = await fetch("/api/app-repo");
		apps = await resp.json();
	});

  let cur = null;
  const view = (e) => {
    if (cur === e.target) {
      cur = null;
      return;
    }
    console.log(111)
    console.log(cur?.parentElement);
    cur?.parentElement.toggleAttribute("open");
    cur = e.target;
  };

  const search = (e) => {
    console.log(e.target.value);
  };
</script>

<div class="main">
<form>
  <input type="search" placeholder="Search" on:input={search} />
</form>

<aside>
  <nav>
    <ul>
      {#each apps as app}
        <li>
          <article>
            <div>
              <a href="/app/{app.slug}" class="logo">
                <Icon icon="{app.icon}" width="50" height="50" />
              </a>
            </div>
            <div>
              <a href="/app/{app.slug}">
                {app.name}
              </a>
              {app.shortDescription}
            </div>
          </article>
        </li>
      {/each}
    </ul>
  </nav>
</aside>
</div>

<style>
  .main {
    max-width: 70%;
    margin: 0 auto;
  }

  article {
    margin: 0.3em;
    margin-bottom: 0.3em;

    display: flex;
    flex-direction: row;
  }

  .logo {
    display: table-cell;
    vertical-align: middle;
  }
  nav li {
    padding-top: 0;
    padding-bottom: 0;
  }

  input[type="search"] {
    margin-bottom: 0;
  }
</style>
