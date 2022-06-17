# API

**Contents**

<!-- toc -->

The Simple IoT server currently provides both Http and NATS.io APIs. We've tried
to keep the two APIs a similar as possible so it is easy to switch from one to
the other. The Http API currently accepts JSON, and the NATS API uses protobuf.

**NOTE, the Simple IoT API is not final and will continue to be refined in the
coming months.**

## HTTP

For details on data payloads, it is simplest to just refer to the Go types which
have JSON tags.

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

`curl -i -H "Content-Type: application/json" -H "Accept: application/json" -X POST -d '[{"type":"value", "value":100}]' http://localhost:8080/v1/nodes/be183c80-6bac-41bc-845b-45fa0b1c7766/points`

If you want HTTP authorization, set the `SIOT_AUTH_TOKEN` environment variable
before starting Simple IoT and then pass the token in the authorization header:

`curl -i -H "Authorization: f3084462-3fd3-4587-a82b-f73b859c03f9" -H "Content-Type: application/json" -H "Accept: application/json" -X POST -d '[{"type":"value", "value":100}]' http://localhost:8080/v1/nodes/be183c80-6bac-41bc-845b-45fa0b1c7766/points`

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
  - `node.<id>`
    - returns an array of `data.EdgeNode` structs that meets the specified `id`
      and `parent`.
    - If id = "root", then the root node is fetched.
    - the `parent` can can optionally by specified by setting the message
      payload to one of the following:
      - the ID of the parent node
      - "all" to find all instances of the node. If "all" is specified,
        tombstoned nodes are not returned.
  - `node.<id>.children`
    - can be used to request the immediate children of a node
  - `node.<id>.points`
    - used to listen for or publish node point changes.
  - `node.<id>.<parent>.points`
    - used to publish/subscribe node edge points. The `tombstone` point type is
      used to track if a node has been deleted or not.
  - `node.<id>.not`
    - used when a node sends a [notification](notifications.md) (typically a
      rule, or a message sent directly from a node)
  - `node.<id>.msg`
    - used when a node sends a message (SMS, email, phone call, etc). This is
      typically initiated by a [notification](notifications.md).
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
- System
  - `error`
    - any errors that occur are sent to this subject
