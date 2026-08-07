# Simple IoT Store

Simple IoT stores all application data in [NATS
JetStream](https://docs.nats.io/nats-concepts/jetstream), using the NATS
server that is already embedded in every SIOT instance. There is no
separate database: the same technology that moves messages between
components also persists them, retains their history, and (see [Data
Synchronization](sync.md)) replicates them between instances.

[ADR-7](../adr/7-jetstream-store.md) records the full analysis behind
this design. Earlier versions of SIOT used SQLite; existing SQLite data
can be migrated with `siot export` / `siot import`.

## Why JetStream

A JetStream stream is an append-only, persistent log of messages with
sequence numbers and per-subject indexing. That shape matches IoT data
unusually well:

- **Points are already messages.** SIOT components communicate by
  publishing points over NATS. Persisting them is a matter of capturing
  the same messages in a stream, not translating them into a second
  data model.
- **History is the natural byproduct.** A stream retains every point
  written to a subject, so time-series history is stored in the same
  place as current state, rather than requiring a separate time-series
  database.
- **Sequence numbers replace the hash tree.** Streams are ordered and
  sequence-tracked, so another instance can replicate one and know
  exactly what it has and has not seen. This is the foundation of the
  synchronization design.
- **Small and embedded.** JetStream runs inside the NATS server SIOT
  already ships, on cloud instances and small edge devices alike.

## Boundaries and streams

Streams are created per **boundary**, not per node. A boundary is a
node that represents a SIOT instance:

- the local instance's root node, and
- any device-type node, which corresponds to a (potentially synced)
  remote instance.

Every node is owned by the nearest boundary found walking up the tree
through undeleted edges. A node reachable from no boundary, or from
more than one, is owned by the instance root boundary. Boundaries align
with the natural units of synchronization and authorization: a device's
subtree syncs as a unit, and permissions are typically granted at
device or group level.

Each (boundary, origin instance) pair gets one stream:

```
inst-<boundaryID>-<originID>
```

where `originID` is the root node ID of the instance that writes the
stream. The `inst` (instance) prefix keeps the word "node" reserved for
nodes in the data tree — both stream tokens identify instances, since a
boundary is a node that represents one. **Only the origin instance ever
appends to its stream** — this single-writer property is what makes
synchronization simple and echo-free. A standalone instance with root
`R` has a single stream, `inst-R-R`, holding its entire tree.

A hub `R` with a synced device `X` sees three streams:

| Stream       | Written by | Contains                                     |
| ------------ | ---------- | -------------------------------------------- |
| `inst-R-R`   | hub        | the hub's own tree, including the edge to X  |
| `inst-X-R`   | hub        | configuration the hub writes into X's subtree |
| `inst-X-X`   | device     | everything the device writes (a replica on the hub) |

## Subjects

Two subject spaces are in play. **Wire subjects** are how points move
between components in real time — they are plain NATS, not stored:

| Subject                                   | Purpose                    |
| ----------------------------------------- | -------------------------- |
| `p.<nodeID>.<type>.<key>`                 | node point                 |
| `ep.<nodeID>.<parentID>`                  | edge points (batched)      |
| `up.<upID>.<nodeID>.<type>.<key>`         | point fan-out up the tree  |

**Storage subjects** are what streams capture. They carry both routing
tokens so stream subject spaces never overlap:

| Subject                                             | Purpose     |
| --------------------------------------------------- | ----------- |
| `inst.<boundaryID>.<originID>.<nodeID>.p.<type>.<key>` | node point  |
| `inst.<boundaryID>.<originID>.<parentID>.ep.<childID>` | edge points |

The stream `inst-<b>-<o>` captures `inst.<b>.<o>.>`. Edges are stored
with the **parent** node's boundary, so the edge attaching a device
into a hub's tree lives in the hub's stream — the device never needs
it.

## Current state: merge of subject tips

The current value of a point is the **tip** (last message) of its
storage subject. Because a boundary can have streams from several
origins (the device's own data plus hub-written configuration), current
state is the merge of tips across all `inst-<boundaryID>-*` streams,
under one rule:

1. The newest point timestamp wins (timestamps are embedded in the
   point, not taken from the stream).
2. Equal timestamps from different origins resolve to the lexically
   greater origin ID, so every instance converges on the same winner.
3. An identical (timestamp, origin) delivery is a no-op, which makes
   the merge idempotent when the same point arrives more than once.

The store holds this merged state in two in-memory caches — an edge
cache (the tree) and a point cache (current points) — populated by
reading every stream's subject tips at startup. The caches are the read
path; queries never touch JetStream. Writes check the cache tip first,
append to the stream, then update the cache, with a load-on-miss
backstop.

## Writes, deletes, and moves

A local write routes to `inst-<owningBoundary>-<self>`. Deleting a node
writes a tombstone point on its parent edge — history is preserved and
the delete can be undone. Moving a node (or subtree) across boundaries
republishes its subject tips into the new boundary's stream, preserving
original point timestamps, then purges the old subjects; ownership
follows the tree.

## Retention and durability

Streams keep the most recent **5000 messages per subject** by default.
Because the limit is per subject, current state — including
rarely-written configuration points — is always preserved, which time-
or size-based retention could not guarantee. The default is sized so
that:

- data reported every 10 minutes keeps about a month of local history,
- configuration subjects, written a handful of times, are effectively
  unlimited, and
- disk use on unattended edge devices stays bounded (a 1-per-minute
  subject would otherwise grow by ~525k messages a year).

History is tiered by write rate: fast subjects wrap sooner locally,
and long-term history for them belongs in an external time-series
database fed by the Db client, which reads the streams gap-free.

`--storeMaxMsgsPerSubject` (or `SIOT_STORE_MAX_MSGS_PER_SUBJECT`)
overrides the default; `-1` means unlimited. Each instance applies its
own policy to every stream on its own disk — including replica streams,
which the sync pumps create bare and the store configures when it
discovers them — so a hub and a device can retain different amounts of
the same data.

The JetStream file store fsyncs on a 2-minute interval by default.
`--storeSyncInterval` (or `SIOT_STORE_SYNC_INTERVAL`) accepts a Go
duration to shorten that window, or `always` to fsync every write, for
edge devices with unreliable power, at a write-throughput cost.

## Instance metadata

A small `META` key/value bucket (also JetStream) holds the instance's
root node ID and JWT signing key.
