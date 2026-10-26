import { describe, expect, test } from "vitest"
import { createRegistry, defineNodeType, type Icon } from "./registry.ts"

const icon = (() => undefined) as unknown as Icon

const registry = createRegistry([
	defineNodeType({
		type: "rule",
		label: "Rule",
		icon,
		parents: ["group"],
		hasChildren: true,
		sortKey: "E",
	}),
	defineNodeType({ type: "user", label: "User", icon, parents: ["group", "device"], sortKey: "B" }),
	defineNodeType({
		type: "condition",
		label: "Condition",
		icon,
		parents: ["rule"],
		ownedBy: "rule",
	}),
	defineNodeType({ type: "file", label: "File", icon, parents: ["group", "rule"] }),
])

describe("registry", () => {
	test("addableUnder lists the types for a parent in sort order", () => {
		expect(registry.addableUnder("group").map((d) => d.type)).toEqual(["user", "rule", "file"])
		expect(registry.addableUnder("rule").map((d) => d.type)).toEqual(["condition", "file"])
		expect(registry.addableUnder("variable")).toEqual([])
	})

	test("answers for known and unknown types", () => {
		expect(registry.hasChildren("rule")).toBe(true)
		expect(registry.hasChildren("user")).toBe(false)
		expect(registry.hasChildren("nope")).toBe(false)
		expect(registry.ownerOf("condition")).toBe("rule")
		expect(registry.ownerOf("user")).toBeUndefined()
		expect(registry.sortKey("user")).toBe("B")
		expect(registry.sortKey("nope")).toBe("nope")
		expect(registry.editorFor("user")).toBeUndefined()
	})
})
