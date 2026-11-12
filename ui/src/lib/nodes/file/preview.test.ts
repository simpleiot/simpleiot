import { describe, expect, test } from "vitest"
import { testBound, writeText } from "./preview.ts"

describe("file preview test values", () => {
	test("empty is stale, a number is a value, anything else is text", () => {
		expect(testBound(" ").state).toBe("stale")
		expect(testBound("2.5")).toMatchObject({ state: "live", value: 2.5, text: "" })
		expect(testBound("open")).toMatchObject({ state: "live", value: 0, text: "open" })
	})

	test("a click puts its value in the test value", () => {
		expect(writeText({ value: 1, text: "" })).toBe("1")
		expect(writeText({ value: 0, text: "auto" })).toBe("auto")
	})
})
