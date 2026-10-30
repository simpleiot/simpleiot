import { describe, expect, it } from "vitest"
import { nodeStatus, onlineStatus, type Node } from "./node.ts"
import { DataType, newText, type Point } from "./point.ts"
import { PointType } from "./schema.ts"

const num = (type: string, value: number): Point => ({
	type,
	key: "0",
	time: "",
	dataType: DataType.Float,
	value,
	text: "",
	tombstone: 0,
})

const node = (points: Point[]): Node => ({
	id: "a",
	type: "t",
	parent: "p",
	points,
	edgePoints: [],
})

describe("nodeStatus", () => {
	it("reads a plain node as ok", () => {
		expect(nodeStatus(node([newText(PointType.Description, "0", "x")]))).toBe("ok")
	})

	it("reads an error, and an offline device", () => {
		expect(nodeStatus(node([newText(PointType.Error, "0", "timeout")]))).toBe("error")
		expect(nodeStatus(node([num(PointType.Offline, 1)]))).toBe("offline")
		expect(nodeStatus(node([num(PointType.Online, 0)]))).toBe("offline")
		expect(nodeStatus(node([num(PointType.Online, 1)]))).toBe("ok")
		expect(nodeStatus({ ...node([num(PointType.Online, 0)]), parent: "root" })).toBe("ok")
	})

	it("puts disabled ahead of the rest", () => {
		const pts = [num(PointType.Disabled, 1), newText(PointType.Error, "0", "timeout")]
		expect(nodeStatus(node(pts))).toBe("disabled")
	})
})

describe("onlineStatus", () => {
	const at = (type: string, value: number, time: string): Point => ({ ...num(type, value), time })

	it("is empty without an online point", () => {
		expect(onlineStatus([], 0)).toBe("")
	})

	it("reads online and offline with how long ago", () => {
		const t = "2026-10-29T20:00:00Z"
		const now = Date.parse(t) + 90_000
		expect(onlineStatus([at(PointType.Online, 1, t)], now)).toMatch(
			/^Online since .* \(1m 30s ago\)$/
		)
		expect(onlineStatus([at(PointType.Online, 0, t)], now)).toMatch(/^Offline since /)
	})
})
