// The NATS connection, ported from the Elm UI's public/main.js. It
// connects with the sign-in token and retries every 5 s, asks the server
// who the user is on every connect, keeps the subscriptions in step with
// the subjects the page wants, batches incoming points per animation frame,
// and drops the subscriptions while the tab is hidden. It reports through
// plain callbacks so it can be tested without a browser or a server.

import type { Message, Node, PointInput, SiotConnection, SiotSubscription } from "simpleiot-js"
import type { Point } from "./point.ts"

export interface PointsItem {
	nodeId: string
	points: Point[]
}

export interface EdgePointsItem {
	nodeId: string
	parentId: string
	points: Point[]
}

export interface ConnectionEvents {
	// connected reports the connection is up and which anchors the user
	// may reach. The page refetches what it shows in response, which also
	// covers messages missed while the connection was down or the tab
	// was hidden.
	connected(userId: string, anchors: string[]): void
	disconnected(reason: string): void
	authFailed(): void
	points(items: PointsItem[]): void
	edgePoints(items: EdgePointsItem[]): void
}

// Env is what the connection needs from its surroundings; tests replace it.
export interface Env {
	connect(token: string): Promise<SiotConnection>
	isAuthError(err: unknown): boolean
	hidden(): boolean
	onVisibilityChange(f: () => void): () => void
	// frame runs f on the next animation frame, or soon when hidden
	frame(f: () => void): void
	setTimeout(f: () => void, ms: number): unknown
	clearTimeout(t: unknown): void
	warn(...args: unknown[]): void
}

export const retryDelay = 5000

// mergeNewest keeps the newest point per type and key, so a batch never
// carries two values for one point, and a point that arrives late does not
// replace a newer one.
export function mergeNewest(into: Point[], points: readonly Point[]) {
	for (const p of points) {
		const i = into.findIndex((q) => q.type === p.type && q.key === p.key)
		if (i < 0) {
			into.push(p)
		} else if (Date.parse(p.time) >= Date.parse((into[i] as Point).time)) {
			into[i] = p
		}
	}
}

// Batcher collects points per node and hands them on once per frame, so a
// burst from a device is one update.
export class Batcher {
	#points = new Map<string, Point[]>()
	#edges = new Map<string, EdgePointsItem>()
	#scheduled = false

	constructor(
		private readonly frame: (f: () => void) => void,
		private readonly onPoints: (items: PointsItem[]) => void,
		private readonly onEdgePoints: (items: EdgePointsItem[]) => void
	) {}

	queue(m: Pick<Message, "nodeId" | "parentId" | "points">) {
		if (m.parentId) {
			const k = `${m.nodeId}|${m.parentId}`
			const e = this.#edges.get(k) ?? { nodeId: m.nodeId, parentId: m.parentId, points: [] }
			mergeNewest(e.points, m.points)
			this.#edges.set(k, e)
		} else {
			const pts = this.#points.get(m.nodeId) ?? []
			mergeNewest(pts, m.points)
			this.#points.set(m.nodeId, pts)
		}
		if (!this.#scheduled) {
			this.#scheduled = true
			this.frame(() => {
				this.flush()
			})
		}
	}

	flush() {
		this.#scheduled = false
		if (this.#points.size > 0) {
			const items = [...this.#points].map(([nodeId, points]) => ({ nodeId, points }))
			this.#points = new Map()
			this.onPoints(items)
		}
		if (this.#edges.size > 0) {
			const items = [...this.#edges.values()]
			this.#edges = new Map()
			this.onEdgePoints(items)
		}
	}
}

const errorText = (err: unknown) => (err instanceof Error ? err.message : String(err))

export class Connection {
	#conn: SiotConnection | null = null
	#token = ""
	#subs = new Map<string, SiotSubscription>()
	#wanted: string[] = []
	#retry: unknown = null
	#batcher: Batcher
	#stopVisibility: () => void

	constructor(
		private readonly events: ConnectionEvents,
		private readonly env: Env
	) {
		this.#batcher = new Batcher(
			(f) => {
				env.frame(f)
			},
			(items) => {
				events.points(items)
			},
			(items) => {
				events.edgePoints(items)
			}
		)
		this.#stopVisibility = env.onVisibilityChange(() => {
			this.#visibilityChanged()
		})
	}

	get connected(): boolean {
		return this.#conn !== null
	}

	async connect(token: string) {
		this.#token = token
		await this.#connect()
	}

	// close disconnects and stops retrying, for sign out.
	async close() {
		this.#stopVisibility()
		this.env.clearTimeout(this.#retry)
		this.#retry = null
		this.#wanted = []
		const c = this.#conn
		this.#conn = null
		this.#subs = new Map()
		if (c) {
			await c.close()
		}
	}

	// watch sets the subjects the page wants to hear about.
	watch(subjects: string[]) {
		this.#wanted = subjects
		this.#reconcile()
	}

	async getNodes(anchor: string, parent: string, id: string, depth: number): Promise<Node[]> {
		const c = this.#conn
		if (!c) {
			throw new Error("not connected")
		}
		return c.getNodes(anchor, parent, id, { depth })
	}

	async sendPoints(anchor: string, id: string, points: readonly PointInput[]) {
		const c = this.#conn
		if (!c) {
			throw new Error("not connected")
		}
		await c.sendNodePoints(anchor, id, points)
	}

	async sendEdgePoints(anchor: string, id: string, parent: string, points: readonly PointInput[]) {
		const c = this.#conn
		if (!c) {
			throw new Error("not connected")
		}
		await c.sendEdgePoints(anchor, id, parent, points)
	}

	async #connect() {
		this.env.clearTimeout(this.#retry)
		this.#retry = null
		if (this.#conn) {
			const old = this.#conn
			this.#conn = null
			this.#subs = new Map()
			await old.close()
		}

		let c: SiotConnection
		try {
			c = await this.env.connect(this.#token)
		} catch (err) {
			if (this.env.isAuthError(err)) {
				this.events.authFailed()
				return
			}
			this.events.disconnected(errorText(err))
			this.#retry = this.env.setTimeout(() => void this.#connect(), retryDelay)
			return
		}
		this.#conn = c

		void c.closed().then((err) => {
			if (this.#conn !== c) {
				return
			}
			this.#conn = null
			this.#subs = new Map()
			if (this.env.isAuthError(err)) {
				this.events.authFailed()
			} else {
				this.events.disconnected(err ? errorText(err) : "closed")
				this.#retry = this.env.setTimeout(() => void this.#connect(), retryDelay)
			}
		})

		void this.#watchStatus(c)
		await this.#announce(c)
	}

	// announce tells the page the connection is up and what it may reach.
	async #announce(c: SiotConnection) {
		const me = await c.me()
		if (this.#conn !== c) {
			return
		}
		this.events.connected(me.userId, me.anchors)
		this.#reconcile()
	}

	async #watchStatus(c: SiotConnection) {
		for await (const s of c.status()) {
			if (this.#conn !== c) {
				return
			}
			const type: string = s.type
			switch (type) {
				case "disconnect":
					this.events.disconnected("connection lost")
					break
				case "reconnect":
					// the server may have closed the connection because the
					// user's groups changed, so ask again
					this.#subs = new Map()
					this.#announce(c).catch((err: unknown) => {
						this.env.warn("siot: reconnect", err)
					})
					break
				case "error":
					this.env.warn("siot: connection error", s.data)
					break
				default:
					break
			}
		}
	}

	// reconcile brings the subscriptions in line with the wanted subjects.
	#reconcile() {
		const c = this.#conn
		if (!c || this.env.hidden()) {
			return
		}
		const want = new Set(this.#wanted)
		for (const [subject, sub] of this.#subs) {
			if (!want.has(subject)) {
				this.#subs.delete(subject)
				sub.unsubscribe()
			}
		}
		for (const subject of want) {
			if (!this.#subs.has(subject)) {
				this.#subscribe(c, subject)
			}
		}
	}

	#subscribe(c: SiotConnection, subject: string) {
		const sub = c.subscribe(subject)
		this.#subs.set(subject, sub)
		void (async () => {
			try {
				for await (const m of sub) {
					this.#batcher.queue(m)
				}
			} catch (err) {
				this.env.warn("siot: subscription", subject, err)
			} finally {
				if (this.#conn === c && this.#subs.get(subject) === sub) {
					this.#subs.delete(subject)
				}
			}
		})()
	}

	// A hidden tab drops its subscriptions so the server is not sending to
	// a page nobody is looking at, and on return the page fetches what it
	// shows afresh, as it does after a reconnect.
	#visibilityChanged() {
		const c = this.#conn
		if (!c) {
			return
		}
		if (this.env.hidden()) {
			for (const [subject, sub] of this.#subs) {
				this.#subs.delete(subject)
				sub.unsubscribe()
			}
		} else {
			this.#announce(c).catch((err: unknown) => {
				this.env.warn("siot: visible", err)
			})
		}
	}
}
