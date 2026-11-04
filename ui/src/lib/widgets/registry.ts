// The widget registry. Each widget is a directory under src/lib/widgets
// holding its component and an index.ts that exports a descriptor made
// with defineWidget; index.ts in this directory collects them. A widget
// takes resolved values and options as props and never talks to the
// server, so it renders the same in a dashboard, an editor preview, and a
// test.

import type { Component } from "svelte"
import type { MimicPort } from "#lib/vendor/nautilus/mimic.ts"

// BindState is how current a bound value is: live when the node and the
// point are there, stale when the node is there but the point is not yet,
// and dead when the node cannot be found.
export type BindState = "live" | "stale" | "dead"

export interface Bound {
	state: BindState
	value: number
	text: string
	// time is the point's time as ISO 8601, "" when there is no point
	time: string
}

// Write is what a control asks to send: a number, or text when text is
// not empty.
export interface Write {
	value: number
	text: string
}

export interface WidgetProps {
	label: string
	// values holds one entry per binding the widget declares
	values: Readonly<Record<string, Bound>>
	// options are the widget's prop.* points, with the descriptor's
	// defaults filled in
	options: Readonly<Record<string, string>>
	// width is the rendered width in canvas units, on a canvas
	width?: number
	// onWrite is set when the bound node accepts writes, and makes the
	// widget a control: it sends a value for one of its bindings
	onWrite?: (prop: string, write: Write) => void
	// control says the widget would have an onWrite outside edit mode,
	// and editing that the dashboard is being edited, so a control can
	// show itself as one while it is inactive
	control?: boolean
	editing?: boolean
}

export interface WidgetDescriptor {
	name: string
	label: string
	// host is where the widget can be placed
	host: "grid" | "canvas" | "both"
	// bindings names the props the widget reads live values for
	bindings: readonly string[]
	// options are the display options and their defaults
	options: Readonly<Record<string, string>>
	// ports are where links attach on a canvas, as fractions of the box
	ports?: readonly MimicPort[]
	// width is the default width on a canvas, in canvas units
	width?: number
	// fit says how a canvas widget follows its width: a drawing takes the
	// width as a prop and draws at that size; anything else is zoomed by
	// the ratio of its width to the default
	fit?: "width" | "zoom"
	component: Component<WidgetProps>
}

export function defineWidget(d: WidgetDescriptor): WidgetDescriptor {
	return d
}

export interface WidgetRegistry {
	all: readonly WidgetDescriptor[]
	get(name: string): WidgetDescriptor | undefined
}

export function createWidgetRegistry(descriptors: readonly WidgetDescriptor[]): WidgetRegistry {
	const byName = new Map(descriptors.map((d) => [d.name, d]))
	const all = descriptors
		.slice()
		.sort((a, b) => (a.label < b.label ? -1 : a.label > b.label ? 1 : 0))
	return { all, get: (name) => byName.get(name) }
}

// autoWidget is the component name that means "pick by the bound node's
// type", and fallbackWidget what is used when the type names none.
export const autoWidget = "auto"
export const fallbackWidget = "value"

// withDefaults fills a widget's options from its descriptor's defaults.
export function withDefaults(
	d: WidgetDescriptor,
	options: Readonly<Record<string, string>>
): Record<string, string> {
	const out: Record<string, string> = { ...d.options }
	for (const [k, v] of Object.entries(options)) {
		if (v !== "") {
			out[k] = v
		}
	}
	return out
}

// wantsConfirm reads the confirm option a control honours: on, yes, true,
// or 1 ask before a write.
export function wantsConfirm(options: Readonly<Record<string, string>>): boolean {
	return ["on", "yes", "true", "1"].includes((options.confirm ?? "").trim().toLowerCase())
}

// num reads a numeric option, or the fallback when it is not a number.
export function num(
	options: Readonly<Record<string, string>>,
	name: string,
	fallback: number
): number {
	const v = Number(options[name])
	return options[name] !== undefined && options[name] !== "" && Number.isFinite(v) ? v : fallback
}
