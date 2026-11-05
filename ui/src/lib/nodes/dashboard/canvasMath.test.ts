import { describe, expect, test } from "vitest"
import { canvasDelta, fitScale, parsePair, resized, toCanvas, zoomAt } from "./canvasMath.ts"

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

	test("a corner drag scales along the diagonal and holds the far corner", () => {
		const start = { x: 100, y: 100, w: 200, h: 0 }
		const box = { w: 200, h: 100 }
		// the bottom right corner dragged out along the diagonal doubles it
		expect(resized(start, box, "se", 200, 100)).toEqual({ x: 100, y: 100, w: 400, h: 0 })
		// a purely vertical drag resizes too
		expect(resized(start, box, "se", 0, 50)).toEqual({ x: 100, y: 100, w: 220, h: 0 })
		// the top left corner dragged in keeps the bottom right at 300,200
		expect(resized(start, box, "nw", 100, 50)).toEqual({ x: 200, y: 150, w: 100, h: 0 })
		// the top right corner keeps the bottom left, and the bottom left the top right
		expect(resized(start, box, "ne", 200, -100)).toEqual({ x: 100, y: 0, w: 400, h: 0 })
		expect(resized(start, box, "sw", -200, 100)).toEqual({ x: 0, y: 100, w: 400, h: 0 })
		// no smaller than 40, and never off the plane
		expect(resized(start, box, "nw", 400, 200).w).toBe(40)
		expect(resized(start, box, "nw", -400, -200)).toEqual({ x: 0, y: 0, w: 600, h: 0 })
	})

	test("pairs parse or not", () => {
		expect(parsePair("12, 34")).toEqual([12, 34])
		expect(parsePair("12")).toBeUndefined()
		expect(parsePair("a,b")).toBeUndefined()
	})
})
