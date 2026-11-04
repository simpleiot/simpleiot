// The tree page's state: the node tree, the connection that keeps it
// current, pending edits, and the node operation in progress. It is the
// one place, with the page, that talks to the server.

import { registry } from "#lib/nodes/index.ts"
import { prefs, type Expanded } from "#lib/prefs.svelte.ts"
import type { Clipboard, GeneratedToken } from "#lib/nodes/types.ts"
import type { Env } from "./connection.ts"
import { Connection } from "./connection.ts"
import * as http from "./http.ts"
import { edgeRole, isTombstone, type Node } from "./node.ts"
import {
	clearText,
	DataType,
	getBestDesc,
	getBool,
	getText,
	newText,
	updatePoints,
	type Point,
} from "./point.ts"
import { PointType } from "./schema.ts"
import * as tree from "./tree.ts"
import type { Forest, TreeNode } from "./tree.ts"
import { SvelteMap } from "svelte/reactivity"
import { pickChild, splitPath } from "#lib/nodes/dashboard/paths.ts"
import { buildMatches, rankMatches, type NodeMatch } from "./search.ts"

// Edit is the pending edit of one node, by its key on the page.
export interface Edit {
	key: string
	points: Point[]
	// viewRaw shows the node as its points
	viewRaw: boolean
}

export type NodeOp =
	| { kind: "none" }
	| { kind: "add"; key: string; type: string | null }
	| { kind: "message"; key: string; message: string }
	| { kind: "delete"; key: string }
	| { kind: "paste"; key: string }

export interface NodeMsg {
	key: string
	text: string
	time: number
}

// a notice says an operation went through, and fades; an error stays
// until it is dismissed
const noticeShowMs = 3000
// searchDepth is how far down the first search of a session fetches
const searchDepth = 20
const nodeMsgShowMs = 3000

const errorText = (err: unknown) => (err instanceof Error ? err.message : String(err))

export class TreeStore {
	forest = $state.raw<Forest>([])
	connected = $state(false)
	// ready is set once the groups have been fetched after a connect
	ready = $state(false)
	userId = $state("")
	anchors = $state.raw<string[]>([])
	error = $state<string | null>(null)
	notice = $state<{ text: string; time: number } | null>(null)
	now = $state(Date.now())
	readonly zone = Intl.DateTimeFormat().resolvedOptions().timeZone

	edit = $state<Edit | null>(null)
	op = $state<NodeOp>({ kind: "none" })
	clipboard = $state<Clipboard | null>(null)
	scratch = $state("")
	nodeMsg = $state<NodeMsg | null>(null)
	generatedToken = $state<GeneratedToken | null>(null)
	addPoint = $state({ type: "", key: "" })
	customNodeType = $state("")

	// query filters the tree; searchKeys are the nodes it shows, or null
	query = $state("")
	searching = $state(false)
	searchKeys = $derived(this.query.trim() === "" ? null : tree.search(this.forest, this.query))
	// focusKey is the row that takes focus in the tree
	focusKey = $state("")
	// rows are the nodes on screen, in order
	rows = $derived(tree.shown(this.forest, this.searchKeys))
	// tabKey is the row the tree's tab stop is on
	tabKey = $derived(
		this.rows.some((r) => r.key === this.focusKey) ? this.focusKey : (this.rows[0]?.key ?? "")
	)

	// onConnected runs after each connect, once the anchors are fetched;
	// the page uses it to open a deep link
	onConnected: (() => void) | null = null
	// onDetail runs when a node's detail is opened or closed from the tree
	onDetail: ((node: Node, open: boolean) => void) | null = null

	#conn: Connection | null = null
	#token = ""
	#watching: string[] = []
	#tick: ReturnType<typeof setInterval> | null = null
	#onAuthFailed: () => void = () => undefined
	#deepFetched = false
	#saved = ""
	// resolved holds nodes fetched by ID for display off the tree; it is
	// dropped on each connect, since those nodes are not watched
	#resolved = new Map<string, Node>()

	// liveNodes are nodes a page off the tree, such as a dashboard, keeps
	// current: fetched by ID, then updated from their points as the tree
	// is. liveAnchors records the anchor each was reached through, and
	// liveEdges the nodes whose child edges the page wants to hear about.
	liveNodes = new SvelteMap<string, Node>()
	#liveAnchors = new Map<string, string>()
	#liveEdges = new Set<string>()
	// liveWants is what each watcher on the page asked for, by key; the
	// nodes kept live are the union
	#liveWants = new Map<string, { ids: string[]; edges: string[] }>()
	// paths caches relative bindings resolved from a parent
	#paths = new Map<string, string | undefined>()
	// onLiveEdge runs when an edge below a node in liveEdges changes
	onLiveEdge: ((parentId: string) => void) | null = null

	constructor(private readonly env: Env) {}

	// ---- lifecycle

	start(token: string, onAuthFailed: () => void) {
		if (this.#conn && token === this.#token) {
			return
		}
		void this.stop()
		this.#token = token
		this.#onAuthFailed = onAuthFailed
		this.#tick = setInterval(() => {
			this.#ticked()
		}, 1000)
		this.#conn = new Connection(
			{
				connected: (userId, anchors) => {
					this.#connected(userId, anchors)
				},
				disconnected: () => {
					this.connected = false
				},
				authFailed: () => {
					this.popError("Signed Out")
					this.#onAuthFailed()
				},
				points: (items) => {
					let f = this.forest
					for (const i of items) {
						f = tree.applyPoints(i.nodeId, i.points, f)
						const live = this.liveNodes.get(i.nodeId)
						if (live) {
							this.liveNodes.set(i.nodeId, {
								...live,
								points: updatePoints(live.points, i.points),
							})
						}
					}
					this.#setForest(f)
				},
				edgePoints: (items) => {
					this.#edgePoints(items)
				},
			},
			this.env
		)
		void this.#conn.connect(token).catch((err: unknown) => {
			this.popError(errorText(err))
		})
	}

	async stop() {
		if (this.#tick) {
			clearInterval(this.#tick)
			this.#tick = null
		}
		const c = this.#conn
		this.#conn = null
		this.#token = ""
		this.connected = false
		this.ready = false
		this.forest = []
		this.query = ""
		this.#deepFetched = false
		this.#saved = ""
		this.#watching = []
		this.liveNodes.clear()
		this.#liveAnchors = new Map()
		this.#liveEdges = new Set()
		this.#liveWants = new Map()
		this.#paths = new Map()
		this.edit = null
		this.op = { kind: "none" }
		this.clipboard = null
		this.generatedToken = null
		if (c) {
			await c.close()
		}
	}

	#ticked() {
		this.now = Date.now()
		if (this.notice && this.now - this.notice.time > noticeShowMs) {
			this.notice = null
		}
		if (this.nodeMsg && this.now - this.nodeMsg.time > nodeMsgShowMs) {
			this.nodeMsg = null
		}
	}

	popError(message: string) {
		this.error = message
	}

	#notify(text: string) {
		this.notice = { text, time: Date.now() }
	}

	dismissError() {
		this.error = null
	}

	// ---- the tree

	// setForest finishes a changed forest and keeps the subscriptions in
	// step with what is on screen.
	#setForest(f: Forest) {
		this.forest = tree.finish(f, (type) => registry.sortKey(type))
		this.#updateWatch()
		this.#remember()
	}

	#updateWatch() {
		if (!this.#conn || !this.connected) {
			return
		}
		const subjects = new Set(tree.watchSubjects(this.forest, this.searchKeys))
		for (const [id, anchor] of this.#liveAnchors) {
			subjects.add(`up.${anchor}.${id}.*.*`)
			if (this.#liveEdges.has(id)) {
				subjects.add(`up.${anchor}.*.${id}.*.*`)
			}
		}
		const sorted = [...subjects].sort()
		if (sorted.join("\n") !== this.#watching.join("\n")) {
			this.#watching = sorted
			this.#conn.watch(sorted)
		}
	}

	#connected(userId: string, anchors: string[]) {
		this.userId = userId
		this.anchors = anchors
		this.connected = true
		this.#watching = []
		this.#resolved = new Map()
		this.#paths = new Map()
		// live nodes are fetched afresh, as the tree is
		this.#liveAnchors = new Map()
		void this.#reconcileLive()
		// fetch every group afresh, and every expanded node, since points
		// sent while the connection was down were missed
		const f = this.forest.filter((t) => anchors.includes(t.anchor))
		const first = f.length === 0
		const expanded = tree.expandedIds(f)
		this.#setForest(f)
		void Promise.all([
			...anchors.map((a) => this.fetch(a, "all", a, 2)),
			...expanded.map((e) => this.fetchChildren(e.anchor, e.id)),
		])
			.then(async () => {
				if (first) {
					await this.#restore(prefs.expanded(userId))
				}
				if (this.#deepFetched) {
					await this.#deepFetch()
				}
			})
			.then(() => {
				this.ready = true
				this.#remember()
				this.onConnected?.()
			})
	}

	// ---- remembered expansion

	// remember keeps what is expanded and open, per user, so a reload
	// shows the tree as it was.
	#remember() {
		if (!this.ready || this.userId === "" || this.searchKeys) {
			return
		}
		const list: Expanded[] = []
		for (const n of tree.shown(this.forest)) {
			if (n.expChildren || n.expDetail) {
				list.push({
					anchor: n.anchor,
					id: n.node.id,
					key: n.key,
					...(n.expChildren ? { children: true } : {}),
					...(n.expDetail ? { detail: true } : {}),
				})
			}
		}
		const sig = JSON.stringify(list)
		if (sig !== this.#saved) {
			this.#saved = sig
			prefs.setExpanded(this.userId, list)
		}
	}

	// restore expands remembered nodes a level at a time, each level once
	// the one above has its children. A node that is gone is dropped.
	async #restore(list: Expanded[]) {
		const depth = (e: Expanded) => e.key.split("/").length
		const levels = [...new Set(list.map(depth))].sort((a, b) => a - b)
		for (const level of levels) {
			await Promise.all(
				list
					.filter((e) => depth(e) === level)
					.map(async (e) => {
						if (!this.find(e.key)) {
							return
						}
						if (e.detail) {
							this.setDetail(e.key, true)
						}
						if (e.children) {
							await this.expand(e.key)
						}
					})
			)
		}
	}

	// ---- search

	// setQuery filters the tree. The first search of a session fetches
	// each group deeply, so the whole tree is searched; after that the
	// filtering is local.
	async setQuery(q: string) {
		this.query = q
		if (q.trim() !== "" && !this.#deepFetched) {
			this.#deepFetched = true
			this.searching = true
			await this.#deepFetch()
			this.searching = false
		}
		this.#updateWatch()
	}

	async #deepFetch() {
		await Promise.all(this.anchors.map((a) => this.fetch(a, "all", a, searchDepth)))
	}

	// fetch asks for nodes and puts them in the tree: parent "all" fetches
	// an anchor, id "all" the children of parent, and otherwise one child.
	async fetch(anchor: string, parent: string, id: string, depth: number) {
		const c = this.#conn
		if (!c) {
			return
		}
		let nodes: Node[]
		try {
			nodes = await c.getNodes(anchor, parent, id, depth)
		} catch (err) {
			this.popError(errorText(err))
			this.#setForest(tree.clearLoading(this.forest))
			return
		}
		if (c !== this.#conn) {
			return
		}
		let f: Forest
		if (parent === "all") {
			f = tree.replaceAnchor(anchor, depth, nodes, this.forest)
		} else if (id === "all") {
			f = tree.replaceChildren(anchor, parent, depth, nodes, this.forest)
		} else {
			f = tree.replaceChild(anchor, parent, id, depth, nodes, this.forest)
		}
		this.#setForest(f)
	}

	// fetchChildren asks for the children of a node, with their children,
	// so each child knows whether it can be expanded.
	fetchChildren(anchor: string, id: string) {
		return this.fetch(anchor, id, "all", 1)
	}

	// getNodes fetches nodes without putting them in the tree.
	async getNodes(anchor: string, parent: string, id: string, depth = 0): Promise<Node[]> {
		if (!this.#conn) {
			return []
		}
		return this.#conn.getNodes(anchor, parent, id, depth)
	}

	// searchNodes asks every anchor for nodes matching query and merges the
	// replies, ranked, each with the path it sits on. With a type, only
	// nodes of that type match, and an empty query lists them all.
	async searchNodes(query: string, limit = 20, type = ""): Promise<NodeMatch[]> {
		const c = this.#conn
		if (!c || (query.trim() === "" && type === "")) {
			return []
		}
		const replies = await Promise.all(
			this.anchors.map(async (anchor) => {
				try {
					return { anchor, nodes: await c.searchNodes(anchor, query, limit, type) }
				} catch {
					return { anchor, nodes: [] }
				}
			})
		)
		const matches: NodeMatch[] = []
		for (const r of replies) {
			const top = this.findNode(r.anchor)
			matches.push(
				...buildMatches(
					r.nodes,
					query,
					r.anchor,
					top ? getText(top.points, PointType.Description) : undefined,
					type
				)
			)
		}
		return rankMatches(matches).slice(0, limit)
	}

	// locate finds the anchor a node sits under, and the node itself.
	async #locate(id: string): Promise<{ anchor: string; node: Node } | undefined> {
		const c = this.#conn
		if (!c || id === "") {
			return undefined
		}
		const onPage = tree.findById(id, this.forest)
		if (onPage) {
			return { anchor: onPage.anchor, node: onPage.node }
		}
		for (const anchor of this.anchors) {
			try {
				const nodes = await c.getNodes(anchor, "all", id, 0)
				const n = nodes[0]
				if (n) {
					return { anchor, node: n }
				}
			} catch {
				// not under this anchor
			}
		}
		return undefined
	}

	// fetchSubtree fetches a node with depth levels below it, through the
	// anchor it sits under, without putting it in the tree.
	async fetchSubtree(id: string, depth: number): Promise<{ anchor: string; nodes: Node[] } | null> {
		const c = this.#conn
		const where = await this.#locate(id)
		if (!c || !where) {
			return null
		}
		try {
			return { anchor: where.anchor, nodes: await c.getNodes(where.anchor, "all", id, depth) }
		} catch (err) {
			this.popError(errorText(err))
			return null
		}
	}

	// watchNodes keeps the given nodes current in liveNodes, and reports
	// edge changes below those in edgesOf through onLiveEdge. Each watcher
	// names itself with a key; nodes no watcher names any more are
	// dropped.
	async watchNodes(ids: readonly string[], edgesOf: readonly string[] = [], key = "page") {
		if (ids.length === 0 && edgesOf.length === 0) {
			this.#liveWants.delete(key)
		} else {
			this.#liveWants.set(key, { ids: [...ids], edges: [...edgesOf] })
		}
		await this.#reconcileLive()
	}

	async #reconcileLive() {
		const want = new Set<string>()
		const edges = new Set<string>()
		for (const w of this.#liveWants.values()) {
			for (const id of w.ids) {
				want.add(id)
			}
			for (const id of w.edges) {
				edges.add(id)
			}
		}
		for (const id of [...this.#liveAnchors.keys()]) {
			if (!want.has(id)) {
				this.#liveAnchors.delete(id)
				this.liveNodes.delete(id)
			}
		}
		this.#liveEdges = edges
		await Promise.all(
			[...want]
				.filter((id) => !this.#liveAnchors.has(id))
				.map(async (id) => {
					const where = await this.#locate(id)
					if (where && want.has(id)) {
						this.#liveAnchors.set(id, where.anchor)
						this.liveNodes.set(id, where.node)
					} else {
						this.liveNodes.delete(id)
					}
				})
		)
		this.#updateWatch()
	}

	// resolvePath follows a relative binding, "type/description" segments,
	// down from a node, and returns the node it reaches. Results are kept
	// until the next connect.
	async resolvePath(from: string, path: string): Promise<string | undefined> {
		const cacheKey = `${from}|${path}`
		if (this.#paths.has(cacheKey)) {
			return this.#paths.get(cacheKey)
		}
		const segments = splitPath(path)
		const c = this.#conn
		if (!segments || !c || from === "") {
			return undefined
		}
		const where = await this.#locate(from)
		let cur = where?.node.id
		for (const seg of segments) {
			if (!cur || !where) {
				break
			}
			try {
				const kids = await c.getNodes(where.anchor, cur, "all", 0)
				cur = pickChild(
					kids.filter((k) => !isTombstone(k)),
					seg
				)?.id
			} catch {
				cur = undefined
			}
		}
		this.#paths.set(cacheKey, cur)
		return cur
	}

	// resolveNode finds a node by ID for display: on the page if it is
	// there, else fetched once through any anchor it sits under.
	async resolveNode(id: string): Promise<Node | undefined> {
		const onPage = this.findNode(id)
		if (onPage) {
			return onPage
		}
		const cached = this.#resolved.get(id)
		if (cached) {
			return cached
		}
		const c = this.#conn
		if (!c || id === "") {
			return undefined
		}
		for (const anchor of this.anchors) {
			try {
				const nodes = await c.getNodes(anchor, "all", id, 0)
				const n = nodes[0]
				if (n) {
					this.#resolved.set(id, n)
					return n
				}
			} catch {
				// not under this anchor
			}
		}
		return undefined
	}

	#edgePoints(items: { nodeId: string; parentId: string; points: Point[] }[]) {
		// an edge the tree does not have is a new child of a node on
		// screen: fetch it, with its children if the parent is open
		let f = this.forest
		const fetches: Promise<void>[] = []
		for (const i of items) {
			if (this.#liveEdges.has(i.parentId)) {
				this.onLiveEdge?.(i.parentId)
			}
			const [next, missing] = tree.applyEdgePoints(i.nodeId, i.parentId, i.points, f)
			f = next
			if (!getBool(i.points, PointType.Tombstone)) {
				for (const m of missing) {
					fetches.push(this.fetch(m.anchor, m.parentId, i.nodeId, m.expanded ? 1 : 0))
				}
			}
		}
		this.#setForest(f)
		void Promise.all(fetches)
	}

	find(key: string): TreeNode | undefined {
		return tree.findByKey(key, this.forest)
	}

	findNode(id: string): Node | undefined {
		return tree.findById(id, this.forest)?.node
	}

	// toggleChildren expands or collapses a node.
	async toggleChildren(key: string) {
		const n = this.find(key)
		if (!n) {
			return
		}
		if (n.expChildren) {
			this.#setForest(tree.setExpanded(key, false, this.forest))
		} else {
			await this.expand(key)
		}
	}

	// expand opens a node and fetches its children afresh, along with those
	// of the nodes below it left expanded, since nothing under a collapsed
	// node is watched.
	async expand(key: string) {
		const n = this.find(key)
		if (!n) {
			return
		}
		this.#setForest(tree.setExpanded(key, true, this.forest))
		const expanded = tree.expandedIds([{ ...n, expChildren: true }])
		await Promise.all(expanded.map((e) => this.fetchChildren(e.anchor, e.id)))
	}

	setDetail(key: string, open: boolean) {
		this.#setForest(tree.setDetail(key, open, this.forest))
	}

	toggleDetail(key: string) {
		const n = this.find(key)
		if (!n) {
			return
		}
		this.setDetail(key, !n.expDetail)
		this.onDetail?.(n.node, !n.expDetail)
	}

	// reveal expands the tree down to a node and opens its detail,
	// returning its key, or null when the node is not in reach.
	async reveal(id: string): Promise<string | null> {
		for (const anchor of this.anchors) {
			const path = await this.#pathTo(anchor, id, new Set())
			if (!path) {
				continue
			}
			let key = `${anchor}/${anchor}`
			for (const nid of path.slice(1)) {
				const n = this.find(key)
				if (!n) {
					return null
				}
				if (!n.expChildren || n.children.every((c) => c.node.id !== nid)) {
					await this.expand(key)
				}
				key = `${key}/${nid}`
			}
			if (!this.find(key)) {
				return null
			}
			this.setDetail(key, true)
			return key
		}
		return null
	}

	// pathTo finds the IDs from an anchor down to a node by walking up its
	// parents.
	async #pathTo(anchor: string, id: string, seen: Set<string>): Promise<string[] | null> {
		if (id === anchor) {
			return [anchor]
		}
		if (seen.has(id) || seen.size > 64) {
			return null
		}
		seen.add(id)
		let edges: Node[]
		try {
			edges = await this.getNodes(anchor, "all", id)
		} catch {
			return null
		}
		for (const e of edges) {
			if (isTombstone(e) || e.parent === "root" || e.parent === "") {
				continue
			}
			const up = await this.#pathTo(anchor, e.parent, seen)
			if (up) {
				return [...up, id]
			}
		}
		return null
	}

	// ---- edits

	editPoints(key: string, points: Point[]) {
		const prev = this.edit && this.edit.key === key ? this.edit : null
		this.edit = {
			key,
			points: updatePoints(prev?.points ?? [], points),
			viewRaw: prev?.viewRaw ?? false,
		}
		this.scratch = ""
	}

	get hasEdits(): boolean {
		return this.edit !== null && this.edit.points.length > 0
	}

	discardEdits() {
		this.edit = null
	}

	// save sends the pending edit of a node and shows it at once.
	save(key: string) {
		const edit = this.edit
		const n = this.find(key)
		if (!edit || edit.key !== key || !n) {
			return
		}
		const points = clearText(edit.points)
		this.edit = null
		if (points.length === 0) {
			return
		}
		const id = n.node.id
		this.#setForest(
			tree.mapForest(this.forest, (t) =>
				t.node.id === id
					? { ...t, node: { ...t.node, points: updatePoints(t.node.points, points) } }
					: t
			)
		)
		void this.sendPoints(id, points).then((ok) => {
			if (ok) {
				this.#notify("Saved")
			}
		})
	}

	// sendPoints sends points to a node, reporting whether they went. A
	// node shown live off the tree is reached through its own anchor.
	async sendPoints(id: string, points: Point[]): Promise<boolean> {
		const anchor = tree.anchorOf(id, this.forest) ?? this.#liveAnchors.get(id)
		if (!anchor || !this.#conn) {
			return false
		}
		const live = this.liveNodes.get(id)
		if (live) {
			this.liveNodes.set(id, { ...live, points: updatePoints(live.points, points) })
		}
		try {
			await this.#conn.sendPoints(anchor, id, points)
			return true
		} catch (err) {
			this.popError(`Error saving points: ${errorText(err)}`)
			return false
		}
	}

	toggleRaw(key: string) {
		const viewRaw = this.edit && this.edit.key === key ? !this.edit.viewRaw : true
		this.edit = viewRaw ? { key, points: [], viewRaw: true } : null
	}

	addRawPoint(key: string) {
		const k = this.addPoint.key === "" ? "0" : this.addPoint.key
		this.editPoints(key, [
			{
				type: this.addPoint.type,
				key: k,
				time: new Date(this.now).toISOString(),
				dataType: DataType.Unknown,
				value: 0,
				text: "",
				tombstone: 0,
			},
		])
	}

	async uploadFile(id: string, binary: boolean, file: File) {
		let contents: string
		if (binary) {
			const bytes = new Uint8Array(await file.arrayBuffer())
			let s = ""
			for (const b of bytes) {
				s += String.fromCharCode(b)
			}
			contents = btoa(s)
		} else {
			contents = await file.text()
		}
		const time = new Date(this.now).toISOString()
		const ok = await this.sendPoints(id, [
			newText(PointType.Name, "0", file.name, time),
			newText(PointType.Data, "0", contents, time),
		])
		if (ok) {
			this.#notify(`Sent ${file.name}`)
		}
	}

	async generateKey(id: string) {
		try {
			const token = await http.generateKey(this.#token, id)
			this.generatedToken = { id, token }
		} catch (err) {
			this.popError(`Error generating token: ${errorText(err)}`)
		}
	}

	// ---- node operations

	cancelOp() {
		this.op = { kind: "none" }
	}

	async addNode(parentKey: string, type: string) {
		const parent = this.find(parentKey)
		this.op = { kind: "none" }
		if (!parent || type === "") {
			return
		}
		try {
			await http.insertNode(this.#token, {
				id: "",
				type,
				parent: parent.node.id,
				points: [newText(PointType.Description, "", "New, please edit")],
				edgePoints: [],
			})
		} catch (err) {
			this.popError(`Error adding node: ${errorText(err)}`)
			return
		}
		this.#notify("Node added")
		await this.expand(parentKey)
	}

	// addChild adds a node under a parent by ID, with the given points
	// beside the usual description, and returns the new node's ID.
	async addChild(parentId: string, type: string, points: Point[] = []): Promise<string> {
		try {
			const resp = await http.insertNode(this.#token, {
				id: "",
				type,
				parent: parentId,
				points: [newText(PointType.Description, "", "New, please edit"), ...points],
				edgePoints: [],
			})
			this.#notify("Node added")
			return resp.id ?? ""
		} catch (err) {
			this.popError(`Error adding node: ${errorText(err)}`)
			return ""
		}
	}

	// removeNode deletes the edge from a node to a parent, by ID.
	async removeNode(id: string, parentId: string): Promise<boolean> {
		try {
			await http.deleteNode(this.#token, id, parentId)
		} catch (err) {
			this.popError(`Error deleting node: ${errorText(err)}`)
			return false
		}
		this.#notify("Node deleted")
		return true
	}

	async deleteNode(key: string) {
		const n = this.find(key)
		this.op = { kind: "none" }
		if (!n) {
			return
		}
		// the tombstone arrives as a live edge point and hides the node
		try {
			await http.deleteNode(this.#token, n.node.id, n.node.parent)
		} catch (err) {
			this.popError(`Error deleting node: ${errorText(err)}`)
			return
		}
		this.#notify(edgeRole(n.node) === "mirror" ? "Mirror removed" : "Node deleted")
		if (n.expDetail) {
			this.onDetail?.(n.node, false)
		}
	}

	async sendMessage(key: string, message: string) {
		const n = this.find(key)
		if (!n) {
			return
		}
		try {
			await http.notify(this.#token, n.node.id, message)
			this.op = { kind: "none" }
			this.#notify("Message sent")
		} catch (err) {
			this.popError(`Error messaging node: ${errorText(err)}`)
		}
	}

	copy(key: string) {
		const n = this.find(key)
		if (!n) {
			return
		}
		this.clipboard = {
			id: n.node.id,
			parent: n.node.parent,
			description: getBestDesc(n.node.points),
		}
		this.nodeMsg = { key, text: "Node copied\nclick paste in destination node", time: Date.now() }
		writeClipboard(n.node.id)
	}

	clearClipboard() {
		this.clipboard = null
	}

	// paste moves, mirrors, or duplicates the copied node under a node.
	async paste(destKey: string, how: "move" | "mirror" | "duplicate") {
		const dest = this.find(destKey)
		const src = this.clipboard
		if (!dest || !src) {
			return
		}
		try {
			if (how === "move") {
				await http.moveNode(this.#token, src.id, src.parent, dest.node.id)
				this.clipboard = null
			} else {
				await http.copyNode(this.#token, src.id, src.parent, dest.node.id, how === "duplicate")
			}
		} catch (err) {
			const verb = { move: "moving", mirror: "mirroring", duplicate: "duplicating" }[how]
			this.popError(`Error ${verb} node: ${errorText(err)}`)
			return
		}
		this.op = { kind: "none" }
		this.#notify({ move: "Node moved", mirror: "Node mirrored", duplicate: "Node duplicated" }[how])
		await this.expand(destKey)
	}

	// ownerOf is the parent type the copied node must live under to work.
	clipboardOwner(): string | undefined {
		const src = this.clipboard
		if (!src) {
			return undefined
		}
		const n = this.findNode(src.id)
		return n ? registry.ownerOf(n.type) : undefined
	}
}

export function writeClipboard(text: string) {
	if ("clipboard" in navigator) {
		navigator.clipboard.writeText(text).catch((err: unknown) => {
			console.log("clipboard write failed", err)
		})
	}
}
