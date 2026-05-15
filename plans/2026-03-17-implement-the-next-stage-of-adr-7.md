# Plan: JetStream Store (ADR-7 Stage 2)

**Branch:** `feat/js-store` **ADR:** docs/adr/7-jetstream-store.md (Stage 2:
switch store from SQLite to JetStream)

## Context

Stage 1 (message/subject encoding changes) is complete and merged. The Point
struct now uses `DataType`/`Data`, NATS subjects encode type/key
(`p.<nodeId>.<type>.<key>`), and binary encoding replaces protobuf for the wire
format. Stage 2 replaces SQLite with NATS JetStream as the persistent store.

**Why JetStream?** JetStream is already embedded via the NATS server. It
provides linearizable consistency, sequence-number-based change tracking,
subject-indexed message retrieval, and built-in stream synchronization between
hub and leaf instances. This eliminates the hash tree, simplifies
synchronization (Stage 3), and enables history storage without an external
time-series database.

## Design Decisions

### Stream Organization

Per-node JetStream streams, plus a KV bucket for metadata:

- **Per-node stream** `node.<nodeID>`: each node gets its own stream capturing
  subjects `node.<nodeID>.p.>` (node points) and `node.<nodeID>.ep.>` (edge
  points for edges where this node is the parent). The stream retains full
  history (time-series data). Current state is always the tip of each subject,
  retrieved via `GetLastMsgForSubject`.

  **Retention:** Use `MaxMsgsPerSubject` (default: no limit) as the only
  retention mechanism. This keeps the last N messages *per subject*, ensuring
  current state is always preserved for every point type/key — including
  rarely-updated config points. Avoid `MaxAge` and `MaxBytes`/`MaxMsgs`
  (stream-level), as these can silently drop config points that were set long ago
  but are still the current state. Since each point type/key is a unique subject,
  `MaxMsgsPerSubject` naturally bounds storage for high-frequency sensor data
  while preserving infrequent config.
- **`META`** KV bucket: stores `rootID`, `jwtKey`, `version`.

A typical edge device has ~30 nodes = ~30 streams. The NATS server handles this
efficiently. Per-node streams provide natural boundaries for:

- **AuthZ** (Stage 3): permissions can be set per stream, restricting which
  users/devices can access which nodes.
- **Selective sync** (Stage 3): remote instances create durable consumers only
  for the node streams in their subtree. On reconnect, durable consumers resume
  from their last ack'd sequence — only missed messages are delivered.
- **Subject structure**: `node.<nodeID>.p.<type>.<key>` for node points,
  `node.<nodeID>.ep.<childID>.<type>.<key>` for edge points (edges stored under
  the parent node's stream).

The `node.` prefix separates JetStream storage subjects from core NATS
real-time subjects (`p.>`, `ep.>`) to avoid the store handler seeing its own
writes. The store handler remains the gatekeeper: it receives on `p.>` / `ep.>`
(core NATS), validates timestamps/merge logic, then persists to
`node.<nodeID>.p.>` / `node.<parentID>.ep.>` (JetStream).

Streams are created on demand when a node is first written to, and the store
maintains a registry of known streams via the edge cache.

### Edge Cache

An in-memory edge cache provides fast tree traversal (parent-child lookups)
without scanning per-node streams on every query. It is populated on startup by
reading edge subject tips from each node's stream, and kept current as edge
points arrive.

### Hash Tree

Dropped entirely. JetStream provides sequence numbers and linearizable
consistency, making the CRDT hash tree unnecessary. The `Hash` field is removed
from `Edge` (`data/edge.go`) and `NodeEdge` (`data/node.go`). `updateHash`,
`updateHashHelper`, `verifyNodeHashes`, `CalcHash` are all removed. The protobuf
`Node` message and `ToPbNodes`/`PbToNode` conversions are updated to drop hash.

### Data Migration

Users export via `siot export` on the old (SQLite) version and `siot import` on
the new (JetStream) version. This uses existing tested code paths and preserves
timestamps.

## Implementation Plan

### Phase 1: Enable JetStream in Embedded NATS Server

**Goal:** Turn on JetStream with a configurable store directory.

**Files:**

- `server/nats-server.go` -- add `StoreDir` to options, set
  `opts.JetStream = true` and `opts.StoreDir`
- `server/server.go` -- add `JetStreamDir` to `Options` (default:
  `<DataDir>/jetstream`), pass through to NATS server options
- `server/test-server.go` -- use temp directory for JetStream data, clean up on
  stop

**Verify:** Server starts, logs confirm JetStream is active, existing SQLite
store still works.

### Phase 2: Implement DbJetStream Backend

**Goal:** New `DbJetStream` type providing the same methods as `DbSqlite`.

**New files:**

- `store/jetstream.go` -- `DbJetStream` struct and all storage methods
- `store/edge_cache.go` -- in-memory edge index (`byUp`, `byDown` maps)

**DbJetStream struct:**

```go
type DbJetStream struct {
    js        nats.JetStreamContext
    nc        *nats.Conn
    metaKV    nats.KeyValue
    meta      Meta
    edgeCache *EdgeCache
    mu        sync.RWMutex
}
```

**Methods (mirroring DbSqlite interface):**

- `NewJetStreamDb(nc, rootID)` -- create META KV, load edge cache from existing
  streams, init root if needed
- `nodePoints(id, points)` -- ensure stream `node.<id>` exists, for each point:
  get last msg from `node.<id>.p.<type>.<key>`, compare timestamps, publish if
  newer
- `edgePoints(nodeID, parentID, points)` -- ensure stream `node.<parentID>`
  exists, merge with existing edge points, publish to
  `node.<parentID>.ep.<nodeID>`, update edge cache
- `getNodes(parent, id, typ, includeDel)` -- use edge cache for tree structure,
  load node/edge points from per-node streams
- `up(id, includeDeleted)` -- read from edge cache `byDown[id]`
- `userCheck(email, password)` -- iterate user edges from cache, load points,
  check credentials
- `initRoot(rootID)` -- create root node stream, publish points and edges
- `reset()` -- delete all `node.*` streams, purge META KV, re-initialize
- `Close()` -- no-op
- `ensureStream(nodeID)` -- create stream `node.<nodeID>` if it doesn't exist,
  with subjects `node.<nodeID>.>`, retention `Limits`, no `MaxAge`. Current
  state read from tip via `GetLastMsgForSubject`.

**Helper functions:**

- `loadNodePoints(id)` -- enumerate subjects matching `node.<id>.p.>` via
  `StreamInfo(SubjectsFilter)`, get last msg for each, decode
- `loadEdgePoints(parentID, nodeID)` -- get last msg for
  `node.<parentID>.ep.<nodeID>`, decode
- `loadEdgeCache()` -- list all `node.*` streams, for each stream use
  `StreamInfo(SubjectsFilter)` to enumerate `ep.>` subjects, then
  `GetLastMsgForSubject` for each to read the tip (current state). No stream
  replay needed.

**Verify:** Unit tests for `DbJetStream` covering node CRUD, edge
creation/deletion, point merge logic, user auth.

### Phase 3: Wire Store to Use DbJetStream

**Goal:** Replace `DbSqlite` with `DbJetStream` in the `Store` struct.

**Files:**

- `store/store.go`:
  - Change `Store.db` from `*DbSqlite` to `*DbJetStream`
  - Update `NewStore` to create JetStream context and call `NewJetStreamDb`
  - Update `Params`: replace `File string` with `DataDir string` (for JetStream
    store directory)
  - Stub or remove `handleStoreVerify` / `handleStoreMaint` (hash verification
    is gone)
- `server/server.go`:
  - Update `storeParams` construction (pass DataDir instead of StoreFile)
- `server/test-server.go`:
  - Remove `StoreFile` references, use temp dir for JetStream data
  - Update cleanup to remove JetStream data directory

**Verify:** All existing `store/store_test.go` tests pass. Run
`go test -race ./...`.

### Phase 4: Remove SQLite Code

**Goal:** Clean removal of SQLite backend.

**Files:**

- Delete `store/sqlite.go`
- Delete `store/hash.go` (already dead code)
- Rename/rewrite `store/sqlite_test.go` to `store/jetstream_test.go`
- Remove `StoreFile` from `server.Options` if no longer used
- Run `go mod tidy` to remove `modernc.org/sqlite` dependency

**Verify:** `go build ./...`, `go test -race ./...`, `golangci-lint run`

### Phase 5: Point Cache (Performance)

**Goal:** Cache node points in memory to avoid repeated JetStream lookups.

**Files:**

- `store/jetstream.go` -- add `pointCache map[string]data.Points`
  - On `nodePoints()`: update cache after write
  - On `getNodes()`: read from cache, fall back to JetStream
  - On startup: optionally pre-populate by replaying each node's stream

For a typical system (~100 nodes, ~20 points each), this is ~2000 points in
memory -- trivial.

**Verify:** All tests pass. Manual test with `siot_run` to verify UI works.

### Phase 6: Documentation and Migration

**Goal:** Update docs and ADR.

- Update `docs/adr/7-jetstream-store.md`: mark Stage 2 complete
- Document migration path (export/import) in user docs
- Update CLAUDE.md if any build commands change
- Update changelog

## Key Files

| File                    | Role                 | Change                              |
| ----------------------- | -------------------- | ----------------------------------- |
| `server/nats-server.go` | NATS server config   | Enable JetStream                    |
| `server/server.go`      | Server wiring        | JetStream context, updated params   |
| `server/test-server.go` | Test infrastructure  | JetStream temp dirs                 |
| `store/store.go`        | NATS handlers        | Use DbJetStream instead of DbSqlite |
| `data/edge.go`          | Edge struct          | Remove `Hash` field                 |
| `data/node.go`          | NodeEdge struct      | Remove `Hash`, `CalcHash`           |
| `store/sqlite.go`       | SQLite backend       | **Delete**                          |
| `store/hash.go`         | Hash verification    | **Delete**                          |
| `store/jetstream.go`    | JetStream backend    | **New**                             |
| `store/edge_cache.go`   | In-memory edge index | **New**                             |

## Commits

| Hash | Description | Status |
|------|-------------|--------|
| 0c97576b | feat: enable JetStream in embedded NATS server | Implemented |
| 3693def8 | feat: add EdgeCache for in-memory edge index | Implemented |
| 06bad0c1 | feat: add DbJetStream backend with core storage methods | Implemented |
| ce818075 | feat: wire store to use DbJetStream, update server plumbing | Implemented |
| f85108e4 | refactor: remove SQLite store and hash tree code | Implemented |
| 99c28ca2 | feat: add point cache for node point lookups | Implemented |
| 80d9d51e | docs: update ADR-7 and changelog for Stage 2 completion | Implemented |

**Note:** Hash field kept in Edge/NodeEdge structs for API/protobuf compatibility
(always 0). CalcHash, ByHash, hash computation removed. Full Hash field removal
deferred to Stage 3 when sync client is rewritten.

## Risks

1. **JetStream subject enumeration speed** -- mitigated by point cache (Phase 5)
2. **Edge cache consistency** -- store is sole writer, updates cache in-line
3. **Power loss durability** -- JetStream uses WAL; configure sync options for
   embedded/edge deployments
4. **Breaking change** -- no SQLite fallback; users must export/import to
   migrate

## Verification

1. `go build ./...` compiles at each phase
2. `go test -race ./...` passes at each phase
3. `golangci-lint run` clean
4. `siot_test` full suite passes after Phase 3
5. Manual: `siot_run`, verify UI loads, create/edit/delete nodes, check points
   persist across restart
