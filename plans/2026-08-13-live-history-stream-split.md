# Plan: Split Replication into Live and History Streams

**Branch:** `feat/sync-live-history` **Branched from:** `640fe7de`

## Context

Sync replicates whole streams rather than comparing tree state, so a receiving
instance that holds none of a sender's data starts at sequence 1 and works
forward. That is correct, and it is what makes the stream sequence the
synchronization state, but it means current values are the *last* thing to
arrive rather than the first.

The cost shows up whenever a receiver loses its identity. A store reset gives an
instance a new root ID, the durable consumer is named for the receiving
instance, so the sender sees a new reader and replays its retained history from
the beginning. On a recent upstream reset that meant 499,215 messages at roughly
64 messages per second — about two hours during which the upstream's UI showed a
device last updated a day and a half earlier, while the device beside it was
publishing every second.

Windowed replication (`96747936`) made that replay several times faster, but it
did not change the shape of the problem: the receiver still processes *O(history)*
messages before it holds *O(subjects)* worth of current state. A device with a
year of data behind it takes proportionally longer, and the fix is not to make
the replay faster but to stop making current state wait behind it.

This plan separates the two concerns. Current state arrives in one pass over the
subject list; history fills in behind it without blocking anything.

## The enabling property

The merge rule already tolerates out-of-order arrival. `tipWins`
(`store/merge.go:19`) resolves by embedded timestamp with an origin tie-break,
and is explicitly idempotent:

```go
func tipWins(curTime time.Time, curOrigin string, inTime time.Time, inOrigin string) bool {
	if inTime.After(curTime) { return true }
	if curTime.After(inTime) { return false }
	return inOrigin > curOrigin
}
```

A point backfilled from last week cannot displace a live point that already
arrived, and the same point delivered twice is a no-op. So live and historical
messages may be interleaved freely without corrupting in-memory state, and no
part of this plan needs to sequence them against each other.

Exactly one mechanism depends on arrival order: the cold-start loader
(`store/jetstream.go:805` and `:857`, plus the lazy per-node variant behind
`ensureNodePointsCached`) reads `GetLastMsgForSubject`, which is the
last message by *sequence*, and merges it as the tip. That call trusts sequence
order to agree with timestamp order per subject. Appending backfilled history
into the same stream as live traffic breaks that assumption, and a receiver
restarting mid-backfill would load stale values and keep them. That single
constraint is what makes this a stream-layout change rather than a consumer
change.

## Design Decisions

**Two lanes, not a catch-up mode.** The pump gains a second consumer rather than
a state machine. A *history* consumer reads from sequence 1 in order, as today. A
*live* consumer starts at `DeliverLastPerSubjectPolicy`, which delivers the newest
message for every subject and then follows the stream. The live lane's first pass
is one message per subject — hundreds, not hundreds of thousands — so the
receiver is current within seconds of connecting. Both lanes run permanently and
neither has to detect that the other has finished. There is no handoff sequence,
no completion signal, and no tip sweep to write.

**History keeps the canonical subjects; live takes the prefix.** JetStream
rejects overlapping subjects within an account, verified against a running
server:

```
nats: error: could not create Stream: subjects overlap with an existing stream (10065)
```

So the two streams cannot both carry `inst.<boundary>.<origin>.>` and one of them
must be renamed. It should be the live one. History is the lane carrying data
that cannot be reconstructed, it is the lane a future multi-hop topology would
forward, and it is the lane the docs describe as needing no translation because a
copied message keeps its subject. The live stream is a derived view that can be
deleted and rebuilt in one pass, so it can afford a rewrite:

| Lane    | Stream                     | Subjects                          |
| ------- | -------------------------- | --------------------------------- |
| history | `inst_<boundary>_<origin>` | `inst.<boundary>.<origin>.>`      |
| live    | `live_<boundary>_<origin>` | `live.inst.<boundary>.<origin>.>` |

Choosing it this way also makes the upgrade non-destructive: an existing replica
stream already *is* a history stream, under the right name, with the right
subjects and the right retention. Nothing has to be migrated, renamed, or
re-replicated. The live streams are new and empty and fill themselves.

**Only replica streams split.** An instance's own origin stream is written by one
writer in timestamp order, so its tips are already correct and its cache is
updated at write time rather than by consuming. Own streams are untouched. This
keeps the change to the receiving path and leaves the write path, the wire
subjects, and the `Siot-Origin` rule exactly as they are.

**The receiver's cache is fed by live streams only.** Today `replicaManager`
consumes every replica stream and merges it. After the split it consumes only
`live_*`. History streams are for sinks — the Db client and anything else that
wants every point — and are never loaded into the node cache. That is what makes
cold start cheap: `loadStream` currently walks `info.State.Subjects` doing one
`GetLastMsgForSubject` round trip per subject against a stream holding the full
history, and against a live stream that call runs against a stream whose entire
contents are the answer.

**The backlog gating goes away.** `consumeReplica` (`store/replica.go:190-221`)
carries a `pending` map and a `caughtUp` flag purely so state clients do not see
a replay of stale intermediate values while a mixed stream drains. A live stream
has no meaningful backlog by construction, and history consumers always wanted
every point. The machinery is deleted rather than maintained.

**The Db client is not touched.** It consumes `inst_*` with its own durable
consumers, and `inst_*` still means "every point, in order, full retention."
Its behavior after this change is identical to before.

## What this costs

Every point crosses the wire twice in steady state: once in the live lane, once
in the history lane. Points are small — the 30,000-message stream measured on the
test pair was 6.8 MiB, about 230 bytes per message — but this is a real doubling
on a metered link, and it is the main thing to weigh against the design.

It can be removed later by idling the live lane while the history lane has no
backlog, since `NumPending` is already available in `fillWindow` and the live
consumer is ephemeral and cheap to recreate. That is deliberately not in this
plan: it adds a mode and a hysteresis threshold to a design whose whole appeal is
that it has neither, and it should be justified by a measurement rather than
assumed. See [Open Questions](#open-questions).

## Implementation Plan

### Phase 1: Live stream naming and the tip pump

**Goal:** A second pump that carries current state, with no change to what the
receiver does with it yet.

1. Add `liveStreamName(boundary, origin)` and `liveSubjects(boundary, origin)`
   beside `streamName` in `store/jetstream.go`, plus `liveSubject(subject)` and
   `histSubject(subject)` to add and strip the `live.` prefix. Export what
   `client` needs.
2. In `client/sync.go`, rename `runPump` to `runHistoryPump` with no behavior
   change, and add `runLiveTipPump`. The live pump creates the destination
   `live_*` stream if missing, opens an *ephemeral* consumer with
   `DeliverPolicy: jetstream.DeliverLastPerSubjectPolicy`, and publishes each
   message to the prefixed subject. JetStream requires a filter subject with
   that policy, so the consumer sets
   `FilterSubject: "inst.<boundary>.<origin>.>"`. The prefix is applied by
   wrapping each message in a `pumpMsg` adapter whose `Subject()` returns the
   `live.` form — `pumpMsg` is already an interface, so `sendWindow` and
   `fillWindow` are reused unchanged.
3. The live consumer is ephemeral on purpose: it holds no durable state, a
   reconnect re-derives current state in one pass, and there is no second durable
   name to collide with a store reset.
4. When a window fails, the live pump re-ensures the destination `live_*` stream
   exists before retrying. That is what makes "delete a live stream to rebuild
   it" true while a session is running, rather than only at the next reconnect.
5. Start the live pump beside the history pump in both places pumps start: the
   push in `runSession` (which today calls `runPump` directly) and each pull in
   `scanPulls`. The push side gains a second stopper next to `pushCC`, and the
   `pulls` map value becomes a pair of stoppers.

**Verify:** `go build ./... && go test -race ./client/ && golangci-lint run`

**Test:** Extend `client/sync_window_test.go` with the subject rewrite —
`liveSubject`/`histSubject` round-trip, including a subject holding a `.` inside a
point key. Add a fake-publisher case asserting the live pump publishes to the
`live.inst.…` subject and the history pump to `inst.…`, so the two lanes cannot
be crossed without a test failing.

### Phase 2: Receive live streams

**Goal:** The receiver's cache is fed by the live lane.

1. `replicaManager.scan` lists `live.>` in addition to `inst.>`, and consumes
   only `live_*` streams. Replica `inst_*` streams are discovered for policy
   purposes but no longer consumed into the cache.
2. `mergeReplicaMsg` strips the `live.` prefix before tokenizing. The existing
   `parentID == "root"` guard stays exactly as it is.
3. Delete the `pending`/`caughtUp` gating from `consumeReplica` and broadcast
   every tip change directly.
4. Apply a live retention policy when a `live_*` stream is discovered:
   `MaxMsgsPerSubject: 10` (a named constant) rather than the history default.
   Count-based retention only, never `MaxAge`: cold start depends on the live
   stream holding at least one message for every subject ever written, and an
   age limit would silently drop subjects that update rarely. Ten per subject
   costs almost nothing and absorbs a consumer that stalls briefly.

**Verify:** `go test -race ./store/ ./client/`

**Test:** A store-level test that publishes into a `live_*` replica stream and
asserts the tip reaches the node cache and a broadcast is emitted, and that a
message arriving with an older timestamp than the current tip changes nothing —
the `tipWins` property this design leans on, asserted at the replica boundary
rather than only in `store/merge_test.go`.

### Phase 3: Cold start

**Goal:** Startup loads current state from live streams and skips replica
history.

1. `loadAllStreams` loads the instance's own `inst_*` stream as today, loads each
   `live_*` replica stream, and skips replica `inst_*` streams.
2. `loadPointSubjects` and `loadEdgeSubjects` accept the prefixed subject form.
   The root-anchor guard added in `bf7f483a` stays in place and applies to both.
3. `loadNodePoints` (the cache-miss path behind `ensureNodePointsCached`,
   `store/jetstream.go:883`) makes the same lane choice: it lists `live_*`
   streams alongside the instance's own and skips replica `inst_*` streams, so
   a lazy load during backfill cannot pull a stale history tip into the cache.
4. `reset` (`store/jetstream.go:1136`) deletes `live_*` streams along with
   `inst_*`. The motivating scenario for this plan is a store reset, and a
   reset that left the old identity's live replicas behind would load them into
   the new instance's cache at the next cold start.

**Verify:** `go test -race ./store/`

**Test:** Extend `TestDbJetStreamReplicaRootNotAdopted` to the live stream form,
so the root-adoption regression is covered on the path that now feeds the cache.
Add a restart test that is the point of the whole plan: publish current tips into
`live_*`, publish *older* history into the replica `inst_*` stream after them,
restart the store on the same directory, and assert the loaded tips are the
current ones. This is the case that fails today and is the reason for the split.
Also assert that `reset` removes `live_*` streams along with `inst_*`.

### Phase 4: End-to-end catch-up test

**Goal:** Prove the property the plan exists for.

1. A `client` test with two instances where the downstream has a large stream and
   the upstream starts empty: assert the upstream serves current values well
   before the history stream has finished filling, and that the history stream
   reaches parity afterward.
2. Assert the upstream's `live_*` stream holds no more than the live retention
   limit per subject rather than the full history, so a retention regression is
   caught here.

**Verify:** `go test -race ./client/ -run TestSync`

### Phase 5: Documentation, dump, and changelog

1. `docs/ref/sync.md` — rewrite the durable-consumer section for two lanes, the
   subject layout table, why history keeps the canonical subjects, and the
   duplication cost stated plainly.
2. `docs/ref/store.md` — the two stream kinds and their retention.
3. `client/dump.go` — label streams `live` and `history` in `-streams` output, so
   a dump shows which lane is behind.
4. `CHANGELOG.md` — a Changed entry describing what an operator sees: an upstream
   that has just been reset shows current values immediately instead of after a
   long replay.
5. `plans/plans.md` — add this plan.

## Files Touched

| File                          | Change                                                             |
| ----------------------------- | ------------------------------------------------------------------ |
| `store/jetstream.go`          | live naming, prefix helpers, cold-start lane split incl. `loadNodePoints`, `reset` deletes `live_*` |
| `store/replica.go`            | consume `live_*`, strip prefix, delete backlog gating, live policy |
| `store/jetstream_test.go`     | live-form root-adoption test, stale-history restart test           |
| `client/sync.go`              | `runHistoryPump` rename, `runLiveTipPump`, both lanes in the scans |
| `client/sync_window_test.go`  | subject rewrite and lane-routing tests                             |
| `client/sync_test.go`         | end-to-end catch-up test                                           |
| `client/dump.go`              | lane labels in `-streams`                                          |
| `docs/ref/sync.md`            | two-lane model, subject table, duplication cost                    |
| `docs/ref/store.md`           | stream kinds and retention                                         |
| `CHANGELOG.md`                | Changed entry                                                      |
| `plans/plans.md`              | index row                                                          |

No frontend change, no schema change, no new dependency. Roughly 200 lines of
implementation and a similar amount of test.

## Risks

**Wire traffic doubles in steady state.** Stated above and accepted for this
plan. The mitigation exists and is deliberately deferred; the number to watch is
bytes per point on a metered link, not message count.

**Stream count doubles on a receiver.** An upstream with fifty devices goes from
fifty replica streams to a hundred. JetStream handles this, but file handles,
consumer count, and `stream ls` legibility all grow, and the practical ceiling
should be measured before a large fleet meets it.

**A live stream that is too small loses messages for a lagging consumer.**
`MaxMsgsPerSubject: 1` would mean a consumer that stalls for one sample interval
misses points. Nothing in this design should consume the live stream for history
— that is what the history lane is for — and the chosen retention of ten per
subject leaves room for a brief stall; a consumer that falls further behind than
that has the history lane.

**The first cold start after upgrading shows remote nodes late.** Startup now
skips replica history, and on the first boot after this change the live streams
do not exist yet, so remote nodes are absent from the cache until the first live
pass completes — seconds after the sync connects, but a visible gap if the peer
is offline at the time. Every later cold start reads the local `live_*` streams
and is complete without a connection. Worth a sentence in the changelog entry.

**Deleting a live stream is safe; deleting a history stream is not.** The
asymmetry is deliberate and useful operationally, but it is a new thing an
operator can get wrong, and it deserves a plain sentence in the store reference
rather than only in this plan.

**The prefix is a translation on the sync path.** The history lane keeps the
no-translation property, but the live lane does not, and a rewrite that can be
wrong now sits between two instances. The round-trip test in Phase 1 is the guard,
and the lane-routing assertion is what stops the two from being crossed silently.

## Open Questions

**Should the live lane idle when history is caught up?** This removes the
doubling in steady state, and `NumPending` already tells the pump when history is
behind. It adds a mode and a threshold to a design that currently has neither, so
it should follow a measurement on a real link rather than land with the plan.

**Should a receiver be able to decline history entirely?** An upstream that only
wants current state — a dashboard instance, or one whose Db client already
carries retention — could run the live lane alone and skip the history stream
completely. That is close to free once the lanes are separate, but it is a point
on the Sync node and therefore a schema change, and it should wait until someone
wants it.

**Does the boundary case of a rarely-updating subject need attention?** A subject
that has not been written since before the live consumer's first pass is
delivered by `DeliverLastPerSubject` correctly, so current state is complete. It
is worth an explicit test rather than an argument, and it is folded into Phase 4.
