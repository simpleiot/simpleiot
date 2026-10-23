import { describe, expect, test } from "vitest"
import { isTombstone, type Node } from "./node.ts"
import { DataType, newText, type Point } from "./point.ts"
import {
	applyEdgePoints,
	finish,
	mergeForest,
	nodeView,
	replaceChildren,
	setExpanded,
	watchSubjects,
	type TreeNode,
} from "./tree.ts"

const tombstone = (v: number): Point => ({
	type: "tombstone",
	key: "0",
	time: new Date(0).toISOString(),
	dataType: DataType.Float,
	value: v,
	text: "",
	tombstone: 0,
})

const node = (id: string, type: string, parent: string): Node => ({
	id,
	type,
	parent,
	points: [newText("description", "0", id)],
	edgePoints: [tombstone(0)],
})

const tree = (anchor: string, n: Node, children: TreeNode[] = []) => nodeView(anchor, n, children)
const expand = (t: TreeNode): TreeNode => ({ ...t, expChildren: true })
const ids = (ts: TreeNode[]) => ts.map((t) => t.node.id)

// groups sort before variables, as the registry orders them
const sortKey = (type: string) => (type === "group" ? "C" : type)

describe("tree", () => {
	test("mergeForest keeps expansion and the levels the reply did not reach", () => {
		const old = [
			expand(
				tree("G", node("a", "group", "G"), [
					tree("G", node("a1", "variable", "a"), [tree("G", node("a1x", "variable", "a1"))]),
					tree("G", node("stale", "variable", "a")),
				])
			),
		]
		const fresh = [
			tree("G", node("a", "group", "G"), [
				tree("G", node("a1", "variable", "a")),
				tree("G", node("a2", "variable", "a")),
			]),
		]
		const [m] = mergeForest(1, old, fresh)
		expect(m?.expChildren).toBe(true)
		expect(ids(m?.children ?? [])).toEqual(["a1", "a2"])
		// a1 sat at the depth the reply stopped at, so its children stay
		expect(ids(m?.children[0]?.children ?? [])).toEqual(["a1x"])
	})

	test("replaceChildren fills in a fetched subtree and clears loading", () => {
		let forest = finish([tree("G", node("G", "group", "root"))], sortKey)
		forest = setExpanded(forest[0]?.key ?? "", true, forest)
		expect(forest[0]?.loading).toBe(true)
		const reply = [
			node("c1", "variable", "G"),
			node("c2", "group", "G"),
			node("c2a", "variable", "c2"),
		]
		const [top] = finish(replaceChildren("G", "G", 1, reply, forest), sortKey)
		expect(top?.loading).toBe(false)
		expect(ids(top?.children ?? [])).toEqual(["c2", "c1"])
		expect(top?.children.map((c) => c.hasChildren)).toEqual([true, false])
		expect(top?.children[0]?.key).toBe("G/G/c2")
	})

	test("watchSubjects covers the nodes on screen", () => {
		const gone: Node = { ...node("gone", "variable", "G"), points: [], edgePoints: [tombstone(1)] }
		const forest = [
			expand(
				tree("G", node("G", "group", "root"), [
					tree("G", node("c1", "variable", "G"), [tree("G", node("hidden", "variable", "c1"))]),
					tree("G", gone),
				])
			),
		]
		expect(watchSubjects(forest)).toEqual([
			"up.G.*.G.*.*",
			"up.G.*.c1.*.*",
			"up.G.G.*.*",
			"up.G.c1.*.*",
		])
	})

	test("a tombstone edge point hides a node", () => {
		const forest = [tree("G", node("G", "group", "root"), [tree("G", node("c1", "variable", "G"))])]
		const [result, missing] = applyEdgePoints("c1", "G", [tombstone(1)], forest)
		const child = result[0]?.children[0]
		expect(child && isTombstone(child.node)).toBe(true)
		expect(missing).toEqual([])
	})

	test("an edge for a node the tree lacks names the parent to fetch under", () => {
		const forest = [expand(tree("G", node("G", "group", "root")))]
		const [, missing] = applyEdgePoints("new", "G", [tombstone(0)], forest)
		expect(missing).toEqual([{ anchor: "G", parentId: "G", expanded: true }])
	})

	test("finish leaves an unchanged forest as it was", () => {
		const forest = finish(
			[tree("G", node("G", "group", "root"), [tree("G", node("c1", "variable", "G"))])],
			sortKey
		)
		expect(finish(forest, sortKey)[0]).toBe(forest[0])
	})
})
