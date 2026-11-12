# simpleiot-js

A JavaScript client for [Simple IoT](https://simpleiot.org) that runs in the
browser and talks to the server over NATS WebSockets. The Simple IoT web UI is
built on it.

![a JavaScript app in the browser signs in, then connects to the SIOT HTTP port over WebSocket as that user, and reaches only the subtree of the groups the user belongs to](https://docs.simpleiot.org/js/simpleiot-js/images/connection.svg)

A connection is made as a signed-in user. The app signs in with `POST /v1/auth`,
then opens a WebSocket on the server's HTTP port with the user's node ID and the
returned JWT as the NATS user and password. The server limits the connection to
the groups the user belongs to, called _anchors_, and every read and write names
the anchor it is made under. See the
[API reference](https://docs.simpleiot.org/docs/ref/api.html) for the subjects.

## Install

```
npm i simpleiot-js
```

The package is an ES module and depends on
[nats.ws](https://github.com/nats-io/nats.ws). TypeScript declarations are in
`siot-nats.d.ts` and `codec.d.ts`.

## Usage

```js
import { connect } from "simpleiot-js"

const res = await fetch("http://localhost:4223/v1/auth", {
	method: "POST",
	body: new URLSearchParams({ email: "admin", password: "admin" }),
})
const { token } = await res.json()

const c = await connect({ url: "ws://localhost:4223/", token })

const { anchors } = await c.me()
for (const anchor of anchors) {
	const nodes = await c.getNodes(anchor, anchor, "all", { depth: 1 })
	console.log(nodes)
}

const sub = c.subscribe(`up.${anchors[0]}.>`)
for await (const m of sub) {
	console.log(m.nodeId, m.points)
}
```

## API

`connect({ url, token, user, ...opts })` opens a connection and returns a
`SiotConnection`. `url` is the WebSocket URL of the Simple IoT HTTP port (4223
by default), which relays the connection to NATS. `user` defaults to the ID in
the token. Other options go to nats.ws; reconnection is on without limit by
default.

`SiotConnection` methods:

- `me()` returns `{ userId, anchors, nodes }`: the nodes the user sits under and
  the user's node at each of them.
- `getNodes(anchor, parent, id, { depth, type, includeDel })` fetches nodes the
  way `nodes.<parent>.<id>` does. `depth` also returns descendants that many
  levels down, in one flat array.
- `searchNodes(anchor, query, { limit, type })` finds nodes below an anchor
  whose description, ID, or type matches `query`, with their ancestors.
- `sendNodePoints(anchor, id, points)` and
  `sendEdgePoints(anchor, id, parent, points)` write points. An error reply is
  thrown.
- `subscribe(subject)` returns a nats.ws subscription whose async iterator
  yields `{ subject, anchor, nodeId, parentId, type, key, points }`. `parentId`
  is set for edge points. A user connection may subscribe to `up.<anchor>.>` for
  each of its anchors.
- `request(subject, payload, timeout)` sends a raw NATS request, and `nc` is the
  underlying nats.ws connection, for anything the methods above do not cover.
- `status()`, `closed()`, and `close()` are the nats.ws methods.

Each request method also takes a `timeout` option in milliseconds, 10 seconds by
default.

The package also exports the codec (`encodePoints`, `decodePoints`,
`decodeNodes`, and `DataType`), `userIdFromToken(token)`, and
`inboxPrefix(userId)`, the reply subject prefix a user connection may subscribe
to.

A point is `{ type, key, time, dataType, value, text, tombstone, origin }`, with
`time` an ISO 8601 string. When sending, `text` makes a string point and
otherwise a non-zero `value` makes a number point; a missing `time` is now. A
node is `{ id, type, parent, points, edgePoints }`.

## Tests

`npm test` checks the codec against fixtures written by the Go tests in
`data/point_fixture_test.go`, so the two encoders cannot drift.
