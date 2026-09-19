# Plan: JetStream Sync Follow-ups

**Branch:** `cbrake/master` **Branched from:** `a85c26f6`

## Context

The [Stage 3 sync plan](2026-08-06-stage3-jetstream-sync.md) is complete: two
instances replicate their boundary-origin streams in both directions, survive
disconnection, and converge, and per-device credentials
([plan](2026-08-20-per-device-credentials.md)) scope each device to its own
streams. The items below are what it left open, collected here so the old plan
reads as a record of what shipped. The
[remaining work](../docs/adr/7-jetstream-store.md#remaining-work) section of
ADR-7 points at this plan.

Items 1 and 2 extend where sync works; items 4 to 7 make it observable. Each
stands alone.

## Checklist

### 1. Replicate nested device boundaries

- [ ] Sync a device that sits beneath another device's boundary.

**Problem:** the sync client pushes and pulls only the streams of its own root
boundary (`client/sync.go`, `scanPulls`). A device X under device Y on the hub
has its own boundary; nothing replicates `inst_X_*` through Y.

**Change:** have the sync client discover the device boundaries in its tree
(`OwningBoundary` already resolves nested boundaries, see
`TestOwningBoundaryNested`) and run a push and a pull scan per boundary, and
extend `devicePermissions` in `server/auth.go` to grant the nested boundaries.

**Verify:** a point written on a nested device reaches the hub, and a write on
the hub reaches the nested device.

### 2. Test multi-hop chaining

- [ ] Add a three-instance test: device → intermediate → hub.

**Problem:** each hop replicates independently, so chaining is expected to work,
but no test covers it.

**Change:** a test in `client/sync_test.go` with three instances, covering
points and node create/delete in both directions and an intermediate restart.

**Verify:** the test passes; any defect it finds gets its own item.

### 3. Move a node between boundaries

- [ ] Support moving a node from one boundary to another.

**Problem:** a node's point subjects live in its owning boundary's streams.
Moving it into another device's boundary leaves its tips in the old streams, so
the new device never sees them.

**Change:** on a move that changes the owning boundary, republish the node's
subject tips (and its subtree's) into the new boundary's stream and purge the
old subjects.

**Verify:** a node moved from the hub root into a device boundary arrives on the
device with its current values; moving it back works the same way.

### 4. Per-boundary retention overrides

- [ ] Let a hub keep deeper history for selected devices.

**Problem:** retention defaults to 5000 messages per subject for every stream,
replicas included. There is no way to keep more for one device.

**Change:** a point on the device node (or the sync node) read at the
`maxMsgsForStream` resolution point in the store, with the server option as the
default.

**Verify:** a replica stream for a device with the override carries the larger
limit; others keep the default.

### 5. Sync status points

- [ ] Report per-replica lag and last-delivered time on the sync node.

**Problem:** `SyncCount` counts replication sessions, which says little about
whether data is current.

**Change:** each pump reports its pending count and the time of its last
delivered message as points on the sync node, keyed by stream. Replace
`SyncCount` and `SyncCountReset` if nothing else needs them.

**Verify:** lag rises while the upstream is unreachable and returns to zero
after catch-up.

### 6. Frontend sync status

- [ ] Show sync lag and last-delivered time in the sync node UI.

Depends on item 5.

### 7. History sink gaps

- [ ] Store edge points in the Db client and surface sink lag.

**Problem:** the Db client's durable consumer filters out edge points, so they
are not stored, and its lag is not visible.

**Change:** widen the consumer filter to edge points and report consumer lag as
a point on the Db node.

**Verify:** an edge change appears in the database; lag is reported while the
database is unreachable.

### 8. Revisit JetStream sourcing over leaf connections

- [ ] Replace durable-consumer replication with sourcing once instance identity
      can drive server configuration.

**Problem:** sourcing across a leaf connection works
(`store/leafnode_spike_test.go`), but JetStream domains are static server
configuration and an instance learns its root ID only after the store starts.
Chained sourcing is also unverified.

**Change:** persist the instance identity before the NATS server starts, then
evaluate sourcing against the current pumps on reconnect catch-up, bandwidth,
and multi-hop behavior. Adopt it only if it is simpler or cheaper on constrained
links.

**Verify:** the evaluation and decision are recorded in ADR-7.
