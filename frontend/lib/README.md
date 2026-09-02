# simpleiot-js

A Simple IoT client for the browser, over NATS WebSockets. The web UI in
`frontend/` is built on it.

A connection is made as a signed-in user: the user's node ID and the JWT from
`POST /v1/auth` are the NATS user and password, and the server limits the
connection to the groups the user belongs to. Every read and write names the
group (the _anchor_) it is made under. See the
[API reference](https://docs.simpleiot.org/docs/ref/api.html) for the subjects.

## Install

`npm i simpleiot-js`

The package is an ES module and depends on
[nats.ws](https://github.com/nats-io/nats.ws).

## Usage

```js
import { connect } from "simpleiot-js"

const token = "..." // from POST /v1/auth
const c = await connect({ url: "ws://localhost:8118/", token })

const { userId, anchors } = await c.me()
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

- `connect({ url, token, user, ...opts })` opens a connection. `url` is the
  WebSocket URL of the Simple IoT HTTP server, which proxies to NATS. `user`
  defaults to the ID in the token. Other options go to nats.ws; reconnection is
  on without limit by default.
- `me()` returns `{ userId, anchors, nodes }`: the nodes the user sits under and
  the user's node at each of them.
- `getNodes(anchor, parent, id, { depth, type, includeDel })` fetches nodes the
  way `nodes.<parent>.<id>` does. `depth` also returns descendants that many
  levels down, in one flat array.
- `sendNodePoints(anchor, id, points)` and
  `sendEdgePoints(anchor, id, parent, points)` write points. An error reply is
  thrown.
- `subscribe(subject)` returns a nats.ws subscription whose async iterator
  yields `{ subject, anchor, nodeId, parentId, type, key, points }`. `parentId`
  is set for edge points. A user connection may subscribe to `up.<anchor>.>` for
  each of its anchors.
- `status()`, `closed()`, and `close()` are the nats.ws methods.
- `encodePoints`, `decodePoints`, and `decodeNodes` are the codec.

A point is `{ type, key, time, dataType, value, text, tombstone, origin }`, with
`time` an ISO 8601 string. When sending, `text` makes a string point and
otherwise a non-zero `value` makes a number point; a missing `time` is now. A
node is `{ id, type, parent, points, edgePoints }`.

## Tests

`npm test` checks the codec against fixtures written by the Go tests in
`data/point_fixture_test.go`, so the two encoders cannot drift.
