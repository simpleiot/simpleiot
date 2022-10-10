# Simple IoT Store

We currently use SQLite to implement the persistent store for Simple IoT. Each
instance (cloud, edge, etc.) has its own store that must be synchronized with
replicas of the data located in other instances.

## Reasons for using SQLite

We have evaluated BoltDB, Genji, and various other Go key/value stores in the
past and settled on SQLite for the following reasons:

- **Reliability**: SQLite is very well tested and
  [handles things](https://www.sqlite.org/transactional.html) like program/OS
  crashes, power failures, etc. It is important that the configuration for a
  system never become corrupt to the point where it won't load.
- **Stable file format**: Dealing with file format changes is not something we
  want to deal with when we have 100's of systems in the field. A SQLite file is
  very portable across time and between systems.
- **Pure Go**: There is now a
  [pure Go version](https://pkg.go.dev/modernc.org/sqlite) of SQLite. If more
  performance is needed or smaller binary size, the native version of SQLite can
  still be used.
- **The relational model**: it seems to make sense to store points and nodes in
  separate tables. This allows us to update points more quickly as it is a
  separate line in the DB. It also seems like flat data structures are generally
  a good thing versus deeply nested objects.
- **Fast**: SQLite does read caching, and other things that make it quite fast.
- **Lots of innovation around SQLite**:
  [LiteFS](https://github.com/superfly/litefs),
  [Litestream](https://litestream.io/), etc.
- **Multi-process**: SQLite
  [supports multiple processes](https://www.sqlite.org/faq.html#q5). While we
  don't really need this for core functionality, it is very handy for debugging,
  and there may be instances where you need multiple applications in your stack.

## Node hash

The edge `Hash` field is a hash of:

- edge and node point timestamps except for repetitive or high rate sample
  points.
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
prioritize performance and efficiency over hash strength.

Two desirable properties of the hash algorithm include:

- **commutability**: the ability to process elements in any order and get the
  same answer.
- **incremental**: the ability to incrementally add values to the hash without
  recomputing the entire array of inputs.

The hash of a node is calculated by computing the CRC-32 of all point
timestamps, and then XOR'ing these CRC values. The hash of child nodes is also
XOR'd. If a point or child hash changes, the hash can be updated by XOR'ing the
old value and the new value with the current hash. This allows the hash to be
updated incrementally without requiring a bunch of DB reads every time something
changes.

See
[hash_test.go](https://github.com/simpleiot/simpleiot/blob/master/store/hash_test.go)
for an test of this concept.
