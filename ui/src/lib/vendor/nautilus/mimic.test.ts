import { describe, expect, test } from "vitest"
import {
	inferPortDir,
	makeGetPort,
	orthogonalPoints,
	pointsToPath,
	portAbsolute,
	resolvePipeEndpoints,
	routedPoints,
} from "./mimic.ts"

describe("ports", () => {
	test("direction is inferred from the edge a port sits on", () => {
		expect(inferPortDir({ x: 0, y: 0.5 })).toBe("left")
		expect(inferPortDir({ x: 1, y: 0.5 })).toBe("right")
		expect(inferPortDir({ x: 0.5, y: 0 })).toBe("up")
		expect(inferPortDir({ x: 0.5, y: 1 })).toBe("down")
		expect(inferPortDir({ x: 0, y: 0 })).toBeUndefined()
		expect(inferPortDir({ x: 0.5, y: 0.5 })).toBeUndefined()
	})

	test("a port resolves to canvas units in its box", () => {
		const box = { x: 100, y: 50, w: 200, h: 100 }
		expect(portAbsolute(box, { x: 0.5, y: 1 })).toEqual({ x: 200, y: 150, dir: "down" })
		expect(portAbsolute(box, { x: 0, y: 0.5, dir: "right" }).dir).toBe("right")
	})
})

describe("routing", () => {
	test("orthogonal routing adds one corner per diagonal, continuing the last axis", () => {
		expect(
			orthogonalPoints([
				[0, 0],
				[10, 4],
				[14, 20],
			])
		).toEqual([
			[0, 0],
			[10, 0],
			[10, 4],
			[10, 20],
			[14, 20],
		])
		expect(
			orthogonalPoints([
				[0, 0],
				[0, 10],
			])
		).toEqual([
			[0, 0],
			[0, 10],
		])
		expect(orthogonalPoints([[3, 3]])).toEqual([[3, 3]])
	})

	test("points become a path", () => {
		expect(
			pointsToPath([
				[0, 0],
				[5, 5],
			])
		).toBe("M 0 0 L 5 5")
	})

	test("anchored ends are derived from ports and leave by a stub", () => {
		const getPort = makeGetPort(
			(id) => (id === "pump" ? { x: 0, y: 0, w: 100, h: 100 } : undefined),
			(id) => (id === "pump" ? [{ name: "out", x: 1, y: 0.5 }] : undefined)
		)
		const pts = routedPoints({ points: [[200, 50]], from: { equip: "pump", port: "out" } }, getPort)
		expect(pts).toEqual([
			[100, 50],
			[112, 50],
			[200, 50],
		])
	})

	test("an anchor that does not resolve is flagged, not lost", () => {
		const r = resolvePipeEndpoints(
			{ points: [[30, 30]], from: { equip: "gone", port: "x" }, to: { equip: "gone", port: "y" } },
			() => undefined
		)
		expect(r.startFlagged && r.endFlagged).toBe(true)
		expect(r.points).toEqual([
			[30, 30],
			[30, 30],
			[30, 30],
		])
		const empty = resolvePipeEndpoints(
			{ points: [], from: { equip: "a", port: "b" } },
			() => undefined
		)
		expect(empty.points).toEqual([[0, 0]])
	})
})
