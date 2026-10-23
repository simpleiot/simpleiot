// Nodes as the client reads them, ported from the Elm UI's Api/Node.elm.
// The per-type knowledge (labels, where a type may be added, which parent
// owns it) lives in the node type registry, not here.

import { getBool, getText, type Point } from "./point.ts"
import { PointType } from "./schema.ts"

// Node is a node reached through one of its edges: points are the node's
// own and edgePoints those of the edge to parent.
export interface Node {
	id: string
	type: string
	parent: string
	points: Point[]
	edgePoints: Point[]
}

// EdgeRole is what an edge means for the node below it. A node that owns
// something outside the tree (a bus, a line, a socket) has one primary edge
// that runs its client, and any number of mirror edges that display it and
// run nothing. A node with no primary location, such as a user, has edges
// with no role.
export type EdgeRole = "none" | "primary" | "mirror"

// edgeRole reads the role of the edge a node was reached through. An edge
// carrying both points reads as a mirror, matching the backend.
export function edgeRole(node: Node): EdgeRole {
	if (getBool(node.edgePoints, PointType.Mirror)) {
		return "mirror"
	}
	if (getBool(node.edgePoints, PointType.Primary)) {
		return "primary"
	}
	return "none"
}

export function description(node: Node): string {
	return getText(node.points, PointType.Description)
}

// isTombstone is set when the edge to the parent is deleted.
export function isTombstone(node: Node): boolean {
	return getBool(node.edgePoints, PointType.Tombstone)
}

// NodeStatus is how a node is doing, from the points most node types share,
// for a glance at the tree without opening each node.
export type NodeStatus = "ok" | "disabled" | "error" | "offline"

// nodeStatus reads a node's status. A disabled node is only disabled, since
// its error or connection state no longer matters.
export function nodeStatus(node: Node): NodeStatus {
	if (getBool(node.points, PointType.Disabled)) {
		return "disabled"
	}
	if (getText(node.points, PointType.Error) !== "") {
		return "error"
	}
	if (
		getBool(node.points, PointType.Offline) ||
		getText(node.points, PointType.SysState) === "offline"
	) {
		return "offline"
	}
	return "ok"
}
