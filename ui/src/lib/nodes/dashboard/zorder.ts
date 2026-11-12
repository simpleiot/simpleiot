// Stacking order of widgets on a canvas. Each widget has a z point, 0 when
// it has none, and widgets are drawn from the lowest z to the highest, so
// a higher one sits on top. Widgets with the same z are drawn in ID order,
// so the order is the same on every page.

import type { Node } from "#lib/siot/node.ts"
import { get } from "#lib/siot/point.ts"
import { PointType } from "#lib/siot/schema.ts"

export interface Stacked {
	id: string
	z: number
}

// widgetZ reads a widget's z, 0 when it has none.
export function widgetZ(widget: Node): number {
	const p = get(widget.points, PointType.Z)
	return p && p.tombstone === 0 && Number.isFinite(p.value) ? p.value : 0
}

// byStack orders items from the bottom of the stack to the top.
export function byStack(a: Stacked, b: Stacked): number {
	return a.z !== b.z ? a.z - b.z : a.id < b.id ? -1 : a.id > b.id ? 1 : 0
}

// topZ is the z that puts a new widget above all the others.
export function topZ(items: readonly Stacked[]): number {
	return items.length === 0 ? 0 : Math.max(...items.map((i) => i.z)) + 1
}

export type StackMove = "front" | "back" | "forward" | "backward"

// restack moves one item in the stack and returns the z values to write,
// by ID. Bringing to the front or sending to the back writes one value.
// Moving one step swaps the item with its neighbor, and renumbers the
// stack when that is needed to separate equal values. Nothing is returned
// when the item is already where the move would put it.
export function restack(
	items: readonly Stacked[],
	id: string,
	move: StackMove
): Map<string, number> {
	const out = new Map<string, number>()
	const order = items.slice().sort(byStack)
	const i = order.findIndex((s) => s.id === id)
	const item = order[i]
	if (!item) {
		return out
	}
	const others = order.filter((s) => s.id !== id)
	if (move === "front") {
		if (i < order.length - 1) {
			out.set(id, topZ(others))
		}
		return out
	}
	if (move === "back") {
		if (i > 0) {
			out.set(id, Math.min(...others.map((s) => s.z)) - 1)
		}
		return out
	}
	const j = move === "forward" ? i + 1 : i - 1
	const other = order[j]
	if (!other) {
		return out
	}
	if (other.z !== item.z) {
		out.set(id, other.z)
		out.set(other.id, item.z)
		return out
	}
	// equal values cannot be swapped, so the stack is numbered from 0 in
	// its new order
	order[i] = other
	order[j] = item
	order.forEach((s, k) => {
		if (s.z !== k) {
			out.set(s.id, k)
		}
	})
	return out
}
