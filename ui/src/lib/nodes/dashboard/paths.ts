// Relative bindings name a node by a path from the dashboard's parent:
// "type/description" segments, each picking one child of the node before
// it, so one dashboard placed under many devices shows each device's own
// nodes.

import type { Node } from "#lib/siot/node.ts"
import { getBestDesc } from "#lib/siot/point.ts"

export interface Segment {
	type: string
	description: string
}

// splitPath reads "type/description/type/description..." into segments,
// or undefined when the path is not well formed.
export function splitPath(path: string): Segment[] | undefined {
	const parts = path.split("/").map((s) => s.trim())
	if (parts.length === 0 || parts.length % 2 !== 0 || parts.some((p) => p === "")) {
		return undefined
	}
	const out: Segment[] = []
	for (let i = 0; i < parts.length; i += 2) {
		out.push({ type: parts[i] ?? "", description: parts[i + 1] ?? "" })
	}
	return out
}

// joinPath writes segments back as a path.
export function joinPath(segments: readonly Segment[]): string {
	return segments.map((s) => `${s.type}/${s.description}`).join("/")
}

// pickChild finds the child a segment names: the first of the type whose
// description matches, ignoring case.
export function pickChild(children: readonly Node[], seg: Segment): Node | undefined {
	const want = seg.description.toLowerCase()
	return children.find((c) => c.type === seg.type && getBestDesc(c.points).toLowerCase() === want)
}

// pathBelow is the path from an ancestor to a node, given the chain of
// nodes from the top down to the node itself, or undefined when the
// ancestor is not in the chain.
export function pathBelow(
	chain: readonly { id: string; type: string; description: string }[],
	ancestorId: string
): string | undefined {
	const i = chain.findIndex((c) => c.id === ancestorId)
	if (i < 0 || i === chain.length - 1) {
		return undefined
	}
	return joinPath(chain.slice(i + 1))
}
