# Plan: JetStream Synchronization (ADR-7 Stage 3) — DRAFT

**Status:** DRAFT — written before Stage 2 (boundary-origin streams) is
implemented, to think through the sync design far enough to catch anything that
should change in the
[Stage 2 store plan](2026-08-06-boundary-origin-streams.md). The phases here
are a sketch, not a commitment; they get firmed up after the Stage 2 work and
the Phase 7 spikes land. **ADR:** docs/adr/7-jetstream-store.md (Stage 3
decision).

## Context

### What sync does today

`client/sync.go` implements upstream sync over a second NATS *client*
connection to the remote instance:

- **Live path:** subscribes to all local `p.>` / `ep.>` traffic and forwards it
  to the remote; subscribes per-node to remote point subjects and forwards them
  back locally. Echo is avoided with `NoEcho` connections and by the
  subscription topology, not by design.
- **Catch-up path:** a periodic full-tree walk comparing node hashes
  (`syncNode`), pushing whole subtrees in either direction when they differ.
- **New-node discovery:** watches remote `up.<localRoot>.>` for tombstone
  transitions and replays subtrees.

### Why it must be rewritten, not ported

The Stage 2 store removes the hash tree (`store/store.go` no longer computes or
maintains hashes). `syncNode` compares `Hash` fields that are now always zero,
so the periodic catch-up path silently believes everything is in sync. On the
current branch, only the live forwarding path does anything. **Upstream sync is
effectively broken from the moment Stage 2 merges until Stage 3 lands.** This
constrains release planning: either Stage 3 follows Stage 2 before the next
release, or the release notes must state that upstream sync is nonfunctional.

### The Stage 3 model (from ADR-7)

Every stream is single-writer: instance R appends only to `node-*-R` streams.
Instances hold local **replicas** of remote-origin streams for the boundaries
they participate in, via JetStream sourcing (durable consumers as fallback).
Current state is merged at read in the edge/point caches. Echo is impossible
because no instance ever writes remote data into its own streams.

For a device X (root node ID X) synced to hub R:

| Stream       | Origin | On device X   | On hub R      |
| ------------ | ------ | ------------- | ------------- |
| `node-X-X`   | X      | owned, writes | replica       |
| `node-X-R`   | R      | replica       | owned, writes |
| `node-R-R`   | R      | not present   | owned, writes |

The edge attaching X into the hub's tree (group → X) lives in the hub's root
boundary stream (`node-R-R`, edges belong to the parent's boundary), so the
device never needs any hub-boundary stream.

## Design Walkthrough

### 1. Transport: how replication crosses instances

JetStream sourcing is a server-to-server feature. For the hub to source
`node-X-X` from the device (and vice versa), each side's JetStream API must be
reachable from the other, which in NATS terms means a **leaf node connection**
(possibly with distinct JS domains) rather than the plain client connection
sync uses today.

- The embedded NATS server currently has no leafnode listener or remote
  configuration (`server/server.go` configures client ports and auth token
  only). Stage 3 adds server options: a leafnode listen port on the hub side,
  and leafnode remote config (URI + credentials) on the device side, likely
  still driven by the Sync node's URI/AuthToken points.
- **Fallback** (if the Phase 7 spikes show sourcing across leaf connections is
  unsuitable — e.g. domain handling, reconnect behavior, or bandwidth): the
  sync client replicates manually with a durable consumer over the existing
  client connection and appends the messages to a local replica stream,
  preserving order and tracking sequence. The stream remains logically
  single-writer (it holds only origin X's messages); the local appender is a
  replication agent, not a second writer. This changes who creates the replica
  stream (the sync client, with direct publish allowed) versus sourcing (the
  store, with a `Sources` config and direct publish denied).

The decision is made after the Stage 2 Phase 7 spikes and recorded in ADR-7.

### 2. Replica lifecycle

- **Which replicas exist where:** device X replicates `node-X-*` except its own
  `node-X-X`. The hub replicates `node-X-X` for every synced device X — and in
  multi-hop topologies, `node-X-I` for any intermediate instance I that has
  written config for X. The set is dynamic: it changes when devices are
  adopted, when a new origin first writes to a boundary, and when devices are
  removed.
- **Discovery:** an instance must learn which `node-<boundary>-*` streams exist
  on the remote for boundaries it participates in. Simplest: query the remote's
  `$JS.API.STREAM.NAMES` with subject filter `node.<boundaryID>.>` on connect
  and periodically; re-check when a replica delivers an edge point referencing
  an origin with no known stream.
- **Creation/teardown:** replicas are created on first discovery and removed
  (or retained, configurable) when a device is detached. Stream names are
  identical on every instance that holds a copy, which keeps the store's
  "enumerate `node-<boundaryID>-*`" read path unchanged — a replica is just
  another stream matching the pattern.

### 3. Live path (latency) vs replica path (persistence)

Real-time delivery continues over core NATS `p.>` / `ep.>` wire subjects as
today; the replica covers the offline/startup gap and is the only path that
persists remote data.

This splits message handling on the receiving instance in a way the store must
support (see Implications below):

- A point arriving on the wire for a node owned by a **local-origin** stream is
  persisted (published to JetStream) and merged into the cache. This is the
  Stage 2 behavior.
- A point arriving on the wire that was **forwarded from a remote origin** must
  NOT be persisted (single-writer; the replica will deliver and persist it).
  It is merged into the cache and fanned out to `up.>` so rules, dbs, and the
  UI react in real time.
- A point delivered by a **replica stream** while connected is usually a
  duplicate of one already seen live — the cache merge must be an idempotent
  no-op. After an offline gap, replica delivery is the first sighting; how it
  reaches clients depends on the kind of client (next section).

#### Catch-up delivery: state clients vs. history sinks

Clients divide into two kinds with different catch-up needs:

- **State clients** (rules, protocol clients, the UI) care about current
  state. Replaying every intermediate point missed during an outage is
  actively harmful — a rule would fire actions on stale values in burst. After
  replica catch-up completes for a subject, the store emits one notification
  per *changed subject tip* (normal `p.>`/`ep.>`/`up.>` fan-out, carrying the
  origin header), so state converges without a replay storm.
- **History sinks** (Db/InfluxDB, VictoriaMetrics, or any external TSDB
  client) need every point, gap-free. They should not depend on core NATS
  wire delivery at all: each history sink owns a **durable JetStream
  consumer** on the relevant `node-*` streams (filtered to its subtree's
  subjects). Sequence tracking makes delivery resumable and gap-free across
  *both* sync outages and the sink's own downtime or instance restarts — a
  guarantee the current fire-and-forget `up.>` delivery never provided.
  Replica streams participate naturally: when the hub's replica of
  `node-X-X` catches up after a device was offline, the hub-side sink's
  consumer receives those messages in stream order with original embedded
  timestamps, so external TSDB backfill happens automatically.

A dedicated "catch-up subject" falls out of this for free: a push consumer
with a deliver subject is exactly that, and the pull-consumer API is the more
robust equivalent. The Db client rework to a durable consumer can ship as its
own phase; until then it keeps the same best-effort behavior it has today.

One retention caveat: `MaxMsgsPerSubject` guarantees tips, not unbounded
history. A sink offline longer than the per-subject window on a fast-changing
subject loses the intermediate points that aged out. Per-stream retention
(hub-long, device-short) is the knob, and sink lag should be visible in sync
status.

Distinguishing the three cases requires **origin attribution on wire
messages**: the flat `p.<nodeID>.<type>.<key>` subject cannot carry it, so
forwarded and re-broadcast messages carry an origin marker in a NATS message
header. Absence of the header means "originated here." Any defect here is
caught structurally: the worst case is a remote point written into a local
stream, which the single-writer stream naming makes visible and auditable,
not an invisible echo loop.

### 4. Merge determinism

With two origins writing the same subject tip (hub config write racing a
device-local write), every instance must converge to the same winner:

- Later timestamp wins (existing rule).
- Equal timestamps: deterministic tie-break on origin ID (e.g. lexically
  greater origin wins). Without this, hub and device caches can disagree
  forever while both believe they are in sync.
- Same timestamp and same origin: identical message (live + replica duplicate)
  — no-op.

### 5. Adoption / first connect

The boundary contract is: **the device node's ID on the hub equals the device
instance's root node ID.** Streams only line up if this holds.

- Device-initiated (matches today's flow): device connects, hub sees traffic
  for an unknown boundary X, creates a device node with ID X under a configured
  default group, writing the attachment edge into its own `node-R-R`. Policy
  point: auto-adopt vs. a pending-approval state.
- Hub-initiated (pre-provisioning): hub creates the device node (ID X) and
  writes config into `node-X-R` before the device ever connects. On first
  connect the device replicates `node-X-R` and comes up configured. This is a
  capability the old sync did not have and pairs well with the YAML
  provisioning work.
- The device's first connect replaces today's `sendNodesRemote` tree push: the
  hub simply replicates `node-X-X` from sequence 1 — full history, config, and
  structure arrive by the same mechanism as steady-state sync.

### 6. Sync status

Hash comparison and `SyncCount` disappear. Replication is sequence-tracked, so
status becomes observable directly: per-replica lag (source last sequence vs.
replica last sequence), last-delivered time, connected state — reported as
points on the Sync node so the UI and rules can use them.

### 7. Deletion and detach

- In-boundary deletes (tombstone edge points) replicate like any other point.
- Deleting the device *from the hub* tombstones the group→X edge in
  `node-R-R`, which the device does not replicate — the device does not learn
  it was detached, matching the AuthZ view (detach revokes the hub's interest;
  the device keeps operating standalone). Actual disconnect enforcement is
  AuthZ (revoking stream export/credentials), not data sync.
- Permanent removal on the hub: delete replica `node-X-X`, purge hub subjects
  for boundary X (`node.X.R.>`), remove the edge. Ordering and retention of
  history for departed devices is a policy decision to settle here.

### 8. Multi-hop

Intermediate instance I between device X and hub R: R sources `node-X-X` and
`node-X-I` through I (chained sourcing — I's replica is itself a source for
R's replica). The Phase 7 spikes must confirm chained sourcing behaves over two
leaf connections; the fallback replicator chains trivially since each hop is
independent.

### 9. AuthZ

- Writes: the wire subjects can't express "only nodes in boundary X" (flat
  UUIDs), so wire-level write enforcement stays coarse. The structural
  guarantee comes from the stream side: an instance can only *export* streams
  it is entitled to, and receivers never persist wire messages from remote
  origins anyway (section 3). Receiving instances should additionally validate
  that a message's claimed origin is entitled to the node's owning boundary
  before fan-out.
- Reads: per-stream JetStream API permissions — device X may replicate
  `node-X-*` and export only `node-X-X`. Dynamic grants (NATS auth callout) as
  devices are adopted/removed.
- Initial implementation can ship with today's shared-token trust model and
  tighten with auth callout as a follow-on phase; the stream layout is what
  makes the tightening possible later without redesign.

### 10. Retention asymmetry

A hub typically wants long history for every device; the device wants a small
local buffer. With sourcing, each side applies its own retention to its own
copy, so replica retention can exceed source retention. Consequence: retention
must be configurable **per stream** on each instance (hub-side long
`MaxMsgsPerSubject` on replicas, short on the device's own stream). If the
device prunes history the hub has not yet replicated (long offline gap),
sourcing resumes from what remains — the gap is real data loss at the hub and
should be surfaced in sync status.

## Open Questions

1. Sourcing behavior across leaf connections/domains — reconnect catch-up,
   chained sourcing, bandwidth on Cat-M-class links (Stage 2 Phase 7 spikes).
2. Leafnode vs. client connection: can the fallback replicator match sourcing
   closely enough that both share the replica-management code?
3. Adoption policy: auto-adopt unknown devices vs. pending approval.
4. Nodes mirrored across device boundaries (explicitly deferred from Stage 2).
5. Departed-device history retention policy on the hub.
6. Frontend changes: sync status UI (lag instead of SyncCount), and whether
   any UI flows assumed hash-based state.

## Implications for the Stage 2 Store Plan

Working through the design above surfaces these items that are cheaper to
accommodate in the Stage 2 rework than to retrofit:

1. **Thread origin through the write path (Phase 3).** The store's subject
   builders and handlers should take origin as a parameter rather than assuming
   "self", and the point/edge write path needs one branch point: *persist +
   merge* (local origin) vs. *merge + fan-out only* (remote origin, identified
   by an origin header on the wire message). Stage 2 always runs the first
   branch, but shaping the seam now means Stage 3 does not have to rework
   `handleNodePoints` / `handleEdgePoints`.
2. **Specify the merge tie-break now (Phase 3).** "Newest timestamp wins" is
   under-specified for Stage 3: equal timestamps from different origins need a
   deterministic winner (origin ID tie-break), and equal timestamp + same
   origin must be an idempotent no-op (live/replica duplicates). Defining and
   testing this in the Stage 2 cache merge costs little and locks in
   cross-instance convergence.
3. **Make cache loading incremental (Phase 3).** Replica streams appear at
   runtime (device adopted, new origin writes to a boundary). `loadNodePoints`
   / `loadEdgeCache` should be callable for a single stream and safe to run
   concurrently with live writes — not a startup-only path. Startup then
   becomes "for each stream, load one."
4. **`ensureStream` must anticipate replica configs (Phase 3).** Same stream
   name exists on multiple instances; a replica is created with a `Sources`
   config (or by the fallback replicator) rather than for direct publish.
   Separate "ensure my origin stream" from "ensure replica of remote stream"
   in the API even if Stage 2 only implements the first, and make the
   config-equality/update check tolerant of sourcing fields.
5. **Assert the boundary ID contract in Phase 2 tests.** `IsBoundary` /
   `OwningBoundary` must produce identical answers on hub and device given
   their respective trees, with the device node ID equal to the remote root
   ID. Add tests that model both perspectives of the same synced pair (hub
   tree with device node; device tree standalone) and assert the stream set
   each side computes lines up.
6. **Per-stream retention override moves up (Phase 5).** The Stage 2 plan
   defers per-boundary retention overrides; Stage 3's hub-long/device-short
   asymmetry makes them a requirement, and on *replica* streams specifically.
   Shape the Phase 5 config so a per-stream override slots in without
   reworking the option (e.g. resolve retention via a function of stream name,
   with the server option as default), even if the override itself ships in
   Stage 3.
7. **Notifications must be reachable from stream deliveries (Phase 3).**
   After offline catch-up, changed subject tips must reach state clients via
   the normal fan-out even though they arrived from a stream consumer rather
   than a wire subscription. Keep the store's cache-merge/fan-out logic
   callable from a non-wire entry point instead of embedding it in the NATS
   handler closures.
8. **Release sequencing.** Upstream sync is nonfunctional between the Stage 2
   merge and Stage 3 (hash comparison is inert). Decided 2026-08-06: the gap
   is acceptable; upstream sync returns, rebuilt, with Stage 3. Recorded in
   the Stage 2 plan.

*(Items 1–7 were folded into the Stage 2 plan on 2026-08-06.)*

## Implementation Status (2026-08-06)

The initial implementation landed on `feat/js-store-boundary-stream`
immediately after the Stage 2 rework:

- **Spike:** JetStream sourcing across a leaf connection with distinct
  domains verified, including restart catch-up
  (`store/leafnode_spike_test.go`). Chained sourcing and the
  consumer-create permission form remain open.
- **Transport:** durable-consumer replication over the existing
  upstream client connection (`client/sync.go` `runPump`); no leafnode
  or domain configuration needed. Sourcing remains the intended
  replacement once instance identity can drive server configuration.
- **Store:** origin-header ingress (merge and fan out, never persist),
  replica stream consumers with catch-up gating (tip-only broadcasts
  after the backlog drains), wire re-broadcast (`store/replica.go`).
  Instance-local "root" edges are never merged from replicas.
- **SyncClient:** rewritten around push/pull replication; adoption
  announcement on first connect; detach semantics honored (an upstream
  delete is final until the upstream restores the edge).
- **Tests:** two-instance flows in both directions (points, node
  create/delete), detach, offline catch-up
  (`client/sync_test.go`).

Remaining, in rough priority order:

1. Nested device boundaries (only the root boundary replicates today).
2. Multi-hop chaining test (expected to work: each hop is independent).
3. Per-replica retention overrides (replica streams are currently
   unlimited; the resolution point exists in `maxMsgsForStream`).
4. AuthZ tightening: shared token today; per-stream permissions via
   auth callout.
5. History sinks as durable stream consumers (Db client unchanged).
6. Sync status points (lag, last-delivered); `SyncCount` currently
   counts replication sessions.
7. Frontend sync status UI.

## Retrospective (2026-08-07)

The initial implementation followed the Stage 2 rework in the same
effort, and the seams folded into that plan were all used — the origin
parameter, per-stream cache loading, and the merge tie-break dropped in
without rework. Findings worth keeping:

1. **The sourcing spike passed, but domains blocked v1.** JetStream
   sourcing across a leaf connection with distinct domains works,
   including restart catch-up (`store/leafnode_spike_test.go`). It was
   still not used for the first implementation: JetStream domains are
   static server configuration, while a SIOT instance only learns its
   identity (root ID) after the store initializes. Durable-consumer
   replication over the existing client connection avoids the ordering
   problem entirely. Revisit sourcing once identity can drive server
   configuration (for example, persisted before the NATS server
   starts).

2. **Replicated streams contain instance-local state.** A device's
   origin stream carries its own `root` edge
   (`node.X.X.root.ep.X`); merging that on the hub would create a
   second root and break tree traversal and auth path checks. Edges
   with the virtual `root` parent are instance-local by definition and
   are never merged from a replica. Lesson: reason through what is
   actually in a stream being copied — single-instance tests cannot
   surface this class of defect.

3. **The client manager filters own-node points with an empty
   `Origin`.** A test (or any caller) changing a client's
   configuration must set `Origin` on the point, as UI edits do;
   otherwise the manager assumes the client wrote the point itself and
   drops it. This cost a debugging cycle on the offline catch-up test.

4. **The model deletes more than it adds.** Hash walks, tree pushes,
   and per-node remote subscriptions were replaced by two symmetric
   copy pumps, and echo prevention became structural (no instance
   writes remote data into its own streams) rather than a runtime
   discipline. The catch-up gating fell out of the ordered consumer's
   pending count.

## Phase Sketch (to be firmed up after Stage 2)

1. **Transport:** leafnode server options (listen + remotes), or commit to the
   consumer-based replicator, per spike results. Two-instance test harness
   (in-process servers connected leaf-to-hub) — this harness is the foundation
   for everything after it.
2. **Replica lifecycle:** discovery, creation, teardown of replica streams;
   incremental cache load on replica appearance.
3. **Store ingress:** origin headers, persist-vs-merge branch, catch-up
   re-broadcast (whatever was not already landed in Stage 2 per the
   implications above).
4. **SyncClient rewrite:** replace hash walk and tree push with replica
   management + live forwarding with origin tags; sync status points
   (lag, last-delivered, connected).
5. **Adoption:** device-initiated and hub-initiated flows; boundary ID
   contract enforcement.
6. **Detach/removal:** hub-side detach, purge, history retention policy.
7. **Multi-hop:** chained replication through an intermediate instance.
8. **AuthZ tightening:** per-stream permissions via auth callout (may ship
   separately).
9. **History sinks:** move the Db client (and document the pattern for
   external TSDB sinks like VictoriaMetrics) to durable stream consumers for
   gap-free delivery; surface sink lag in status.
10. **Tests/docs:** offline catch-up, convergence (equal-timestamp races),
   echo-absence (assert no instance's origin streams ever contain remote
   points), bandwidth sanity on constrained links; update ADR-7, user docs,
   changelog.

## Key Files (expected)

| File                            | Change                                              |
| ------------------------------- | --------------------------------------------------- |
| `client/sync.go`                | Rewritten around replica management                 |
| `store/jetstream.go`            | Replica ensure/load, origin-aware ingress           |
| `store/store.go`                | Fan-out callable from stream deliveries             |
| `server/server.go` / `args.go`  | Leafnode options, replica retention defaults        |
| `frontend/`                     | Sync status UI (lag instead of hash/SyncCount)      |
| `docs/adr/7-jetstream-store.md` | Transport decision, adoption policy, spike results  |
