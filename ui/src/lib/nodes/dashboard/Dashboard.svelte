<script lang="ts">
	// Dashboard renders a dashboard node and its children through a layout
	// host. It takes nodes and a lookup for the bound ones, so it draws the
	// same from a page, a preview, or a test.
	import type { Node } from "#lib/siot/node.ts"
	import { getBestDesc, getText } from "#lib/siot/point.ts"
	import { NodeType, PointType, PointValue } from "#lib/siot/schema.ts"
	import { widgets } from "#lib/widgets/index.ts"
	import { autoWidget, fallbackWidget, withDefaults } from "#lib/widgets/registry.ts"
	import { registry } from "../index.ts"
	import {
		parseBindings,
		resolveBindings,
		widgetLayout,
		widgetOptions,
		type Layout,
		type Placed,
	} from "./bindings.ts"
	import Grid from "./Grid.svelte"

	let {
		dashboard,
		children,
		lookup,
		edit = false,
		selected = "",
		onSelect,
		onOpen,
		onLayout,
	}: {
		dashboard: Node
		children: readonly Node[]
		lookup: (id: string) => Node | undefined
		edit?: boolean
		selected?: string
		onSelect?: (id: string) => void
		onOpen?: (p: Placed) => void
		onLayout?: (id: string, layout: Layout) => void
	} = $props()

	// componentOf picks the widget for a node: its component, or for
	// "auto" the one the bound node's type names.
	function componentOf(w: Node, bindings: ReturnType<typeof parseBindings>): string {
		const named = getText(w.points, PointType.Component) || autoWidget
		if (named !== autoWidget) {
			return named
		}
		const first = Object.values(bindings)[0]
		const bound = first ? lookup(first.nodeId) : undefined
		return (bound && registry.get(bound.type)?.widget) || fallbackWidget
	}

	const placed = $derived.by((): Placed[] =>
		children
			.filter((c) => c.type === NodeType.Widget)
			.map((w) => {
				const bindings = parseBindings(w)
				const component = componentOf(w, bindings)
				const descriptor = widgets.get(component)
				const first = Object.values(bindings)[0]
				const bound = first ? lookup(first.nodeId) : undefined
				const own = getText(w.points, PointType.Description)
				return {
					node: w,
					label: own !== "" ? own : bound ? getBestDesc(bound.points) : "",
					component,
					descriptor,
					values: resolveBindings(bindings, lookup),
					options: descriptor ? withDefaults(descriptor, widgetOptions(w)) : widgetOptions(w),
					layout: widgetLayout(w),
				}
			})
	)

	const layout = $derived(getText(dashboard.points, PointType.Layout) || PointValue.Grid)
	const cols = $derived.by(() => {
		const c = Math.floor(Number(getText(dashboard.points, PointType.Cols)) || 0)
		return c > 0 ? c : 12
	})
</script>

{#if layout === PointValue.Canvas}
	<p class="pending">Canvas layouts are not drawn yet; showing the widgets as a grid.</p>
{/if}
<Grid {cols} {placed} {edit} {selected} {onSelect} {onOpen} {onLayout} />

<style>
	.pending {
		margin: 0 0 12px;
		color: var(--muted);
	}
</style>
