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
	// dataType is the point's wire data type, when there is a point
	dataType?: number
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
	// writable names the bindings onWrite accepts, for a widget with more
	// than one
	writable?: readonly string[]
	// opens holds, per binding that names a dashboard, the address of that
	// dashboard, which a click on the widget opens. The host gives none in
	// edit mode, where a click selects the widget.
	opens?: Readonly<Record<string, string>>
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
	// bindingPoints names the point type a binding reads when the widget
	// node does not say: a drawing reads a file node's data, for instance.
	// Any binding not listed reads value.
	bindingPoints?: Readonly<Record<string, string>>
	// source is the binding that holds a document the widget draws, and
	// bindingsIn and portsIn read the further bindings and the ports that
	// document declares, for the editor and the dashboard
	source?: string
	bindingsIn?: (text: string) => readonly DeclaredBinding[]
	portsIn?: (text: string) => readonly MimicPort[]
	component: Component<WidgetProps>
}

// DeclaredBinding is a binding a document names, with what the document
// does with it, in words the editor can show: "toggle", "text, rotate"
export interface DeclaredBinding {
	name: string
	use: string
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

// navigateBinding is the binding any widget may take: the dashboard a
// click on the widget opens, where the widget does not act on the click
// itself.
export const navigateBinding = "navigate"

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

// multiplier reads the multiplier option, which scales a reading for
// display: seconds shown as days, bytes as megabytes. It is 1 when the
// option is not set, not a number, or zero.
export function multiplier(options: Readonly<Record<string, string>>): number {
	const m = num(options, "multiplier", 1)
	return m === 0 ? 1 : m
}

// a color the background option may name: a hex value, a name, or an
// rgb() or hsl() function, and nothing that could load a resource
const color = /^(#[0-9a-f]{3,8}|[a-z]+|(rgba?|hsla?)\([\d\s.,%/]+\))$/i

// background reads the background option every widget takes, which draws
// a card behind the widget so it reads cleanly over another: on, yes,
// true, or 1 for the page's card color, or a CSS color. It is undefined
// when the option is off or is not a color.
export function background(options: Readonly<Record<string, string>>): string | undefined {
	const v = (options.background ?? "").trim()
	const lower = v.toLowerCase()
	if (["", "off", "no", "false", "0", "none"].includes(lower)) {
		return undefined
	}
	if (["on", "yes", "true", "1"].includes(lower)) {
		return "var(--bg)"
	}
	return color.test(v) ? v : undefined
}
