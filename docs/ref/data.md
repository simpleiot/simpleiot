# Data

**Contents**

<!-- toc -->

See also:

- [Data store](store.md)
- [Data syncronization](sync.md)

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

### The Point `Key` field constraint

The Point data structure has a `Key` field that can be used to construct Array
and Map data structures in a node. This is a flexible idea in that it is easy to
transition from a scaler value to an array or map. However, it can also cause
problems if one client is writing key values of `""` and another client (say a
rule action) is writing value of `"0"`. One solution is to have fancy logic that
equates `""` to `"0"` on point updates, compares, etc. Another approach is to
consider `""` and invalid key value and set key to `"0"` for scaler values. This
incurs a slight amount of overhead, but leads to more predictable operation and
eliminates the possibility of having two points in a node that mean the same
things.

**The Simple IoT Store always sets the Key field to `"0"` on incoming points if
the Key field is blank.**

Clients should be written with this in mind.

### Arrays and Maps

Points can be used to represent arrays and maps. For an array, the `key` field
contains the index `"0"`, `"1"`, `"2"`, etc. For maps, the `key` field contains
the key of the map. An example:

| Type            | Key   | Text             | Value |
| --------------- | ----- | ---------------- | ----- |
| description     | 0     | Node Description |       |
| ipAddress       | 0     | 192.168.1.10     |       |
| ipAddress       | 1     | 10.0.0.3         |       |
| diskPercentUsed | /     |                  | 43    |
| diskPercentUsed | /home |                  | 75    |
| switch          | 0     |                  | 1     |
| switch          | 1     |                  | 0     |

The above would map to the following Go type:

```go
type myNode struct {
    ID              string      `node:"id"`
    Parent          string      `node:"parent"`
    Description     string      `node:"description"`
    IpAddresses     []string    `point:"ipAddress"`
    Switches        []bool      `point:"switch"`
    DiscPercentUsed []float64   `point:"diskPercentUsed"`
}
```

The
[`data.Decode()`](https://pkg.go.dev/github.com/simpleiot/simpleiot/data#Decode)
function can be used to decode an array of points into the above type. The
[`data.Merge()`](https://pkg.go.dev/github.com/simpleiot/simpleiot/data#MergePoints)
function can be used to update an existing struct from a new point.

## Node Topology changes

Nodes can exist in multiple locations in the tree. This allows us to do things
like include a user in multiple groups.

### Add

Node additions are detected in real-time by sending the points for the new node
as well as points for the edge node that adds the node to the tree.

### Copy

Node copies are are similar to add, but only the edge points are sent.

### Delete

Node deletions are recorded by setting a tombstone point in the edge above the
node to true. If a node is deleted, this information needs to be recorded,
otherwise the synchronization process will simply re-create the deleted node if
it exists on another instance.

### Move

Move is just a combination of Copy and Delete.

If the any real-time data is lost in any of the above operations, the catch up
synchronization will propagate any node changes.

## Tracking who made changes

The `Point` type has an `Origin` field that is used to track who generated this
point. If the node that owned the point generated the point, then Origin can be
left blank -- this saves data bandwidth -- especially for sensor data which is
generated by the client managing the node. There are several reasons for the
`Origin` field:

- track who made changes for auditing and debugging purposes. If a rule or some
  process other than the owning node modifies a point, the Origin should always
  be populated. Tests that generate points should generally set the origin to
  "test".
- eliminate echos where a client may be subscribed to a subject as well as
  publish to the same subject. With the Origin field, the client can determine
  if it was the author of a point it receives, and if so simply drop it. See
  [client documentation](client.md#message-echo) for more discussion of the echo
  topic.

## Converting Nodes to other data structures

Nodes and Points are convenient for storage and synchronization, but cumbersome
to work with in application code that uses the data, so we typically convert
them to another data structure.
[`data.Decode`](https://pkg.go.dev/github.com/simpleiot/simpleiot/data#Decode),
[`data.Encode`](https://pkg.go.dev/github.com/simpleiot/simpleiot/data#Encode),
and
[`data.MergePoints`](https://pkg.go.dev/github.com/simpleiot/simpleiot/data#MergePoints)
can be used to convert Node data structures to your own custom `struct`, much
like the Go `json` package.

## Evolvability

One important consideration in data design is the can the system be easily
changed. With a distributed system, you may have different versions of the
software running at the same time using the same data. One version may use/store
additional information that the other does not. In this case, it is very
important that the other version does not delete this data, as could easily
happen if you decode data into a type, and then re-encode and store it.

With the Node/Point system, we don't have to worry about this issue because
Nodes are only updated by sending Points. It is not possible to delete a Node
Point. So it one version writes a Point the other is not using, it will be
transferred, stored, synchronized, etc and simply ignored by version that don't
use this point. This is another case where SIOT solves a hard problem that
typically requires quite a bit of care and effort.
