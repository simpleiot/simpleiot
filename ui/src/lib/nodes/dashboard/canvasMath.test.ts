import { describe, expect, test } from "vitest"
import { canvasDelta, fitScale, parsePair, toCanvas, zoomAt } from "./canvasMath.ts"

describe("canvas math", () => {
	test("fit scale spans the frame, and falls back to one", () => {
		expect(fitScale(450, 900)).toBe(0.5)
		expect(fitScale(0, 900)).toBe(1)
	})

	test("frame positions map back to canvas units", () => {
		const v = { scale: 0.5, panX: 10, panY: 20 }
		expect(toCanvas(60, 70, v)).toEqual([100, 100])
		expect(canvasDelta(25, -25, v)).toEqual([50, -50])
	})

	test("zoom keeps the point under the pointer still, within bounds", () => {
		const v = { scale: 1, panX: 0, panY: 0 }
		const z = zoomAt(v, 2, 100, 50, 1)
		expect(z.scale).toBe(2)
		expect(toCanvas(100, 50, z)).toEqual([100, 50])
		expect(zoomAt(v, 100, 0, 0, 1).scale).toBe(8)
		expect(zoomAt(v, 0.01, 0, 0, 1).scale).toBe(0.25)
	})

	test("pairs parse or not", () => {
		expect(parsePair("12, 34")).toEqual([12, 34])
		expect(parsePair("12")).toBeUndefined()
		expect(parsePair("a,b")).toBeUndefined()
	})
})
