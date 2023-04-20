import { connect as natsConnect, StringCodec } from "nats.ws"

// The syntax: `import { Timestamp } from ...` will not work properly
// in Node.js because of how protobuf generates the CommonJS code, so we
// have to do a little more work...
import timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb.js"
const { Timestamp } = timestamp_pb
import point_pb from "./protobuf/point_pb.js"
const { Points, Point } = point_pb
import node_pb from "./protobuf/node_pb.js"
const { NodesRequest } = node_pb
import message_pb from "./protobuf/message_pb.js"
const { Message } = message_pb
import notification_pb from "./protobuf/notification_pb.js"
const { Notification } = notification_pb

// eslint-disable-next-line new-cap
const strCodec = StringCodec()

// connect opens and returns a connection to SIOT / NATS via WebSockets
export * from "nats.ws"
export async function connect(opts = {}) {
	const { servers = ["ws://localhost:9222"] } = opts
	const nc = await natsConnect({ ...opts, servers })

	// Force SIOTConnection to inherit from `nc` prototype
	SIOTConnection.prototype = Object.create(
		Object.getPrototypeOf(nc),
		Object.getOwnPropertyDescriptors(SIOTConnection.prototype)
	)
	// Create new instance of SIOTConnection and then assign `nc` properties
	return Object.assign(new SIOTConnection(), nc)
}

// SIOTConnection is a wrapper around a NatsConnectionImpl
function SIOTConnection() {
	// do nothing
}

Object.assign(SIOTConnection.prototype, {
	// getNode sends a request to `nodes.<parent>.<id>` to retrieve an array of
	// NodeEdges for the specified Node id.
	// - If `id` is "all" or falsy, this calls `getNodeChildren` instead;
	// however we strongly recommend using `getNodeChildren` directly
	// - If `parent` is "all" or falsy, all instances of the specified node are
	// returned
	// - If `parent` is "root", only root nodes are returned
	// - `opts` are options passed to the NATS request
	async getNode(id, { parent, type, includeDel, opts } = {}) {
		if (id === "all" || !id) {
			return this.getNodeChildren(parent, { type, includeDel, opts })
		}

		const points = [
			{ type: "nodeType", text: type },
			{ type: "tombstone", value: includeDel ? 1 : 0 },
		]
		const payload = encodePoints(points)
		const m = await this.request(
			"nodes." + (parent || "all") + "." + id,
			payload,
			opts
		)
		return decodeNodesRequest(m.data)
	},

	// getNodeChildren sends a request to `nodes.<parentID>.<id>` to retrieve
	// an array of child NodeEdges of the specified parent node.
	// - If `parent` is "root", all root nodes are retrieved
	// - `type` - can be used to filter nodes of a specified type (defaults to "")
	// - `includeDel` - set to true to include deleted nodes (defaults to false)
	// - `recursive` - set to true to recursively retrieve all descendants matching
	//   the criteria. In this case, each returned NodeEdge will contain a
	//   `children` property, which is an array of that Node's descendant NodeEdges.
	//
	//   Set to "flat" to return a single flattened array of NodeEdges.
	//
	//   Note: If `type` is also set when `recursive` is truthy, `type` still
	//   restricts which nodes are recursively searched. Consider removing the
	//   `type` filter and filter manually.
	// - `opts` are options passed to the NATS request
	async getNodeChildren(
		parentID,
		{ type, includeDel, recursive, opts, _cache = {} } = {}
	) {
		if (parentID === "all" || parentID === "none" || !parentID) {
			throw new Error("parent node ID must be specified")
		}

		const points = [
			{ type: "nodeType", text: type },
			{ type: "tombstone", value: includeDel ? 1 : 0 },
		]

		const payload = encodePoints(points)
		const m = await this.request("nodes." + parentID + ".all", payload, opts)
		const nodeEdges = decodeNodesRequest(m.data)
		if (recursive) {
			const flat = recursive === "flat"
			const nodeChildren = []
			// Note: recursive calls are done serially to fully utilize
			// the temporary `_cache`
			for (const n of nodeEdges) {
				const children =
					_cache[n.id] ||
					(await this.getNodeChildren(n.id, {
						type,
						includeDel,
						recursive,
						opts,
						_cache,
					}))
				// update cache
				// eslint-disable-next-line require-atomic-updates
				_cache[n.id] = children
				if (!flat) {
					// If not flattening, add `children` key to `n`
					n.children = children
				}
				nodeChildren.push(children)
			}
			if (flat) {
				// If flattening, simply return flat array of node edges
				return nodeEdges.concat(nodeChildren.flat())
			}
		}
		return nodeEdges
	},

	// getNodesForUser returns the parent nodes for the given userID along
	// with their descendants if `recursive` is truthy.
	// - `type` - can be used to filter nodes of a specified type (defaults to "")
	// - `includeDel` - set to true to include deleted nodes (defaults to false)
	// - `recursive` - set to true to recursively retrieve all descendants matching
	//   the criteria. In this case, each returned NodeEdge will contain a
	//   `children` property, which is an array of that Node's descendant NodeEdges.
	//   Set to "flat" to return a single flattened array of NodeEdges.
	// - `opts` are options passed to the NATS request
	async getNodesForUser(userID, { type, includeDel, recursive, opts } = {}) {
		// Get root nodes of the user node
		const rootNodes = await this.getNode(userID, {
			parent: "all",
			includeDel,
			opts,
		})

		// Create function to filter nodes based on `type` and `includeDel`
		const filterFunc = (n) => {
			const tombstone = n.edgepointsList.find((e) => e.type === "tombstone")
			return (
				(!type || n.type === type) &&
				(includeDel || !tombstone || tombstone.value % 2 === 0)
			)
		}

		const _cache = {}
		const flat = recursive === "flat"
		const parentNodes = await Promise.all(
			rootNodes.filter(filterFunc).map(async (n) => {
				const [parentNode] = await this.getNode(n.parent, { opts })
				if (!recursive) {
					return parentNode
				}
				const children = await this.getNodeChildren(n.parent, {
					// TODO: Not sure if `type` should be passed here since we need
					// to do recursive search
					type,
					includeDel,
					recursive,
					opts,
					_cache,
				})
				if (flat) {
					return [parentNode].concat(children)
				}
				return Object.assign(parentNode, { children })
			})
		)
		if (flat) {
			return parentNodes.flat()
		}
		return parentNodes
	},

	// subscribePoints subscribes to `p.<nodeID>` and returns an async
	// iterable for Point objects
	subscribePoints(nodeID) {
		const sub = this.subscribe("p." + nodeID)
		// Return subscription wrapped by new async iterator
		return Object.assign(Object.create(sub), {
			async *[Symbol.asyncIterator]() {
				// Iterator reads and decodes Points from subscription
				for await (const m of sub) {
					const { pointsList } = Points.deserializeBinary(m.data).toObject()
					// Convert `time` to JavaScript date and return each point
					for (const p of pointsList) {
						p.time = new Date(p.time.seconds * 1e3 + p.time.nanos / 1e6)
						yield p
					}
				}
			},
		})
	},

	// sendNodePoints sends an array of `points` for a given `nodeID`
	// - `ack` - true if function should block waiting for send acknowledgement
	// - `opts` are options passed to the NATS request
	async sendNodePoints(nodeID, points, { ack, opts } = {}) {
		const payload = encodePoints(points)
		if (!ack) {
			await this.publish("p." + nodeID, payload, opts)
		}

		const m = await this.request("p." + nodeID, payload, opts)

		// Assume message data is an error message
		if (m.data && m.data.length > 0) {
			throw new Error(
				`error sending points for node '${nodeID}': ` + strCodec.decode(m.data)
			)
		}
	},

	// TODO: subscribeEdgePoints

	// sendEdgePoints sends an array of `edgePoints` for the edge between
	// `nodeID` and `parentID`
	// - `ack` - true if function should block waiting for send acknowledgement
	// - `opts` are options passed to the NATS request
	async sendEdgePoints(nodeID, parentID, edgePoints, { ack, opts } = {}) {
		const payload = encodePoints(edgePoints)
		if (!ack) {
			await this.publish("p." + nodeID + "." + parentID, payload, opts)
		}

		const m = await this.request("p." + nodeID + "." + parentID, payload, opts)

		// Assume message data is an error message
		if (m.data && m.data.length > 0) {
			throw new Error(
				`error sending edge points between nodes '${nodeID}' and '${parentID}': ` +
					strCodec.decode(m.data)
			)
		}
	},

	// subscribeMessages subscribes to `node.<nodeID>.msg` and returns an async
	// iterable for Message objects
	subscribeMessages(nodeID) {
		const sub = this.subscribe("node." + nodeID + ".msg")
		// Return subscription wrapped by new async iterator
		return Object.assign(Object.create(sub), {
			async *[Symbol.asyncIterator]() {
				// Iterator reads and decodes Messages from subscription
				for await (const m of sub) {
					yield Message.deserializeBinary(m.data).toObject()
				}
			},
		})
	},

	// subscribeNotifications subscribes to `node.<nodeID>.not` and returns an async
	// iterable for Notification objects
	subscribeNotifications(nodeID) {
		const sub = this.subscribe("node." + nodeID + ".not")
		// Return subscription wrapped by new async iterator
		return Object.assign(Object.create(sub), {
			async *[Symbol.asyncIterator]() {
				// Iterator reads and decodes Messages from subscription
				for await (const m of sub) {
					yield Notification.deserializeBinary(m.data).toObject()
				}
			},
		})
	},
})

// decodeNodesRequest decodes a protobuf-encoded NodesRequest and returns
// the array of nodes returned by the request
function decodeNodesRequest(data) {
	const { nodesList, error } = NodesRequest.deserializeBinary(data).toObject()
	if (error) {
		throw new Error("NodesRequest decode error: " + error)
	}

	for (const n of nodesList) {
		// Convert `time` to JavaScript date for each point
		for (const p of n.pointsList) {
			p.time = new Date(p.time.seconds * 1e3 + p.time.nanos / 1e6)
		}
		for (const p of n.edgepointsList) {
			p.time = new Date(p.time.seconds * 1e3 + p.time.nanos / 1e6)
		}
	}
	return nodesList
}

// encodePoints returns protobuf encoded Points
function encodePoints(points) {
	const payload = new Points()
	// Convert `time` from JavaScript date if needed
	points = points.map((p) => {
		if (p instanceof Point) {
			return p
		}
		let { time = new Date() } = p
		const { type, key, index, value, text, tombstone, data } = p
		p = new Point()
		if (!(time instanceof Timestamp)) {
			let { seconds, nanos } = time
			if (time instanceof Date) {
				const ms = time.valueOf()
				seconds = Math.round(ms / 1e3)
				nanos = (ms % 1e3) * 1e6
			}
			time = new Timestamp()
			time.setSeconds(seconds)
			time.setNanos(nanos)
		}
		p.setTime(time)
		p.setType(type)
		if (key) {
			p.setKey(key)
		}
		if (index) {
			p.setIndex(index)
		}
		if (value || value === 0) {
			p.setValue(value)
		}
		if (text) {
			p.setText(text)
		}
		if (tombstone) {
			p.setTombstone(tombstone)
		}
		if (data) {
			p.setData(data)
		}
		return p
	})
	payload.setPointsList(points)
	return payload.serializeBinary()
}
