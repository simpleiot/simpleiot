# Simple IoT Store

Simple IoT stores all application data in
[NATS JetStream](https://docs.nats.io/nats-concepts/jetstream), using the NATS
server that is already embedded in every SIOT instance. There is no separate
database: the same technology that moves messages between components also
persists them, retains their history, and (see [Data Synchronization](sync.md))
replicates them between instances.

[ADR-7](../adr/7-jetstream-store.md) records the full analysis behind this
design. Earlier versions of SIOT used SQLite; existing SQLite data can be
migrated with `siot export` / `siot import`.

## Why JetStream

A JetStream stream is an append-only, persistent log of messages with sequence
numbers and per-subject indexing. That shape matches IoT data unusually well:

- **Points are already messages.** SIOT components communicate by publishing
  points over NATS. Persisting them is a matter of capturing the same messages
  in a stream, not translating them into a second data model.
- **History is the natural byproduct.** A stream retains every point written to
  a subject, so time-series history is stored in the same place as current
  state, rather than requiring a separate time-series database.
- **Sequence numbers replace the hash tree.** Streams are ordered and
  sequence-tracked, so another instance can replicate one and know exactly what
  it has and has not seen. This is the foundation of the synchronization design.
- **Small and embedded.** JetStream runs inside the NATS server SIOT already
  ships, on cloud instances and small edge devices alike.

## Boundaries and streams

Streams are created per **boundary**, not per node. A boundary is a node that
represents a SIOT instance:

- the local instance's root node, and
- any device-type node, which corresponds to a (potentially synced) remote
  instance.

Every node is owned by the nearest boundary found walking up the tree through
undeleted edges. A node reachable from no boundary, or from more than one, is
owned by the instance root boundary. Boundaries align with the natural units of
synchronization and authorization: a device's subtree syncs as a unit, and
permissions are typically granted at device or group level.

Each (boundary, origin instance) pair gets one stream:

```
inst_<boundaryID>_<originID>
```

where `originID` is the root node ID of the instance that writes the stream. The
`inst` (instance) prefix keeps the word "node" reserved for nodes in the data
tree — both stream tokens identify instances, since a boundary is a node that
represents one. **Only the origin instance ever appends to its stream** — this
single-writer property is what makes synchronization simple and echo-free. A
standalone instance with root `R` has a single stream, `inst_R_R`, holding its
entire tree.

A hub `R` with a synced device `X` sees three streams:

| Stream     | Written by | Contains                                            |
| ---------- | ---------- | --------------------------------------------------- |
| `inst_R_R` | hub        | the hub's own tree, including the edge to X         |
| `inst_X_R` | hub        | configuration the hub writes into X's subtree       |
| `inst_X_X` | device     | everything the device writes (a replica on the hub) |

## Subjects

Two subject spaces are in play. **Wire subjects** are how points move between
components in real time — they are plain NATS, not stored:

| Subject                                      | Purpose                   |
| -------------------------------------------- | ------------------------- |
| `p.<nodeID>.<type>.<key>`                    | node point                |
| `ep.<nodeID>.<parentID>`                     | edge points (batched)     |
| `up.<upID>.<nodeID>.<type>.<key>`            | point fan-out up the tree |
| `up.<upID>.<nodeID>.<parentID>.<type>.<key>` | edge point fan-out        |

Listeners tell the two fan-out subjects apart by counting tokens, so a point
type or key may not contain a period. The store checks this on every point it
accepts — see `checkPoints` in `store/store.go` and
[the data reference](./data.md).

**Storage subjects** are what streams capture. They carry both routing tokens so
stream subject spaces never overlap:

| Subject                                                | Purpose     |
| ------------------------------------------------------ | ----------- |
| `inst.<boundaryID>.<originID>.<nodeID>.p.<type>.<key>` | node point  |
| `inst.<boundaryID>.<originID>.<parentID>.ep.<childID>` | edge points |

The stream `inst_<b>_<o>` captures `inst.<b>.<o>.>`. Edges are stored with the
**parent** node's boundary, so the edge attaching a device into a hub's tree
lives in the hub's stream — the device never needs it.

## Current state: merge of subject tips

The current value of a point is the **tip** (last message) of its storage
subject. Because a boundary can have streams from several origins (the device's
own data plus hub-written configuration), current state is the merge of tips
across all `inst_<boundaryID>_*` streams, under one rule:

1. The newest point timestamp wins (timestamps are embedded in the point, not
   taken from the stream).
2. Equal timestamps from different origins resolve to the lexically greater
   origin ID, so every instance converges on the same winner.
3. An identical (timestamp, origin) delivery is a no-op, which makes the merge
   idempotent when the same point arrives more than once.

The store holds this merged state in two in-memory caches — an edge cache (the
tree) and a point cache (current points) — populated by reading every stream's
subject tips at startup. The caches are the read path; queries never touch
JetStream. Writes check the cache tip first, append to the stream, then update
the cache, with a load-on-miss backstop.

## Writes, deletes, and moves

A local write routes to `inst_<owningBoundary>_<self>`. Deleting a node writes a
tombstone point on its parent edge — history is preserved and the delete can be
undone. Moving a node (or subtree) across boundaries republishes its subject
tips into the new boundary's stream, preserving original point timestamps, then
purges the old subjects; ownership follows the tree.

## Retention and durability

Streams keep the most recent **5000 messages per subject** by default. Because
the limit is per subject, current state — including rarely-written configuration
points — is always preserved, which time- or size-based retention could not
guarantee. The default is sized so that:

- data reported every 10 minutes keeps about a month of local history,
- configuration subjects, written a handful of times, are effectively unlimited,
  and
- disk use on unattended edge devices stays bounded (a 1-per-minute subject
  would otherwise grow by ~525k messages a year).

History is tiered by write rate: fast subjects wrap sooner locally, and
long-term history for them belongs in an external time-series database fed by
the Db client, which reads the streams gap-free.

`--storeMaxMsgsPerSubject` (or `SIOT_STORE_MAX_MSGS_PER_SUBJECT`) overrides the
default; `-1` means unlimited. Each instance applies its own policy to every
stream on its own disk — including replica streams, which the sync pumps create
bare and the store configures when it discovers them — so a hub and a device can
retain different amounts of the same data.

The store logs the policy it resolved when it starts, so the effective value is
visible without inspecting a stream:

```
STORE: retention: 5000 points per subject (default); current state is always preserved
STORE: retention: 20000 points per subject; current state is always preserved
STORE: retention: unlimited points per subject
```

The setting is otherwise invisible once an instance is running, particularly
when it comes from the environment rather than the command line. To read it back
from a running system instead, `nats stream info` reports the limit each stream
was given.

### Compression

Streams are compressed with
[S2](https://github.com/klauspost/compress/tree/master/s2) by default. Point
data compresses unusually well, because the same point type repeats in every
message, keys come from a small set, and timestamps march forward — the kind of
repetition a compressor is built for. The cost is a small amount of CPU on write
and read, far below what any SIOT write rate produces.

JetStream compresses a block when it seals it, not while it is the active block
being written, so the saving appears once a store outgrows its first block.
Measured on scraped Prometheus points:

| Messages | Uncompressed | S2      | Saving |
| -------- | ------------ | ------- | ------ |
| 20,000   | 6.7 MB       | 6.7 MB  | none   |
| 100,000  | 33.4 MB      | 11.7 MB | 65%    |

The sealed blocks themselves compress to roughly a sixth of their size; the
uncompressed active block is what holds the whole-store figure short of that. A
small store therefore gives up nothing and gains nothing, and compression starts
paying exactly where disk begins to matter.

`--storeCompression` (or `SIOT_STORE_COMPRESSION`) accepts `s2` or `none`.
Turning it on for an instance that already has data is safe: existing messages
stay readable and are recompressed as their blocks are rewritten. As with
retention, each instance applies its own setting to every stream on its own
disk, replica streams included, and the effective value appears in the startup
log:

```
STORE: compression: s2 (default)
```

### Durability

The JetStream file store fsyncs on a 2-minute interval by default.
`--storeSyncInterval` (or `SIOT_STORE_SYNC_INTERVAL`) accepts a Go duration to
shorten that window, or `always` to fsync every write, for edge devices with
unreliable power, at a write-throughput cost.

## Message and payload limits

Retention bounds how much history a subject keeps. A separate limit bounds how
many points a single node can hold, and it is worth knowing because exceeding it
fails in a way that looks unrelated to the node that caused it.

When a node is requested, the store encodes the node, its points, and — for a
subtree request — its children into **one** NATS message and publishes it as the
reply (`getNodes` in `store/store.go`). SIOT runs the NATS default `max_payload`
of 1 MB and does not raise it, so that reply has to fit in 1 MB.

Point sizes measured with `cmd/point-size` and against real data:

| Point                                                             | Encoded size |
| ----------------------------------------------------------------- | ------------ |
| Typical reading (short type, no key)                              | ~34 bytes    |
| Long type with a multi-label key, as a Prometheus scrape produces | ~100 bytes   |

A node holding 10,000 scraped points therefore encodes to about 1 MB on its own.
`data.DecodePoints` independently refuses an array of more than 10,000 points.

### What exceeding it looks like

The publish is rejected, nothing is sent, and the requester waits out its
timeout:

```
NATS: Error publishing response to node request: nats: maximum payload exceeded
Error getting nodes for user: error getting children: nats: timeout
```

Because a reply carries a subtree rather than a single node, **every tree fetch
covering the node fails**, so the UI stops loading for that user entirely rather
than showing one broken node. The messages name neither the node nor the point
count, which is what makes this worth documenting.

Lowering whatever setting produced the points does not fix it. Points are
current state and persist until removed, so recovery means deleting the node —
or, if the UI cannot load, stopping SIOT, purging that node's subjects, and
restarting so the caches repopulate from the tips:

```sh
nats stream ls
nats stream purge <stream> --subject "inst.*.*.<nodeID>.p.>"
```

### Keeping nodes within it

A node with points in the hundreds is comfortable; one in the thousands deserves
thought. Clients that can generate points in bulk bound themselves:

- The [metrics client](../user/metrics.md) caps a Prometheus scrape at 3000
  series, about 350 KB, and reports a larger configured limit on the node rather
  than honoring it.
- The same client's `allProcesses` type is disabled in the UI, since a modern
  system has thousands of processes.

A source too large for one node is better split across several. Nodes are
inexpensive, and a limit or a failure then affects only the part of the source
it belongs to.

## Instance metadata

A small `META` key/value bucket (also JetStream) holds the instance's root node
ID and JWT signing key.
