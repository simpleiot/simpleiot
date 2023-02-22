# simpleiot-js

SimpleIoT JavaScript API using NATS / WebSockets

This package allows JavaScript clients (especially web browsers) to connect to
SimpleIoT using [nats.ws](https://github.com/nats-io/nats.ws).

## Install

`npm i simpleiot-js`

## Usage

```js
import { connect } from "simpleiot-js"
;(async function siotConnect() {
	try {
		// Note: nats.ws has built-in reconnection logic by default
		const nc = await connect({
			servers: "localhost:9222",
			// Pass in options as documented in nats.ws package
		})
		// `getServer()` is a method documented by nats.ws
		console.log(`connected to ${nc.getServer()}`)
		// `closed()` is a nats.ws method that returns a promise
		// indicating the client closed
		const done = nc.closed()

		// Example: get root nodes from SimpleIoT tree
		const n = await nc.getNodeChildren("root")

		// close the connection
		await nc.close()
		// check if the close was OK
		const err = await done
		if (err) {
			console.log(`error closing:`, err)
		}
	} catch (err) {
		console.error("connection error:", err)
	}
})()
```

## API

The SimpleIoT package is simply a wrapper of the
[nats.ws](https://github.com/nats-io/nats.ws) package. Any API documented in the
nats.ws package will work. We have also added the following functions specific
to SimpleIoT.

- `getNode(id, { parent, type, includeDel, opts } = {})`
- `getNodeChildren(parentID, { type, includeDel, recursive, opts } = {} )`
- `getNodesForUser(userID, { type, includeDel, recursive, opts } = {})`
- `subscribePoints(nodeID)`
- `sendNodePoints(nodeID, points, { ack, opts })`
- `sendEdgePoints(nodeID, parentID, edgePoints, { ack, opts })`
- `subscribeMessages(nodeID)`
- `subscribeNotifications(nodeID)`
