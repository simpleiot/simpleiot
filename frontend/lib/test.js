// Codec tests against the fixtures data/point_fixture_test.go writes, so
// the JavaScript and Go encoders are checked against the same bytes.
// Run with `npm test` (node --test).

import assert from "node:assert/strict"
import { readFile } from "node:fs/promises"
import { test } from "node:test"
import { decodeNodes, decodePoints, encodePoints } from "./codec.js"
import { inboxPrefix, userIdFromToken } from "./siot-nats.js"

const fixture = (name) =>
	readFile(new URL(`./testdata/${name}`, import.meta.url))
const fixtureJSON = async (name) => JSON.parse(await fixture(name))

test("decodes points the way Go reads them", async () => {
	const bytes = new Uint8Array(await fixture("points.bin"))
	assert.deepEqual(decodePoints(bytes), await fixtureJSON("points.json"))
})

test("encodes points to the bytes Go wrote", async () => {
	const want = new Uint8Array(await fixture("points.bin"))
	const got = encodePoints(await fixtureJSON("points.json"))
	assert.deepEqual(got, want)
})

test("round-trips points", async () => {
	const points = await fixtureJSON("points.json")
	assert.deepEqual(decodePoints(encodePoints(points)), points)
})

test("fills in a missing time", () => {
	const before = Date.now()
	const [p] = decodePoints(
		encodePoints([{ type: "value", key: "0", value: 1 }])
	)
	const t = new Date(p.time).valueOf()
	assert.ok(t >= before - 1 && t <= Date.now() + 1, `time ${p.time} is not now`)
})

test("decodes a node reply", async () => {
	const bytes = new Uint8Array(await fixture("nodes.bin"))
	assert.deepEqual(decodeNodes(bytes), await fixtureJSON("nodes.json"))
})

test("throws the error in a node reply", async () => {
	const bytes = new Uint8Array(await fixture("nodes-error.bin"))
	assert.throws(() => decodeNodes(bytes), { message: "not in scope" })
	assert.deepEqual(decodeNodes(new Uint8Array(0)), [])
})

test("reads the user ID from a token", () => {
	const payload = btoa(JSON.stringify({ jti: "user-1", exp: 1 }))
	assert.equal(userIdFromToken(`eyJhbGciOiJIUzI1NiJ9.${payload}.sig`), "user-1")
	assert.equal(userIdFromToken("garbage"), "")
	assert.equal(inboxPrefix("user-1"), "_INBOX_user-1")
})
