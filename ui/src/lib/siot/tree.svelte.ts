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
	newText,
	updatePoints,
	type Point,
} from "./point.ts"
import { PointType } from "./schema.ts"
import * as tree from "./tree.ts"
import type { Forest, TreeNode } from "./tree.ts"

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
		const subjects = tree.watchSubjects(this.forest, this.searchKeys)
		if (subjects.join("\n") !== this.#watching.join("\n")) {
			this.#watching = subjects
			this.#conn.watch(subjects)
		}
	}

	#connected(userId: string, anchors: string[]) {
		this.userId = userId
		this.anchors = anchors
		this.connected = true
		this.#watching = []
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

	#edgePoints(items: { nodeId: string; parentId: string; points: Point[] }[]) {
		// an edge the tree does not have is a new child of a node on
		// screen: fetch it, with its children if the parent is open
		let f = this.forest
		const fetches: Promise<void>[] = []
		for (const i of items) {
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

	// toggleChildren expands or collapses a node. Expanding fetches the
	// children afresh, which doubles as a refresh.
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

	async expand(key: string) {
		const n = this.find(key)
		if (!n) {
			return
		}
		this.#setForest(tree.setExpanded(key, true, this.forest))
		await this.fetchChildren(n.anchor, n.node.id)
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

	// sendPoints sends points to a node, reporting whether they went.
	async sendPoints(id: string, points: Point[]): Promise<boolean> {
		const anchor = tree.anchorOf(id, this.forest)
		if (!anchor || !this.#conn) {
			return false
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
		await this.#expandAndRefetch(parentKey)
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
		await this.#expandAndRefetch(destKey)
	}

	async #expandAndRefetch(key: string) {
		const n = this.find(key)
		if (!n) {
			return
		}
		this.#setForest(
			tree.mapForest(this.forest, (t) =>
				t.key === key && !t.expChildren ? { ...t, expChildren: true } : t
			)
		)
		await this.fetchChildren(n.anchor, n.node.id)
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
