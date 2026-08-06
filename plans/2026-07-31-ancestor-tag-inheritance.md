# Plan: Ancestor Tag Inheritance for Time-Series Tags

**Branch:** TBD **Branched from:** `d839042e` **Status:** NOT STARTED

## Context

The Db client copies a configured set of point types onto every Influx point it
writes, so that time-series data carries descriptive labels alongside the raw
value. The Db node holds a `TagPointTypes` list (`client/db.go:28`, point type
`tagPointType`), and `nodeCache` (`client/node-tag-cache.go`) caches, per node,
the values of any points whose type appears in that list. When a point is
written, `CopyTags` (`client/node-tag-cache.go:53`) stamps on `node.id`,
`node.type`, `node.description`, and one `node.<type>.<key>` tag per cached tag
point. The behavior is described in the 0.15.0 changelog entries
(`CHANGELOG.md:142-153`).

The limitation is that tags resolve against the *emitting* node only. A
temperature point published by a sensor node carries that sensor's tags and
nothing else. To label data by machine, line, site, or customer, the same tag
points have to be repeated on every node that emits data. On a machine with a
few dozen Modbus registers, one label costs a few dozen edits, and every node
added later has to remember to repeat them. In practice the labels drift out of
sync, which is worse than not having them.

The goal is to set a tag once, on the node that represents the thing being
described, and have every point emitted beneath it carry that tag.

## Design Decisions

**Inherit from ancestors rather than from a single designated node.** The
obvious narrow version of this feature is "tags on the root device node apply to
everything." Ancestor inheritance costs the same to implement, subsumes that
case, and avoids two problems the narrow version has.

The first is granularity upstream. When an edge instance syncs to a cloud
instance, the edge's root node is grafted in as a child of the cloud root
(`client/sync.go:181`, `client/sync.go:592`). A Db client running in the cloud
sees the cloud's own root, not the per-device roots, so root-scoped tags there
would collapse to one label spanning every device. With an ancestor walk, each
device's root node is an ancestor of that device's data, and per-device tags
work in the cloud without any additional configuration.

The second is that the thing being labeled is usually not the root. A machine is
a subtree: a group node with sensors beneath it. Inheriting along the ancestor
chain lets a tag be set at whatever level the thing actually lives, and
root-level tagging becomes the case where the ancestor happens to be the root.

**Nearest ancestor wins, merged into the existing `node.tag.<key>` namespace.**
Tags resolve into the same flat namespace used today, so queries do not need to
know at what depth a tag was set. When the same key is defined at more than one
level, the value closest to the emitting node wins, which makes a local tag an
override of an inherited one. The alternative, namespacing tags by the level
they came from, keeps both values but makes every query depend on tree shape.

**Reuse the existing `TagPointTypes` configuration.** Inheritance applies to the
same per-Db-client list that gates tagging today, so the feature stays opt-in
and no new configuration point is introduced. Note that this does change output
for existing users who already populate `tagPointType`, since ancestor tags did
not previously propagate. This warrants a changelog entry.

**`node.description` and `node.type` remain local.** These describe the node
that emitted the point and are useful precisely because they are specific. If
they were inherited, a sensor's own description would be shadowed by, or shadow,
its machine's. Only the configured tag point types participate in inheritance.

**Bound the walk at the Db client's configured `Parent`.** The Db client
subscribes to `up.<Parent>.>` (`client/db.go:67`) and therefore only ever sees
data from that subtree. Stopping the ancestor walk at the same node keeps
resolution consistent with the subscription scope and bounds the walk depth.

**Resolve deterministically in the presence of multiple parents.** Simple IoT is
a DAG, and `store.up()` returns a slice, so the set of ancestors is not a chain.
Resolution walks breadth-first by depth, nearest depth wins, and ties at the same
depth are broken by lowest node ID with a log message the first time a tie is
observed for a given key. Non-determinism here would be difficult to notice and
expensive to recover from, because it would silently split an Influx series in
two.

**Cache the resolved tag set, not the walk.** The ancestor walk runs once per
cache miss, and steady-state point writes remain a single map lookup. This
preserves the current cost profile on the hot path.

**Influx cardinality is not a concern here.** `node.id` is already part of the
tag set, so the series count is already at least one per node per point type.
Adding stable ancestor tags multiplies that by approximately one, since the
inherited values are constant for a given node. The usual caution applies only
if a user places high-cardinality values such as serial numbers in a tag point,
which is equally true of the existing per-node tags.

## Tag Resolution Semantics

Given this tree, with `tagPointType` configured as `["tag"]`:

```
root (device)          tag:site=plant-a, tag:customer=acme
└── press-3 (group)    tag:machine=press-3, tag:site=plant-b
    └── temp-1 (modbusIo)   tag:sensor=inlet
```

A point emitted by `temp-1` resolves to:

| Tag                 | Value    | Source                          |
| ------------------- | -------- | ------------------------------- |
| `node.id`           | `temp-1` | emitting node, not inherited    |
| `node.type`         | `modbusIo` | emitting node, not inherited  |
| `node.description`  | ...      | emitting node, not inherited    |
| `node.tag.sensor`   | `inlet`  | emitting node                   |
| `node.tag.machine`  | `press-3` | inherited from `press-3`       |
| `node.tag.site`     | `plant-b` | `press-3` wins over `root`     |
| `node.tag.customer` | `acme`   | inherited from `root`           |

## Cache Invalidation

This is the substantive work in the plan. Today `nodeCache.Update`
(`client/node-tag-cache.go:74`) refreshes only the entry for the node the points
belong to, which is correct when tags cannot be inherited. Once they can, a tag
edit on an ancestor changes the resolved tags of every node beneath it, and every
one of those cached entries is stale.

Two events invalidate inherited tags:

**A tag point changes on any node.** The Db client already observes this.
`processPointsUpstream` (`store/store.go:532`) republishes each point to
`up.<ancestor>.<nodeID>` for every ancestor, starting with the node itself
(`store/store.go:275`), so any point within the Db client's subtree arrives on
its existing subscription. The simplest correct response is to `Clear()` the
whole cache (`client/node-tag-cache.go:128`) when a point of a configured tag
type arrives for any node. Tag edits are rare, entries repopulate lazily on the
next point from each node, and this avoids maintaining a reverse index from
ancestor to descendants. If profiling later shows the repopulation storm matters
on a large tree, a descendant index can replace it without changing semantics.

**A node is reparented.** Moving a node under a different machine changes its
inherited tags, and that arrives as an edge point rather than a node point. This
path needs attention because the Db client does not currently distinguish the
two. Node points arrive on `up.<up>.<node>.<type>.<key>` and edge points on
`up.<up>.<node>.<parent>.<type>.<key>` (`client/manager.go:277-278`), and the
handler at `client/db.go:76` splits the subject and takes `chunks[2]` as the node
ID in both cases. The result is that edge points are presently decoded and
written to Influx as though they were node points, which is what the `FIXME` at
`client/db.go:65` anticipates. Distinguishing them by chunk count lets edge
points invalidate the cache instead of being written as data.

## Implementation Plan

### Phase 1: Ancestor resolution in the tag cache

- Extend `nodeCache` with the Db client's boundary node ID, so the walk knows
  where to stop. Pass it in from `NewDbClient` (`client/db.go:56`) and from the
  rebuild on `tagPointType` change (`client/db.go:234`).
- Add an ancestor walk to `nodeCache.Update`. Starting from the emitting node,
  use `GetNodes(nc, "all", id, "", false)` to retrieve every living instance of a
  node with its `Parent` populated, which is the same call the cache already
  makes at `client/node-tag-cache.go:81`. Walk breadth-first, level by level,
  until the boundary node is reached or a node has no living parents.
- Merge tag entries with nearest-depth-wins. Within a level, apply in ascending
  node ID order and log once per key when two nodes at the same depth define it.
- Store the merged result in `nodeCacheEntry.Tags`. `CopyTags` needs no change,
  since it already emits every entry in that map under `node.<type>.<key>`.
- Keep `Description` and `Type` populated from the emitting node only.

### Phase 2: Invalidation on tag point changes

- In the `newDbPoints` handler (`client/db.go:243`), check arriving points
  against `TagPointTypes` before the cache update. When one matches, `Clear()`
  the cache so the emitting node and every descendant re-resolve.
- Confirm the high-rate path (`client/db.go:101`), which calls `Update` with an
  empty point set purely to populate the cache, still behaves correctly against a
  cleared cache. It should, since `Update` fetches on miss.

### Phase 3: Edge point handling

- Split the `up.<Parent>.>` handler at `client/db.go:68` on chunk count: five
  chunks is a node point, six is an edge point.
- Route edge points to cache invalidation rather than to `writeAPI.WritePoint`.
  A tombstone or new edge changes tree shape, so `Clear()` is the appropriate
  response.
- This resolves the `FIXME` at `client/db.go:65` for the read path. Deciding
  whether edge points should be stored as time-series data in their own right is
  out of scope here.

### Phase 4: Tests

- Unit tests for the resolution rules against a constructed tree: inheritance
  from a grandparent, nearest-ancestor override, boundary node stopping the walk,
  and deterministic tie-breaking with two parents at equal depth.
- A test that a tag edit on an ancestor changes the tags resolved for a
  descendant on the next point.
- A test that reparenting a node changes its resolved tags.
- Verify tombstoned tag points remove the inherited tag rather than leaving a
  stale value, matching the existing behavior at
  `client/node-tag-cache.go:113-120`.

### Phase 5: Documentation and changelog

- Document the resolution rules, including precedence and the DAG tie-breaking
  rule, in the Db client documentation.
- Changelog entry noting the behavior change for existing `tagPointType` users.
- No frontend work is expected. The device node already exposes a tag editor
  (`frontend/src/Components/NodeDevice.elm:124`), as do group nodes and most
  client nodes, so the configuration surface needed for this feature exists.

## Files Touched

| File                                       | Change                                   |
| ------------------------------------------ | ---------------------------------------- |
| `client/node-tag-cache.go`                 | ancestor walk, merge rules, boundary node |
| `client/db.go`                             | pass boundary node, invalidation, edge point routing |
| `client/node-tag-cache_test.go`            | new tests                                |
| `docs/` (Db client reference)              | resolution rules                         |
| `CHANGELOG.md`                             | behavior change note                     |

## Open Questions

- Should an inherited `description` be available as a separate tag, for example
  `node.ancestor.description`, for dashboards that want to label a chart with the
  machine name without requiring a tag point to be set? This is additive and can
  be deferred.
- Should inheritance be switchable per Db client, for users who prefer the
  current local-only behavior? The argument against is that a user who does not
  want inherited tags can simply not set tags on ancestor nodes. Worth revisiting
  only if the behavior change proves disruptive.
- Tag changes are not retroactive. Renaming a machine produces a new Influx
  series from that point forward, and queries spanning the change see two series.
  The same is already true of `node.description`. This is documented rather than
  solved, but it is worth confirming that is acceptable before implementing.
- If the cache clear proves too coarse on large trees, is a reverse index from
  ancestor to cached descendants worth the added state? Deferred until there is
  evidence it matters.
