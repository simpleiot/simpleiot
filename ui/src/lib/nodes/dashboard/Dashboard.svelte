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
	import type { Point as PointXY } from "#lib/vendor/nautilus/mimic.ts"
	import type { Point } from "#lib/siot/point.ts"
	import {
		parseBindings,
		parseLink,
		parsePorts,
		resolveBindings,
		widgetLayout,
		widgetOptions,
		writeTargets,
		type Layout,
		type Placed,
	} from "./bindings.ts"
	import Canvas from "./Canvas.svelte"
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
		onLink,
		onVertex,
		onWrite,
		resolvePath = () => undefined,
		ancestors = [],
	}: {
		dashboard: Node
		children: readonly Node[]
		lookup: (id: string) => Node | undefined
		// resolvePath turns a relative binding into a node ID
		resolvePath?: (path: string) => string | undefined
		// ancestors are the dashboards this one is nested in, for the
		// cycle guard
		ancestors?: readonly string[]
		edit?: boolean
		selected?: string
		onSelect?: (id: string) => void
		onOpen?: (p: Placed) => void
		onLayout?: (id: string, layout: Layout) => void
		onLink?: (from: { equip: string; port: string }, to: { equip: string; port: string }) => void
		onVertex?: (id: string, index: number, point: PointXY) => void
		// onWrite sends a point to a node, for the widgets that are controls
		onWrite?: (nodeId: string, point: Point) => void
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

	const layout = $derived(getText(dashboard.points, PointType.Layout) || PointValue.Grid)
	const canvas = $derived(layout === PointValue.Canvas)
	const cols = $derived.by(() => {
		const c = Math.floor(Number(getText(dashboard.points, PointType.Cols)) || 0)
		return c > 0 ? c : 12
	})
	const size = $derived.by(() => {
		const w = Number(getText(dashboard.points, PointType.CanvasWidth)) || 0
		const h = Number(getText(dashboard.points, PointType.CanvasHeight)) || 0
		return { width: w > 0 ? w : 900, height: h > 0 ? h : 500 }
	})

	const placed = $derived.by((): Placed[] =>
		children
			.filter((c) => c.type === NodeType.Widget)
			.map((w) => {
				const bindings = parseBindings(w, resolvePath)
				const component = componentOf(w, bindings)
				const descriptor = widgets.get(component)
				const first = Object.values(bindings)[0]
				const bound = first ? lookup(first.nodeId) : undefined
				const own = getText(w.points, PointType.Description)
				// on a canvas the default size is the component's
				const defaults = canvas
					? { x: 20, y: 20, w: descriptor?.width ?? 160, h: 80 }
					: { x: 0, y: 0, w: 3, h: 2 }
				const options = descriptor ? withDefaults(descriptor, widgetOptions(w)) : widgetOptions(w)
				return {
					node: w,
					label: own !== "" ? own : bound ? getBestDesc(bound.points) : "",
					component,
					descriptor,
					values: resolveBindings(bindings, lookup),
					options,
					writes: writeTargets(bindings, lookup, (t) => registry.get(t)?.writeType, options),
					layout: widgetLayout(w, defaults),
					ports: parsePorts(w, descriptor?.ports),
					...(component === nestedWidget && first ? { nested: first.nodeId } : {}),
					ancestors,
				}
			})
	)

	const links = $derived(
		children.filter((c) => c.type === NodeType.Link).map((l) => parseLink(l, lookup))
	)

	// the widget that shows another dashboard in place
	const nestedWidget = "dashboard"
</script>

{#if canvas}
	<Canvas
		width={size.width}
		height={size.height}
		{placed}
		{links}
		{edit}
		{selected}
		{onSelect}
		{onOpen}
		{onLayout}
		{onLink}
		{onVertex}
		{onWrite}
	/>
{:else}
	<Grid {cols} {placed} {edit} {selected} {onSelect} {onOpen} {onLayout} {onWrite} />
{/if}
