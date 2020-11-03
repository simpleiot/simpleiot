---
id: development
title: Development
sidebar_label: Development
---

## Code Organization

Currently, there are a lot of subdirectories. One reason for this is to limit
the size of application binaries when building edge/embedded Linux binaries. In
some use cases, we want to deploy app updates over cellular networks, therefore
we want to keep packages as small as possible. For instance, if we put the
`natsserver` stuff in the `nats` package, then app binaries grow a couple MB,
even if you don't start a nats server. It is not clear yet what Go does for dead
code elimination, but at this point, it seems referencing a package increases
the binary size, even if you don't use anything in it. (Clarification welcome!)

For edge applications on Embedded Linux, we'd eventually like to get rid of
net/http, since we can do all network communications over NATS. We're not there
yet, but be careful about pulling in dependencies that require net/http into the
nats package, and other low level packages intended for use on devices.

## Coding Standards

Please run `siot_test` from `envsetup.sh` before submitting pull requests. All
code should be formatted and linted before committing.

Please configure your editor to run code formatters:

- Go: `goimports`
- Elm: `elm-format`
- Markdown: `prettier` (note, there is a `.prettierrc` in this project that
  configures prettier to wrap markdown to 80 characters. Whether to wrap
  markdown or not is debatable, as wrapping can make diff's harder to read, but
  Markdown is much more pleasant to read in an editor if it is wrapped. Since
  more people will be reading documentation than reviewing, lets optimize for
  the reading in all scenarios -- editor, Github, and generated docs)

* [Environment Variables](environment-variables.md)

## Running unit tests

There are not a lot of unit tests in the project yet, but below are some
examples of running tests:

- test everything: `go test ./...`
- test only db directory: `go test ./db`

The leading `./` is important, otherwise Go things you are giving it a package
name, not a directory. The `...` tells Go to recursively test all subdirs.

## Device communication and messaging

This project is moving toward using NATS.io for messaging. Some reasons:

- allows us to [push realtime data](https://youtu.be/REZ6DKvRVv0) to an edge
  device behind a NAT, on cellular network, etc -- no public IP address, VPN,
  etc required.
- is more efficient than HTTP as it shares one persistent TCP connection for all
  messages. The overhead and architecture is similar to MQTT, which is proven to
  be a good IoT solution. It may also use less resources than something like
  observing resources is CoAP systems, where each observation requires a
  separate persistent connection.
- can scale out with multiple servers to provide redundancy or more capacity.
- is written in Go, so possible to embed the server to make deployments simpler
  for small systems. Also, Go services are easy to manage as there are no
  dependencies.
- focus on simplicity -- values fit this project.
- good security model.

For systems that only need to send one value several times a day, CoAP is
probably a better solution than NATS. Initially we are focusing on systems that
send more data -- perhaps 5-30MB/month. There is no reason we can't support CoAP
as well in the future.

## Flexible data structures

As we work on IoT systems, data structures (types) tend to emerge. Common data
structures allow us to develop common algorithms and mechanism to process data.
Instead of defining a new data type for each type of sensor, define one type
that will work with all sensors. Then it is trivial to support new sensors and
applications.

These are currently defined in the `data` directory for Go code, and
`frontend/src/Data` directory for Elm code. The fundamental data structures for
the system are [`Nodes`](../data/node.go), [`Points`](../data/point.go), and
[`Edges`](../data/edge.go). A `Device` can have one or more `Points`. A `Point`
can represent a sensor value, or a configuration parameter for the device. With
sensor values and configuration represented as `Points`, it becomes easy to use
both sensor data and configuration in rule or equations because the mechanism to
use both is the same. Additionally, if all `Point` changes are recorded in a
time series database (for instance Influxdb), you automatically have a record of
all configuration changes for a `node`.

Treating most data as `Points` also has another benefit in that we can easily
simulate a device -- simply provide a UI or write a program to modify any point
and we can shift from working on real data to simulating scenarios we want to
test.

`Edges` are used to describe the relationships between nodes as a graph. `Nodes`
can have parents or children and thus be represented in a hierarchy. To add
structure to the system, you simply add nested `Nodes`. The `Node` hierarchy can
represent the physical structure of the system, or it could also contain virtual
`Nodes`. These virtual nodes could contain logic to process data from sensors.
Several examples of virtual nodes:

- a pump `Node` that converts motor current readings into pump events.
- implement moving averages, scaling, etc on sensor data.
- combine data from multiple sensors
- implement custom logic for a particular application
- a component in an edge device such as a cellular modem

Being able to arranged nodes in an arbitrary hierarchy also opens up some
interesting possibilities such as creating virtual nodes that have a number of
children that are collecting data. The parent virtual nodes could have rules or
logic that operate off data from child nodes. In this case, the virtual parent
nodes might be a town or city, and the child nodes are physical gateways
collecting data.

Eventually, it seems logical to have a scripting language where formulas can be
written to operate on any `Device:Point` data. While there are likely many other
systems that have this type of functionality (for instance Node-RED), the focus
of Simple IoT is not for one-off systems where every device is manually
configured, but rather for a system that can be programmed or configured once,
and then scales with no manual effort as additional devices and users are added.

As this is a distributed system where devices may be created on any number of
connected systems, node IDs need to be unique. A unique serial number or UUID is
recommended.

When a `Point` changes, all `Points` that depend on this data need to be
updated. One _simple_ way to handle this is notify all parent nodes in the
hierarchy and re-run any rules or computed values on these nodes. This keeps
point/node dependency management simple -- nodes can only depend on data from
child nodes. No extra data is required to track relationships. If nodes need to
share data, do that through a shared parent node.

## Configuration and Synchronization

Typically, configuration is modified through a user interface. As mentioned
above, the configuration of a `Node` will be stored as `Points`. Typically the
UI for a node will present fields for the needed configuration based on the
`Node:Type`.

As Simple IoT is evolving into a distributed system, the question of
configuration and the synchronization of config needs to be considered. Both
client (edge) and server (cloud) siot instances can make configuration changes.
An example might be a edge device that has a local LCD/keypad that allows the
user to make configuration changes in the field. Both client and server will
need to communicate changes to the other instance and know if they are in sync.
`Point` changes can be communicated as they are changed via NATS which both the
cloud and device instance can listen to. If one of the systems is not connected,
they will miss the `Point` change. For sensor data it is not huge deal if a
sensor reading is lost -- at some point in the future another sensor reading
will be sent. But for configuration data, it may never be changed again and it
is import that any configuration changes be synchronized. When a system comes
online (say an edge device), it requests the `Device:Points` data for all
devices it is interested in. All systems then respond with their `Point` data.
If the timestamp of a `Point` coming in is newer than the one stored locally, it
is then processed on that system. This ensures the latest information for all
`Points` is propagated (even sensor data) and should cover most scenarios even
where two people edit two different configuration parameters on the same device
on two different systems and these systems later reconnect. It may not be
appropriate for cloud systems to request `Points` for all connected `Devices` if
the service in the cloud is simply restarted (say during a new version
deployment as downtime is minimal) as then it would have a flood of `Points` to
deal with. On the other hand, NATS can handle millions of messages per second,
so with small medium/scale systems (1000's of IoT devices), it seems this is
probably not a big deal. It may also make sense to occasionally request
`Device:Points` synchronization -- say once per hour. Perhaps the `Device` data
structure could have a `LastSychronized` field and the server could initiate a
synchronization at some interval.

Synchronization is managed using the node `Hash` field. The `Hash` field is a
hash of the node point timestamps, and child node `Hash` fields. Comparing the
node `Hash` field allows us to detect node differences.

Any time a node Point is modified, its `Hash` field is updated, and the `Hash`
field in parents, grand-parents, etc are also computed. This may seem like a lot
of overhead, but if the database is local, and the graph is reasonably
constructed, then each update might require reading a dozen or so nodes. An
indexed read in Genji is orders of magnitude faster than a write (at least for
Bolt), so this overhead should be minimal. Again, we are optimizing for
small/mid size IoT systems. If a point update requires 50ms, the system can
handle 20 points/sec. If the average device sends 0.05pt/sec, then we can handle
400 devices. Switching from Bolt to Badger will likely improve this by an order
of magnitude, so that puts us well into the 1000's of devices.

There are two things that need to be synchronized:

1. Node point changes (this happens when sensor data changes).
1. Node topology changes (includes adding/deleting nodes).

There are two synchronization cases:

1. _Catch up_ -- This is the case where one system starts after another and must
   catchup to any changes.
1. _Run time_ -- This is the case where two systems have "caught up" and need to
   stay synchronized.

### Catch up synchronization

So for every node modification, the root node of the graph is updated. To
synchronize the graph, you run the following steps:

1. Start at root node.
1. Does the `Hash` field match?
1. If not push the node into a queue, fetch node's children and compare `Hash`
   fields. For nodes where `Hash` does not match, continue fetching children
   until you reach a point where all children match.
1. Once you are at the bottom of the graph, walk back up the graph by popping a
   node ID off the queue and synchronize that node's data by comparing `Hash`
   fields.

### Run Time Point Synchronization

Point changes are handled by sending points to NATS topic for a node. There are
two primary instance types:

1. Cloud: will subscribe to point changes on all nodes.
1. Edge: will subscribe to point changes only for the nodes that exist on the
   instance.

To accomplish the above, an edge instance will subscribe and publish point
changes at its root node. This ensures it will only receive messages for the
root node and lower. A cloud instance will publish point changes to every node
in the parent/grandparent/etc hierarchy, and will subscribe to changes at the
root node. NATS is very efficient, so publishing to many nodes is not a problem.
This will require each instance to be configured as either a cloud or edge
instance so the instance knows whether to publish points to every node in the
hierarchy or only the root (to save bandwidth on cellular systems). With Point
Synchronization, each instance is responsible for updating the node data in its
local database.

## Extendible architecture

Any `siot` app can function as a standalone, client, server or both. As an
example, `siot` can function both as an edge (client) and cloud apps (server).

- full client: full siot node that initiates and maintains connection with
  another siot instance on a server. Can be behind a firewall, NAT, etc. May
  eventually use
  [NATS leaf node](https://docs.nats.io/nats-server/configuration/leafnodes)
  functionality for this.
- server: needs to be on a network that is accessible by clients

We also need the concept of a lean client where an effort is made to minimize
the application size to facilitate updates over IoT cellular networks where data
is expensive.

## Frontend architecture

Much of the frontend architecture is already defined by the Elm architecture.
However, we still have to decide how data flows between various modules in the
frontend. If possible, we'd like to keep the UI
[optimistic](https://blog.meteor.com/optimistic-ui-with-meteor-67b5a78c3fcf) if
possible. Thoughts on how to accomplish this:

- single data model at top level
- modifications to the backend database are sent to the top level, the model is
  modified first, and then a request is sent to the backend to modify the
  database. This ensures the value does not flash or revert to old value while
  the backend request is being made.

## Backend architecture

Currently the backend architecture is very simple as everything is driven by
REST APIs. Eventually, we'll need to have goroutines running collecting data,
running rules, etc. and figure out how to flow data through the system.
