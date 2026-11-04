<script lang="ts">
	// DashList lists every dashboard the user can see, found by type under
	// each anchor, with where each one sits.
	import X from "@lucide/svelte/icons/x"
	import { untrack } from "svelte"
	import { resolve } from "$app/paths"
	import { auth } from "#lib/auth.svelte.ts"
	import Header from "#lib/components/Header.svelte"
	import IconButton from "#lib/components/IconButton.svelte"
	import type { NodeMatch } from "#lib/siot/search.ts"
	import { store } from "#lib/siot/store.ts"
	import { dashboardType } from "./index.ts"

	let list = $state<NodeMatch[] | null>(null)

	$effect(() => {
		const user = auth.user
		if (user) {
			store.start(user.token, () => {
				auth.signOut()
			})
		}
	})

	async function load() {
		list = await store.searchNodes("", 100, dashboardType)
	}

	$effect(() => {
		store.onConnected = () => void load()
		if (store.ready) {
			untrack(() => void load())
		}
	})

	function signOut() {
		void store.stop()
		auth.signOut()
	}
</script>

<svelte:head>
	<title>Dashboards - SIOT</title>
</svelte:head>

<div class="page">
	<Header
		email={auth.user?.email ?? ""}
		badge={store.connected ? "" : "connecting..."}
		onSignOut={signOut}
	/>
	{#if store.error}
		<div class="error" role="alert">
			<span>{store.error}</span>
			<IconButton label="Dismiss" onclick={() => store.dismissError()}><X size={18} /></IconButton>
		</div>
	{/if}
	<main>
		<h2>Dashboards</h2>
		{#if list === null}
			<p class="muted">Loading...</p>
		{:else if list.length === 0}
			<p class="muted">
				No dashboards yet. Add a Dashboard node under a device or group in
				<a href={resolve("/")}>Nodes</a>.
			</p>
		{:else}
			<ul>
				{#each list as d (d.id)}
					<li>
						<a href={resolve("/dash/[id]/[[widget]]", { id: d.id })}>{d.description}</a>
						{#if d.path.length > 0}<span class="path">{d.path.join(" / ")}</span>{/if}
					</li>
				{/each}
			</ul>
		{/if}
	</main>
</div>

<style>
	.page {
		max-width: 1280px;
		margin: 0 auto;
		padding: 20px 16px;
		display: flex;
		flex-direction: column;
		gap: 24px;
	}

	h2 {
		margin: 0 0 16px;
	}

	ul {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	li {
		display: flex;
		flex-wrap: wrap;
		align-items: baseline;
		gap: 4px 12px;
		padding: 10px 14px;
		border: 1px solid var(--border);
		border-radius: 8px;
	}

	li a {
		font-weight: 600;
	}

	.path,
	.muted {
		color: var(--muted);
	}

	.path {
		font-size: 13px;
	}

	.error {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		padding: 4px 16px;
		font-size: 14px;
		font-weight: 600;
		color: #fff;
		background: var(--error-bg);
		border-radius: 4px;
	}
</style>
