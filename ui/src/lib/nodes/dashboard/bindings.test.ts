import { describe, expect, test } from "vitest"
import type { Node } from "#lib/siot/node.ts"
import { newText, newValue } from "#lib/siot/point.ts"
import {
	bindingPaths,
	boundNodeIds,
	parseBindings,
	parseLink,
	parsePorts,
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

	test("a relative path resolves through the resolver and wins over the ID", () => {
		const w: Node = {
			id: "w",
			type: "widget",
			parent: "d",
			points: [
				newText("bindNode", "value", "old"),
				newText("bindPath", "value", "modbusIo/pump1"),
				newText("bindPath", "other", "modbusIo/none"),
			],
			edgePoints: [],
		}
		expect(bindingPaths(w)).toEqual(["modbusIo/pump1", "modbusIo/none"])
		const b = parseBindings(w, (p) => (p === "modbusIo/pump1" ? "n9" : undefined))
		expect(b.value?.nodeId).toBe("n9")
		expect(b.other?.nodeId).toBe("")
		expect(parseBindings(w).value?.nodeId).toBe("")
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

describe("canvas", () => {
	test("ports override the defaults by name, and an empty one removes it", () => {
		const w: Node = {
			id: "w",
			type: "widget",
			parent: "d",
			points: [
				newText("port", "in", "0.5,0,down"),
				newText("port", "out", ""),
				newText("port", "bad", "x"),
			],
			edgePoints: [],
		}
		expect(
			parsePorts(w, [
				{ name: "in", x: 0.5, y: 0, dir: "up" },
				{ name: "out", x: 0.5, y: 1 },
			])
		).toEqual([{ name: "in", x: 0.5, y: 0, dir: "down" }])
	})

	test("a link reads its ends, ordered vertices, routing, and flow", () => {
		const link: Node = {
			id: "l",
			type: "link",
			parent: "d",
			points: [
				newText("from", "0", "pump.discharge"),
				newText("to", "0", "tank.in"),
				newText("routing", "0", "orthogonal"),
				newText("vertex", "1", "300,40"),
				newText("vertex", "0", "200,40"),
				newText("bindNode", "flowing", "p"),
			],
			edgePoints: [],
		}
		const pump: Node = {
			id: "p",
			type: "variable",
			parent: "d",
			points: [newValue("value", "0", 1)],
			edgePoints: [],
		}
		const l = parseLink(link, (id) => (id === "p" ? pump : undefined))
		expect(l.from).toEqual({ equip: "pump", port: "discharge" })
		expect(l.to).toEqual({ equip: "tank", port: "in" })
		expect(l.routing).toBe("orthogonal")
		expect(l.points).toEqual([
			[200, 40],
			[300, 40],
		])
		expect(l.flowing).toBe(true)
		expect(l.rate).toBe(1)
		expect(l.dead).toBe(false)
		expect(parseLink(link, () => undefined).dead).toBe(true)
	})
})
