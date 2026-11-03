// Node search results, as the node reference picker shows them. The
// server answers a search with the matches and their ancestors in one flat
// list; this module tells them apart, ranks the matches as the server does,
// and gives each one the path it sits on.

import type { Node } from "./node.ts"
import { getBestDesc } from "./point.ts"
import { PointType } from "./schema.ts"

export interface NodeMatch {
	id: string
	type: string
	description: string
	// path is the descriptions of the ancestors below the anchor, nearest
	// last, with the anchor's own description first when it is known
	path: string[]
	// rank is 0 for a description starting with the query, 1 for one
	// containing it, 2 for an equal ID, and 3 for a type containing it
	rank: number
}

// rankMatch scores a node against a query the way the store does, or -1
// when it does not match.
export function rankMatch(node: Node, query: string): number {
	const q = query.trim().toLowerCase()
	if (q === "") {
		return -1
	}
	const desc = node.points
		.find((p) => p.type === PointType.Description && p.key === "0" && p.tombstone === 0)
		?.text.toLowerCase()
	if (desc !== undefined && desc !== "" && desc.startsWith(q)) {
		return 0
	}
	if (desc?.includes(q)) {
		return 1
	}
	if (node.id.toLowerCase() === q) {
		return 2
	}
	if (node.type.toLowerCase().includes(q)) {
		return 3
	}
	return -1
}

// buildMatches turns one anchor's search reply into matches with paths.
export function buildMatches(
	nodes: readonly Node[],
	query: string,
	anchor: string,
	anchorName?: string,
	type = ""
): NodeMatch[] {
	const byId = new Map<string, Node>()
	for (const n of nodes) {
		if (!byId.has(n.id)) {
			byId.set(n.id, n)
		}
	}
	const matches: NodeMatch[] = []
	for (const n of nodes) {
		if (type !== "" && n.type !== type) {
			continue
		}
		const rank = query.trim() === "" && type !== "" ? 0 : rankMatch(n, query)
		if (rank < 0) {
			continue
		}
		const path: string[] = []
		const seen = new Set<string>([n.id])
		for (
			let p = byId.get(n.parent);
			p && p.id !== anchor && !seen.has(p.id);
			p = byId.get(p.parent)
		) {
			seen.add(p.id)
			path.unshift(getBestDesc(p.points))
		}
		if (anchorName !== undefined && anchorName !== "") {
			path.unshift(anchorName)
		}
		matches.push({ id: n.id, type: n.type, description: getBestDesc(n.points), path, rank })
	}
	return matches
}

// rankMatches orders matches by rank then description, keeping the first
// of any node reached twice.
export function rankMatches(matches: readonly NodeMatch[]): NodeMatch[] {
	const seen = new Set<string>()
	return matches
		.slice()
		.sort(
			(a, b) =>
				a.rank - b.rank || a.description.toLowerCase().localeCompare(b.description.toLowerCase())
		)
		.filter((m) => !seen.has(m.id) && seen.add(m.id))
}

// moveActive is the keyboard rule for a list: the arrows move and wrap,
// and a list with nothing in it has nothing active.
export function moveActive(active: number, key: "ArrowDown" | "ArrowUp", length: number): number {
	if (length === 0) {
		return -1
	}
	if (key === "ArrowDown") {
		return active < 0 || active >= length - 1 ? 0 : active + 1
	}
	return active <= 0 ? length - 1 : active - 1
}
