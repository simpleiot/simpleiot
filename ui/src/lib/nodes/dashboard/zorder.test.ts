import { describe, expect, test } from "vitest"
import type { Node } from "#lib/siot/node.ts"
import { newValue } from "#lib/siot/point.ts"
import { byStack, restack, topZ, widgetZ, type Stacked } from "./zorder.ts"

const node = (z?: number): Node => ({
	id: "w",
	type: "widget",
	parent: "d",
	points: z === undefined ? [] : [newValue("z", "0", z)],
	edgePoints: [],
})

const ids = (items: readonly Stacked[]) =>
	items
		.slice()
		.sort(byStack)
		.map((s) => s.id)

const apply = (items: readonly Stacked[], changes: Map<string, number>) =>
	items.map((s) => ({ ...s, z: changes.get(s.id) ?? s.z }))

describe("z-order", () => {
	test("a widget without z is at 0", () => {
		expect(widgetZ(node())).toBe(0)
		expect(widgetZ(node(-2))).toBe(-2)
		expect(widgetZ({ ...node(3), points: [{ ...newValue("z", "0", 3), tombstone: 1 }] })).toBe(0)
	})

	test("equal z values are ordered by ID", () => {
		expect(
			ids([
				{ id: "b", z: 0 },
				{ id: "a", z: 0 },
				{ id: "c", z: -1 },
			])
		).toEqual(["c", "a", "b"])
	})

	test("a new widget goes on top", () => {
		expect(topZ([])).toBe(0)
		expect(
			topZ([
				{ id: "a", z: 2 },
				{ id: "b", z: -1 },
			])
		).toBe(3)
	})

	const stack: Stacked[] = [
		{ id: "a", z: 0 },
		{ id: "b", z: 0 },
		{ id: "c", z: 0 },
	]

	test("front and back write one value", () => {
		expect([...restack(stack, "a", "front")]).toEqual([["a", 1]])
		expect([...restack(stack, "c", "back")]).toEqual([["c", -1]])
		expect(restack(stack, "c", "front").size).toBe(0)
		expect(restack(stack, "a", "back").size).toBe(0)
		expect(restack(stack, "missing", "front").size).toBe(0)
	})

	test("a step swaps with the neighbor, renumbering equal values", () => {
		expect(ids(apply(stack, restack(stack, "a", "forward")))).toEqual(["b", "a", "c"])
		expect(ids(apply(stack, restack(stack, "c", "backward")))).toEqual(["a", "c", "b"])
		const spread = [
			{ id: "a", z: 1 },
			{ id: "b", z: 5 },
		]
		expect([...restack(spread, "a", "forward")]).toEqual([
			["a", 5],
			["b", 1],
		])
		expect(restack(spread, "b", "forward").size).toBe(0)
	})
})
