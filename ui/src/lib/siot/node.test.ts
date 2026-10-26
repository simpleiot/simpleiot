import { describe, expect, it } from "vitest"
import { nodeStatus, type Node } from "./node.ts"
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
		expect(nodeStatus(node([newText(PointType.SysState, "0", "offline")]))).toBe("offline")
	})

	it("puts disabled ahead of the rest", () => {
		const pts = [num(PointType.Disabled, 1), newText(PointType.Error, "0", "timeout")]
		expect(nodeStatus(node(pts))).toBe("disabled")
	})
})
