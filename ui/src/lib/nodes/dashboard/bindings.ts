// Bindings join a widget's props to points on other nodes. A widget node
// carries, per prop, the node (bindNode), the point type (bindType, "value"
// when absent), and the key (bindKey, "0" when absent). This module reads
// them and resolves them against the nodes a page keeps live.

import type { Node } from "#lib/siot/node.ts"
import { get, getAll, getText, type Point } from "#lib/siot/point.ts"
import { PointType } from "#lib/siot/schema.ts"
import type { Bound, WidgetDescriptor } from "#lib/widgets/registry.ts"

export interface Binding {
	nodeId: string
	type: string
	key: string
}

const live = (points: readonly Point[], type: string) =>
	getAll(points, type).filter((p) => p.tombstone === 0 && p.text !== "")

// parseBindings reads a widget node's bindings, keyed by prop.
export function parseBindings(widget: Node): Record<string, Binding> {
	const out: Record<string, Binding> = {}
	for (const p of live(widget.points, PointType.BindNode)) {
		out[p.key] = {
			nodeId: p.text,
			type: getText(widget.points, PointType.BindType, p.key) || PointType.Value,
			key: getText(widget.points, PointType.BindKey, p.key) || "0",
		}
	}
	return out
}

// boundNodeIds lists the nodes a set of bindings reads, once each.
export function boundNodeIds(bindings: Readonly<Record<string, Binding>>): string[] {
	return [...new Set(Object.values(bindings).map((b) => b.nodeId))]
}

// resolveBindings turns bindings into values, through a lookup of the
// live nodes. A node the lookup does not have is dead; a node without the
// point yet is stale.
export function resolveBindings(
	bindings: Readonly<Record<string, Binding>>,
	lookup: (id: string) => Node | undefined
): Record<string, Bound> {
	const out: Record<string, Bound> = {}
	for (const [prop, b] of Object.entries(bindings)) {
		const n = lookup(b.nodeId)
		if (!n) {
			out[prop] = { state: "dead", value: 0, text: "", time: "" }
			continue
		}
		const p = get(n.points, b.type, b.key)
		if (!p || p.tombstone !== 0) {
			out[prop] = { state: "stale", value: 0, text: "", time: "" }
			continue
		}
		out[prop] = { state: "live", value: p.value, text: p.text, time: p.time }
	}
	return out
}

// widgetOptions reads a widget's prop.* points as display options.
export function widgetOptions(widget: Node): Record<string, string> {
	const out: Record<string, string> = {}
	for (const p of getAll(widget.points, PointType.Prop)) {
		if (p.tombstone === 0) {
			out[p.key] = p.text
		}
	}
	return out
}

export interface Layout {
	x: number
	y: number
	w: number
	h: number
}

// widgetLayout reads a widget's place; a widget with none sits at the
// origin with a default size.
export function widgetLayout(widget: Node, defaults: Layout = { x: 0, y: 0, w: 3, h: 2 }): Layout {
	const n = (type: string, d: number) => {
		const p = get(widget.points, type)
		return p && p.tombstone === 0 && Number.isFinite(p.value) ? p.value : d
	}
	return {
		x: Math.max(0, Math.floor(n(PointType.X, defaults.x))),
		y: Math.max(0, Math.floor(n(PointType.Y, defaults.y))),
		w: Math.max(1, Math.floor(n(PointType.W, defaults.w))),
		h: Math.max(1, Math.floor(n(PointType.H, defaults.h))),
	}
}

// Placed is a widget ready to draw: its node, the widget chosen for it,
// its resolved values and options, and where it sits.
export interface Placed {
	node: Node
	label: string
	component: string
	descriptor: WidgetDescriptor | undefined
	values: Record<string, Bound>
	options: Record<string, string>
	layout: Layout
}
