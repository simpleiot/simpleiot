# Plan: Shared Nodes Across Devices

**Branch:** `cbrake/master` **Branched from:** `cda79cb7`

## Context

A node's points live in one stream, the one for its owning boundary
(`EdgeCache.OwningBoundary`, `store/edge_cache.go`), and a device replicates
only its own boundary's streams. A node mirrored into two devices resolves to
the upstream root, which neither device replicates, so it reaches neither of
them. The docs describe this limit in
[one node reaches one device](../docs/ref/data.md#one-node-reaches-one-device),
and the workaround is one node per device, written from a rule.

The common case this leaves unsolved is **an upstream node shared with several
devices**: a setpoint, a schedule, or a configuration value set in the portal
and used by every device at a site. This plan solves that case by having the
upstream deliver its own writes to every device a node is mirrored into.

Two related cases stay open and are described under
[future work](#future-device-writes-reach-the-other-devices): a node that lives
on one device and is mirrored into another, and a sensor on one device displayed
on another. Both need the upstream to relay one device's writes to the other
devices, which is a larger change to how sync works.

## Design Decisions

**Ownership stays as it is, and delivery is a separate set.** `OwningBoundary`
still decides where the upstream stores a node and which instance runs its
client. A new `EdgeCache.DeliveryBoundaries(id, rootID)` returns every device
boundary the node is reachable from, walking live edges of any role, owner
included. The owner decides where the node lives. The delivery set decides who
receives it.

**Ownership can still move when a mirror is added.** Mirroring a node of a type
that is not primary (a variable, a rule, a group) makes an edge with no role
(`mirrorRoleFor`, `client/node.go`), so a variable under device A mirrored into
device B has two roleless edges and resolves to the upstream root, as it does
today. That is fine: delivery is {A, B} either way, and `migrateBoundary`
already moves the stored subjects. What changes is that the move and the
delivery change land on the same edge write, so the seed and purge in Phase 3
handle both in one pass.

**Only the origin instance appends to a stream, still.** The upstream copies its
own writes into the streams it already writes for each device,
`inst.<device>.<upstream>`. Devices never forward what they replicate, and the
upstream never copies a device's data. Sync stays echo-free for the same reason
it is today.

**No change to sync or credentials.** A device credential already allows pulling
`inst_<device>_<upstream>` (`devicePermissions`, `server/auth.go`), and the sync
client already pulls it. All the work is in the upstream's store.

**Timestamps decide, as they do everywhere.** A copied point keeps its original
time. The tie-break for equal times uses the stream's origin (`tipWins`,
`store/merge.go`), so a copy the upstream appends carries the upstream as its
origin on the device, even when the upstream first learned the value from that
device. A copy of a device's own tip arriving back at the device can therefore
count as a change and be rebroadcast locally. This is benign, and
`migrateBoundary` already does it today, but it is why the design does not claim
a returning copy is a no-op.

**Seed from the cache when delivery starts.** Adding a mirror into a device
republishes the node's current tips into that device's upstream-written stream.
The point cache holds tips from every origin, so the seed carries whatever the
upstream currently knows, including a value a device wrote. Removing the mirror
purges them. This generalizes `migrateBoundary`, which already republishes and
purges when the owner changes.

**Purge keeps the delivery set.** `purgeNodeSubjectsExcept` removes a node's
subjects from every local-origin stream except one. Once delivery copies exist,
it has to keep every boundary in the delivery set, not just the owner, or an
ownership change deletes the copies it just made.

## Incompatibilities to Note in the Changelog

- A node mirrored into several devices now syncs to all of them, where before it
  reached none. After the upgrade, existing mirrors receive new writes right
  away and current values on the next change to the node's edges; removing and
  re-adding the mirror delivers the current values at once.
- The "one node reaches one device" section in `docs/ref/data.md`,
  `docs/ref/sync.md`, and `docs/user/ui.md` is replaced.

## Phase 1 — Delivery set

- [x] Add `EdgeCache.DeliveryBoundaries(id, rootID)`: the device boundaries
      reachable from `id` walking live edges of any role, stopping at the first
      boundary on each path. The instance root is left out.
- [x] Unit tests in `store/edge_cache_test.go`: a node under one device, a node
      under two devices, a no-role node under a device and a group, a sensor
      under device A mirrored into device B, a descendant of a node mirrored
      into two devices, and a tombstoned mirror edge.

## Phase 2 — Fan out node and edge points

- [x] `nodePoints` (`store/jetstream.go`) writes each point to the owning
      boundary as today, and also to `inst.<b>.<self>` for each boundary in the
      delivery set other than the owner.
- [x] `edgePoints` does the same for a child edge, using the parent's delivery
      set, so a node's subtree reaches every device the node does.
- [x] The fan-out also runs on a device. `DeliveryBoundaries` is empty there
      unless the device has nested devices, which item 1 of the
      [sync follow-ups plan](2026-09-19-jetstream-sync-followups.md) covers.
- [x] Test: on a hub with two devices, a variable under the hub root mirrored
      into both devices. A write on the hub lands in both `inst_A_<hub>` and
      `inst_B_<hub>`.

## Phase 3 — Seed and purge when the delivery set changes

- [x] In `edgePoints`, capture the child's delivery set before and after the
      edge lands, as it does for the owner. For each added boundary, republish
      the node's tips from the point cache and its child edges into that
      boundary's upstream-written stream, recursing into the subtree. For each
      removed boundary, purge the node's subjects there.
- [x] Fold this into `migrateBoundary` rather than adding a parallel path, and
      change `purgeNodeSubjectsExcept` to keep the whole delivery set.
- [x] Sync test in `client/sync_test.go`: a hub, two devices, a variable on the
      hub mirrored into both. Both devices receive its current value, a hub
      write reaches both, and removing one mirror removes the node from that
      device only.
- [x] Sync test: a variable created on device A, mirrored into device B on the
      hub. B receives A's value at the time of the mirror, and a hub write
      reaches both. A write on A reaches the hub and not B, which is the
      documented limit until the relay exists.
- [x] Sync test: a hub restart does not append anything to the device streams.

## Phase 4 — Docs and changelog

- [x] Replace "One node reaches one device" in `docs/ref/data.md` with how
      delivery works, including that a roleless node mirrored into a second
      device still moves to the upstream root, and update `docs/ref/sync.md` and
      `docs/user/ui.md`.
- [x] Changelog entry, noting that a device's writes reach the upstream and not
      the other devices.

## Future: Device Writes Reach the Other Devices

Not planned. Tracked as
[issue 810](https://github.com/simpleiot/simpleiot/issues/810) and recorded here
so the shape is not lost.

With delivery in place, the remaining gap is a write made on one device reaching
the other devices a node is shared with: a variable that lives on device A and
is edited on device B, or a sensor on A displayed on B. Solving it means the
upstream relays a device's writes, appending copies of what it consumes from
`inst_A_A` into `inst_B_<hub>` for each other device in the delivery set,
relaying only tips and holding relays during catch-up as `consumeReplica`
already does for broadcasts.

That changes what a hub-origin stream contains. Today every message in it is
something the hub decided; with the relay, it also carries copies of device
data. The costs are a shared sensor's readings multiplied by the number of
devices, equal-timestamp conflicts that can settle differently on each instance
because the tie-break origin is the stream rather than the writer, and a seed,
purge, and restart path that all have to agree with the relay. A rule on the
upstream covers most of what this would give, so it waits for a concrete need.

## Out of Scope

- **Nested device boundaries.** A device under another device is item 1 of the
  [sync follow-ups plan](2026-09-19-jetstream-sync-followups.md). Delivery here
  assumes each device syncs directly to the upstream.
- **Merging strategies other than latest-write-wins.** Two instances driving one
  value converge on the later write, the same as any other point.
