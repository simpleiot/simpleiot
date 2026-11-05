// @vitest-environment jsdom
import { describe, expect, test } from "vitest"
import type { Bound } from "../registry.ts"
import { applyBindings, clickTarget, parseDrawing } from "./drawing.ts"

const lamp = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 160">
  <style>[data-siot-state="on"] { fill: #0a0; }</style>
  <circle class="glass" cx="50" cy="55" r="40" data-siot-bind="lit" data-siot-toggle />
  <circle class="glow" cx="50" cy="55" r="46" data-siot-bind="lit" />
  <text x="50" y="150" data-siot-bind="temp" data-siot-text data-siot-decimals="1" data-siot-unit="°C">?</text>
  <rect class="needle" x="48" y="10" width="4" height="40" data-siot-bind="pressure" data-siot-rotate="-90,90" data-siot-range="0,10" data-siot-origin="50% 100%" />
  <rect class="liquid" x="10" y="100" width="80" height="50" data-siot-bind="level" data-siot-level />
  <g data-siot-bind="mode"><rect x="0" y="0" width="10" height="10" data-siot-set="1" /><rect x="20" y="0" width="10" height="10" data-siot-set="stop" /></g>
  <circle data-siot-port="inlet,left" cx="0" cy="80" r="2" />
  <rect data-siot-port="outlet,right" x="90" y="70" width="10" height="20" />
  <path data-siot-port="nowhere" d="M0 0" />
</svg>`

const live = (value: number, text = ""): Bound => ({ state: "live", value, text, time: "t" })
const stale: Bound = { state: "stale", value: 0, text: "", time: "" }
const dead: Bound = { state: "dead", value: 0, text: "", time: "" }

function mount(svg: string): HTMLElement {
	const host = document.createElement("div")
	host.innerHTML = svg
	return host
}

describe("parseDrawing", () => {
	const d = parseDrawing(lamp)

	test("lists the bindings once each, in document order, with what the drawing does with them", () => {
		expect(d.error).toBe("")
		expect(d.bindings).toEqual([
			{ name: "lit", use: "toggle" },
			{ name: "temp", use: "text" },
			{ name: "pressure", use: "rotate" },
			{ name: "level", use: "level" },
			{ name: "mode", use: "set" },
		])
	})

	test("a binding with no effect is state alone", () => {
		const d = parseDrawing(
			'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect data-siot-bind="run" /></svg>'
		)
		expect(d.bindings).toEqual([{ name: "run", use: "state" }])
	})

	test("reads ports from a circle and a box as fractions, skipping one without a position", () => {
		expect(d.ports).toEqual([
			{ name: "inlet", x: 0, y: 0.5, dir: "left" },
			{ name: "outlet", x: 0.95, y: 0.5, dir: "right" },
		])
	})

	test("keeps the markup, without a size of its own", () => {
		expect(d.svg).toContain('viewBox="0 0 100 160"')
		expect(d.svg).toContain("data-siot-toggle")
	})

	test("makes a viewBox from the size when the file has none, and drops the size", () => {
		const s = parseDrawing(
			'<svg xmlns="http://www.w3.org/2000/svg" width="200" height="100"><rect/></svg>'
		)
		expect(s.error).toBe("")
		expect(s.svg).toContain('viewBox="0 0 200 100"')
		expect(s.svg).not.toContain('width="200"')
	})

	test("says what is wrong with text that is not a drawing", () => {
		expect(parseDrawing("").error).toBe("no drawing")
		expect(parseDrawing("hello").error).toBe("not an SVG")
		expect(parseDrawing("<svg><rect/></svg>").error).toBe("the SVG has no viewBox or size")
	})

	test("returns the same drawing for the same text", () => {
		expect(parseDrawing(lamp)).toBe(d)
	})
})

describe("applyBindings", () => {
	const writes = new Set(["lit", "mode"])

	test("sets the state from the value and the binding's state", () => {
		const host = mount(parseDrawing(lamp).svg)
		applyBindings(host, { lit: live(1), temp: stale, pressure: dead }, writes)
		expect(host.querySelector(".glass")?.getAttribute("data-siot-state")).toBe("on")
		expect(host.querySelector(".glow")?.getAttribute("data-siot-state")).toBe("on")
		expect(host.querySelector("text")?.getAttribute("data-siot-state")).toBe("stale")
		expect(host.querySelector(".needle")?.getAttribute("data-siot-state")).toBe("dead")
		// an unbound name is dead too
		expect(host.querySelector(".liquid")?.getAttribute("data-siot-state")).toBe("dead")
		applyBindings(host, { lit: live(0) }, writes)
		expect(host.querySelector(".glass")?.getAttribute("data-siot-state")).toBe("off")
	})

	test("writes text with decimals and a unit, or the point's text, or a dash", () => {
		const host = mount(parseDrawing(lamp).svg)
		const t = host.querySelector("text")
		applyBindings(host, { temp: live(21.456) }, writes)
		expect(t?.textContent).toBe("21.5°C")
		applyBindings(host, { temp: live(0, "warm") }, writes)
		expect(t?.textContent).toBe("warm")
		applyBindings(host, { temp: stale }, writes)
		expect(t?.textContent).toBe("--")
	})

	test("rotates over the degrees range by the value over its range, clamped", () => {
		const host = mount(parseDrawing(lamp).svg)
		const n = host.querySelector<SVGElement>(".needle")
		applyBindings(host, { pressure: live(5) }, writes)
		expect(n?.style.transform).toBe("rotate(0deg)")
		expect(n?.style.transformOrigin).toBe("50% 100%")
		applyBindings(host, { pressure: live(10) }, writes)
		expect(n?.style.transform).toBe("rotate(90deg)")
		applyBindings(host, { pressure: live(-3) }, writes)
		expect(n?.style.transform).toBe("rotate(-90deg)")
		applyBindings(host, { pressure: stale }, writes)
		expect(n?.style.transform).toBe("rotate(-90deg)")
	})

	test("scales a level from the bottom, 0..100 when no range is given", () => {
		const host = mount(parseDrawing(lamp).svg)
		const l = host.querySelector<SVGElement>(".liquid")
		applyBindings(host, { level: live(25) }, writes)
		expect(l?.style.transform).toBe("scaleY(0.25)")
		expect(l?.style.transformOrigin).toBe("bottom")
		applyBindings(host, { level: live(250) }, writes)
		expect(l?.style.transform).toBe("scaleY(1)")
	})

	test("marks only writable click targets with a pointer", () => {
		const host = mount(parseDrawing(lamp).svg)
		applyBindings(host, { lit: live(1), mode: live(0) }, new Set(["lit"]))
		expect(host.querySelector<SVGElement>(".glass")?.style.cursor).toBe("pointer")
		expect(host.querySelector<SVGElement>(".glow")?.style.cursor).toBe("")
		expect(host.querySelector<SVGElement>("[data-siot-set]")?.style.cursor).toBe("")
		applyBindings(host, { lit: live(1), mode: live(0) }, new Set(["mode"]))
		expect(host.querySelector<SVGElement>(".glass")?.style.cursor).toBe("")
		expect(host.querySelector<SVGElement>("[data-siot-set]")?.style.cursor).toBe("pointer")
	})
})

describe("clickTarget", () => {
	const writes = new Set(["lit", "mode"])
	const host = mount(parseDrawing(lamp).svg)
	const glass = host.querySelector(".glass")
	const sets = host.querySelectorAll("[data-siot-set]")

	test("a toggle sends the other state", () => {
		expect(clickTarget(host, glass, { lit: live(1) }, writes)).toEqual({
			prop: "lit",
			write: { value: 0, text: "" },
		})
		expect(clickTarget(host, glass, { lit: live(0) }, writes)?.write.value).toBe(1)
		expect(clickTarget(host, glass, { lit: stale }, writes)?.write.value).toBe(1)
	})

	test("a set sends its value, as text when it is not a number", () => {
		expect(clickTarget(host, sets[0] ?? null, {}, writes)).toEqual({
			prop: "mode",
			write: { value: 1, text: "" },
		})
		expect(clickTarget(host, sets[1] ?? null, {}, writes)).toEqual({
			prop: "mode",
			write: { value: 0, text: "stop" },
		})
	})

	test("a click elsewhere, or on a binding that is not writable, sends nothing", () => {
		expect(clickTarget(host, host.querySelector(".glow"), { lit: live(1) }, writes)).toBeUndefined()
		expect(clickTarget(host, host.querySelector("text"), {}, writes)).toBeUndefined()
		expect(clickTarget(host, glass, { lit: live(1) }, new Set())).toBeUndefined()
		expect(clickTarget(host, null, {}, writes)).toBeUndefined()
	})
})
