import assert from "node:assert"
import WebSocket from "ws"
import { connect } from "./siot-nats.mjs"

import { inspect } from "node:util"
inspect.defaultOptions.depth = 10
// Using the Mocha test framework: https://mochajs.org/

// Tell eslint about the additional globals that will be used
/* global global, it, describe, before */

global.WebSocket = WebSocket
let c, rootID, adminID

it("connects", async () => {
	c = await connect()
})
describe("simpleiot-js", () => {
	// abort tests if connection failed
	before(function () {
		if (!c) {
			this.skip()
		}
	})

	it("gets root node", async () => {
		const roots = await c.getNodeChildren("root")
		assert.strictEqual(roots.length, 1, "should have 1 root")

		// Check root node
		const [root] = roots
		assert.strictEqual(root.type, "device", "root type should be 'device'")
		assert.strictEqual(root.parent, "root", "root parent should be 'root'")
		assert.strictEqual(root.edgepointsList.length, 1, "should have 1 edgepoint")
		const [tombstone] = root.edgepointsList
		assert.strictEqual(tombstone.type, "tombstone", "should have tombstone")
		assert.strictEqual(tombstone.tombstone, 0, "tombstone is 0")

		rootID = root.id
	})

	it("gets root node by ID", async function () {
		if (!rootID) {
			return this.skip()
		}

		const roots = await c.getNode(rootID)
		assert.strictEqual(roots.length, 1, "should have 1 root")

		const [root] = roots
		assert.strictEqual(root.id, rootID, "root id should match")
		assert.strictEqual(root.type, "device", "root type should be 'device'")
		assert.strictEqual(root.parent, "root", "parent should be root")
		assert.strictEqual(
			root.edgepointsList.length,
			1,
			"only tombstone edge point"
		)
		assert.strictEqual(
			root.edgepointsList[0].type,
			"tombstone",
			"edge point should be tombstone"
		)
		assert.strictEqual(root.edgepointsList[0].value, 0, "tombstone should be 0")
	})

	it("gets nodes recursively", async () => {
		const roots = await c.getNodeChildren("root", { recursive: true })
		assert.strictEqual(roots.length, 1, "should have 1 root")

		// Check root node
		const [root] = roots
		assert.strictEqual(root.children.length, 1, "root should have 1 child")
		const [user] = root.children
		assert.strictEqual(user.type, "user", "child node type should be 'user'")
		assert.strictEqual(user.parent, root.id, "invalid parent")
		const [tombstone] = root.edgepointsList
		assert.strictEqual(tombstone.type, "tombstone", "should have tombstone")
		assert.strictEqual(tombstone.tombstone, 0, "tombstone is 0")

		// Check default points
		const { text: first } = user.pointsList.find((p) => p.type === "firstName")
		assert.strictEqual(
			first,
			"admin",
			"firstName of admin user is not default value"
		)
		const { text: email } = user.pointsList.find((p) => p.type === "email")
		assert.strictEqual(
			email,
			"admin@admin.com",
			"email of admin user is not default value"
		)

		adminID = user.id
	})

	it("publishes points for user name", async function () {
		if (!adminID) {
			return this.skip()
		}
		await c.sendNodePoints(
			adminID,
			[
				{ type: "firstName", text: "John" },
				{ type: "lastName", text: "Doe" },
			],
			{ ack: true }
		)

		// Get updated node
		const [user] = await c.getNode(adminID)
		const first = user.pointsList.find((p) => p.type === "firstName")
		const last = user.pointsList.find((p) => p.type === "lastName")
		assert.strictEqual(first.text, "John", "unchanged firstName " + first.text)
		assert.strictEqual(last.text, "Doe", "unchanged lastName " + last.text)
	})

	const nodeID = "faux-sensor-1"
	it("creates a device node", async function () {
		if (!rootID) {
			return this.skip()
		}

		// Create `nodeID`
		await c.sendNodePoints(
			nodeID,
			[{ type: "description", text: nodeID + " (created by test suite)" }],
			{ ack: true }
		)
		await c.sendEdgePoints(
			nodeID,
			rootID,
			[{ type: "tombstone" }, { type: "nodeType", text: "device" }],
			{
				ack: true,
			}
		)
	})

	it("subscribes / publishes node points", async function () {
		const ITERATIONS = 10
		let temperature = 30
		let humidity = 55
		let pointsRx = 0

		// Subscribe and start reading points asynchronously
		const sub = c.subscribePoints(nodeID)
		const readPromise = (async function readPoints() {
			for await (const { points } of sub) {
				points.forEach((p) => {
					if (p.type === "temperature") {
						assert.strictEqual(p.value, temperature)
					} else if (p.type === "humidity") {
						assert.strictEqual(p.value, humidity)
					} else {
						throw new Error("unknown point type: " + p.type)
					}
					pointsRx++
				})
			}
		})()
		// Note: Add no-op `catch` to avoid unhandled Promise rejection
		readPromise.catch((err) => {})

		// Send out node points
		for (let i = 0; i < ITERATIONS; i++) {
			await c.sendNodePoints(
				nodeID,
				[
					{ type: "temperature", value: ++temperature },
					{ type: "humidity", value: ++humidity },
				],
				{ ack: true }
			)
		}

		// Close subscription and handle errors
		await sub.close()
		await readPromise

		// Check to ensure we read all points
		assert.strictEqual(
			pointsRx,
			ITERATIONS * 2,
			"did not receive all published points"
		)
	})

	it("closes", async () => {
		await c.close()
	})
})
