import { describe, expect, test } from "vitest"
import type { Node } from "./node.ts"
import { newText } from "./point.ts"
import { buildMatches, moveActive, rankMatch, rankMatches } from "./search.ts"

const node = (id: string, parent: string, desc: string, type = "variable"): Node => ({
	id,
	type,
	parent,
	points: desc === "" ? [] : [newText("description", "0", desc)],
	edgePoints: [],
})

describe("rankMatch", () => {
	test("ranks description prefix, description, ID, then type", () => {
		expect(rankMatch(node("a", "G", "Pump 1"), "pump")).toBe(0)
		expect(rankMatch(node("a", "G", "sump pump"), "pump")).toBe(1)
		expect(rankMatch(node("abc123", "G", "tank"), "ABC123")).toBe(2)
		expect(rankMatch(node("a", "G", "tank", "modbusIo"), "modbus")).toBe(3)
		expect(rankMatch(node("a", "G", "tank"), "pump")).toBe(-1)
		expect(rankMatch(node("a", "G", "tank"), "  ")).toBe(-1)
	})
})

describe("buildMatches", () => {
	test("tells matches from ancestors and gives each its path", () => {
		const reply = [
			node("p1", "devA", "pump 1"),
			node("devA", "G", "station A", "device"),
			node("p2", "devB", "Pump 2"),
			node("devB", "G", "station B", "device"),
		]
		const m = buildMatches(reply, "pump", "G", "site")
		expect(m.map((x) => [x.id, x.rank, x.path])).toEqual([
			["p1", 0, ["site", "station A"]],
			["p2", 0, ["site", "station B"]],
		])
	})

	test("an ancestor that matches is a match too, and a cycle stops", () => {
		const reply = [node("pumps", "G", "pumps", "group"), node("p1", "pumps", "pump 1")]
		const m = buildMatches(reply, "pump", "G")
		expect(m.map((x) => [x.id, x.path])).toEqual([
			["pumps", []],
			["p1", ["pumps"]],
		])
		const loop = [node("a", "b", "pump a"), node("b", "a", "b")]
		expect(buildMatches(loop, "pump", "G")[0]?.path).toEqual(["b"])
	})
})

describe("rankMatches", () => {
	test("orders by rank then description and drops repeats", () => {
		const m = rankMatches([
			{ id: "x", type: "", description: "sump pump", path: [], rank: 1 },
			{ id: "y", type: "", description: "pump b", path: [], rank: 0 },
			{ id: "z", type: "", description: "Pump a", path: [], rank: 0 },
			{ id: "y", type: "", description: "pump b", path: ["other"], rank: 0 },
		])
		expect(m.map((x) => x.id)).toEqual(["z", "y", "x"])
	})
})

describe("moveActive", () => {
	test("arrows move and wrap, and an empty list stays inactive", () => {
		expect(moveActive(-1, "ArrowDown", 3)).toBe(0)
		expect(moveActive(0, "ArrowDown", 3)).toBe(1)
		expect(moveActive(2, "ArrowDown", 3)).toBe(0)
		expect(moveActive(0, "ArrowUp", 3)).toBe(2)
		expect(moveActive(-1, "ArrowUp", 3)).toBe(2)
		expect(moveActive(1, "ArrowDown", 0)).toBe(-1)
	})
})
