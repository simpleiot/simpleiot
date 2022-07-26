# Data

**Contents**

<!-- toc -->

## Data Structures

As a client developer, there are two main primary structures:
[`NodeEdge`](https://pkg.go.dev/github.com/simpleiot/simpleiot/data#NodeEdge)
and [`Point`](https://pkg.go.dev/github.com/simpleiot/simpleiot/data#Point). A
`Node` can be considered a collection of `Points`.

These data structures describe most data that is stored and transferred in a
Simple IoT system.

The core data structures are currently defined in the
[`data`](https://github.com/simpleiot/simpleiot/tree/master/data) directory for
Go code, and
[`frontend/src/Api`](https://github.com/simpleiot/simpleiot/tree/master/frontend/src/Api)
directory for Elm code.

A `Point` can represent a sensor value, or a configuration parameter for the
node. With sensor values and configuration represented as `Points`, it becomes
easy to use both sensor data and configuration in rule or equations because the
mechanism to use both is the same. Additionally, if all `Point` changes are
recorded in a time series database (for instance Influxdb), you automatically
have a record of all configuration and sensor changes for a `node`.

Treating most data as `Points` also has another benefit in that we can easily
simulate a device -- simply provide a UI or write a program to modify any point
and we can shift from working on real data to simulating scenarios we want to
test.

Edges are used to describe the relationships between nodes as a
[directed acyclic graph](https://en.wikipedia.org/wiki/Directed_acyclic_graph).

![dag](images/dag.svg)

`Nodes` can have parents or children and thus be represented in a hierarchy. To
add structure to the system, you simply add nested `Nodes`. The `Node` hierarchy
can represent the physical structure of the system, or it could also contain
virtual `Nodes`. These virtual nodes could contain logic to process data from
sensors. Several examples of virtual nodes:

- a pump `Node` that converts motor current readings into pump events.
- implement moving averages, scaling, etc on sensor data.
- combine data from multiple sensors
- implement custom logic for a particular application
- a component in an edge device such as a cellular modem

Like Nodes, Edges also contain a Point array that further describes the
relationship between Nodes. Some examples:

- role the user plays in the node (viewer, admin, etc)
- order of notifications when sequencing notifications through a node's users
- node is enabled/disabled -- for instance we may want to disable a Modbus IO
  node that is not currently functioning.

Being able to arranged nodes in an arbitrary hierarchy also opens up some
interesting possibilities such as creating virtual nodes that have a number of
children that are collecting data. The parent virtual nodes could have rules or
logic that operate off data from child nodes. In this case, the virtual parent
nodes might be a town or city, service provider, etc., and the child nodes are
physical edge nodes collecting data, users, etc.

## Synchronization

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

### Real-time Point synchronization

Point changes are handled by sending points to a NATS topic for a node any time
it changes. There are three primary instance types:

1. Cloud: will subscribe to point changes on all nodes (wildcard)
1. Edge: will subscribe to point changes only for the nodes that exist on the
   instance -- typically a handful of nodes.
1. WebUI: will subscribe to point changes for nodes currently being viewed --
   again, typically a small number.

With Point Synchronization, each instance is responsible for updating the node
data in its local store.

### Catch-up/non real-time synchronization

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
each `edge` will have a `Hash` field.

The edge `Hash` field is a hash of:

- edge and node point timestamps except for sample points. Sample points are
  excluded from the hash as discussed above.
- child edge `Hash` fields

We store the hash in the `edge` structures because nodes (such as users) can
exist in multiple places in the tree.

The points are sorted by timestamp and child nodes are sorted by hash so that
the order is consistent when the hash is computed.

This is essentially a Merkle Tree -- see [research](research.md).

Comparing the node `Hash` field allows us to detect node differences. We then
compare the node points and child nodes to determine the actual differences.

Any time a node point (except for sample date) is modified, the node's `Hash`
field is updated, and the `Hash` field in parents, grand-parents, etc are also
computed and updated. This may seem like a lot of overhead, but if the database
is local, and the graph is reasonably constructed, then each update might
require reading a dozen or so nodes and perhaps writing 3-5 nodes. Additionally,
non sample-data changes are relatively infrequent.

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

The md5 algorithm is used to compute the hash fields because it is relatively
efficient to compute and reasonably small. While sha246 might be more _secure_,
the application of this hash is not security, but rather verifiability. The
failure mode is that two different trees will generate the same hash. Because
the root hash is always changing, this is not really a problem as the next
change to the tree will likely trigger a new hash -- usually within a short
amount of time.

### Node Topology changes

Nodes can exist in multiple locations in the tree. This allows us to do things
like include a user in multiple groups.

#### Add

Node additions are detected in real-time by sending the points for the new node
as well as points for the edge node that adds the node to the tree.

#### Copy

Node copies are are similar to add, but only the edge points are sent.

#### Delete

Node deletions are recorded by setting a tombstone point in the edge above the
node to true. If a node is deleted, this information needs to be recorded,
otherwise the synchronization process will simply re-create the deleted node if
it exists on another instance.

#### Move

Move is just a combination of Copy and Delete.

If the any real-time data is lost in any of the above operations, the catch up
synchronization will propagate any node changes.

## Tracking who made changes

The `Point` type has an `Origin` field that is used to track who generated this
point. If the node that owned the point generated the point, then Origin can be
left blank -- this saves data bandwidth -- especially for sensor data which is
generated by the client managing the node. There are several reasons for the
`Origin` field:

- track who made changes for auditing and debugging purposes
- eliminate echos where a client may be subscribed to a subject as well as
  publish to the same subject. With the Origin field, the client can determine
  if it was the author of a point it receives, and if so simply drop it. See
  [client documentation](client.md#message-echo) for more discussion of the echo
  topic.

## Converting Nodes to other data structures

Nodes and Points are convenient for storage and synchronization, but cumbersome
to work with in application code that uses the data, so we typically convert
them to another data structure. `data.Decode` and `data.Encode` can be used to
convert Node data structures to your own custom `struct`, much like the Go
`json` package.
