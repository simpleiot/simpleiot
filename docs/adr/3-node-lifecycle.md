# Node Lifecycle

- Author: Cliff Brake, last updated: 2022-02-16
- PR/Discussion:
- Status: discussion

## Context

In the process of implementing a
[feature to duplicate a node tree](https://github.com/simpleiot/simpleiot/issues/312),
several problems have surfaced related to the lifecycle of creating and updating
nodes.

- traditionally, if a point was sent and node did not exist, SIOT created a
  "device" node as a child of the root node with this point. This was based on
  this initial use of SIOT with 1-wire devices.
- the creation process for a node involves:
  - sending all the points of a node including a meta point with the node type.
  - sending the edge points of a node to describe the upstream connection
- there is also a feature where if we send a point to a Device node that does
  not have an upstream path to root, or that path is tombstoned, we create this
  path. This ensures that we don't have orphaned device nodes in an upstream if
  they are still active.

In the past, when creating a node, we sent all the node points, then the edge
points. However this can create an issue in that an upstream edge for a device
node does not exist yet, so in a multi-level upstream configuration A->B->C, if
B is syncing to C for the first time, multiple instances of A will be created on
C.

An attempt was made to switch the sending edge points of new nodes before node
points, however this created other issues (TODO: detail these).

Sending node and edge points separately for new nodes creates an issue in that
these don't happen in one communication transaction, so there is a period of
time between the two where the node state is indeterminate. Consideration was
given to adding a NATS endpoint to create nodes where everything could be sent
at once. However, this is problematic in that now there is another NATS subject
for everyone to listen to and process, rather than just listening for new
points. It seems less than ideal to have multiple subjects that can
create/modify node points.

It seems at this point we can probably deprecate the feature to create new
devices nodes based on a single point. This will force new nodes to be
explicitly created. This is probably OK as new nodes are created in several
ways:

1. by the user in the UI
2. by the upstream sync mechanism -- if the hash does match or a node does not
   exist upstream, it is sent. This is continuously checked so if a message does
   not succeed, it will eventually get resent.
3. plug-n-play discovery mechanisms that detect new devices and automatically
   populate new nodes. Again, it is not a big deal if a message gets lost as the
   discovery mechanism will continue to try to create the new device if it does
   not find it.
