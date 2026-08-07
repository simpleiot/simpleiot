# Plan: Boundary-Origin Streams (ADR-7 Stage 2 Revision)

**Branch:** `feat/js-store` (continues the existing branch)
**ADR:** docs/adr/7-jetstream-store.md (Stream Granularity and Synchronization
Model section, 2026-08-06 revision)

## Context

Stage 2 replaced SQLite with JetStream using one stream per node. A design
review on 2026-08-06 found two structural problems before the branch merged:

1. The Stage 3 sync sketch (merge-on-receive with durable consumers) creates
   an echo loop and gives up the single-writer property.
2. Per-node streams scale hub storage with total fleet node count rather than
   instance count (500 devices x 30 nodes = ~15,000 hub streams).

The review also found a point cache coherency bug and two smaller correctness
issues in the current implementation (previously tracked as Phases 7-9 of the
2026-03-17 plan, now superseded by this plan).

Because `feat/js-store` is unmerged and no release ships the JetStream store,
the layout is revised now, before Stage 3 builds on it. ADR-7 records the full
analysis and the adopted model; this plan implements it.

## Design Summary (ADR-7 is authoritative)

- **Boundary:** a node representing a SIOT instance: the instance root node,
  plus device-type nodes (which correspond to potentially synced remote
  instances). Every node is owned by the nearest boundary walking up the
  tree. Nodes reachable from more than one boundary are owned by the
  instance root boundary.
- **Streams:** one per (boundary, origin instance), named
  `node-<boundaryID>-<originID>`. Only the origin instance appends. A
  standalone instance therefore has a single stream,
  `node-<rootID>-<rootID>`, plus one per device-type node.
- **Storage subjects:** `node.<boundaryID>.<originID>.<nodeID>.p.<type>.<key>`
  for node points, `node.<boundaryID>.<originID>.<parentID>.ep.<childID>` for
  edge points (edges stored with the parent's boundary). Stream captures
  `node.<boundaryID>.<originID>.>`; subject spaces never overlap between
  streams. Core NATS wire subjects (`p.>`, `ep.>`) are unchanged.
- **Current state:** merge of subject tips across all `node-<boundaryID>-*`
  streams, newest timestamp wins. The edge and point caches hold the merged
  state; the caches are populated fully at startup and are the read path.
- **Retention:** `MaxMsgsPerSubject` per stream (per boundary), configurable,
  default no limit.

## Implementation Plan

### Phase 1: Layout-Independent Correctness Fixes

**Goal:** Fix the issues from the 2026-08-06 review that apply regardless of
stream layout.

- `store/store.go` `handleEdgePoints`: add the missing `return` after
  replying with a DB write error (it currently replies twice and still
  propagates failed points upstream), and correct the stale comment claiming
  upstream processing happens before the DB write.
- `store/jetstream.go` `edgePoints`: validate that new edges carry a
  `nodeType` point before publishing to JetStream, so the stream and edge
  cache cannot diverge.

**Verify:** `go test -race ./...` passes.

### Phase 2: Boundary Resolution

**Goal:** Deterministic mapping from node to owning boundary.

- Add to the edge cache: `IsBoundary(nodeID)` (instance root or device-type
  node) and `OwningBoundary(nodeID)` (walk up undeleted edges to the nearest
  boundary; fall back to the instance root when none or more than one is
  reachable).
- A boundary node's own points are owned by its own boundary. An edge is
  owned by the parent's boundary, so the edge from a group to a device root
  lives in the parent (hub-side) boundary stream, matching the sync model
  where a device's parent edge lives above the synced subtree.
- Unit tests: nested boundaries, multi-parent nodes, nodes above all device
  boundaries, tombstoned paths.

**Verify:** New unit tests pass.

### Phase 3: Store Layout Rewrite

**Goal:** Replace per-node streams with boundary-origin streams.

- `store/jetstream.go`:
  - `streamName(boundaryID, originID)` and subject builders carrying both
    routing tokens.
  - `ensureStream` keyed by (boundary, origin=self); local writes route via
    `OwningBoundary`.
  - `loadNodePoints` / `loadEdgeCache`: enumerate subjects across all
    `node-<boundaryID>-*` streams, merge tips with newest-timestamp-wins.
  - **Mandatory point cache pre-population at startup** (this fixes the
    cache poisoning bug: the previous implementation seeded the cache from
    the first write after restart and then trusted the partial entry). As a
    backstop, `nodePoints` loads current points from JetStream on a cache
    miss before adding new points.
  - `reset`: delete all `node-*` streams, purge META, re-init.
  - Node deletion remains tombstone-based; add subject purge
    (`node.<b>.<o>.<nodeID>.>`) as part of permanent removal paths.
- `store/jetstream_test.go`: migrate existing tests; add a restart test that
  re-opens the store against existing streams, writes a point, then reads
  the node back and confirms config points are intact.

**Verify:** All store tests pass with `go test -race`. Manual `siot_run`:
UI loads, nodes persist across restart.

### Phase 4: Cross-Boundary Node Moves

**Goal:** Handle re-parenting a node (or subtree) into a different boundary.

- In `edgePoints`, detect when a new or undeleted edge changes a node's
  owning boundary. For the node and its owned descendants: republish current
  subject tips into the new boundary stream preserving original point
  timestamps, then purge the old subjects.
- Tests: move a node between two device boundaries, move a subtree from the
  root boundary into a device boundary, verify history tips and edge cache
  stay consistent.

**Verify:** Move tests pass; `go test -race ./...`.

### Phase 5: Retention and Durability

**Goal:** Close the retention and durability gaps carried from the original
plan.

- Configurable `MaxMsgsPerSubject` applied at stream creation/update (server
  option; per-boundary override can come later as a node point). Document
  whether changing it retroactively updates existing streams.
- Decide and document the JetStream `SyncInterval` posture for power-loss
  durability on edge devices (default is a 2-minute fsync interval); expose
  a server option if warranted. Record the decision in ADR-7.
- Test: a stream with a retention limit drops old messages per subject while
  preserving tips of rarely-written subjects.

**Verify:** Retention test passes.

### Phase 6: Tests, Documentation, Cleanup

**Goal:** Close remaining gaps from the 2026-08-06 review.

- Tests: tombstone delete and undelete, `reset()`, edge point merge with
  older incoming timestamps.
- Resolve `Meta.Version`: persist it to the META KV bucket or remove the
  field and its references.
- Update changelog, ADR-7 status, and CLAUDE.md as needed.
- Deferred (measure first): `nodePoints` currently performs a synchronous
  `GetLastMsgForSubject` plus `Publish` per point. Once the point cache is
  trustworthy (Phase 3), the timestamp comparison can come from the cache
  and `PublishAsync` can pipeline batches. Revisit if `metricCycleNodePoint`
  shows it matters.

**Verify:** `go test -race ./...`, `golangci-lint run`, `siot_test` all pass.

### Phase 7: Stage 3 Prerequisite Spikes

**Goal:** Verify the NATS behaviors the Stage 3 design depends on. These can
run in parallel with earlier phases (in `nats-exp` or a scratch area, not in
this repo's production code).

- JetStream sourcing across leaf connections/domains: replica creation,
  catch-up after disconnect, chained sourcing through an intermediate
  instance.
- The filter-carrying consumer-create permission form
  (`$JS.API.CONSUMER.CREATE.<stream>.<consumer>.<filter>`) on the NATS
  version SIOT pins, including denying the legacy create forms.
- Record results in ADR-7 under the Stage 3 decision.

## Key Files

| File                       | Change                                          |
| -------------------------- | ----------------------------------------------- |
| `store/store.go`           | Handler fix (Phase 1)                           |
| `store/jetstream.go`       | Routing, subjects, streams, caches (Phases 2-4) |
| `store/edge_cache.go`      | Boundary resolution (Phase 2)                   |
| `store/jetstream_test.go`  | Migrated plus new tests (Phases 3-6)            |
| `server/args.go`           | Retention/durability options (Phase 5)          |
| `docs/adr/7-jetstream-store.md` | Status updates, spike results              |

## Risks

1. **Boundary resolution divergence between instances** would route the same
   node to different streams. Mitigated by keeping the rule simple, pure,
   and heavily unit tested; mirrored nodes across device boundaries are
   explicitly deferred to Stage 3.
2. **Cross-boundary moves** are the most intricate new code path. Mitigated
   by dedicated tests and by their rarity in practice.
3. **Sourcing may behave unexpectedly over leaf connections.** Phase 7 spikes
   de-risk this before Stage 3; durable consumers remain the fallback.

## Verification

1. `go build ./...` and `go test -race ./...` at each phase
2. `golangci-lint run` clean
3. `siot_test` full suite after Phase 3
4. Manual: `siot_run`, create/edit/delete/move nodes, verify persistence
   across restart

## Commits

| Hash | Description | Status |
|------|-------------|--------|
| -- | fix: edge handler reply and edge validation order (Phase 1) | Planned |
| -- | feat: boundary resolution in edge cache (Phase 2) | Planned |
| -- | feat: boundary-origin stream layout (Phase 3) | Planned |
| -- | feat: cross-boundary node moves (Phase 4) | Planned |
| -- | feat: retention and durability options (Phase 5) | Planned |
| -- | test/docs: coverage and cleanup (Phase 6) | Planned |
| -- | docs: Stage 3 spike results in ADR-7 (Phase 7) | Planned |
