# Plan: JetStream Store Point Encoding Changes

**Branch:** `feat/js-subject-point-changes` **PR:** simpleiot/simpleiot#742
**ADR:** docs/adr/7-jetstream-store.md

## Context

This is Step 1 of the JetStream migration: message/subject encoding changes. The
goal is to replace the `Value float64` and `Text string` fields in the Point
struct with a unified `Data []byte` + `DataType` field, and encode point
Type/Key in NATS subjects. This simplifies the wire format, eliminates the need
for protobuf in point encoding, and prepares for JetStream storage.

The PR TODO list from #742 drives the work. Several items are already done (new
Point struct, DataType constants, get/put helper methods, API doc updates). The
remaining work is substantial — the old `Value`/`Text` fields are referenced in
~30 files across the codebase.

## Status

- **Phase 1-3**: COMPLETE — New Point struct, all callers updated, store layer
  updated. JSON/YAML backward-compat marshal/unmarshal added.
- **Phase 4**: COMPLETE — Protobuf schema updated (value/text removed, dataType
  added). New binary Encode/DecodePoints replaces protobuf for NATS and serial
  wire format. Point.ToPb/PbToPoint retained for Node encoding only.
- **Phase 5**: COMPLETE — Elm Point type updated with dataType field. All
  positional constructors updated. Frontend compiles and tests pass.
- **Phase 6a**: COMPLETE — Edge points moved to `ep.` prefix.
- **Phase 6b**: COMPLETE — Node point subjects now use `p.<nodeId>.<type>.<key>`
  with one point per message. Edge points remain batched on `ep.<nodeId>.<parentId>`
  for atomicity. All upstream subscriptions updated.
- **Phase 7-8**: TODO

## What's Already Done

- New `Point` struct with `DataType` and `Data` fields (data/point.go)
- `PointOld` struct preserved for migration
- `PointDataType` constants (Unknown, Float, Int, String, JSON)
- Value get/put methods: `ValueInt()`, `ValueFloat()`, `ValueString()`,
  `PutInt()`, `PutFloat()`, `PutString()`
- Updated CRC function to use `Data` instead of `Value`/`Text`
- API documentation updated with new subject formats (`p.<nodeId>.<type>.<key>`,
  `ep.<nodeId>.<parentId>.<type>.<key>`)

## Implementation Plan

### Phase 1: Core Data Layer (data/point.go)

**Goal:** Make the codebase compile with the new Point struct.

1. **Update helper methods on Points collection** (`data/point.go`)
   - `Points.Value()` → use `p.ValueFloat()` internally
   - `Points.ValueInt()` → use `p.ValueInt()` internally
   - `Points.ValueBool()` → use `p.ValueFloat()` internally
   - `Points.Text()` → use `p.ValueString()` internally
   - `Point.Bool()` → use `p.ValueFloat()` internally
   - `Point.String()` → update to use DataType/Data

2. **Update protobuf conversion functions** (`data/point.go`)
   - `ToPb()` — encode Data/DataType into pb fields (keep pb format for now as
     transitional, or convert Data→Value/Text for backward compat)
   - `ToSerial()` — same approach
   - `PbToPoint()` — decode pb Value/Text into Data/DataType
   - `SerialToPoint()` — same
   - Decision: keep protobuf as wire format temporarily for backward compat, but
     populate new Point fields internally

3. **Update merge/comparison logic** (`data/point.go`)
   - `Merge()` function references `p.Value` and `p.Text`
   - `ProcessPoint()` references `.Value`

4. **Add encoding/decoding for wire packets** (`data/point.go`)
   - Functions to encode Point → binary (Time + Tombstone + Origin + DataType +
     Data)
   - Functions to decode binary → Point
   - These will eventually replace protobuf

### Phase 2: Store Layer

5. **Update SQLite storage** (`store/sqlite.go`)
   - Modify point storage to use `Data`/`DataType` instead of `Value`/`Text`
   - Create migration from old schema (add `data_type` column, migrate existing
     `value`→float Data, `text`→string Data)
   - Update queries in `store/sqlite.go` (~11 references)

6. **Update store handlers** (`store/store.go`)
   - `handleNodePoints()` and `handleEdgePoints()` — update point handling
   - ~3 references to `.Value`/`.Text`

### Phase 3: Client Updates

7. **Update all client code** that accesses `.Value`/`.Text`:
   - `client/rule.go` (~20 refs) — heaviest user, rules evaluate point values
   - `client/serial.go` (~7 refs) — serial protocol
   - `client/node.go` (~7 refs) — node operations
   - `client/node-tag-cache.go` (~5 refs)
   - `client/network-manager.go` (~5 refs)
   - `client/update.go` (~5 refs)
   - `client/metrics.go` (~4 refs)
   - `client/db.go` (~3 refs) — InfluxDB writes
   - `client/can.go`, `client/sync.go`, `client/particle.go`,
     `client/manager.go`, `client/browser.go`, `client/shelly-io-client.go`,
     `client/auth.go` (~2 refs each)
   - `client/serial_test.go`, `client/rule_test.go`, `client/db_test.go`,
     `client/node_test.go`, `client/manager_test.go` — tests
   - `node/onewire-io.go`, `node/onewire-io-node.go`, `node/node.go`
   - `modbus/reg.go`
   - `api/auth.go`, `api/client.go`
   - `store/sqlite_test.go`

   **Strategy:** For each reference, determine if it's reading a float or
   string, then use the appropriate getter/setter. Most `.Value` refs become
   `p.ValueFloat()` calls and `.Text` refs become `p.ValueString()` calls. For
   writes, use `p.PutFloat()` / `p.PutString()` / `p.PutInt()`.

### Phase 4: Protobuf Removal

8. **Update protobuf schema** (`internal/pb/point.proto`)
   - Remove `value` and `text` fields from Point and SerialPoint messages
   - Add `int32 dataType` field
   - Keep `data` field (already exists)
   - Regenerate `point.pb.go`

9. **Remove protobuf dependencies from point.go**
   - Remove `ToPb()`, `ToSerial()`, `PbToPoint()`, `SerialToPoint()` or replace
     with new wire encoding
   - Remove protobuf imports

10. **Update serial protocol** (`client/serial.go`, `client/serial-wrapper.go`)
    - Replace protobuf serial encoding with new binary encoding

### Phase 5: Frontend

11. **Update Elm Point type** (`frontend/src/Api/Point.elm`)
    - Replace `value : Float` and `text : String` with `dataType : Int` and
      `data` field
    - Update JSON encoder/decoder
    - Add helper functions for accessing typed values
    - Update all components that reference point value/text

12. **Update/remove protobuf JS library** (`frontend/lib/protobuf/point_pb.js`)
    - Remove or regenerate once protobuf schema changes

### Phase 6a: Separate Edge Point Subjects

13. **Move edge points to `ep.` prefix**
    - Change `SubjectEdgePoints` from `p.<nodeId>.<parentId>` to
      `ep.<nodeId>.<parentId>`
    - Change `SubjectEdgeAllPoints` from `p.*.*` to `ep.*.*`
    - Update store subscription from `p.*.*` to `ep.*.*`
    - Update debug subscriptions
    - Update `DecodeEdgePointsMsg` to expect `ep.` prefix (chunks[0]="ep")
    - Update sync client upstream edge subjects (`pup` → `epup` if applicable)
    - Verify all tests pass — this is a clean prefix swap with no wildcard
      behavior change

### Phase 6b: Add Type/Key to Node Point Subjects

14. **Update node point subjects to `p.<nodeId>.<type>.<key>`**
    - Update `SubjectNodePoints` to accept type/key parameters
    - Update `SendPoints` / `SendNodePoints` to build subject from point
      type/key
    - Change store subscription from `p.*` to `p.>` (safe now that edge
      points are on `ep.`)
    - Update `DecodeNodePointsMsg` if type/key should be extracted from subject
    - Update debug subscriptions

15. **Update edge point subjects to `ep.<nodeId>.<parentId>.<type>.<key>`**
    - Similar changes for edge points
    - Change store subscription from `ep.*.*` to `ep.>`

### Phase 7: Export/Import & Migration

14. **Fix export/import functions** (`client/node.go`)
    - Ensure node export/import works with new Point format

15. **Create database migration**
    - SQLite migration to convert existing data

### Phase 8: JS Library

16. **Update JavaScript library** (`frontend/lib/`)
    - Update any JS point handling code

## Verification

1. `go build ./...` — must compile cleanly after each phase
2. `go test -race ./...` — all Go tests pass
3. `siot_test` — full test suite passes
4. `golangci-lint run` — no lint errors
5. Manual test: `siot_run` and verify UI works, points display correctly
6. Test serial device communication if hardware available

## Notes

- This is a large refactor touching ~30 files. Work phase by phase, ensuring
  compilation after each phase.
- Phase 1-3 are the critical path — they make the code compile and run.
- Phases 4-8 can be done incrementally.
- The `PointOld` struct should be kept temporarily for reference during
  migration and removed once complete.
