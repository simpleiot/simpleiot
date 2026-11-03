import { describe, expect, test } from "vitest"
import type { Node } from "#lib/siot/node.ts"
import { newText, newValue } from "#lib/siot/point.ts"
import {
	boundNodeIds,
	parseBindings,
	resolveBindings,
	widgetLayout,
	widgetOptions,
} from "./bindings.ts"

const widget: Node = {
	id: "w",
	type: "widget",
	parent: "d",
	points: [
		newText("bindNode", "value", "n1"),
		newText("bindNode", "setpoint", "n2"),
		newText("bindType", "setpoint", "valueSet"),
		newText("bindKey", "setpoint", "3"),
		newText("prop", "unit", "°C"),
		{ ...newText("prop", "gone", "x"), tombstone: 1 },
		newValue("x", "0", 2),
		newValue("w", "0", 4),
	],
	edgePoints: [],
}

describe("bindings", () => {
	test("parse fills in the default type and key", () => {
		expect(parseBindings(widget)).toEqual({
			value: { nodeId: "n1", type: "value", key: "0" },
			setpoint: { nodeId: "n2", type: "valueSet", key: "3" },
		})
		expect(boundNodeIds(parseBindings(widget))).toEqual(["n1", "n2"])
	})

	test("resolve reports live, stale, and dead", () => {
		const nodes: Record<string, Node> = {
			n1: {
				id: "n1",
				type: "variable",
				parent: "d",
				points: [{ ...newValue("value", "0", 21.5, "2026-11-03T00:00:00.000Z") }],
				edgePoints: [],
			},
			n2: { id: "n2", type: "variable", parent: "d", points: [], edgePoints: [] },
		}
		const r = resolveBindings(parseBindings(widget), (id) => nodes[id])
		expect(r.value).toEqual({
			state: "live",
			value: 21.5,
			text: "",
			time: "2026-11-03T00:00:00.000Z",
		})
		expect(r.setpoint?.state).toBe("stale")
		expect(resolveBindings(parseBindings(widget), () => undefined).value?.state).toBe("dead")
	})

	test("options skip deleted points and layout fills defaults", () => {
		expect(widgetOptions(widget)).toEqual({ unit: "°C" })
		expect(widgetLayout(widget)).toEqual({ x: 2, y: 0, w: 4, h: 2 })
	})
})
