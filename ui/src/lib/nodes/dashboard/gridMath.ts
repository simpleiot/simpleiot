// Grid geometry for dragging and resizing widgets: pointer movement in
// pixels becomes whole columns and rows, and a layout is kept on the grid.

import type { Layout } from "./bindings.ts"

export interface GridMetrics {
	// colWidth and rowHeight are the cell sizes in pixels, gap the space
	// between cells
	colWidth: number
	rowHeight: number
	gap: number
}

// metrics derives the cell sizes from the grid's width.
export function metrics(width: number, cols: number, rowHeight = 72, gap = 12): GridMetrics {
	return { colWidth: Math.max(1, (width - gap * (cols - 1)) / cols), rowHeight, gap }
}

// cellDelta turns a pointer movement into whole cells, rounding to the
// nearest so a drag snaps when it crosses the middle of a cell.
export function cellDelta(dx: number, dy: number, m: GridMetrics): { dc: number; dr: number } {
	return {
		dc: Math.round(dx / (m.colWidth + m.gap)),
		dr: Math.round(dy / (m.rowHeight + m.gap)),
	}
}

// clampLayout keeps a layout on a grid of cols columns: at least one cell,
// never past the right edge, never above the top.
export function clampLayout(l: Layout, cols: number): Layout {
	const w = Math.max(1, Math.min(l.w, cols))
	const x = Math.max(0, Math.min(l.x, cols - w))
	return { x, y: Math.max(0, l.y), w, h: Math.max(1, l.h) }
}

// moved applies a drag to a layout: move shifts the place, resize grows
// the size.
export function moved(
	start: Layout,
	mode: "move" | "resize",
	dc: number,
	dr: number,
	cols: number
): Layout {
	return clampLayout(
		mode === "move"
			? { ...start, x: start.x + dc, y: start.y + dr }
			: { ...start, w: start.w + dc, h: start.h + dr },
		cols
	)
}

export function sameLayout(a: Layout, b: Layout): boolean {
	return a.x === b.x && a.y === b.y && a.w === b.w && a.h === b.h
}

// nextRow is the first free row below every widget, where a new one goes.
export function nextRow(layouts: readonly Layout[]): number {
	return layouts.reduce((max, l) => Math.max(max, l.y + l.h), 0)
}
