import { describe, expect, test } from "vitest"
import { cellDelta, clampLayout, metrics, moved, nextRow, sameLayout } from "./gridMath.ts"

describe("grid math", () => {
	test("metrics divide the width into columns less the gaps", () => {
		const m = metrics(1200, 12)
		expect(m.colWidth).toBeCloseTo((1200 - 12 * 11) / 12)
		expect(metrics(0, 12).colWidth).toBe(1)
	})

	test("cell delta snaps at half a cell", () => {
		const m = { colWidth: 88, rowHeight: 72, gap: 12 }
		expect(cellDelta(40, 0, m)).toEqual({ dc: 0, dr: 0 })
		expect(cellDelta(60, 0, m)).toEqual({ dc: 1, dr: 0 })
		expect(cellDelta(-160, 130, m)).toEqual({ dc: -2, dr: 2 })
	})

	test("clamp keeps a layout on the grid", () => {
		expect(clampLayout({ x: 11, y: -1, w: 3, h: 0 }, 12)).toEqual({ x: 9, y: 0, w: 3, h: 1 })
		expect(clampLayout({ x: 0, y: 0, w: 20, h: 2 }, 12)).toEqual({ x: 0, y: 0, w: 12, h: 2 })
	})

	test("moved shifts or grows, within the grid", () => {
		const start = { x: 2, y: 1, w: 3, h: 2 }
		expect(moved(start, "move", 1, 1, 12)).toEqual({ x: 3, y: 2, w: 3, h: 2 })
		expect(moved(start, "resize", 2, -1, 12)).toEqual({ x: 2, y: 1, w: 5, h: 1 })
		expect(moved(start, "move", 20, 0, 12)).toEqual({ x: 9, y: 1, w: 3, h: 2 })
		expect(sameLayout(start, moved(start, "move", 0, 0, 12))).toBe(true)
	})

	test("the next row is below everything", () => {
		expect(nextRow([])).toBe(0)
		expect(
			nextRow([
				{ x: 0, y: 0, w: 3, h: 2 },
				{ x: 3, y: 1, w: 3, h: 3 },
			])
		).toBe(4)
	})
})
