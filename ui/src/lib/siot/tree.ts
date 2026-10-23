// The node tree the UI shows: one tree per anchor (a group the user belongs
// to), fetched one subtree at a time and kept current from live points.
// Ported from the Elm UI's Utils/NodeTree.elm. Everything here is pure, and
// the functions leave unchanged subtrees as they were, so a component given
// one can tell it did not change.
//
// A reply to a fetch is a flat list of nodes; the parent field of each says
// where it goes. depth is how many levels below the requested nodes the
// reply reached: nodes at that level keep whatever children the tree
// already had for them, since the reply did not look.

import { isTombstone, type Node } from "./node.ts"
import { getBestDesc, getText, getValue, mergePoints, type Point } from "./point.ts"
import { PointType } from "./schema.ts"

export interface TreeNode {
	// anchor is the group the node was fetched under, which every request
	// about it names
	anchor: string
	node: Node
	// key is unique on the page: the IDs on the path from the anchor
	key: string
	// parentId is the node's parent in this tree, "" for the anchor
	parentId: string
	expChildren: boolean
	expDetail: boolean
	// loading is set while the node's children are being fetched
	loading: boolean
	// hasChildren is set when the node has children that are not deleted
	hasChildren: boolean
	children: TreeNode[]
}

export type Forest = TreeNode[]

// nodeView wraps a node fetched under an anchor.
export function nodeView(anchor: string, node: Node, children: TreeNode[] = []): TreeNode {
	return {
		anchor,
		node,
		key: "",
		parentId: node.parent,
		expChildren: false,
		expDetail: false,
		loading: false,
		hasChildren: false,
		children,
	}
}

// ---- building

// buildForest builds the subtrees of the nodes in the list whose parent is
// the one given, taking their descendants from the same list.
export function buildForest(anchor: string, parent: string, nodes: readonly Node[]): TreeNode[] {
	return nodes.filter((n) => n.parent === parent).map((n) => buildTree(anchor, nodes, [], n))
}

function buildTree(anchor: string, nodes: readonly Node[], path: string[], node: Node): TreeNode {
	const children = path.includes(node.id)
		? []
		: nodes
				.filter((n) => n.parent === node.id)
				.map((n) => buildTree(anchor, nodes, [node.id, ...path], n))
	return nodeView(anchor, node, children)
}

// ---- merging

// mergeForest merges freshly fetched subtrees into the ones already there,
// matching by node ID. Expansion state is kept, nodes that are gone are
// dropped, and at the depth the reply did not reach the old children stay.
export function mergeForest(
	depth: number,
	old: readonly TreeNode[],
	fresh: readonly TreeNode[]
): TreeNode[] {
	return fresh.map((n) => {
		const o = old.find((t) => t.node.id === n.node.id)
		return o ? mergeTree(depth, o, n) : n
	})
}

function mergeTree(depth: number, old: TreeNode, fresh: TreeNode): TreeNode {
	return {
		...fresh,
		expChildren: old.expChildren,
		expDetail: old.expDetail,
		children: depth <= 0 ? old.children : mergeForest(depth - 1, old.children, fresh.children),
	}
}

// replaceAnchor puts the tree fetched for an anchor (a request for
// nodes.all.<anchor>) in place of the one already there, or adds it.
export function replaceAnchor(
	anchor: string,
	depth: number,
	nodes: readonly Node[],
	forest: Forest
): Forest {
	const mine = forest.filter((t) => t.anchor === anchor)
	const others = forest.filter((t) => t.anchor !== anchor)
	const top = nodes.find((n) => n.id === anchor)
	if (!top) {
		return others
	}
	const fresh = nodeView(anchor, { ...top, parent: "root" }, buildForest(anchor, anchor, nodes))
	return [...others, ...mergeForest(depth, mine, [fresh])]
}

// replaceChildren puts the children fetched for a node (a request for
// nodes.<parent>.all) under every copy of that node in the anchor's tree,
// and clears its loading flag.
export function replaceChildren(
	anchor: string,
	parentId: string,
	depth: number,
	nodes: readonly Node[],
	forest: Forest
): Forest {
	const fresh = buildForest(anchor, parentId, nodes)
	return inAnchor(anchor, forest, (t) =>
		replaceIn(parentId, t, (p) => ({
			...p,
			loading: false,
			children: mergeForest(depth, p.children, fresh),
		}))
	)
}

// replaceChild puts one child fetched for a node (a request for
// nodes.<parent>.<id>) under every copy of that node in the anchor's tree,
// replacing the copy that was there or adding it.
export function replaceChild(
	anchor: string,
	parentId: string,
	childId: string,
	depth: number,
	nodes: readonly Node[],
	forest: Forest
): Forest {
	const fresh = buildForest(anchor, parentId, nodes).find((t) => t.node.id === childId)
	if (!fresh) {
		return forest
	}
	return inAnchor(anchor, forest, (t) =>
		replaceIn(parentId, t, (p) => ({
			...p,
			loading: false,
			children: p.children.some((c) => c.node.id === childId)
				? p.children.map((c) => (c.node.id === childId ? mergeTree(depth, c, fresh) : c))
				: [...p.children, fresh],
		}))
	)
}

function inAnchor(anchor: string, forest: Forest, f: (t: TreeNode) => TreeNode): Forest {
	return forest.map((t) => (t.anchor === anchor ? f(t) : t))
}

// replaceIn applies a function to every subtree whose root has the ID.
function replaceIn(id: string, tree: TreeNode, f: (t: TreeNode) => TreeNode): TreeNode {
	if (tree.node.id === id) {
		return f(tree)
	}
	return withChildren(
		tree,
		mapSame(tree.children, (c) => replaceIn(id, c, f))
	)
}

// ---- mapping with sharing

// mapSame maps a list and returns the same list when no element changed.
function mapSame<T>(list: T[], f: (t: T) => T): T[] {
	let out: T[] | undefined
	for (let i = 0; i < list.length; i++) {
		const v = list[i] as T
		const w = f(v)
		if (w !== v) {
			out ??= list.slice()
			out[i] = w
		}
	}
	return out ?? list
}

function withChildren(t: TreeNode, children: TreeNode[]): TreeNode {
	return children === t.children ? t : { ...t, children }
}

// mapTree applies f to every node of a tree, top down. f returns the node
// unchanged to leave it as it was.
export function mapTree(t: TreeNode, f: (n: TreeNode) => TreeNode): TreeNode {
	const n = f(t)
	return withChildren(
		n,
		mapSame(n.children, (c) => mapTree(c, f))
	)
}

export function mapForest(forest: Forest, f: (n: TreeNode) => TreeNode): Forest {
	return mapSame(forest, (t) => mapTree(t, f))
}

// ---- live points

// applyPoints merges points into every copy of a node.
export function applyPoints(nodeId: string, points: readonly Point[], forest: Forest): Forest {
	return mapForest(forest, (n) =>
		n.node.id === nodeId
			? { ...n, node: { ...n.node, points: mergePoints(n.node.points, points) } }
			: n
	)
}

export interface Missing {
	anchor: string
	parentId: string
	expanded: boolean
}

// applyEdgePoints merges points into the edge between a node and its parent
// wherever the tree has it. Where the tree has the parent but not the
// child, it reports the parent to fetch the child under, once per anchor,
// along with whether the parent is expanded.
export function applyEdgePoints(
	nodeId: string,
	parentId: string,
	points: readonly Point[],
	forest: Forest
): [Forest, Missing[]] {
	const updated = mapForest(forest, (n) =>
		n.node.id === nodeId && n.node.parent === parentId
			? { ...n, node: { ...n.node, edgePoints: mergePoints(n.node.edgePoints, points) } }
			: n
	)
	const missing: Missing[] = []
	for (const t of forest) {
		const all = flatten(t)
		const parents = all.filter((n) => n.node.id === parentId)
		const hasChild = all.some((n) => n.node.id === nodeId && n.node.parent === parentId)
		if (parents.length > 0 && !hasChild) {
			missing.push({ anchor: t.anchor, parentId, expanded: parents.some((p) => p.expChildren) })
		}
	}
	return [updated, missing]
}

// ---- state

export function setExpanded(key: string, expanded: boolean, forest: Forest): Forest {
	return mapForest(forest, (n) =>
		n.key === key ? { ...n, expChildren: expanded, loading: expanded } : n
	)
}

export function setDetail(key: string, open: boolean, forest: Forest): Forest {
	return mapForest(forest, (n) =>
		n.key === key && n.expDetail !== open ? { ...n, expDetail: open } : n
	)
}

export function clearLoading(forest: Forest): Forest {
	return mapForest(forest, (n) => (n.loading ? { ...n, loading: false } : n))
}

export function flatten(t: TreeNode): TreeNode[] {
	const out: TreeNode[] = [t]
	for (const c of t.children) {
		out.push(...flatten(c))
	}
	return out
}

// anchorOf is the anchor a node was fetched under, which a request about it
// has to name. A node in two groups may have two; the first is as good as
// any.
export function anchorOf(id: string, forest: Forest): string | undefined {
	return findById(id, forest)?.anchor
}

export function findById(id: string, forest: Forest): TreeNode | undefined {
	for (const t of forest) {
		const n = flatten(t).find((n) => n.node.id === id)
		if (n) {
			return n
		}
	}
	return undefined
}

export function findByKey(key: string, forest: Forest): TreeNode | undefined {
	for (const t of forest) {
		const n = flatten(t).find((n) => n.key === key)
		if (n) {
			return n
		}
	}
	return undefined
}

// visible lists the nodes a tree shows: the root, and the live children of
// every expanded node below it.
export function visible(t: TreeNode): TreeNode[] {
	const out: TreeNode[] = [t]
	if (t.expChildren) {
		for (const c of t.children) {
			if (!isTombstone(c.node)) {
				out.push(...visible(c))
			}
		}
	}
	return out
}

// expandedIds lists every expanded node that is on screen, as the anchor
// and ID to refetch its children with.
export function expandedIds(forest: Forest): { anchor: string; id: string }[] {
	const seen = new Set<string>()
	const out: { anchor: string; id: string }[] = []
	for (const t of forest) {
		for (const n of visible(t)) {
			const k = `${n.anchor}/${n.node.id}`
			if (n.expChildren && !seen.has(k)) {
				seen.add(k)
				out.push({ anchor: n.anchor, id: n.node.id })
			}
		}
	}
	return out
}

// watchSubjects is what the page needs to hear about: the points of every
// node on screen, and the edges of its children, so a new, deleted, or
// re-roled child is noticed.
export function watchSubjects(forest: Forest): string[] {
	const subjects = new Set<string>()
	for (const t of forest) {
		for (const n of visible(t)) {
			subjects.add(`up.${n.anchor}.${n.node.id}.*.*`)
			subjects.add(`up.${n.anchor}.*.${n.node.id}.*.*`)
		}
	}
	return [...subjects].sort()
}

// ---- finishing

// SortKey maps a node type to the string siblings of that type sort by.
export type SortKey = (type: string) => string

// finish brings a changed forest back to what the view expects: each node
// knows its parent, its key, and whether it has live children, and
// siblings are sorted.
export function finish(forest: Forest, sortKey: SortKey): Forest {
	const order = (a: TreeNode, b: TreeNode) => compareNodes(a, b, sortKey)
	return sortSame(forest, order).map((t) => populate(t, "", "", order))
}

function populate(
	t: TreeNode,
	parentId: string,
	parentKey: string,
	order: (a: TreeNode, b: TreeNode) => number
): TreeNode {
	const key = parentKey === "" ? `${t.anchor}/${t.node.id}` : `${parentKey}/${t.node.id}`
	const children = mapSame(sortSame(t.children, order), (c) => populate(c, t.node.id, key, order))
	const hasChildren = children.some((c) => !isTombstone(c.node))
	if (
		t.key === key &&
		t.parentId === parentId &&
		t.hasChildren === hasChildren &&
		t.children === children
	) {
		return t
	}
	return { ...t, key, parentId, hasChildren, children }
}

// sortSame sorts a list, returning the same list when it was in order.
function sortSame<T>(list: T[], order: (a: T, b: T) => number): T[] {
	for (let i = 1; i < list.length; i++) {
		if (order(list[i - 1] as T, list[i] as T) > 0) {
			return list.slice().sort(order)
		}
	}
	return list
}

const cmp = (a: string | number, b: string | number) => (a < b ? -1 : a > b ? 1 : 0)

// compareNodes sorts siblings by type, then description, then index, then
// ID point.
function compareNodes(a: TreeNode, b: TreeNode, sortKey: SortKey): number {
	const at = sortKey(a.node.type)
	const bt = sortKey(b.node.type)
	if (at !== bt) {
		return cmp(at, bt)
	}
	const ad = getBestDesc(a.node.points).toLowerCase()
	const bd = getBestDesc(b.node.points).toLowerCase()
	if (ad !== bd) {
		return cmp(ad, bd)
	}
	const ai = getValue(a.node.points, PointType.Index)
	const bi = getValue(b.node.points, PointType.Index)
	if (ai !== bi) {
		return cmp(ai, bi)
	}
	return cmp(getText(a.node.points, PointType.ID), getText(b.node.points, PointType.ID))
}
