# API

**Contents**

<!-- toc -->

The Simple IoT server currently provides both Http and NATS.io APIs. We've tried
to keep the two APIs a similar as possible so it is easy to switch from one to
the other. The Http API currently accepts JSON, and the NATS API uses protobuf.

**NOTE, the Simple IoT API is not final and will continue to be refined in the
coming months.**

## NATS

[NATS.io](https://nats.io/) allows more complex and efficient interactions
between various system components (device, cloud, and web UI). These three parts
of the system make IoT systems inherently distributed. NATS focuses on
simplicity and is written in Go which ensures the Go client is a 1st class
citizen and also allows for interesting possibilities such as embedding in the
NATS server in various parts of the system. This allows us to keep our
one-binary deployment model.

The `siot` binary embeds the NATS server, so there is no need to deploy and run
a separate NATS server.

For the NATS transport, protobuf encoding is used for all transfers and are
defined [here](https://github.com/simpleiot/simpleiot/tree/master/internal/pb).

- Nodes
  - `nodes.<parentId>.<nodeId>`
    - Request/response -- returns an array of `data.EdgeNode` structs.
    - `parent="all"`, then all instances of the node are returned.
    - `parent is set and id="all"`, then all child nodes of the parent are
      returned.
    - `parent="root" and id="all"` to fetch the root node(s).
    - The following combinations are invalid:
      - `parent="all" && id="all"`
    - parameters can be specified as points in payload
      - `tombstone` with value field set to 1 will include deleted points
      - `nodeType` with text field set to node type will limit returned nodes to
        this type
  - `p.<nodeId>.<type>.<key>`
    - used to listen for or publish node point changes.
  - `p.<nodeId>.<parentId>.<type>.<key>`
    - used to publish/subscribe node edge points. The `tombstone` point type is
      used to track if a node has been deleted or not.
  - `phr.<nodeId>` (not currently used)
    - high rate point data
  - `phrup.<upstreamId>.<nodeId>`
    - high rate point data re-broadcasted upstream. `upstreamId` is the parent
      of the node that is interested in HR data (currently the db node).
      `nodeId` is the node that is providing the HR data. In the case of a
      custom HR Dest Node (serial client), the serial client may not be a child
      of the upstream node.
  - `up.<upstreamId>.<nodeId>.<type>.<key>`
    - node points are rebroadcast at every upstream ID so that we can listen for
      point changes at any level. The sending node is also included in this. The
      store is responsible for posting to `up` subjects. Individual clients
      should not do this.
  - `up.<upstreamId>.<nodeId>.<parentId>.<type>.<key>`
    - edge points rebroadcast at every upstream node ID.
  - `history.<nodeId>`
    - Request/response -- payload is a JSON-encoded `HistoryQuery` struct.
      Returns a JSON-encoded `data.HistoryResult`.
- Legacy APIs that are being deprecated
  - `node.<id>.not`
    - used when a node sends a [notification](./notifications.md) (typically a
      rule, or a message sent directly from a node)
  - `node.<id>.msg`
    - used when a node sends a message (SMS, email, phone call, etc). This is
      typically initiated by a [notification](./notifications.md).
  - `node.<id>.file` (not currently implemented)
    - is used to transfer files to a node in chunks, which is optimized for
      unreliable networks like cellular and is handy for transfering software
      update files.
- Auth
  - `auth.user`
    - used to authenticate a user. Send a request with email/password points,
      and the system will respond with the User nodes if valid. There may be
      multiple user nodes if the user is instantiated in multiple places in the
      node graph. A JWT node will also be returned with a token point. This JWT
      should be used to authenticate future requests. The frontend can then
      fetch the parent node for each user node.
  - `auth.getNatsURI`
    - this returns the NATS URI and Auth Token as points. This is used in cases
      where the client needs to set up a new connection to specify the no-echo
      option, or other features.
- Admin
  - `admin.error` (not implemented yet)
    - any errors that occur are sent to this subject
  - `admin.storeVerify`
    - used to initiate a database verification process. This currently verifies
      hash values are correct and responds with an error string.
  - `admin.storeMaint`
    - corrects errors in the store (current incorrect hash values)

## HTTP

For details on data payloads, it is simplest to just refer to the Go types which
have JSON tags. HTTP APIs currently return JSON payloads.

Most APIs that do not return specific data (update/delete) return a
[StandardResponse](https://github.com/simpleiot/simpleiot/blob/master/data/api.go)

- Nodes
  - [data structure](https://github.com/simpleiot/simpleiot/blob/master/data/node.go)
  - `/v1/nodes`
    - GET: return a list of all nodes
    - POST: insert a new node
  - `/v1/nodes/:id`
    - GET: return info about a specific node. Body can optionally include the id
      of parent node to include edge point information.
    - DELETE: delete a node
  - `/v1/nodes/:id/parents`
    - POST: move node to new parent
    - PUT: mirror/duplicate node
    - body is JSON api/nodes.go:NodeMove or NodeCopy structs
  - `/v1/nodes/:id/points`
    - POST: post points for a node
  - `/v1/nodes/:id/cmd`
    - GET: gets a command for a node and clears it from the queue. Also clears
      the CmdPending flag in the Device state.
    - POST: posts a cmd for the node and sets the node CmdPending flag.
  - `/v1/nodes/:id/not`
    - POST: send a
      [notification](https://github.com/simpleiot/simpleiot/blob/master/data/notification.go)
      to all node users and upstream users
- Auth
  - `/v1/auth`
    - POST: accepts `email` and `password` as form values, and returns a JWT
      Auth
      [token](https://github.com/simpleiot/simpleiot/blob/master/data/auth.go)

### HTTP Examples

You can post a point using the HTTP API without authorization using curl:

`curl -i -H "Content-Type: application/json" -H "Accept: application/json" -X POST -d '[{"type":"value", "value":100}]' http://localhost:8118/v1/nodes/be183c80-6bac-41bc-845b-45fa0b1c7766/points`

If you want HTTP authorization, set the `SIOT_AUTH_TOKEN` environment variable
before starting Simple IoT and then pass the token in the authorization header:

`curl -i -H "Authorization: f3084462-3fd3-4587-a82b-f73b859c03f9" -H "Content-Type: application/json" -H "Accept: application/json" -X POST -d '[{"type":"value", "value":100}]' http://localhost:8118/v1/nodes/be183c80-6bac-41bc-845b-45fa0b1c7766/points`
