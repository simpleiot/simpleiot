<script lang="ts">
	// DashView is the page for one dashboard: the node and its widgets,
	// kept live by the store, with the nodes they bind to. ?kiosk hides the
	// chrome for a wall display.
	import X from "@lucide/svelte/icons/x"
	import { untrack } from "svelte"
	import { resolve } from "$app/paths"
	import { page } from "$app/state"
	import { auth } from "#lib/auth.svelte.ts"
	import Header from "#lib/components/Header.svelte"
	import IconButton from "#lib/components/IconButton.svelte"
	import { isTombstone, type Node } from "#lib/siot/node.ts"
	import { getBestDesc } from "#lib/siot/point.ts"
	import { store } from "#lib/siot/store.ts"
	import { boundNodeIds, parseBindings } from "./bindings.ts"
	import Dashboard from "./Dashboard.svelte"

	let { id }: { id: string } = $props()

	const kiosk = $derived(page.url.searchParams.has("kiosk"))

	let notFound = $state(false)
	let childIds = $state.raw<string[]>([])

	$effect(() => {
		const user = auth.user
		if (user) {
			store.start(user.token, () => {
				auth.signOut()
			})
		}
	})

	// load fetches the dashboard and its children; the store then keeps
	// them live, and a changed edge below the dashboard loads again
	let loading = 0
	async function load(dashId: string) {
		const mine = ++loading
		const r = await store.fetchSubtree(dashId, 1)
		if (mine !== loading) {
			return
		}
		if (!r) {
			notFound = true
			childIds = []
			return
		}
		notFound = false
		childIds = r.nodes.filter((n) => n.parent === dashId && !isTombstone(n)).map((n) => n.id)
	}

	$effect(() => {
		const dashId = id
		store.onConnected = () => void load(dashId)
		store.onLiveEdge = (parentId) => {
			if (parentId === dashId) {
				void load(dashId)
			}
		}
		if (store.ready) {
			untrack(() => void load(dashId))
		}
		return () => {
			store.onLiveEdge = null
			void store.watchNodes([])
		}
	})

	const dashboard = $derived(store.liveNodes.get(id))
	const children = $derived(
		childIds.map((c) => store.liveNodes.get(c)).filter((n): n is Node => n !== undefined)
	)

	// the nodes to keep live: the dashboard, its children, and what they
	// bind to
	$effect(() => {
		const wanted = new Set([id, ...childIds])
		for (const c of children) {
			for (const b of boundNodeIds(parseBindings(c))) {
				wanted.add(b)
			}
		}
		void store.watchNodes([...wanted], [id])
	})

	const lookup = (nodeId: string) => store.liveNodes.get(nodeId)

	function signOut() {
		void store.stop()
		auth.signOut()
	}
</script>

<svelte:head>
	<title>{dashboard ? getBestDesc(dashboard.points) : "Dashboard"} - SIOT</title>
</svelte:head>

<div class="page" class:kiosk>
	{#if !kiosk}
		<Header
			email={auth.user?.email ?? ""}
			badge={store.connected ? "" : "connecting..."}
			onSignOut={signOut}
		/>
	{/if}
	{#if store.error}
		<div class="error" role="alert">
			<span>{store.error}</span>
			<IconButton label="Dismiss" onclick={() => store.dismissError()}><X size={18} /></IconButton>
		</div>
	{/if}
	<main>
		{#if notFound}
			<p class="not-found" role="status">Dashboard {id} was not found.</p>
		{:else if dashboard}
			{#if !kiosk}
				<div class="title">
					<h2>{getBestDesc(dashboard.points)}</h2>
					<a class="tree-link" href={resolve("/nodes/[id]", { id })}>edit in the tree</a>
				</div>
			{/if}
			<Dashboard {dashboard} {children} {lookup} />
		{:else if store.ready}
			<p class="not-found" role="status">Loading...</p>
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

	.kiosk {
		max-width: none;
		padding: 12px;
	}

	.title {
		display: flex;
		flex-wrap: wrap;
		align-items: baseline;
		gap: 16px;
		margin-bottom: 16px;
	}

	h2 {
		margin: 0;
	}

	.tree-link {
		color: var(--muted);
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

	.not-found {
		margin: 0;
		color: var(--muted);
	}
</style>
