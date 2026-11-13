// Bindings join a widget's props to points on other nodes. A widget node
// carries, per prop, the node (bindNode), the point type (bindType, "value"
// when absent), and the key (bindKey, "0" when absent). This module reads
// them and resolves them against the nodes a page keeps live.

import type { Node } from "#lib/siot/node.ts"
import { DataType, get, getAll, getText, newText, newValue, type Point } from "#lib/siot/point.ts"
import { PointType } from "#lib/siot/schema.ts"
import type { MimicPort, PortDir } from "#lib/vendor/nautilus/mimic.ts"
import type { Bound, WidgetDescriptor, Write } from "#lib/widgets/registry.ts"
import { parsePair } from "./canvasMath.ts"

export interface Binding {
	nodeId: string
	type: string
	key: string
}

const live = (points: readonly Point[], type: string) =>
	getAll(points, type).filter((p) => p.tombstone === 0 && p.text !== "")

// parseBindings reads a widget node's bindings, keyed by prop. A relative
// binding (bindPath) is resolved through resolvePath when given, and takes
// precedence over the node ID; one that does not resolve is dead. A
// binding without a point type reads the type the widget's descriptor
// gives it in pointTypes, else value.
export function parseBindings(
	widget: Node,
	resolvePath: (path: string) => string | undefined = () => undefined,
	pointTypes: Readonly<Record<string, string>> = {}
): Record<string, Binding> {
	const out: Record<string, Binding> = {}
	const rest = (prop: string) => ({
		type: getText(widget.points, PointType.BindType, prop) || pointTypes[prop] || PointType.Value,
		key: getText(widget.points, PointType.BindKey, prop) || "0",
	})
	for (const p of live(widget.points, PointType.BindNode)) {
		out[p.key] = { nodeId: p.text, ...rest(p.key) }
	}
	for (const p of live(widget.points, PointType.BindPath)) {
		out[p.key] = { nodeId: resolvePath(p.text) ?? "", ...rest(p.key) }
	}
	return out
}

// bindingPaths lists a widget's relative paths, for resolving.
export function bindingPaths(widget: Node): string[] {
	return live(widget.points, PointType.BindPath).map((p) => p.text)
}

// writeTargets picks the bindings a widget may write through: those that
// read the value of a node whose type accepts writes, unless the widget's
// control option is off. The target is the node's write point under the
// binding's key. A write option names the point type to write instead,
// for a node such as an MCU that takes many settings under their own
// types: every binding to a node that is there then writes that type
// under its key, whatever the node's type.
export function writeTargets(
	bindings: Readonly<Record<string, Binding>>,
	lookup: (id: string) => Node | undefined,
	writeTypeOf: (nodeType: string) => string | undefined,
	options: Readonly<Record<string, string>>
): Record<string, Binding> {
	const out: Record<string, Binding> = {}
	if (options.control === "off") {
		return out
	}
	const explicit = (options.write ?? "").trim()
	for (const [prop, b] of Object.entries(bindings)) {
		const n = lookup(b.nodeId)
		if (n && explicit !== "") {
			out[prop] = { nodeId: b.nodeId, type: explicit, key: b.key }
			continue
		}
		const type = n ? writeTypeOf(n.type) : undefined
		if (n && type !== undefined && b.type === PointType.Value) {
			out[prop] = { nodeId: b.nodeId, type, key: b.key }
		}
	}
	return out
}

// writePoint builds the point a control sends to a write target. A number
// goes out as an integer when the integer option is on, or when the bound
// point is one, so a device that reads the point as an integer, as an MCU
// setting often is, gets the value it expects rather than the bits of a
// float. The option is what to rely on for a point the device only
// receives, whose stored type is whatever was last written to it.
export function writePoint(
	target: Binding,
	bound: Bound | undefined,
	w: Write,
	time: string,
	options: Readonly<Record<string, string>> = {}
): Point {
	if (w.text !== "") {
		return newText(target.type, target.key, w.text, time)
	}
	const integer =
		options.integer === "on" || (bound?.state === "live" && bound.dataType === DataType.Int)
	if (integer) {
		return {
			...newValue(target.type, target.key, Math.round(w.value), time),
			dataType: DataType.Int,
		}
	}
	return newValue(target.type, target.key, w.value, time)
}

// openTargets picks the bindings that name a dashboard, as isDashboard
// says of the bound node's type, with the address each one opens.
export function openTargets(
	bindings: Readonly<Record<string, Binding>>,
	lookup: (id: string) => Node | undefined,
	isDashboard: (nodeType: string) => boolean,
	href: (dashboardId: string) => string
): Record<string, string> {
	const out: Record<string, string> = {}
	for (const [prop, b] of Object.entries(bindings)) {
		const n = lookup(b.nodeId)
		if (n && isDashboard(n.type)) {
			out[prop] = href(b.nodeId)
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
		out[prop] = {
			state: "live",
			value: p.value,
			text: p.text,
			time: p.time,
			dataType: p.dataType,
		}
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
// its resolved values and options, where it sits, and its ports.
export interface Placed {
	node: Node
	label: string
	component: string
	descriptor: WidgetDescriptor | undefined
	values: Record<string, Bound>
	options: Record<string, string>
	layout: Layout
	// z is the widget's place in the stack on a canvas, higher on top
	z: number
	ports: MimicPort[]
	// writes are the bindings the widget may write through, as controls
	writes: Record<string, Binding>
	// opens are the addresses of the dashboards the widget's bindings name,
	// which a click on the widget opens
	opens: Record<string, string>
	// nested is the dashboard a nested-dashboard widget shows
	nested?: string
	// ancestors are the dashboards this one is drawn inside, outermost first
	ancestors: readonly string[]
	// via is the parent relative bindings start from, which a nested
	// dashboard takes over from the one it is drawn in
	via: string
}

const dirs: readonly PortDir[] = ["left", "right", "up", "down"]

// parsePorts reads a widget's port.* overrides ("x,y" or "x,y,dir") over
// the component's defaults. A port named with an empty text removes the
// default of that name.
export function parsePorts(widget: Node, defaults: readonly MimicPort[] = []): MimicPort[] {
	const out = new Map(defaults.map((p) => [p.name, { ...p }]))
	for (const p of getAll(widget.points, PointType.Port)) {
		if (p.tombstone !== 0) {
			continue
		}
		if (p.text.trim() === "") {
			out.delete(p.key)
			continue
		}
		const parts = p.text.split(",").map((s) => s.trim())
		const pair = parsePair(parts.slice(0, 2).join(","))
		if (!pair) {
			continue
		}
		const dir = dirs.find((d) => d === parts[2])
		out.set(p.key, { name: p.key, x: pair[0], y: pair[1], ...(dir ? { dir } : {}) })
	}
	return [...out.values()]
}

// PlacedLink is a link ready to draw: its ends, its interior vertices in
// order, its routing, and the flow its bindings report.
export interface PlacedLink {
	node: Node
	from?: { equip: string; port: string }
	to?: { equip: string; port: string }
	routing?: "direct" | "orthogonal"
	points: [number, number][]
	flowing: boolean
	rate: number
	dead: boolean
}

function parseEnd(text: string): { equip: string; port: string } | undefined {
	const i = text.lastIndexOf(".")
	if (i <= 0 || i === text.length - 1) {
		return undefined
	}
	return { equip: text.slice(0, i), port: text.slice(i + 1) }
}

// parseLink reads a link node. Vertices are vertex.<index> points, "x,y",
// in index order; a link with no flowing binding is drawn still, and one
// whose bound node is missing is drawn dead.
export function parseLink(link: Node, lookup: (id: string) => Node | undefined): PlacedLink {
	const points = getAll(link.points, PointType.Vertex)
		.filter((p) => p.tombstone === 0)
		.map((p) => ({ i: Number(p.key), pair: parsePair(p.text) }))
		.filter(
			(v): v is { i: number; pair: [number, number] } =>
				v.pair !== undefined && Number.isFinite(v.i)
		)
		.sort((a, b) => a.i - b.i)
		.map((v) => v.pair)
	const routing = getText(link.points, PointType.Routing)
	const values = resolveBindings(parseBindings(link), lookup)
	const flowing = values.flowing
	const rate = values.rate
	return {
		node: link,
		from: parseEnd(getText(link.points, PointType.From)),
		to: parseEnd(getText(link.points, PointType.To)),
		routing: routing === "orthogonal" ? "orthogonal" : "direct",
		points,
		flowing: flowing?.state === "live" && flowing.value !== 0,
		rate: rate?.state === "live" ? Math.max(0, Math.min(1, rate.value)) : 1,
		dead: Object.values(values).some((v) => v.state === "dead"),
	}
}
