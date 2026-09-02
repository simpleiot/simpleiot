// A Simple IoT client for the browser, over NATS WebSockets. The connection
// is made as a signed-in user: the user's node ID and the JWT from sign-in
// are the NATS user and password, and the server limits the connection to
// the groups (anchors) the user belongs to. Every request names the anchor
// it is made under; see docs/ref/api.md for the subjects.

import { connect as natsConnect } from "nats.ws"
import { encodePoints, decodePoints, decodeNodes } from "./codec.js"

export { encodePoints, decodePoints, decodeNodes }
export { ErrorCode, Events, DebugEvents } from "nats.ws"

const TE = new TextEncoder()
const TD = new TextDecoder()

const defaultTimeout = 10000

// userIdFromToken reads the user ID a sign-in JWT was issued to. The token
// is not verified here; the server does that when the connection is made.
export function userIdFromToken(token) {
	try {
		const [, payload] = token.split(".")
		const json = atob(payload.replace(/-/g, "+").replace(/_/g, "/"))
		return JSON.parse(json).jti || ""
	} catch (e) {
		return ""
	}
}

// inboxPrefix is the reply subject prefix a user connection is allowed to
// subscribe to. _INBOX.> is shared by every client on the server.
export function inboxPrefix(userId) {
	return "_INBOX_" + userId
}

// connect opens a connection as a user.
// - `url` is the WebSocket URL, such as ws://localhost:8118/ (the HTTP
//   server proxies WebSocket connections to NATS)
// - `token` is the JWT from sign-in; `user` defaults to the ID in it
// Any other option is passed to nats.ws. Reconnection is on, without
// limit, by default.
export async function connect({ url, token, user, ...opts } = {}) {
	const userId = user || userIdFromToken(token)
	if (!userId) {
		throw new Error("connect: a user ID or a sign-in token is required")
	}
	const nc = await natsConnect({
		servers: [url || "ws://localhost:8118/"],
		user: userId,
		pass: token,
		inboxPrefix: inboxPrefix(userId),
		maxReconnectAttempts: -1,
		...opts,
	})
	return new SiotConnection(nc, userId, token)
}

// parseSubject splits a point subject into its parts:
//   up.<anchor>.<nodeId>.<type>.<key>            node point
//   up.<anchor>.<nodeId>.<parentId>.<type>.<key> edge point
//   p.<nodeId>.<type>.<key>                      node point
//   ep.<nodeId>.<parentId>.<type>.<key>          edge point
function parseSubject(subject) {
	const t = subject.split(".")
	switch (t[0]) {
		case "up":
			if (t.length === 6) {
				return {
					anchor: t[1],
					nodeId: t[2],
					parentId: t[3],
					type: t[4],
					key: t[5],
				}
			}
			return { anchor: t[1], nodeId: t[2], type: t[3], key: t[4] }
		case "ep":
			return { nodeId: t[1], parentId: t[2], type: t[3], key: t[4] }
		default:
			return { nodeId: t[1], type: t[2], key: t[3] }
	}
}

export class SiotConnection {
	constructor(nc, userId, token) {
		this.nc = nc
		this.userId = userId
		this.token = token
	}

	// status yields nats.ws connection events: disconnect, reconnecting,
	// reconnect, error.
	status() {
		return this.nc.status()
	}

	// closed resolves when the connection is closed, with the error that
	// closed it if there was one.
	closed() {
		return this.nc.closed()
	}

	close() {
		return this.nc.close()
	}

	async request(subject, payload, timeout = defaultTimeout) {
		return this.nc.request(subject, payload, { timeout })
	}

	// me asks the server who this connection is. Returns
	// { userId, anchors, nodes }: the anchors are the nodes the user sits
	// under, and nodes the user's node at each of them.
	async me() {
		const m = await this.request("auth.me", TE.encode(this.token))
		const nodes = decodeNodes(m.data)
		const anchors = [...new Set(nodes.map((n) => n.parent))]
		return { userId: this.userId, anchors, nodes }
	}

	// getNodes fetches nodes under an anchor, as nodes.<parent>.<id> does:
	// parent "all" fetches every edge of node id, id "all" fetches the
	// children of parent. `depth` also returns descendants that many
	// levels down; `type` filters the requested nodes; `includeDel`
	// includes deleted ones.
	async getNodes(
		anchor,
		parent,
		id,
		{ depth = 0, type = "", includeDel = false, timeout } = {}
	) {
		const params = []
		if (type) {
			params.push({ type: "nodeType", text: type })
		}
		if (includeDel) {
			params.push({ type: "tombstone", value: 1 })
		}
		if (depth > 0) {
			params.push({ type: "depth", value: depth })
		}
		const subject = `u.${anchor}.${this.userId}.nodes.${parent || "all"}.${
			id || "all"
		}`
		const m = await this.request(subject, encodePoints(params), timeout)
		return decodeNodes(m.data)
	}

	// sendNodePoints writes points to a node under an anchor. Each point is
	// its own message, as in the Go client, and an error reply is thrown.
	async sendNodePoints(anchor, id, points, { timeout } = {}) {
		for (const p of points) {
			const subject = `u.${anchor}.${this.userId}.p.${id}.${p.type || "_"}.${
				p.key || "0"
			}`
			const m = await this.request(subject, encodePoints([p]), timeout)
			if (m.data && m.data.length > 0) {
				throw new Error(TD.decode(m.data))
			}
		}
	}

	// sendEdgePoints writes points to the edge between a node and its
	// parent, in one message so a new edge arrives whole.
	async sendEdgePoints(anchor, id, parent, points, { timeout } = {}) {
		const subject = `u.${anchor}.${this.userId}.ep.${id}.${parent}`
		const m = await this.request(subject, encodePoints(points), timeout)
		if (m.data && m.data.length > 0) {
			throw new Error(TD.decode(m.data))
		}
	}

	// subscribe returns a nats.ws subscription whose async iterator yields
	// decoded messages: { subject, anchor, nodeId, parentId, type, key,
	// points }. parentId is set for edge points only. A subject the
	// connection may not subscribe to closes the subscription with an
	// error.
	subscribe(subject) {
		const sub = this.nc.subscribe(subject)
		return Object.assign(Object.create(sub), {
			async *[Symbol.asyncIterator]() {
				for await (const m of sub) {
					let points
					try {
						points = decodePoints(m.data)
					} catch (err) {
						console.warn("siot: bad points on", m.subject, err)
						continue
					}
					yield { subject: m.subject, ...parseSubject(m.subject), points }
				}
			},
		})
	}
}
