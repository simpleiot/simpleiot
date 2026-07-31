---
date: 2026-04-27
topic: adr-7-stage-3-jetstream-sync
---

# ADR-7 Stage 3: JetStream Inter-Instance Sync

## Problem Frame

Stages 1 and 2 of `docs/adr/7-jetstream-store.md` are complete: each SIOT
instance now stores its tree as per-node JetStream streams with a unified
point/subject encoding. Stage 3 connects instances together so that a tree
distributed across a cloud and one or more edge devices stays consistent —
including the **history** that accumulates while a link is offline.

The existing inter-instance sync (`client/sync.go`) only transfers current
state. After a long offline period, accumulated sensor history is lost. The
mechanism is a polling reconcile (~20s period) layered on core NATS subscribes —
workable for current state, but it does not match Stage 2's new storage model
and cannot benefit from JetStream's sequence-numbered catchup.

Stage 3 replaces it with a JetStream-native sync that delivers full history,
catches up automatically after disconnection, and removes the need for
period-based reconciliation and hash comparison.

This document supersedes the Stage 3 sketch currently in
`docs/adr/7-jetstream-store.md` (which described a durable-consumer model).

---

## Actors

- A1. **Edge SIOT instance**: runs on a device (often Cat-M cellular). Owns a
  subtree of nodes, generates sensor data, applies config from upstream,
  publishes acks/state.
- A2. **Cloud SIOT instance**: aggregates many devices. Hosts the cloud web UI.
  Sends config and commands downward, reads sensor data and state upward. May
  itself be a leaf of a higher cloud.
- A3. **Cloud user**: browser interaction with cloud UI; edits config across
  many devices; views state and history.
- A4. **Edge user**: browser interaction with the edge instance's local UI (when
  present); edits config and views state for the local subtree.

---

## Key Flows

- F1. **Steady-state telemetry**
  - **Trigger:** Edge sensor produces a new point.
  - **Actors:** A1, A2.
  - **Steps:** Edge writes the point to the local node stream and to its
    outbound link stream. The cloud's mirror replicates the message. Cloud
    clients (UI subscribers, rules, db) react via local NATS subjects as today.
  - **Outcome:** Cloud sees the new point with matching JetStream sequence
    number, available immediately for live UI and durably stored for history
    queries.
  - **Covered by:** R1, R3, R5.

- F2. **Cloud user edits config while edge is online**
  - **Trigger:** A3 changes a config point in the cloud UI.
  - **Actors:** A3, A2, A1.
  - **Steps:** Cloud writes the config point to the link's `down` stream. Edge
    mirror replicates. Edge applies the change locally; resulting state (e.g.,
    new period, ack) is published to the edge's `up` stream and mirrors back to
    cloud.
  - **Outcome:** Cloud and edge converge on the new config; cloud sees the
    edge's published state to confirm application.
  - **Covered by:** R2, R5.

- F3. **Edge offline, then reconnects after extended outage**
  - **Trigger:** Cellular link drops for hours/days, then restores.
  - **Actors:** A1, A2.
  - **Steps:** During outage, edge continues writing to its local streams. Cloud
    continues writing to its outbound `down` stream. On reconnect, both mirrors
    automatically replay missed messages by JetStream sequence number. Both
    sides converge without polling, hash comparison, or application-level
    reconcile logic.
  - **Outcome:** Full history accumulated on both sides during the outage is
    transferred. No data loss aside from intentional retention bounds.
  - **Covered by:** R1, R3, R4, R6.

- F4. **Edge user edits config locally**
  - **Trigger:** A4 changes a config point in the edge UI.
  - **Actors:** A4, A1, A2.
  - **Steps:** Edge writes the config point to local node stream and to its `up`
    stream (edges treat their own UI writes as part of the up flow). Cloud's
    mirror replicates. If a cloud user concurrently wrote the same type/key, the
    merge logic resolves by point timestamp (LWW).
  - **Outcome:** Both sides converge to the latest-timestamped value. Multi-
    writer resolution is silent (a known v1 limitation; see Outstanding
    Questions).
  - **Covered by:** R2, R5, R12.

---

## Requirements

**Sync mechanism**

- R1. Inter-instance sync uses JetStream **stream sources/mirrors**. NATS
  manages catchup, replay, and gap recovery. SIOT does not implement a
  client-driven durable-consumer loop.
- R2. Each inter-instance link uses a **directional pair of streams**:
  `link.<edgeID>.up` (written only by the edge instance) and
  `link.<edgeID>.down` (written only by the cloud-side instance). Each side
  mirrors the opposite stream from the remote server.
- R3. The number of streams introduced for sync scales **per link**, not per
  node. A 30-node edge syncing to one cloud creates two streams per side.

**Subject and data model**

Three subject namespaces coexist after Stage 3:

```
┌─ In-process delivery (core NATS) ────────────────────────┐
│   p.<nodeID>.<type>.<key>                                │
│   ep.<nodeID>.<parentID>.<type>.<key>                    │
└──────────────────────────────────────────────────────────┘
              ↕ store handler (existing, Stage 2)
┌─ Local persistent storage (JetStream) ───────────────────┐
│   node.<nodeID>.p.<type>.<key>                           │
│   node.<nodeID>.ep.<childID>.<type>.<key>                │
└──────────────────────────────────────────────────────────┘
              ↕ link adapter (new, Stage 3)
┌─ Cross-instance sync (mirrored JetStream) ───────────────┐
│   link.<edgeID>.up.p.<nodeID>.<type>.<key>               │
│   link.<edgeID>.up.ep.<nodeID>.<parentID>.<type>.<key>   │
│   link.<edgeID>.down.p.<nodeID>.<type>.<key>             │
│   link.<edgeID>.down.ep.<nodeID>.<parentID>.<type>.<key> │
└──────────────────────────────────────────────────────────┘
```

The position of `p`/`ep` in the link namespace is held constant between `<dir>`
and `<nodeID>` so future shadow prefixes (`dp`, `rp`) slot in additively.

- R4. Subject layout for cross-instance traffic is
  `link.<edgeID>.<dir>.p.<nodeID>.<type>.<key>` for node points and
  `link.<edgeID>.<dir>.ep.<nodeID>.<parentID>.<type>.<key>` for edge points,
  where `<dir>` is `up` or `down`. The position of `p`/`ep` between `<dir>` and
  `<nodeID>` is reserved so that future shadow prefixes (`dp`, `rp`) can be
  added additively without re-mirroring or breaking subscribers.
- R5. Cross-instance traffic carries the existing `data.Point` type unchanged
  from Stage 2. Conflicts on the same `(nodeID, type, key)` resolve by
  `Point.Time` (last-writer-wins), preserving today's merge semantics.
- R6. Stage 2's per-node storage layout (`node.<nodeID>` streams,
  `MaxMsgsPerSubject` retention) is unchanged. Stage 3 adds the `link.*` streams
  alongside; a small adapter publishes from per-node storage to the outbound
  link stream, and from the inbound mirror back into local per-node storage so
  existing in-process subscribers (`p.>`, `ep.>` core NATS) keep working
  unchanged.

**Delivery model**

- R7. All cross-instance messages flow through the mirrored streams. Stage 3
  introduces **no core-NATS-over-the-link** path. In-process delivery on a
  single instance continues to use core NATS as today.
- R8. Live updates and offline catchup share one delivery path. There is no
  separate reconcile loop, period timer, or hash comparison.

**Connection lifecycle**

- R9. The edge instance initiates the NATS client connection to the cloud
  (matches existing IoT NAT/firewall expectations). Auth credentials are
  configured on the existing `Sync` node.
- R10. On connect, both sides ensure their outbound stream and inbound mirror
  declarations exist. Mirror catchup runs automatically; SIOT does not gate on
  its completion before resuming live operation.
- R11. On disconnect, both sides continue writing locally. Reconnect is retried
  with backoff (existing `Sync` node's lifecycle behavior preserved).

**AuthZ boundaries**

- R12. NATS subject-level ACLs gate which subjects each link's credentials may
  publish to and mirror from. The cloud side restricts each device's credentials
  so a device cannot write outside its own `link.<edgeID>.up` subject space, and
  cannot mirror anything beyond its `link.<edgeID>.down`.
- R13. User-level AuthZ (which cloud user sees which subtree) reuses the
  existing SIOT user/group model and is enforced at the application layer
  reading from local stream tips. Stage 3 does not introduce per-user NATS
  accounts.

**Replacement of existing sync**

- R14. The new mirror-based sync **replaces** the implementation of
  `client/sync.go` in-place. The `Sync` config node (URI, AuthToken, Period,
  Disabled, Description) is preserved as the user-facing concept. `Period`
  becomes a vestigial field with no behavioral effect; `SyncCount` /
  `SyncCountReset` are repurposed or removed (decision deferred to planning).
- R15. No automatic data migration is provided. Field instances upgrade by
  taking a backup, installing the new release, and restoring (matches the
  approach used for the Stage 2 SQLite → JetStream transition).

---

## Acceptance Examples

- AE1. **Covers R1, R3, R6.** Given an edge with 30 nodes connected to a cloud,
  when sync starts, then exactly two streams (`link.<id>.up` and
  `link.<id>.down`) plus their two mirrors exist for that link — not 60 or 120.
- AE2. **Covers R3, R4.** Given the edge cycles power and reconnects after a
  one-hour outage during which 10,000 sensor points accumulated locally, when
  the link comes back, then the cloud receives all 10,000 messages in original
  sequence order, and the cloud's local node streams contain the full history.
- AE3. **Covers R2, R5, R12.** Given a cloud user changes a node's `period` from
  30 to 60 while the edge is online, when the message reaches the edge, then the
  edge applies the change, publishes confirmation on the up stream, and the
  cloud UI reflects `period=60` within one round-trip latency.
- AE4. **Covers R7, R8.** Given the link is online and steady, when an edge
  point is published, then it appears on the cloud exactly once via the mirror
  stream — never via a parallel core-NATS path.
- AE5. **Covers R12.** Given an edge's credentials are configured for
  `link.<edgeID>.*`, when that edge attempts to publish to
  `link.<otherEdgeID>.up`, then the publish is rejected by the cloud NATS server
  and surfaces as a sync error.

---

## Success Criteria

- An edge device disconnected from the cloud for hours and reconnected recovers
  full sensor history without operator intervention. This is observable in the
  cloud UI as a populated history graph for the offline period.
- The Stage 3 sync codepath is materially simpler than today's `client/sync.go`:
  no period reconcile, no hash comparison, no per-node subscription bookkeeping.
  Reduction in lines of code in `client/sync.go` + removed helpers is a usable
  proxy.
- The number of long-lived NATS streams introduced by sync is bounded by the
  number of links, not the number of nodes. Verified by counting streams on a
  cloud running a 30-node edge.
- A planner picking up this document can write an implementation plan without
  inventing product behavior, scope boundaries, or success criteria.
  Implementation choices (Go types, exact mirror filter syntax, test rig shape)
  are appropriate for planning to resolve.

---

## Scope Boundaries

- **Shadow / desired-vs-reported data model.** Deferred to a possible future
  stage. Stage 3 ships with current point semantics and silent LWW merge.
- **Optimistic-concurrency writes** (`Nats-Expected-Last-Subject-Sequence`) for
  multi-writer config races. Possible future addition; not in v1.
- **Generation-counter command pattern** (Lion-style). Possible future addition
  once shadow lands.
- **Automatic data migration** from existing deployments. Field upgrades use
  backup/restore.
- **Per-user NATS accounts / JWT-based AuthZ.** Stage 3 keeps the existing
  application-layer user model. Operator/account work is a separate effort.
- **Replacement of in-process core NATS subjects** (`p.>`, `ep.>`) for local
  delivery. Those stay as today; only the inter-instance path changes.
- **Re-architecting the per-node storage layout** introduced in Stage 2. Not
  revisited.
- **Cloud-to-cloud federation features beyond simple parent-child links.**
  Multiple-upstream and cloud-to-cloud links should work by construction
  (per-link pairs compose), but elaborated federation topologies are out of
  scope for v1.

---

## Key Decisions

- **Sources/mirrors over durable consumers.** NATS-managed catchup is strictly
  less SIOT code than client-driven consumers and provides identical semantics.
  The previous ADR-7 Stage 3 sketch (durable consumers) is superseded.
- **Per-link directional pair, not per-node streams for sync.** Per-node sync
  streams would scale to ~60 streams per 30-node edge — a Cat-M concern flagged
  in ADR-7's open questions. Per-link pairs reduce that to ~2 while preserving
  subject-level granularity inside the streams. This does not change Stage 2's
  per-node _storage_ layout; it adds a parallel sync layer.
- **Stream-only cross-instance delivery.** Eliminates dual-path reasoning bugs
  and dedup complexity. The per-message JetStream commit cost is immaterial
  relative to WAN/Cat-M latency.
- **Defer shadow.** Transport and data model are orthogonal. Shipping transport
  now and keeping subject space reserved for shadow lets the team measure real
  pain before committing to the data-model expansion. Industry convergence (AWS
  Device Shadow, Azure Device Twin, Lion IoT) is noted as prior art for the
  future stage.
- **Full replacement of `client/sync.go`, no migration tooling.** Backup/
  restore is acceptable per Stage 2 precedent. Avoids carrying two sync
  implementations indefinitely.

---

## Dependencies / Assumptions

- NATS server version embedded in SIOT supports stream sources/mirrors with
  `FilterSubjects`. (Stage 2 already updated to nats-server v2.12.5; verified to
  support these features.)
- JetStream `MaxMsgsPerSubject` retention behaves as expected on link streams.
  Stage 2 chose this for storage; Stage 3 reuses the same semantics.
- Cellular bandwidth on Cat-M is sufficient to carry mirror keepalives and
  catchup traffic for the target node-count range. **Unverified.** Treated as a
  planning-stage benchmark, not a brainstorm-stage gate.
- The existing `Sync` node config (URI, AuthToken) is sufficient for link
  credentials; per-link NATS user provisioning is an operator concern outside
  Stage 3.
- Edge instances initiate connections to cloud (NAT-friendly). Cloud- initiated
  connections are not in scope.

---

## Outstanding Questions

### Resolve Before Planning

(none)

### Deferred to Planning

- [Affects R6][Technical] Exact adapter shape between per-node storage streams
  (Stage 2) and per-link sync streams (Stage 3). Republish on every store write?
  Subject re-mapping consumer? Decide during planning based on code structure of
  `store/jetstream.go`.
- [Affects R12][Technical] Concrete subject ACL patterns and how they map to the
  existing `Sync.AuthToken` flow. May require introducing per-link NATS user
  records or staying with shared-token for v1. Planning decision.
- [Affects R3][Needs research] Cat-M bandwidth viability of mirror keepalives +
  catchup at target node counts. Build a lab rig (cellular link emulation,
  30-node edge, hour-scale outage) and measure. Could surface retention or
  batching tunings.
- [Affects R14][Technical] What to do with `SyncCount` / `SyncCountReset`
  UI/fields once period reconcile is gone. Repurpose as mirror-lag indicator?
  Remove? Frontend implication.
- [Affects R10][Technical] Concrete stream and mirror creation sequence on
  startup and reconnect, including idempotency. Also: behavior when a stream
  exists but with stale config. Planning question.
- [Affects R3][Needs research] Cloud-side scaling: a 1000-device fleet produces
  ~2000 streams on the cloud. Confirm NATS handles this and decide whether
  stream-count tuning (e.g., grouping multiple small edges into one stream pair)
  is needed.

---

## Next Steps

→ `/ce-plan` for structured implementation planning. The plan should
incorporate:

- The adapter design between per-node and per-link streams (Deferred Question
  1).
- A Cat-M benchmark rig as part of the implementation plan (Deferred Question
  3).
- Updates to `docs/adr/7-jetstream-store.md` Stage 3 section to reflect the
  decisions in this document.
- A `CHANGELOG.md` entry under `## Next` once implementation begins.
