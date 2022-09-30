# Simple IoT Store

We currently use SQLite to implement the persistent store for Simple IoT. Each
instance (cloud, edge, etc.) has its own store that must be synchronized with
replicas of the data located in other instances.

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

The hash of a node is calculated by computing the CRC-32 of each point's `Time`,
`Text`, and `Value` fields, and then XOR'ing these CRC values. The hash of child
nodes is also XOR'd. If a point or child hash changes, the hash can be updated
by XOR'ing the old value (which backs out the old value) and the new value with
the current hash. This allows the hash to be updated incrementally without
requiring a bunch of DB reads every time something changes.

See
[hash_test.go](https://github.com/simpleiot/simpleiot/blob/master/store/hash_test.go)
for a test of the XOR concept.

### Updating the Node Hash

When a point is received by the store, the store:

- starts a transaction
  - loads the edge(s)
  - loads the current point if it exists, and backs the CRC-32 out of the
    current hash (XOR)
  - computes the CRC of the new point and XOR's it with the edge hash
  - updates the point
  - updates the edge
  - ends the transaction
- starts a transaction
  - finds upstream edges of the current edge
  - XOR out old hash value
  - XOR in new hash value
  - end transaction
- repeat for all edges up to root edge

It should again be emphasized that repetitive or high rate points should not be
included in the hash because they will be sent again soon -- we do not need the
hash to ensure they get synchronized. The hash should only include points that
change at slow rates (user changes, etc). Anything machine generated should be
repeated -- even if only every 10m.

## Store Synchronization

### The moving target problem

As long as the connection between instances is solid, they will stay
synchronized as each instance will receive all points it is interested in.
Therefore, verifying synchronization by comparing Node hashes is a backup
mechanism -- that allows us to see what changed when disconnected. The root
hashes changes every time anything in the system changes. This is very useful in
that you only need to compare one value to ensure your entire config is
synchronized, but it is also a disadvantage in that the top level hash is
changing more often so you are trying to compare two moving targets. This is not
a problem if things are changing slow enough that it does not matter if they are
changing. However, this also limits the data rates to which we can scale.

Some systems use a concept called Merkle clocks, where events are stored in a
Merle DAG and existing nodes in the DAG are immutable and new events are always
added as parents to existing events. An immutable DAG has an advantage in that
you can always work back in history, which never changes. The SIOT Node tree is
mutable by definition. Actual budget uses a similar concept in that it
[uses a Merkle Trie](https://github.com/actualbudget/actual/discussions/257) to
represent events in time and then prunes the tree as time goes on.

We could create a separate structure to sync all events (points), but that would
require a separate structure on the server for every downstream device and seems
overly complex.

Is it critical that we see all historical data? In an IoT system, there are
essentially two sets of date -- current state/config, and historical data. The
current state is most critical for most things, but historical data may be used
for some algorithms and viewed by users. The volume of data makes it impractical
to store all data in resource constrained edge systems. However, maybe it's a
mistake to separate these two as synchronizing all data might simplify the
system.

One way to handle the moving target problem is to store an array of previous
hashes for the device node in both instances -- perhaps for as long as the
synchronization interval. The downstream could then fetch the upstream hash
array and see if any of the entries match an entry in the downstream array. This
would help cover the case where there may be some time difference when things
get updated, but the history should be similar. If there is a hash in history
that matches, then we are probably OK.

Another approach would be to track metrics on how often the top level hash is
updating -- if it is too often, then perhaps the system needs tuned.

There could also be some type of stop-the-world lock where both systems stop
processing new nodes during the sync operation. However, if they are not in
sync, this probably won't help.
