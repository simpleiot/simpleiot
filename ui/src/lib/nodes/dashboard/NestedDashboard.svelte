<script lang="ts">
	// NestedDashboard shows one dashboard inside another: its widgets, kept
	// live by the store under their own watch key, drawn in view mode. A
	// dashboard that contains itself, however far down, stops here.
	import { untrack } from "svelte"
	import { isTombstone, type Node } from "#lib/siot/node.ts"
	import { getBestDesc } from "#lib/siot/point.ts"
	import { store } from "#lib/siot/store.ts"
	import { boundNodeIds, parseBindings } from "./bindings.ts"
	import Dashboard from "./Dashboard.svelte"

	let { id, ancestors = [] }: { id: string; ancestors?: readonly string[] } = $props()

	const cycle = $derived(ancestors.includes(id))
	const key = $derived(`nested:${ancestors.join("/")}/${id}`)

	let childIds = $state.raw<string[]>([])
	let missing = $state(false)

	let loading = 0
	async function load(dashId: string) {
		const mine = ++loading
		const r = await store.fetchSubtree(dashId, 1)
		if (mine !== loading) {
			return
		}
		missing = !r
		childIds = r
			? r.nodes.filter((n) => n.parent === dashId && !isTombstone(n)).map((n) => n.id)
			: []
	}

	$effect(() => {
		const dashId = id
		if (cycle) {
			return
		}
		if (store.ready) {
			untrack(() => void load(dashId))
		}
	})

	const dashboard = $derived(store.liveNodes.get(id))
	const children = $derived(
		childIds.map((c) => store.liveNodes.get(c)).filter((n): n is Node => n !== undefined)
	)

	$effect(() => {
		const k = key
		if (cycle) {
			return
		}
		const wanted = new Set([id, ...childIds])
		for (const c of children) {
			for (const b of boundNodeIds(parseBindings(c))) {
				wanted.add(b)
			}
		}
		void store.watchNodes([...wanted], [], k)
		return () => void store.watchNodes([], [], k)
	})

	const lookup = (nodeId: string) => store.liveNodes.get(nodeId)
</script>

<div class="nested">
	{#if cycle}
		<p class="muted">This dashboard contains itself.</p>
	{:else if missing}
		<p class="muted">Dashboard {id} was not found.</p>
	{:else if dashboard}
		<div class="title">{getBestDesc(dashboard.points)}</div>
		<Dashboard {dashboard} {children} {lookup} ancestors={[...ancestors, id]} />
	{:else}
		<p class="muted">Loading...</p>
	{/if}
</div>

<style>
	.nested {
		display: flex;
		flex-direction: column;
		gap: 8px;
		height: 100%;
		overflow: auto;
	}

	.title {
		font-size: 13px;
		font-weight: 600;
		color: var(--muted);
	}

	.muted {
		margin: 0;
		color: var(--muted);
	}
</style>
