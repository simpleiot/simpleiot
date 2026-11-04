<script lang="ts">
	// DashView is the page for one dashboard: the node and its widgets,
	// kept live by the store, with the nodes they bind to. In edit mode
	// widgets are dragged, resized, added, and configured in a panel that
	// is the widget node's own editor, with a link to the bound node. In
	// view mode a click does nothing but work a widget's own controls.
	// ?kiosk hides the chrome for a wall display.
	import X from "@lucide/svelte/icons/x"
	import { untrack } from "svelte"
	import { goto } from "$app/navigation"
	import { resolve } from "$app/paths"
	import { page } from "$app/state"
	import { auth } from "#lib/auth.svelte.ts"
	import Header from "#lib/components/Header.svelte"
	import IconButton from "#lib/components/IconButton.svelte"
	import NodeHost from "#lib/components/NodeHost.svelte"
	import { isTombstone, type Node } from "#lib/siot/node.ts"
	import {
		getBestDesc,
		getText,
		newText,
		newValue,
		type Point as SiotPoint,
	} from "#lib/siot/point.ts"
	import { NodeType, PointType, PointValue } from "#lib/siot/schema.ts"
	import type { Point } from "#lib/vendor/nautilus/mimic.ts"
	import { store } from "#lib/siot/store.ts"
	import { SvelteMap } from "svelte/reactivity"
	import {
		bindingPaths,
		boundNodeIds,
		parseBindings,
		widgetLayout,
		type Layout,
	} from "./bindings.ts"
	import Dashboard from "./Dashboard.svelte"
	import { nextRow } from "./gridMath.ts"

	let {
		id,
		select = "",
	}: {
		id: string
		// select opens edit mode with this widget or link selected, from a
		// link to /dash/<id>/<widget>
		select?: string
	} = $props()

	const kiosk = $derived(page.url.searchParams.has("kiosk"))

	let notFound = $state(false)
	let childIds = $state.raw<string[]>([])
	let edit = $state(false)
	let selected = $state("")

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
		if (selected !== "" && !childIds.includes(selected)) {
			selected = ""
		}
	}

	// the address names the selected widget, so its settings can be
	// linked to. A widget in the address, from a link or from going back
	// to it, opens edit mode with its panel.
	$effect(() => {
		const s = select
		if (s !== "" && s !== selected) {
			edit = true
			selected = s
		} else if (s === "" && selected !== "") {
			selected = ""
		}
	})

	function choose(w: string) {
		selected = selected === w ? "" : w
		const path = resolve("/dash/[id]/[[widget]]", { id, widget: selected || undefined })
		// the path is resolved; only the query is carried over
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		void goto(`${path}${page.url.search}`, { replaceState: true })
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

	// via is the parent relative bindings start from: the one in the
	// address, else the parent the dashboard was reached through
	const via = $derived(page.url.searchParams.get("via") ?? dashboard?.parent ?? "")
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

	// the nodes to keep live: the dashboard, its children, and what they
	// bind to
	$effect(() => {
		const wanted = new Set([id, ...childIds])
		for (const c of children) {
			for (const b of boundNodeIds(parseBindings(c, resolvePath))) {
				wanted.add(b)
			}
		}
		void store.watchNodes([...wanted], [id])
	})

	const lookup = (nodeId: string) => store.liveNodes.get(nodeId)

	// ---- editing

	function onLayout(widgetId: string, l: Layout) {
		const t = new Date().toISOString()
		void store.sendPoints(widgetId, [
			newValue(PointType.X, "0", l.x, t),
			newValue(PointType.Y, "0", l.y, t),
			newValue(PointType.W, "0", l.w, t),
			newValue(PointType.H, "0", l.h, t),
		])
	}

	const canvas = $derived(
		dashboard !== undefined && getText(dashboard.points, PointType.Layout) === PointValue.Canvas
	)

	async function addWidget() {
		const t = new Date().toISOString()
		const place = canvas
			? [
					newValue(PointType.X, "0", 20, t),
					newValue(PointType.Y, "0", 20, t),
					newValue(PointType.W, "0", 160, t),
					newValue(PointType.H, "0", 80, t),
				]
			: [
					newValue(PointType.Y, "0", nextRow(children.map((c) => widgetLayout(c))), t),
					newValue(PointType.W, "0", 3, t),
					newValue(PointType.H, "0", 2, t),
				]
		const added = await store.addChild(id, NodeType.Widget, place)
		if (added) {
			await load(id)
			choose(added)
		}
	}

	// duplicateFrom adds a copy of a widget or link under this dashboard,
	// with the same points but placed on the next free row, or offset on a
	// canvas, and opens its panel
	async function duplicateFrom(source: Node) {
		if (source.type !== NodeType.Widget && source.type !== NodeType.Link) {
			return
		}
		const t = new Date().toISOString()
		const l = widgetLayout(source, { x: 20, y: 20, w: 160, h: 80 })
		const place = canvas
			? [newValue(PointType.X, "0", l.x + 20, t), newValue(PointType.Y, "0", l.y + 20, t)]
			: [
					newValue(PointType.X, "0", 0, t),
					newValue(PointType.Y, "0", nextRow(children.map((c) => widgetLayout(c))), t),
				]
		const skip = new Set<string>([PointType.X, PointType.Y])
		const points = [
			newText(PointType.Description, "0", getText(source.points, PointType.Description), t),
			...source.points
				.filter((p) => p.tombstone === 0 && !skip.has(p.type) && p.type !== PointType.Description)
				.map((p) => ({ ...p, time: t })),
			...(source.type === NodeType.Widget ? place : []),
		]
		const added = await store.addChild(id, source.type, points)
		if (added) {
			await load(id)
			choose(added)
		}
	}

	// the clipboard's node, when it is a widget or a link, may be pasted
	let pasteable = $state<Node | null>(null)
	$effect(() => {
		const clip = store.clipboard
		pasteable = null
		if (!clip) {
			return
		}
		void store.resolveNode(clip.id).then((n) => {
			if (n && store.clipboard?.id === clip.id) {
				pasteable = n.type === NodeType.Widget || n.type === NodeType.Link ? n : null
			}
		})
	})

	async function addLink(
		from: { equip: string; port: string },
		to: { equip: string; port: string }
	) {
		const t = new Date().toISOString()
		const added = await store.addChild(id, NodeType.Link, [
			newText(PointType.From, "0", `${from.equip}.${from.port}`, t),
			newText(PointType.To, "0", `${to.equip}.${to.port}`, t),
			newText(PointType.Routing, "0", PointValue.Orthogonal, t),
		])
		if (added) {
			await load(id)
			choose(added)
		}
	}

	function onVertex(linkId: string, index: number, point: Point) {
		void store.sendPoints(linkId, [
			newText(PointType.Vertex, String(index), `${point[0]},${point[1]}`, new Date().toISOString()),
		])
	}

	async function deleteSelected() {
		const w = selected
		if (w === "" || !window.confirm("Delete this item?")) {
			return
		}
		selected = ""
		await store.removeNode(w, id)
		await load(id)
	}

	function signOut() {
		void store.stop()
		auth.signOut()
	}

	const selectedNode = $derived(selected === "" ? undefined : store.liveNodes.get(selected))
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
					<button
						class="btn {edit ? 'gray' : 'blue'}"
						aria-pressed={edit}
						onclick={() => {
							edit = !edit
							if (selected !== "") {
								choose(selected)
							}
						}}>{edit ? "done" : "edit"}</button
					>
					{#if edit}
						<button class="btn green" onclick={() => void addWidget()}>add widget</button>
						{#if pasteable}
							{@const clip = pasteable}
							<button class="btn green" onclick={() => void duplicateFrom(clip)}
								>paste {getBestDesc(clip.points)}</button
							>
						{/if}
					{/if}
					<a class="tree-link" href={resolve("/nodes/[id]", { id })}>in the tree</a>
				</div>
			{/if}
			<div class="layout" class:editing={edit}>
				<div class="board">
					<Dashboard
						{dashboard}
						{children}
						{lookup}
						{resolvePath}
						{via}
						{edit}
						{selected}
						onSelect={choose}
						{onLayout}
						onLink={(from: { equip: string; port: string }, to: { equip: string; port: string }) =>
							void addLink(from, to)}
						{onVertex}
						onWrite={(nodeId: string, point: SiotPoint) => void store.sendPoints(nodeId, [point])}
					/>
				</div>
				{#if edit}
					<aside class="panel" aria-label="Widget settings">
						{#if selectedNode && dashboard}
							<NodeHost
								id={selected}
								parent={{ ...dashboard, parent: via }}
								onDelete={() => void deleteSelected()}
								onCopy={() => store.copyLive(selected)}
								onDuplicate={() => void duplicateFrom(selectedNode)}
							/>
						{:else if canvas}
							<p class="not-found">
								Select a widget or a link to configure it, add a widget, or drag from one port to
								another to draw a link.
							</p>
						{:else}
							<p class="not-found">Select a widget to configure it, or add one.</p>
						{/if}
					</aside>
				{/if}
			</div>
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
		align-items: center;
		gap: 16px;
		margin-bottom: 16px;
	}

	h2 {
		margin: 0;
	}

	.tree-link {
		color: var(--muted);
	}

	.layout {
		display: grid;
		grid-template-columns: minmax(0, 1fr);
		gap: 16px;
	}

	.layout.editing {
		grid-template-columns: minmax(0, 1fr) minmax(280px, 400px);
	}

	.panel {
		position: sticky;
		top: 12px;
		align-self: start;
		max-height: calc(100vh - 24px);
		overflow-y: auto;
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

	@media (max-width: 900px) {
		.layout.editing {
			grid-template-columns: minmax(0, 1fr);
		}

		.panel {
			position: static;
			max-height: none;
		}
	}
</style>
