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

  getNode sends a request to `nodes.<parent>.<id>` to retrieve an array of
  NodeEdges for the specified Node ID.

  - If `id` is "all" or falsy, this calls `getNodeChildren` instead; however we
    strongly recommend using `getNodeChildren` directly
  - If `parent` is "all" or falsy, all instances of the specified node are
    returned
  - If `parent` is "root", only root nodes are returned
  - `opts` are options passed to the NATS request

- `getNodeChildren(parentID, { type, includeDel, recursive, opts } = {} )`

  getNodeChildren sends a request to `nodes.<parentID>.<id>` to retrieve an
  array of child NodeEdges of the specified parent node.

  - If `parentID` is "root", all root nodes are retrieved
  - `type` - can be used to filter nodes of a specified type (defaults to "")
  - `includeDel` - set to true to include deleted nodes (defaults to false)
  - `recursive` - set to true to recursively retrieve all descendants matching
    the criteria. In this case, each returned NodeEdge will contain a `children`
    property, which is an array of that Node's descendant NodeEdges. Set to
    "flat" to return a single flattened array of NodeEdges.

    Note: If `type` is also set when `recursive` is truthy, `type` restricts
    which nodes are recursively searched. If you need to search descendants that
    do _not_ match the `type`, consider removing the `type` filter and filter
    manually.

  - `opts` are options passed to the NATS request

- `getNodesForUser(userID, { type, includeDel, recursive, opts } = {})`

  getNodesForUser returns the parent nodes for the given `userID` along with
  their descendants if `recursive` is truthy.

  - `type` - can be used to filter nodes of a specified type (defaults to "")
  - `includeDel` - set to true to include deleted nodes (defaults to false)
  - `recursive` - set to true to recursively retrieve all descendants matching
    the criteria. In this case, each returned NodeEdge will contain a `children`
    property, which is an array of that Node's descendant NodeEdges. Set to
    "flat" to return a single flattened array of NodeEdges.
  - `opts` are options passed to the NATS request

- `subscribePoints(nodeID)`

  Subscribes to `p.<nodeID>` and returns an
  [async iterable](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Iteration_protocols#the_async_iterator_and_async_iterable_protocols)
  for an array of Point objects.

- `subscribeEdgePoints(nodeID)`

  Subscribes to `p.<nodeID>.<parentID>` and returns an
  [async iterable](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Iteration_protocols#the_async_iterator_and_async_iterable_protocols)
  for an array of Point objects. `parentID` can be "\*" or "all".

- `subscribeUpstreamPoints(upstreamID, nodeID)`

  Subscribes to `up.<upstreamID>.<nodeID>` and returns an
  [async iterable](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Iteration_protocols#the_async_iterator_and_async_iterable_protocols)
  for an array of Point objects. `nodeID` can be `*` or `all`.

- `subscribeUpstreamEdgePoints(upstreamID, nodeID, parentID)`

  Subscribes to `up.<upstreamID>.<nodeID>.<parentID>` and returns an
  [async iterable](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Iteration_protocols#the_async_iterator_and_async_iterable_protocols)
  for an array of Point objects. `nodeID` and `parentID` can be "\*" or "all".

- `setUserID(userID)`

  setUserID sets the user ID for this connection; any points / edge points sent
  from this connection will have their origin set to the specified userID

- `sendNodePoints(nodeID, points, { ack, opts })`

  sendNodePoints sends an array of `points` for a given `nodeID`

  - `ack` - true if function should block waiting for send acknowledgement
    (defaults to true)
  - `opts` are options passed to the NATS request

- `sendEdgePoints(nodeID, parentID, edgePoints, { ack, opts })`

  sendEdgePoints sends an array of `edgePoints` for the edge between `nodeID`
  and `parentID`

  - `ack` - true if function should block waiting for send acknowledgement
    (defaults to true)
  - `opts` are options passed to the NATS request

- `subscribeMessages(nodeID)`

  subscribeMessages subscribes to `node.<nodeID>.msg` and returns an
  [async iterable](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Iteration_protocols#the_async_iterator_and_async_iterable_protocols)
  for Message objects

- `subscribeNotifications(nodeID)`

  subscribeNotifications subscribes to `node.<nodeID>.not` and returns an
  [async iterable](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Iteration_protocols#the_async_iterator_and_async_iterable_protocols)
  for Notification objects
