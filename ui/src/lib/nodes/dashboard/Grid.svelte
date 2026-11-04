<script lang="ts">
	// Grid is the widget host for tiles: columns and rows, each widget
	// spanning its place. Below the breakpoint it becomes one column in
	// tree order. In edit mode a widget is dragged by its body and resized
	// by its corner, and the new place is reported when the pointer lifts.
	import type { Point } from "#lib/siot/point.ts"
	import type { Layout, Placed } from "./bindings.ts"
	import { cellDelta, metrics, moved, sameLayout } from "./gridMath.ts"
	import WidgetFrame from "./WidgetFrame.svelte"

	let {
		cols,
		placed,
		edit = false,
		selected = "",
		onSelect,
		onLayout,
		onWrite,
	}: {
		cols: number
		placed: readonly Placed[]
		edit?: boolean
		// selected is the widget whose panel is open in edit mode
		selected?: string
		onSelect?: (id: string) => void
		onLayout?: (id: string, layout: Layout) => void
		onWrite?: (nodeId: string, point: Point) => void
	} = $props()

	let gridEl: HTMLDivElement | undefined = $state()

	interface Drag {
		id: string
		mode: "move" | "resize"
		startX: number
		startY: number
		start: Layout
		cur: Layout
		moved: boolean
	}
	let drag = $state<Drag | null>(null)

	function layoutOf(p: Placed): Layout {
		return drag?.id === p.node.id ? drag.cur : p.layout
	}

	function down(e: PointerEvent, p: Placed, mode: "move" | "resize") {
		if (!edit || e.button !== 0) {
			return
		}
		e.preventDefault()
		;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
		drag = {
			id: p.node.id,
			mode,
			startX: e.clientX,
			startY: e.clientY,
			start: p.layout,
			cur: p.layout,
			moved: false,
		}
	}

	function move(e: PointerEvent) {
		if (!drag || !gridEl) {
			return
		}
		const m = metrics(gridEl.clientWidth, cols)
		const { dc, dr } = cellDelta(e.clientX - drag.startX, e.clientY - drag.startY, m)
		const next = moved(drag.start, drag.mode, dc, dr, cols)
		if (!sameLayout(next, drag.cur)) {
			drag = { ...drag, cur: next, moved: true }
		}
	}

	function up() {
		const d = drag
		drag = null
		if (!d) {
			return
		}
		if (d.moved && !sameLayout(d.cur, d.start)) {
			onLayout?.(d.id, d.cur)
		} else if (d.mode === "move") {
			onSelect?.(d.id)
		}
	}
</script>

<div class="grid" class:edit style:--cols={cols} bind:this={gridEl}>
	{#each placed as p (p.node.id)}
		{@const l = layoutOf(p)}
		<div
			class="cell"
			class:selected={edit && selected === p.node.id}
			class:dragging={drag?.id === p.node.id}
			style:--x={Math.min(l.x, cols - 1)}
			style:--w={Math.min(l.w, cols - Math.min(l.x, cols - 1))}
			style:--y={l.y}
			style:--h={l.h}
		>
			{#if edit}
				<!-- svelte-ignore a11y_no_static_element_interactions -->
				<div
					class="handle move"
					onpointerdown={(e) => down(e, p, "move")}
					onpointermove={move}
					onpointerup={up}
					onpointercancel={up}
				>
					<WidgetFrame {p} />
				</div>
				<!-- svelte-ignore a11y_no_static_element_interactions -->
				<div
					class="handle resize"
					title="Resize"
					onpointerdown={(e) => down(e, p, "resize")}
					onpointermove={move}
					onpointerup={up}
					onpointercancel={up}
				></div>
			{:else}
				<WidgetFrame {p} {onWrite} />
			{/if}
		</div>
	{/each}
	{#if placed.length === 0}
		<p class="empty">This dashboard has no widgets yet.</p>
	{/if}
</div>

<style>
	.grid {
		display: grid;
		grid-template-columns: repeat(var(--cols), minmax(0, 1fr));
		grid-auto-rows: 72px;
		grid-auto-flow: dense;
		gap: 12px;
	}

	.cell {
		position: relative;
		grid-column: calc(var(--x) + 1) / span var(--w);
		grid-row: calc(var(--y) + 1) / span var(--h);
		min-width: 0;
	}

	.cell.selected {
		outline: 2px solid var(--blue);
		outline-offset: 2px;
		border-radius: 8px;
	}

	.cell.dragging {
		opacity: 0.8;
		z-index: 2;
	}

	.handle.move {
		height: 100%;
		cursor: grab;
		touch-action: none;
	}

	.dragging .handle.move {
		cursor: grabbing;
	}

	.handle.resize {
		position: absolute;
		right: 2px;
		bottom: 2px;
		width: 18px;
		height: 18px;
		cursor: nwse-resize;
		touch-action: none;
		border-right: 3px solid var(--muted);
		border-bottom: 3px solid var(--muted);
		border-radius: 0 0 6px 0;
	}

	.empty {
		grid-column: 1 / -1;
		color: var(--muted);
	}

	@media (max-width: 900px) {
		.grid {
			grid-template-columns: 1fr;
			grid-auto-rows: auto;
		}

		.cell {
			grid-column: auto;
			grid-row: auto;
			min-height: calc(var(--h) * 72px);
		}
	}
</style>
