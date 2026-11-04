import { describe, expect, test } from "vitest"
import type { Node } from "#lib/siot/node.ts"
import { newText } from "#lib/siot/point.ts"
import { joinPath, pathBelow, pickChild, splitPath } from "./paths.ts"

const node = (id: string, type: string, desc: string): Node => ({
	id,
	type,
	parent: "p",
	points: [newText("description", "0", desc)],
	edgePoints: [],
})

describe("paths", () => {
	test("a path splits into type and description segments", () => {
		expect(splitPath("device/Station A/modbusIo/pump1")).toEqual([
			{ type: "device", description: "Station A" },
			{ type: "modbusIo", description: "pump1" },
		])
		expect(splitPath("modbusIo")).toBeUndefined()
		expect(splitPath("modbusIo//x")).toBeUndefined()
		expect(joinPath([{ type: "a", description: "b" }])).toBe("a/b")
	})

	test("a segment picks the first child of its type by description", () => {
		const kids = [node("1", "variable", "Pump 1"), node("2", "modbusIo", "pump 1")]
		expect(pickChild(kids, { type: "modbusIo", description: "PUMP 1" })?.id).toBe("2")
		expect(pickChild(kids, { type: "modbusIo", description: "pump 2" })).toBeUndefined()
	})

	test("the path below an ancestor comes from the chain", () => {
		const chain = [
			{ id: "g", type: "group", description: "site" },
			{ id: "d", type: "device", description: "Station A" },
			{ id: "p", type: "modbusIo", description: "pump1" },
		]
		expect(pathBelow(chain, "g")).toBe("device/Station A/modbusIo/pump1")
		expect(pathBelow(chain, "d")).toBe("modbusIo/pump1")
		expect(pathBelow(chain, "p")).toBeUndefined()
		expect(pathBelow(chain, "x")).toBeUndefined()
	})
})
