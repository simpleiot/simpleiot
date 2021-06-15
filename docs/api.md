+++
title = "API"
weight = 5
+++

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
  - `/v1/nodes/:id`
    - GET: return info about a specific node
    - DELETE: delete a node
  - `/v1/nodes/:id/parents`
    - POST: move node to new parent
    - PUT: add parent _(not implemented yet)_
  - `/v1/nodes/:id/points`
    - POST: post points for a node
  - `/v1/nodes/:id/cmd`
    - GET: gets a command for a node and clears it from the queue. Also clears
      the CmdPending flag in the Device state.
    - POST: posts a cmd for the node and sets the node CmdPending flag.
  - `/v1/nodes/:id/not`
    - POST: send a [notification](../data/notification.md) to all node users and
      upstream users
- Auth
  - `/v1/auth`
    - POST: accepts `email` and `password` as form values, and returns a JWT
      Auth
      [token](https://github.com/simpleiot/simpleiot/blob/master/data/auth.go)

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
defined [here](../internal/pb).

- Nodes
  - `node.<id>`
    - can be used to request an entire node data structure. If id = "root", then
      the root node is fetched.
    - body can optionally include the ID of the parent node to populate node
      with tombstone info, and other data from the edge data structure.
  - `node.<id>.children`
    - can be used to request the immediate children of a node
  - `node.<id>.points`
    - used to listen for or publish node point changes
  - `node.<id>.not`
    - used when a node sends a [notification](notifications.md) (typically a
      rule, or a message sent directly from a node)
  - `node.<id>.msg`
    - used when a node sends a message (SMS, email, phone call, etc). This is
      typically initiated by a [notification](notifications.md).
  - `node.<id>.file`
    - is used to transfer files to a node in chunks, which is optimized for
      unreliable networks like cellular and is handy for transfering software
      update files. There is Go code [available](../api/nats-file.go) to manage
      both ends of the transfer as well as a utility to [send](../cmd/siotutil)
      files and an example [edge](../cmd/edge) application to receive files.
- Edges
  - `edge.<id>.points`
    - used to listen for or publish node edge point changes
- System
  - `error`
    - any errors that occur are sent to this subject
