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

it("connect", async () => {
	c = await connect()
})
describe("simpleiot-js", () => {
	// abort tests if connection failed
	before(function () {
		if (!c) {
			this.skip()
		}
	})

	it("get root node", async () => {
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

	it("get root node by ID", async function () {
		if (!rootID) {
			return this.skip()
		}

		const roots = await c.getNode(rootID)
		assert.strictEqual(roots.length, 1, "should have 1 root")

		const [root] = roots
		assert.strictEqual(root.type, "device", "root type should be 'device'")
		assert.strictEqual(root.parent, "", "parent should be blank")
		assert.strictEqual(root.edgepointsList.length, 0, "no edges retrieved")
	})

	it("get nodes recursively", async () => {
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

	it("change user first name", async function () {
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

	it("close", async () => {
		await c.close()
	})
})
