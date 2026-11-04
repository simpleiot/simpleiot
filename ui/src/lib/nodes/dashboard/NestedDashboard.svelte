<script lang="ts">
	// NestedDashboard shows one dashboard inside another: its widgets, kept
	// live by the store under their own watch key, drawn in view mode. A
	// dashboard that contains itself, however far down, stops here.
	// Relative bindings start from the parent the outer dashboard was
	// opened under, so an overview mirrored under several devices shows
	// each device's nodes in its tiles.
	import { untrack } from "svelte"
	import { dashHref } from "./href.ts"
	import { isTombstone, type Node } from "#lib/siot/node.ts"
	import { getBestDesc, type Point } from "#lib/siot/point.ts"
	import { store } from "#lib/siot/store.ts"
	import { SvelteMap } from "svelte/reactivity"
	import { bindingPaths, boundNodeIds, parseBindings } from "./bindings.ts"
	import Dashboard from "./Dashboard.svelte"

	let {
		id,
		ancestors = [],
		via = "",
	}: {
		id: string
		ancestors?: readonly string[]
		// via is the parent relative bindings start from
		via?: string
	} = $props()

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

	// relative paths are resolved from via, as the dashboard page does
	const pathIds = new SvelteMap<string, string>()
	$effect(() => {
		const from = via
		const paths = new Set(children.flatMap((c) => bindingPaths(c)))
		for (const p of paths) {
			const k = `${from}|${p}`
			if (pathIds.has(k)) {
				continue
			}
			pathIds.set(k, "")
			void store.resolvePath(from, p).then((nodeId) => {
				pathIds.set(k, nodeId ?? "")
			})
		}
	})
	const resolvePath = (p: string) => pathIds.get(`${via}|${p}`) || undefined

	// the watcher is unregistered when this component goes or its key
	// changes, in an effect of its own, so refreshing the wanted set below
	// never drops what was just loaded
	$effect(() => {
		const k = key
		return () => void store.watchNodes([], [], k)
	})

	$effect(() => {
		if (cycle) {
			return
		}
		const wanted = new Set([id, ...childIds])
		for (const c of children) {
			for (const b of boundNodeIds(parseBindings(c, resolvePath))) {
				wanted.add(b)
			}
		}
		void store.watchNodes([...wanted], [], key)
	})

	const lookup = (nodeId: string) => store.liveNodes.get(nodeId)
</script>

<div class="nested">
	{#if cycle}
		<p class="muted">This dashboard contains itself.</p>
	{:else if missing}
		<p class="muted">Dashboard {id} was not found.</p>
	{:else if dashboard}
		<!-- the path is resolved in dashHref; only the query is appended -->
		<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
		<a class="title" href={dashHref(id, via)} title="Open this dashboard"
			>{getBestDesc(dashboard.points)} &rarr;</a
		>
		<Dashboard
			{dashboard}
			{children}
			{lookup}
			{resolvePath}
			{via}
			ancestors={[...ancestors, id]}
			onWrite={(nodeId: string, point: Point) => void store.sendPoints(nodeId, [point])}
		/>
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

	/* the title opens the nested dashboard on its own page */
	.title {
		align-self: flex-start;
		font-size: 13px;
		font-weight: 600;
		color: var(--muted);
		text-decoration: none;
	}

	.title:hover,
	.title:focus-visible {
		color: var(--blue);
		text-decoration: underline;
	}

	.muted {
		margin: 0;
		color: var(--muted);
	}
</style>
