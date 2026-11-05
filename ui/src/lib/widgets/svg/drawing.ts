// A drawing is an uploaded SVG with data-siot-* attributes on the elements
// that follow live points. This module reads a drawing (sanitized, its
// bindings and ports listed) and applies values to the injected DOM, and
// says what a click inside it sends. It never talks to the server, so the
// widget, the editor, and a test see the same drawing.
import type { MimicPort, PortDir } from "#lib/vendor/nautilus/mimic.ts"
import type { DeclaredBinding } from "../registry.ts"
import type { Bound, Write } from "../registry.ts"
import { sanitizeSvg } from "./sanitize.ts"

export interface Drawing {
	// svg is the sanitized markup to inject, "" when there is none
	svg: string
	// bindings are the distinct data-siot-bind names, in document order,
	// each with what the drawing does with it
	bindings: DeclaredBinding[]
	// ports are the elements marked data-siot-port, as fractions of the box
	ports: MimicPort[]
	// error says why there is nothing to draw
	error: string
}

const empty = (error: string): Drawing => ({ svg: "", bindings: [], ports: [], error })

interface ViewBox {
	x: number
	y: number
	w: number
	h: number
}

function parseViewBox(s: string | null): ViewBox | undefined {
	const [x, y, w, h, ...rest] = (s ?? "")
		.trim()
		.split(/[\s,]+/)
		.map(Number)
	if (rest.length > 0 || [x, y, w, h].some((v) => v === undefined || !Number.isFinite(v))) {
		return undefined
	}
	return w !== undefined && h !== undefined && w > 0 && h > 0
		? { x: x ?? 0, y: y ?? 0, w, h }
		: undefined
}

// a length attribute as a number; units other than px are not understood
function length(el: Element, name: string): number | undefined {
	const v = parseFloat(el.getAttribute(name) ?? "")
	return Number.isFinite(v) ? v : undefined
}

const dirs: readonly PortDir[] = ["left", "right", "up", "down"]

// portOf reads a port from an element's own position attributes, as a
// circle's center or a box's center, ignoring any transform.
function portOf(el: Element, box: ViewBox): MimicPort | undefined {
	const [name, dir] = (el.getAttribute("data-siot-port") ?? "").split(",").map((s) => s.trim())
	if (!name) {
		return undefined
	}
	let cx = length(el, "cx")
	let cy = length(el, "cy")
	if (cx === undefined || cy === undefined) {
		const x = length(el, "x")
		const y = length(el, "y")
		if (x === undefined || y === undefined) {
			return undefined
		}
		cx = x + (length(el, "width") ?? 0) / 2
		cy = y + (length(el, "height") ?? 0) / 2
	}
	const d = dirs.find((v) => v === dir)
	return { name, x: (cx - box.x) / box.w, y: (cy - box.y) / box.h, ...(d ? { dir: d } : {}) }
}

const cache = new Map<string, Drawing>()
const cacheSize = 16

// parseDrawing sanitizes a drawing and reads what the widget needs from
// it. The result is cached by the text, since a value change must not
// sanitize again.
export function parseDrawing(text: string): Drawing {
	const hit = cache.get(text)
	if (hit) {
		return hit
	}
	const d = parse(text)
	if (cache.size >= cacheSize) {
		const first = cache.keys().next().value
		if (first !== undefined) {
			cache.delete(first)
		}
	}
	cache.set(text, d)
	return d
}

function parse(text: string): Drawing {
	if (text.trim() === "") {
		return empty("no drawing")
	}
	const clean = sanitizeSvg(text)
	const doc = new DOMParser().parseFromString(clean, "text/html")
	const svg = doc.querySelector("svg")
	if (!svg) {
		return empty("not an SVG")
	}
	// the drawing scales to the widget, so its own size gives way to a
	// viewBox, made from the size when the file has none
	let box = parseViewBox(svg.getAttribute("viewBox"))
	if (!box) {
		const w = length(svg, "width")
		const h = length(svg, "height")
		if (w === undefined || h === undefined || w <= 0 || h <= 0) {
			return empty("the SVG has no viewBox or size")
		}
		box = { x: 0, y: 0, w, h }
		svg.setAttribute("viewBox", `0 0 ${w} ${h}`)
	}
	svg.removeAttribute("width")
	svg.removeAttribute("height")
	const bindings = declaredBindings(svg)
	const ports = [...svg.querySelectorAll("[data-siot-port]")]
		.map((el) => portOf(el, box))
		.filter((p): p is MimicPort => p !== undefined)
	return { svg: svg.outerHTML, bindings, ports, error: "" }
}

// the attributes that make an element do something with its binding; an
// element with none follows the value through its state alone
const effects = ["text", "rotate", "level", "toggle", "set"]

// declaredBindings lists the bindings the drawing names, in document
// order, each with the effects of the elements that follow it
function declaredBindings(svg: Element): DeclaredBinding[] {
	const uses = new Map<string, Set<string>>()
	for (const el of svg.querySelectorAll("[data-siot-bind]")) {
		const name = bindNameOf(el)
		if (name !== "" && !uses.has(name)) {
			uses.set(name, new Set())
		}
	}
	for (const el of svg.querySelectorAll(effects.map((e) => `[data-siot-${e}]`).join(", "))) {
		const name = bindNameOf(el)
		const set = uses.get(name)
		if (set) {
			for (const e of effects) {
				if (el.hasAttribute(`data-siot-${e}`)) {
					set.add(e)
				}
			}
		}
	}
	return [...uses].map(([name, set]) => ({
		name,
		use: set.size > 0 ? [...set].join(", ") : "state",
	}))
}

// pair reads "a,b" from an attribute as two numbers
function pair(el: Element, name: string): [number, number] | undefined {
	const [a, b, ...rest] = (el.getAttribute(name) ?? "").split(",").map((s) => parseFloat(s))
	return a !== undefined &&
		b !== undefined &&
		rest.length === 0 &&
		Number.isFinite(a) &&
		Number.isFinite(b)
		? [a, b]
		: undefined
}

// range reads "min,max" from an attribute, 0..100 when absent or unusable.
function range(el: Element): [number, number] {
	const r = pair(el, "data-siot-range")
	return r && r[0] !== r[1] ? r : [0, 100]
}

// fraction places a value in a range, clamped
function fraction(v: number, [min, max]: [number, number]): number {
	return Math.max(0, Math.min(1, (v - min) / (max - min)))
}

export type DrawState = "on" | "off" | "stale" | "dead"

// stateOf is what a bound element shows: on when the value is not zero
export function stateOf(b: Bound | undefined): DrawState {
	if (!b || b.state === "dead") {
		return "dead"
	}
	if (b.state === "stale") {
		return "stale"
	}
	return b.value !== 0 ? "on" : "off"
}

// formatValue is the text a data-siot-text element shows: the point's
// text when it has one, else the number to the element's decimals with its
// unit appended as given (an attribute's edges are trimmed, so a space
// before the unit is drawn in the SVG itself), and a dash while there is
// nothing to show.
export function formatValue(el: Element, b: Bound | undefined): string {
	if (!b || b.state !== "live") {
		return "--"
	}
	if (b.text !== "") {
		return b.text
	}
	const d = parseInt(el.getAttribute("data-siot-decimals") ?? "", 10)
	const unit = el.getAttribute("data-siot-unit") ?? ""
	return b.value.toFixed(Number.isFinite(d) && d >= 0 ? d : 0) + unit
}

// bindNameOf is the binding an element follows: its own data-siot-bind,
// or the nearest ancestor's, so a group bound once can hold several
// buttons.
function bindNameOf(el: Element): string {
	return (el.closest("[data-siot-bind]")?.getAttribute("data-siot-bind") ?? "").trim()
}

const clickable = "[data-siot-toggle], [data-siot-set]"

// applyBindings sets the state attribute, text, and transforms on every
// bound element under root from the values, and marks the elements a
// click would act on. It touches attributes only, so the drawing's own
// styles and animations carry on.
export function applyBindings(
	root: ParentNode,
	values: Readonly<Record<string, Bound>>,
	writes: ReadonlySet<string>
): void {
	for (const el of root.querySelectorAll<SVGElement | HTMLElement>(clickable)) {
		el.style.cursor = writes.has(bindNameOf(el)) ? "pointer" : ""
	}
	for (const el of root.querySelectorAll<SVGElement | HTMLElement>("[data-siot-bind]")) {
		const name = (el.getAttribute("data-siot-bind") ?? "").trim()
		const b = values[name]
		const state = stateOf(b)
		el.setAttribute("data-siot-state", state)
		if (el.hasAttribute("data-siot-text")) {
			el.textContent = formatValue(el, b)
		}
		const live = b?.state === "live"
		const v = live ? b.value : NaN
		const f = live ? fraction(v, range(el)) : 0
		// a transform turns or scales the element about its own box: its
		// center for a rotation and its bottom for a level unless the
		// element names another origin, as a needle pivoting at one end does
		const origin = el.getAttribute("data-siot-origin")
		if (el.hasAttribute("data-siot-rotate")) {
			const [a, z] = pair(el, "data-siot-rotate") ?? [0, 360]
			el.style.transformBox = "fill-box"
			el.style.transformOrigin = origin ?? "center"
			el.style.transform = `rotate(${a + (z - a) * f}deg)`
		} else if (el.hasAttribute("data-siot-level")) {
			el.style.transformBox = "fill-box"
			el.style.transformOrigin = origin ?? "bottom"
			el.style.transform = `scaleY(${f})`
		}
	}
}

export interface Click {
	prop: string
	write: Write
}

// clickTarget says what a click on target sends: the nearest element up to
// root with data-siot-toggle or data-siot-set, when the binding it follows
// may be written. A toggle sends the other state; a set sends its value,
// as text when it is not a number.
export function clickTarget(
	root: Node,
	target: Element | null,
	values: Readonly<Record<string, Bound>>,
	writes: ReadonlySet<string>
): Click | undefined {
	for (let el = target; el && el !== root; el = el.parentElement) {
		const toggle = el.hasAttribute("data-siot-toggle")
		const set = el.getAttribute("data-siot-set")
		if (!toggle && set === null) {
			continue
		}
		const prop = bindNameOf(el)
		if (!writes.has(prop)) {
			return undefined
		}
		if (toggle) {
			return { prop, write: { value: stateOf(values[prop]) === "on" ? 0 : 1, text: "" } }
		}
		const n = Number(set)
		const numeric = set !== null && set.trim() !== "" && Number.isFinite(n)
		return { prop, write: numeric ? { value: n, text: "" } : { value: 0, text: set ?? "" } }
	}
	return undefined
}
