import { describe, expect, test } from "vitest"
import { widgets } from "./index.ts"
import { num, withDefaults } from "./registry.ts"

describe("widgets", () => {
	test("every widget declares a host, its bindings, and defaults", () => {
		expect(widgets.all.map((w) => w.name).sort()).toEqual(["gauge", "status", "trend", "value"])
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
})
