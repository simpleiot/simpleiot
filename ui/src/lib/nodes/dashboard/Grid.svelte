<script lang="ts">
	// Grid is the widget host for tiles: columns and rows, each widget
	// spanning its place. Below the breakpoint it becomes one column in
	// tree order.
	import type { Placed } from "./bindings.ts"
	import WidgetFrame from "./WidgetFrame.svelte"

	let { cols, placed }: { cols: number; placed: readonly Placed[] } = $props()
</script>

<div class="grid" style:--cols={cols}>
	{#each placed as p (p.node.id)}
		<div
			class="cell"
			style:--x={Math.min(p.layout.x, cols - 1)}
			style:--w={Math.min(p.layout.w, cols - Math.min(p.layout.x, cols - 1))}
			style:--y={p.layout.y}
			style:--h={p.layout.h}
		>
			<WidgetFrame {p} />
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
		grid-column: calc(var(--x) + 1) / span var(--w);
		grid-row: calc(var(--y) + 1) / span var(--h);
		min-width: 0;
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
