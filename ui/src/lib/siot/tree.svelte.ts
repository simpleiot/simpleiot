// The tree page's state: the node tree, the connection that keeps it
// current, pending edits, and the node operation in progress. It is the
// one place, with the page, that talks to the server.

import { registry } from "#lib/nodes/index.ts"
import type { Clipboard, GeneratedToken } from "#lib/nodes/types.ts"
import type { Env } from "./connection.ts"
import { Connection } from "./connection.ts"
import * as http from "./http.ts"
import { isTombstone, type Node } from "./node.ts"
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

const errorShowMs = 5000
const nodeMsgShowMs = 3000

const errorText = (err: unknown) => (err instanceof Error ? err.message : String(err))

export class TreeStore {
	forest = $state.raw<Forest>([])
	connected = $state(false)
	userId = $state("")
	anchors = $state.raw<string[]>([])
	error = $state<string | null>(null)
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

	// onConnected runs after each connect, once the anchors are fetched;
	// the page uses it to open a deep link
	onConnected: (() => void) | null = null
	// onDetail runs when a node's detail is opened or closed from the tree
	onDetail: ((node: Node, open: boolean) => void) | null = null

	#conn: Connection | null = null
	#token = ""
	#watching: string[] = []
	#tick: ReturnType<typeof setInterval> | null = null
	#errorAt = 0
	#onAuthFailed: () => void = () => undefined

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
		this.forest = []
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
		if (this.error && this.now - this.#errorAt > errorShowMs) {
			this.error = null
		}
		if (this.nodeMsg && this.now - this.nodeMsg.time > nodeMsgShowMs) {
			this.nodeMsg = null
		}
	}

	popError(message: string) {
		this.error = message
		this.#errorAt = Date.now()
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
	}

	#updateWatch() {
		if (!this.#conn || !this.connected) {
			return
		}
		const subjects = tree.watchSubjects(this.forest)
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
		const expanded = tree.expandedIds(f)
		this.#setForest(f)
		void Promise.all([
			...anchors.map((a) => this.fetch(a, "all", a, 2)),
			...expanded.map((e) => this.fetchChildren(e.anchor, e.id)),
		]).then(() => {
			this.onConnected?.()
		})
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
		void this.sendPoints(id, points)
	}

	async sendPoints(id: string, points: Point[]) {
		const anchor = tree.anchorOf(id, this.forest)
		if (!anchor || !this.#conn) {
			return
		}
		try {
			await this.#conn.sendPoints(anchor, id, points)
		} catch (err) {
			this.popError(`Error saving points: ${errorText(err)}`)
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
		await this.sendPoints(id, [
			newText(PointType.Name, "0", file.name, time),
			newText(PointType.Data, "0", contents, time),
		])
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
