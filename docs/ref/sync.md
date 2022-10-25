# Data Synchronization

See [research](research.md) for information on techniques that may be applicable
to this problem.

Typically, configuration is modified through a user interface either in the
cloud, or with a local UI (ex touchscreen LCD) at an edge device. Rules may also
eventually change values that need to be synchronized. As mentioned above, the
configuration of a `Node` will be stored as `Points`. Typically the UI for a
node will present fields for the needed configuration based on the `Node`
`Type`, whether it be a user, rule, group, edge device, etc.

In the system, the Node configuration will be relatively static, but the points
in a node may be changing often as sensor values changes, thus we need to
optimize for efficient synchronization of points. We can't afford the bandwidth
to send the entire node data structure any time something changes.

As IoT systems are fundamentally distributed systems, the question of
synchronization needs to be considered. Both client (edge), server (cloud), and
UI (frontend) can be considered independent systems and can make changes to the
same node.

- An edge device with a LCD/Keypad may make configuration changes.
- Configuration changes may be made in the Web UI.
- Sensor values will be sent by an edge device.
- Rules running in the cloud may update nodes with calculated values.

Although multiple systems may be updating a node at the same time, it is very
rare that multiple systems will update the same node point at the same time. The
reason for this is that a point typically only has one source. A sensor point
will only be updated by an edge device that has the sensor. A configuration
parameter will only be updated by a user, and there are relatively few admin
users, and so on. Because of this, we can assume there will rarely be collisions
in individual point changes, and thus this issue can be ignored. The point with
the latest timestamp is the version to use.

## Real-time Point synchronization

Point changes are handled by sending points to a NATS topic for a node any time
it changes. There are three primary instance types:

1. Cloud: will subscribe to point changes on all nodes (wildcard)
1. Edge: will subscribe to point changes only for the nodes that exist on the
   instance -- typically a handful of nodes.
1. WebUI: will subscribe to point changes for nodes currently being viewed --
   again, typically a small number.

With Point Synchronization, each instance is responsible for updating the node
data in its local store.

## Catch-up/non real-time synchronization

Sending points over NATS will handle 99% of data synchronization needs, but
there are a few cases this does not cover:

1. One system is offline for some period of time
1. Data is lost during transmission
1. Other errors or unforeseen situations

There are two types of data:

1. periodic sensor readings (we'll call sample data) that is being continuously
   updated
1. configuration data that is infrequently updated

Any node that produces sample data should send values every 10m, even if the
value is not changing. There are several reasons for this:

- indicates the data source is still alive
- makes graphing easier if there is always data to plot
- covers the synchronization problem for sample data. A new value will be coming
  soon, so don't really need catch-up synchronization for sample data.

Config data is not sent periodically. To manage synchronization of config data,
each `edge` will have a `Hash` field that can be compared between instances.

## Node hash

The edge `Hash` field is a hash of:

- edge point CRCs
- node points CRCs (except for repetitive or high rate sample points)
- child edge `Hash` fields

We store the hash in the `edge` structures because nodes (such as users) can
exist in multiple places in the tree.

This is essentially a Merkle DAG -- see [research](research.md).

Comparing the node `Hash` field allows us to detect node differences. If a
difference is detected, we can then compare the node points and child nodes to
determine the actual differences.

Any time a node point (except for repetitive or high rate data) is modified, the
node's `Hash` field is updated, and the `Hash` field in parents, grand-parents,
etc are also computed and updated. This may seem like a lot of overhead, but if
the database is local, and the graph is reasonably constructed, then each update
might require reading a dozen or so nodes and perhaps writing 3-5 nodes.
Additionally, non sample-data changes are relatively infrequent.

Initially synchronization between edge and cloud nodes is supported. The edge
device will contain an "upstream" node that defines a connection to another
instance's NATS server -- typically in the cloud. The edge node is responsible
for synchronizing of all state using the following algorithm:

1. occasionally the edge device fetches the edge device root node hash from the
   cloud.
1. if the hash does not match, the edge device fetches the entire node and
   compares/updates points. If local points need updated, this process can
   happen all on the edge device. If upstream points need updated, these are
   simply transmitted over NATS.
1. if node hash still does not match, a recursive operation is started to fetch
   child node hashes and the same process is repeated.

### Hash Algorithm

We don't need cryptographic level hashes as we are not trying to protect against
malicious actors, but rather provide a secondary check to ensure all data has
been synchronized. Normally, all data will be sent via points as it is changes
and if all points are received, the Hash is not needed. Therefore, we want to
prioritize performance and efficiency over hash strength. The XOR function has
some interesting properties:

- **Commutative: A ⊕ B = B ⊕ A** (the ability to process elements in any order
  and get the same answer)
- **Associative: A ⊕ (B ⊕ C) = (A ⊕ B) ⊕ C** (we can group operations in any
  order)
- **Identity: A ⊕ 0 = A**
- **Self-Inverse: A ⊕ A = 0** (we can back out an input value by simply applying
  it again)

See
[hash_test.go](https://github.com/simpleiot/simpleiot/blob/master/store/hash_test.go)
for tests of the XOR concept.

### Point CRC

Point CRCs are calculated using the crc-32 of the following point fields:

- `Time`
- `Type`
- `Key`
- `Text`
- `Value`

### Updating the Node Hash

- edge or node points received
  - for points updated
    - back out previous point CRC
    - add in new point CRC
  - update upstream hash values (stops at device node)
    - create cache of all upstream edges to root
    - for each upstream edge, back out old hash, and xor in new hash
    - write all updated edge hash fields

It should again be emphasized that repetitive or high rate points should not be
included in the hash because they will be sent again soon -- we do not need the
hash to ensure they get synchronized. The hash should only include points that
change at slow rates (user changes, state, etc). Anything machine generated
should be repeated -- even if only every 10m.

The hash is only useful in synchronizing state between a device node tree, and a
subset of the upstream node tree. For instances which do not have an upstream of
peer instances, there is little value in calculating hash values back to the
root node and could be computationally intensive for a cloud instance that had
1000's of child nodes.
