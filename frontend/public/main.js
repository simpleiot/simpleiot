// The web UI's JavaScript: starts the Elm application and owns the NATS
// connection on its behalf. Elm owns the node tree and decides what to
// fetch and watch; this side carries a small JSON protocol over the
// natsOut/natsIn ports (see src/Api/Nats.elm), keeps the subscriptions in
// step with what Elm wants, and batches incoming points per animation
// frame so a burst from a device is one render.

import { connect, ErrorCode } from "/dist/siot-nats.js"

// eslint-disable-next-line no-undef
const app = Elm.Main.init({
	flags: JSON.parse(localStorage.getItem("storage")),
})

app.ports.save_.subscribe((storage) => {
	localStorage.setItem("storage", JSON.stringify(storage))
	app.ports.load_.send(storage)
})

app.ports.out.subscribe(({ action, data }) =>
	actions[action]
		? actions[action](data)
		: console.warn(`I didn't recognize action "${action}".`)
)

const actions = {
	LOG: (message) => console.log(`From Elm:`, message),
	CLIPBOARD: (message) => {
		if (navigator.clipboard) {
			navigator.clipboard.writeText(message).catch((err) => {
				console.log("clipboard write failed", err)
			})
		} else {
			console.log("clipboard not available")
		}
	},
}

// ---- NATS connection

// the connection variables are module state on purpose: each async path
// checks it still owns the connection before acting on it
/* eslint-disable require-atomic-updates */

const retryDelay = 5000

let conn = null
let token = ""
let subs = new Map() // subject -> subscription
let wanted = [] // subjects Elm last asked for
let retryTimer = null

const send = (event) => app.ports.natsIn.send(event)

function wsUrl() {
	const proto = location.protocol === "https:" ? "wss:" : "ws:"
	return `${proto}//${location.host}/`
}

function isAuthError(err) {
	const code = err && err.code
	return (
		code === ErrorCode.AuthorizationViolation ||
		code === ErrorCode.AuthenticationExpired ||
		code === ErrorCode.AuthenticationTimeout
	)
}

app.ports.natsOut.subscribe((cmd) => {
	handle(cmd).catch((err) => {
		console.warn("siot:", cmd.cmd, err)
		send({ event: "error", message: String(err.message || err) })
	})
})

async function handle(cmd) {
	switch (cmd.cmd) {
		case "connect":
			token = cmd.token
			await doConnect()
			break
		case "disconnect":
			await doDisconnect()
			break
		case "fetch":
			await fetchNodes(cmd)
			break
		case "watch":
			wanted = cmd.subjects
			await reconcile()
			break
		case "sendPoints":
			await conn.sendNodePoints(cmd.anchor, cmd.id, cmd.points)
			break
		case "sendEdgePoints":
			await conn.sendEdgePoints(cmd.anchor, cmd.id, cmd.parent, cmd.points)
			break
		default:
			console.warn("siot: unknown command", cmd)
	}
}

async function doDisconnect() {
	clearTimeout(retryTimer)
	retryTimer = null
	wanted = []
	const c = conn
	conn = null
	subs = new Map()
	if (c) {
		await c.close()
	}
}

async function doConnect() {
	clearTimeout(retryTimer)
	retryTimer = null
	if (conn) {
		const c = conn
		conn = null
		subs = new Map()
		await c.close()
	}

	let c
	try {
		c = await connect({ url: wsUrl(), token })
	} catch (err) {
		if (isAuthError(err)) {
			send({ event: "authFailed" })
			return
		}
		send({ event: "disconnected", reason: String(err.message || err) })
		retryTimer = setTimeout(doConnect, retryDelay)
		return
	}
	conn = c

	c.closed().then((err) => {
		if (conn !== c) {
			return
		}
		conn = null
		subs = new Map()
		if (isAuthError(err)) {
			send({ event: "authFailed" })
		} else {
			send({
				event: "disconnected",
				reason: err ? String(err.message || err) : "closed",
			})
			retryTimer = setTimeout(doConnect, retryDelay)
		}
	})

	watchStatus(c)
	await announce(c)
}

// announce tells Elm the connection is up and what it may reach. Elm
// refetches every loaded subtree in response, which also covers messages
// missed while the connection was down.
async function announce(c) {
	const me = await c.me()
	if (conn !== c) {
		return
	}
	send({ event: "connected", userId: me.userId, anchors: me.anchors })
	await reconcile()
}

async function watchStatus(c) {
	for await (const s of c.status()) {
		if (conn !== c) {
			return
		}
		switch (s.type) {
			case "disconnect":
				send({ event: "disconnected", reason: "connection lost" })
				break
			case "reconnect":
				// the server may have closed the connection because the
				// user's groups changed, so ask again
				subs = new Map()
				announce(c).catch((err) => console.warn("siot: reconnect", err))
				break
			case "error":
				console.warn("siot: connection error", s.data)
				break
			default:
				break
		}
	}
}

async function fetchNodes({ anchor, parent, id, depth }) {
	if (!conn) {
		return
	}
	const nodes = await conn.getNodes(anchor, parent, id, { depth })
	send({ event: "nodes", anchor, parent, id, depth, nodes })
}

// reconcile brings the subscriptions in line with the subjects Elm wants.
async function reconcile() {
	if (!conn) {
		return
	}
	const want = new Set(wanted)
	for (const [subject, sub] of subs) {
		if (!want.has(subject)) {
			subs.delete(subject)
			sub.unsubscribe()
		}
	}
	for (const subject of want) {
		if (!subs.has(subject)) {
			subscribe(conn, subject)
		}
	}
}

function subscribe(c, subject) {
	const sub = c.subscribe(subject)
	subs.set(subject, sub)
	;(async () => {
		try {
			for await (const m of sub) {
				queue(m)
			}
		} catch (err) {
			console.warn("siot: subscription", subject, err)
		} finally {
			if (conn === c && subs.get(subject) === sub) {
				subs.delete(subject)
			}
		}
	})()
}

// ---- batching: points are collected per node and handed to Elm once per
// animation frame

let pendingPoints = new Map() // nodeId -> points
let pendingEdges = new Map() // nodeId|parentId -> {nodeId, parentId, points}
let frame = null

function queue(m) {
	if (m.parentId) {
		const k = `${m.nodeId}|${m.parentId}`
		const e = pendingEdges.get(k) || {
			nodeId: m.nodeId,
			parentId: m.parentId,
			points: [],
		}
		e.points.push(...m.points)
		pendingEdges.set(k, e)
	} else {
		const pts = pendingPoints.get(m.nodeId) || []
		pts.push(...m.points)
		pendingPoints.set(m.nodeId, pts)
	}
	if (frame == null) {
		frame = document.hidden
			? setTimeout(flush, 250)
			: requestAnimationFrame(flush)
	}
}

function flush() {
	frame = null
	if (pendingPoints.size > 0) {
		const items = [...pendingPoints].map(([nodeId, points]) => ({
			nodeId,
			points,
		}))
		pendingPoints = new Map()
		send({ event: "points", items })
	}
	if (pendingEdges.size > 0) {
		const items = [...pendingEdges.values()]
		pendingEdges = new Map()
		send({ event: "edgePoints", items })
	}
}
