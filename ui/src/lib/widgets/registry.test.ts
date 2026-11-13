import { describe, expect, test } from "vitest"
import { widgets } from "./index.ts"
import { background, multiplier, num, withDefaults } from "./registry.ts"

describe("widgets", () => {
	test("every widget declares a host, its bindings, and defaults", () => {
		expect(widgets.all.map((w) => w.name).sort()).toEqual([
			"button",
			"dashboard",
			"gauge",
			"image",
			"pump",
			"status",
			"svg",
			"tank",
			"trend",
			"value",
			"valve",
		])
		for (const w of widgets.all) {
			expect(["grid", "canvas", "both"]).toContain(w.host)
			expect(w.bindings.length).toBeGreaterThan(0)
		}
	})

	test("options fall back to the descriptor's defaults", () => {
		const gauge = widgets.get("gauge")
		if (!gauge) {
			throw new Error("no gauge")
		}
		const o = withDefaults(gauge, { max: "250", unit: "" })
		expect(o.max).toBe("250")
		expect(o.min).toBe(gauge.options.min)
		expect(num(o, "max", 0)).toBe(250)
		expect(num({ max: "abc" }, "max", 7)).toBe(7)
	})

	test("the multiplier scales a reading for display, and is 1 when unusable", () => {
		expect(multiplier({ multiplier: "0.001" })).toBe(0.001)
		expect(multiplier({})).toBe(1)
		expect(multiplier({ multiplier: "" })).toBe(1)
		expect(multiplier({ multiplier: "abc" })).toBe(1)
		expect(multiplier({ multiplier: "0" })).toBe(1)
		for (const name of ["value", "gauge", "trend"]) {
			expect(widgets.get(name)?.options.multiplier).toBe("1")
		}
	})

	test("background is off, the card color, or a CSS color", () => {
		expect(background({})).toBeUndefined()
		expect(background({ background: "off" })).toBeUndefined()
		expect(background({ background: "on" })).toBe("var(--bg)")
		expect(background({ background: "#123" })).toBe("#123")
		expect(background({ background: "rgb(0 0 0 / 50%)" })).toBe("rgb(0 0 0 / 50%)")
		expect(background({ background: "white" })).toBe("white")
		expect(background({ background: "url(https://example.com/t.png)" })).toBeUndefined()
		expect(background({ background: "red; color: blue" })).toBeUndefined()
	})
})
