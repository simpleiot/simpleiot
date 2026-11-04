<script lang="ts">
	// Canvas is the widget host for mimics: a plane in canvas units, scaled
	// to the frame, panned by dragging the background and zoomed with the
	// buttons or ctrl and the wheel. Links are drawn under the equipment,
	// anchored to ports so they follow a moved widget. In edit mode
	// equipment is dragged, a link is drawn from port to port, and the
	// selected link's vertices are dragged. Below the breakpoint the plane
	// is replaced by one column of cards.
	import Minus from "@lucide/svelte/icons/minus"
	import Plus from "@lucide/svelte/icons/plus"
	import Maximize from "@lucide/svelte/icons/maximize"
	import { SvelteMap } from "svelte/reactivity"
	import IconButton from "#lib/components/IconButton.svelte"
	import Pipe from "#lib/vendor/nautilus/Pipe.svelte"
	import {
		makeGetPort,
		pointsToPath,
		portAbsolute,
		routedPoints,
		type EquipmentBox,
		type Point,
	} from "#lib/vendor/nautilus/mimic.ts"
	import type { Layout, Placed, PlacedLink } from "./bindings.ts"
	import { canvasDelta, fitScale, toCanvas, zoomAt, type View } from "./canvasMath.ts"
	import Grid from "./Grid.svelte"
	import WidgetFrame from "./WidgetFrame.svelte"

	let {
		width,
		height,
		placed,
		links,
		edit = false,
		selected = "",
		onSelect,
		onOpen,
		onLayout,
		onLink,
		onVertex,
	}: {
		width: number
		height: number
		placed: readonly Placed[]
		links: readonly PlacedLink[]
		edit?: boolean
		selected?: string
		onSelect?: (id: string) => void
		onOpen?: (p: Placed) => void
		onLayout?: (id: string, layout: Layout) => void
		// onLink reports a link drawn from one port to another
		onLink?: (from: { equip: string; port: string }, to: { equip: string; port: string }) => void
		// onVertex reports a moved vertex of a link
		onVertex?: (id: string, index: number, point: Point) => void
	} = $props()

	// ---- the frame and the view

	let frame: HTMLDivElement | undefined = $state()
	let frameWidth = $state(0)
	let narrow = $state(false)
	$effect(() => {
		const el = frame
		if (!el) {
			return
		}
		const update = () => (frameWidth = el.clientWidth)
		update()
		const ro = new ResizeObserver(update)
		ro.observe(el)
		const mq = window.matchMedia("(max-width: 900px)")
		const onMq = () => (narrow = mq.matches)
		onMq()
		mq.addEventListener("change", onMq)
		return () => {
			ro.disconnect()
			mq.removeEventListener("change", onMq)
		}
	})

	const fit = $derived(fitScale(frameWidth, width))
	let zoom = $state(1)
	let pan = $state({ x: 0, y: 0 })
	const view = $derived<View>({ scale: fit * zoom, panX: pan.x, panY: pan.y })

	function zoomBy(factor: number) {
		const fx = frameWidth / 2
		const fy = (height * fit) / 2
		const next = zoomAt(view, factor, fx, fy, fit)
		zoom = next.scale / fit
		pan = { x: next.panX, y: next.panY }
	}

	function onwheel(e: WheelEvent) {
		if (!e.ctrlKey || !frame) {
			return
		}
		e.preventDefault()
		const r = frame.getBoundingClientRect()
		const next = zoomAt(
			view,
			e.deltaY < 0 ? 1.15 : 1 / 1.15,
			e.clientX - r.left,
			e.clientY - r.top,
			fit
		)
		zoom = next.scale / fit
		pan = { x: next.panX, y: next.panY }
	}

	function framePoint(e: PointerEvent): [number, number] {
		const r = frame?.getBoundingClientRect()
		return r ? [e.clientX - r.left, e.clientY - r.top] : [e.clientX, e.clientY]
	}

	// ---- boxes and ports

	// sizes are each equipment's rendered box in canvas units, measured
	// from the DOM, so an anchored link follows a component that changes
	// size with its bound props
	const sizes = new SvelteMap<string, { w: number; h: number }>()
	function measure(node: HTMLElement, id: string) {
		const update = () => sizes.set(id, { w: node.offsetWidth, h: node.offsetHeight })
		update()
		const ro = new ResizeObserver(update)
		ro.observe(node)
		return {
			destroy() {
				sizes.delete(id)
				ro.disconnect()
			},
		}
	}

	function layoutOf(p: Placed): Layout {
		return drag?.kind === "move" && drag.id === p.node.id ? drag.cur : p.layout
	}

	function boxOf(p: Placed): EquipmentBox {
		const size = sizes.get(p.node.id)
		const l = layoutOf(p)
		return { x: l.x, y: l.y, w: size?.w || l.w, h: size?.h || l.h }
	}

	const byId = $derived(new Map(placed.map((p) => [p.node.id, p])))
	const getPort = makeGetPort(
		(id) => {
			const p = byId.get(id)
			return p ? boxOf(p) : undefined
		},
		(id) => byId.get(id)?.ports
	)

	// every port on the plane, for snapping a drawn link
	const allPorts = $derived(
		placed.flatMap((p) =>
			p.ports.map((port) => ({
				equip: p.node.id,
				port: port.name,
				...portAbsolute(boxOf(p), port),
			}))
		)
	)

	function nearestPort(x: number, y: number) {
		let best: (typeof allPorts)[number] | undefined
		let bestD = 14 / view.scale + 8
		for (const port of allPorts) {
			const d = Math.hypot(port.x - x, port.y - y)
			if (d < bestD) {
				best = port
				bestD = d
			}
		}
		return best
	}

	const paths = $derived(
		links.map((l) => {
			const points = drag?.kind === "vertex" && drag.id === l.node.id ? drag.points : l.points
			return {
				link: l,
				d: pointsToPath(routedPoints({ ...l, points }, getPort)),
			}
		})
	)

	// ---- dragging: a widget, the background, a new link, or a vertex

	type Drag =
		| {
				kind: "move"
				id: string
				sx: number
				sy: number
				start: Layout
				cur: Layout
				moved: boolean
		  }
		| { kind: "pan"; sx: number; sy: number; start: { x: number; y: number } }
		| { kind: "link"; from: { equip: string; port: string }; at: Point; to: Point }
		| {
				kind: "vertex"
				id: string
				index: number
				sx: number
				sy: number
				points: Point[]
				start: Point
		  }
	let drag = $state<Drag | null>(null)

	function capture(e: PointerEvent) {
		;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
	}

	function downWidget(e: PointerEvent, p: Placed) {
		if (!edit || e.button !== 0) {
			return
		}
		e.stopPropagation()
		e.preventDefault()
		capture(e)
		drag = {
			kind: "move",
			id: p.node.id,
			sx: e.clientX,
			sy: e.clientY,
			start: p.layout,
			cur: p.layout,
			moved: false,
		}
	}

	function downPort(e: PointerEvent, equip: string, port: string) {
		if (!edit || e.button !== 0) {
			return
		}
		e.stopPropagation()
		e.preventDefault()
		capture(e)
		const at = getPort(equip, port)
		if (at) {
			drag = { kind: "link", from: { equip, port }, at: [at.x, at.y], to: [at.x, at.y] }
		}
	}

	function downVertex(e: PointerEvent, l: PlacedLink, index: number) {
		const start = l.points[index]
		if (!edit || e.button !== 0 || !start) {
			return
		}
		e.stopPropagation()
		e.preventDefault()
		capture(e)
		drag = {
			kind: "vertex",
			id: l.node.id,
			index,
			sx: e.clientX,
			sy: e.clientY,
			points: l.points.slice(),
			start,
		}
	}

	function downBackground(e: PointerEvent) {
		if (e.button !== 0) {
			return
		}
		capture(e)
		drag = { kind: "pan", sx: e.clientX, sy: e.clientY, start: pan }
	}

	function move(e: PointerEvent) {
		const d = drag
		if (!d) {
			return
		}
		switch (d.kind) {
			case "move": {
				const [dx, dy] = canvasDelta(e.clientX - d.sx, e.clientY - d.sy, view)
				const cur = { ...d.start, x: Math.max(0, d.start.x + dx), y: Math.max(0, d.start.y + dy) }
				if (cur.x !== d.cur.x || cur.y !== d.cur.y) {
					drag = { ...d, cur, moved: true }
				}
				break
			}
			case "pan":
				pan = { x: d.start.x + e.clientX - d.sx, y: d.start.y + e.clientY - d.sy }
				break
			case "link": {
				const [fx, fy] = framePoint(e)
				drag = { ...d, to: toCanvas(fx, fy, view) }
				break
			}
			case "vertex": {
				const [dx, dy] = canvasDelta(e.clientX - d.sx, e.clientY - d.sy, view)
				const points = d.points.slice()
				points[d.index] = [d.start[0] + dx, d.start[1] + dy]
				drag = { ...d, points }
				break
			}
		}
	}

	function up() {
		const d = drag
		drag = null
		if (!d) {
			return
		}
		switch (d.kind) {
			case "move":
				if (d.moved) {
					onLayout?.(d.id, d.cur)
				} else {
					onSelect?.(d.id)
				}
				break
			case "link": {
				const to = nearestPort(d.to[0], d.to[1])
				if (to && (to.equip !== d.from.equip || to.port !== d.from.port)) {
					onLink?.(d.from, { equip: to.equip, port: to.port })
				}
				break
			}
			case "vertex": {
				const p = d.points[d.index]
				if (p && (p[0] !== d.start[0] || p[1] !== d.start[1])) {
					onVertex?.(d.id, d.index, p)
				}
				break
			}
			case "pan":
				break
		}
	}

	// the card fallback gives every widget the same short height
	const cards = $derived(placed.map((p) => ({ ...p, layout: { x: 0, y: 0, w: 1, h: 3 } })))
	const layers = ["wall", "bore", "flow"] as const
</script>

{#if narrow}
	<Grid cols={1} placed={cards} {edit} {selected} {onSelect} {onOpen} />
{:else}
	<div class="canvas-host">
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			class="frame"
			class:edit
			bind:this={frame}
			style:height="{height * fit}px"
			{onwheel}
			onpointerdown={downBackground}
			onpointermove={move}
			onpointerup={up}
			onpointercancel={up}
		>
			<div
				class="plane"
				style:width="{width}px"
				style:height="{height}px"
				style:transform="translate({pan.x}px, {pan.y}px) scale({view.scale})"
			>
				<svg class="links" viewBox="0 0 {width} {height}" {width} {height}>
					{#each layers as layer (layer)}
						{#each paths as { link, d } (link.node.id + layer)}
							<Pipe {d} flowing={link.flowing} rate={link.rate} dead={link.dead} {layer} />
						{/each}
					{/each}
					{#if edit}
						{#each paths as { link, d } (link.node.id)}
							<!-- svelte-ignore a11y_no_static_element_interactions -->
							<path
								class="hit"
								class:selected={selected === link.node.id}
								{d}
								onpointerdown={(e) => {
									e.stopPropagation()
									onSelect?.(link.node.id)
								}}
							/>
							{#if selected === link.node.id}
								{#each drag?.kind === "vertex" && drag.id === link.node.id ? drag.points : link.points as v, i (i)}
									<!-- svelte-ignore a11y_no_static_element_interactions -->
									<circle
										class="vertex"
										cx={v[0]}
										cy={v[1]}
										r="6"
										onpointerdown={(e) => downVertex(e, link, i)}
										onpointermove={move}
										onpointerup={up}
										onpointercancel={up}
									/>
								{/each}
							{/if}
						{/each}
						{#if drag?.kind === "link"}
							<path class="drawing" d={pointsToPath([drag.at, drag.to])} />
						{/if}
					{/if}
				</svg>
				{#each placed as p (p.node.id)}
					{@const l = layoutOf(p)}
					<!-- svelte-ignore a11y_no_static_element_interactions -->
					<div
						class="equipment"
						class:selected={edit && selected === p.node.id}
						style:left="{l.x}px"
						style:top="{l.y}px"
						style:width="{l.w}px"
						use:measure={p.node.id}
						onpointerdown={(e) => downWidget(e, p)}
						onpointermove={move}
						onpointerup={up}
						onpointercancel={up}
					>
						{#if edit || !onOpen}
							<WidgetFrame {p} bare />
						{:else}
							<button
								class="open"
								type="button"
								onclick={() => onOpen(p)}
								aria-label="Open {p.label}"
							>
								<WidgetFrame {p} bare />
							</button>
						{/if}
						{#if edit}
							{#each p.ports as port (port.name)}
								<!-- svelte-ignore a11y_no_static_element_interactions -->
								<div
									class="port"
									title={port.name}
									style:left="{port.x * 100}%"
									style:top="{port.y * 100}%"
									onpointerdown={(e) => downPort(e, p.node.id, port.name)}
									onpointermove={move}
									onpointerup={up}
									onpointercancel={up}
								></div>
							{/each}
						{/if}
					</div>
				{/each}
			</div>
			{#if placed.length === 0}
				<p class="empty">This dashboard has no widgets yet.</p>
			{/if}
		</div>
		<div class="zoom">
			<IconButton label="Zoom in" onclick={() => zoomBy(1.25)}><Plus size={18} /></IconButton>
			<IconButton label="Zoom out" onclick={() => zoomBy(1 / 1.25)}><Minus size={18} /></IconButton>
			<IconButton
				label="Fit"
				onclick={() => {
					zoom = 1
					pan = { x: 0, y: 0 }
				}}><Maximize size={18} /></IconButton
			>
		</div>
	</div>
{/if}

<style>
	.canvas-host {
		position: relative;
	}

	.frame {
		position: relative;
		overflow: hidden;
		border: 1px solid var(--border);
		border-radius: 8px;
		background: var(--detail-bg);
		touch-action: none;
		cursor: grab;
	}

	.plane {
		position: absolute;
		top: 0;
		left: 0;
		transform-origin: 0 0;
	}

	.links {
		position: absolute;
		top: 0;
		left: 0;
		overflow: visible;
	}

	.hit {
		fill: none;
		stroke: transparent;
		stroke-width: 16;
		cursor: pointer;
		pointer-events: stroke;
	}

	.hit.selected {
		stroke: var(--focus);
	}

	.vertex {
		fill: var(--bg);
		stroke: var(--blue);
		stroke-width: 2;
		cursor: move;
	}

	.drawing {
		fill: none;
		stroke: var(--blue);
		stroke-width: 3;
		stroke-dasharray: 6 6;
	}

	.equipment {
		position: absolute;
	}

	.edit .equipment {
		cursor: grab;
	}

	.equipment.selected {
		outline: 2px solid var(--blue);
		outline-offset: 4px;
		border-radius: 6px;
	}

	.port {
		position: absolute;
		width: 12px;
		height: 12px;
		margin: -6px 0 0 -6px;
		border-radius: 50%;
		background: var(--bg);
		border: 2px solid var(--blue);
		cursor: crosshair;
	}

	.open {
		display: block;
		padding: 0;
		border: none;
		background: none;
		font: inherit;
		color: inherit;
		cursor: pointer;
	}

	.empty {
		position: absolute;
		top: 12px;
		left: 12px;
		margin: 0;
		color: var(--muted);
	}

	.zoom {
		position: absolute;
		right: 8px;
		bottom: 8px;
		display: flex;
		gap: 2px;
		padding: 2px;
		border-radius: 20px;
		background: var(--bg);
		border: 1px solid var(--border);
	}
</style>
