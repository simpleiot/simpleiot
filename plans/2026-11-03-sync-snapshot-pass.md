# Plan: Send Current Values First on Sync Catch-up

**Branch:** `cbrake/master` **Branched from:** `e8d0390b` **Issue:**
[simpleiot#836](https://github.com/simpleiot/simpleiot/issues/836)

## Context

A pump (`runPump` in `client/sync.go`) copies a boundary-origin stream upstream
in stream order from a durable consumer. After a device has been offline, the
upstream learns the current value of a subject only when the backlog reaches it:
a day of 1 Hz data on a slow link can leave the upstream showing stale values
for hours.

Issue #836 proposes separate current and history streams. Writing every point to
two streams would send every point twice over the link. This plan keeps one
stream per boundary/origin on disk and splits the _sync_ into two passes
instead: a snapshot of the current value of each subject that changed while
offline, followed by each point written while catching up, alongside the history
as now.

![catch-up today and with the snapshot pass](images/snapshot-catchup.svg)

The cost is one extra message per changed subject per reconnect, plus one extra
message for every point written while the history is catching up. Each of those
crosses the link in the snapshot and again when the history reaches it.
Unchanged subjects cost nothing. The second copy is a no-op on the receiving
side: `tipWins` (`store/merge.go`) does not replace a tip with an equal time
from the same origin.

Snapshot messages do not enter the replica stream, so the stream keeps the ADR-7
invariant that the last message per subject is the current value, and the
ordered `DeliverLastPerSubject` consumer in `consumeReplica`
(`store/replica.go`) is unchanged. They are merged into the receiving store's
cache and broadcast locally, like a live replica message. If the receiving
instance restarts before the history catches up, its cache rebuilds from the
stream and the tips arrive again with the history; this is accepted for now.

This changes what clients on the receiving instance see during catch-up. Today a
rule on the upstream receives every backlog point in order, hours late on a slow
link, because each one is newer than the cached tip and is broadcast. With the
snapshot pass it receives the current value at once, and the older history
messages that follow are neither merged nor broadcast: `tipWins` rejects a point
older than the tip, or equal in time from the same origin. A rule that reacts to
a short-lived condition inside the backlog no longer fires for it upstream; the
device's own rules evaluated it live. The history pass does not wait for the
snapshot, so a history message can occasionally be broadcast before the snapshot
of its subject arrives; values still only move forward in time. The database
client is not affected: it reads the streams with its own durable consumers
(`client/db.go`), not the broadcasts, so it still records the full history.

## Approach

Each pump runs a snapshot pass alongside the history pass until the history
catches up:

1. Create (or find) the durable consumer as now and read its ack floor
   (`ConsumerInfo.AckFloor.Stream`).
2. Open an ephemeral consumer on the source stream with
   `DeliverLastPerSubjectPolicy`, filtered to the stream's capture subject. It
   delivers the newest message on each subject, then every message written after
   it was created.
3. Publish each delivered message whose stream sequence is above the ack floor
   on the receiving side as a core NATS message on
   `snap.<boundary>.<origin>.<rest>`: the storage subject with `snap` in place
   of `inst`, same payload, same `OriginHeader`. No acknowledgement; the history
   pass carries the same message durably. Points written during catch-up are
   forwarded as they arrive, one for one, with no coalescing.
4. Start the history pass as now, without waiting for the snapshot.
5. When the history pass sees nothing pending (`NumPending == 0` in
   `fillWindow`), delete the ephemeral consumer. From then on the history pass
   alone keeps the upstream current, as it does today.

![pump snapshot pass and the upstream store](images/snapshot-flow.svg)

A new durable (first sync, ack floor 0) snapshots every subject, so a newly
adopted device shows its full current state on the upstream before any history
is copied.

The receiving store subscribes to `snap.>`, rewrites the first token to `inst`,
and calls `mergeReplicaMsg`; when that returns true it calls `broadcastReplica`.
A snapshot whose origin is this instance's own root ID is dropped: `replicaWait`
trusts streams this instance wrote, and a snapshot must not be able to claim
that trust.

Both directions use the same code. The pull pump snapshots `inst_X_R` into the
device, which matters less (configuration streams are small) but keeps one pump
implementation.

## Phases

### Phase 1: Store accepts snapshot messages

- [x] `store/replica.go`: subscribe to `snap.>` on the store's connection.
      Handler: reject `origin == db.meta.RootID`, rewrite the subject prefix,
      `mergeReplicaMsg(subject, payload, origin, true)`, broadcast if changed.
- [x] Test (`store/replica_test.go` or `store_test.go`): a snapshot for a
      boundary this instance replicates updates the cache and is broadcast on
      the `p.` subject; a later, older history message for the same subject does
      not change the tip and is not broadcast (subscribe to `up.>` and count);
      the history copy of the snapshot point itself is not broadcast again; a
      snapshot claiming this instance's own origin is ignored.
- [x] `go test -race ./store/...`, `golangci-lint run`. Commit.

### Phase 2: Pump snapshot pass

- [x] `client/sync.go`: in `runPump`, after `CreateOrUpdateConsumer`, read the
      ack floor, start the snapshot loop (`runSnapshot`) in its own goroutine
      when the durable consumer has a backlog, and start the window loop. The
      window loop stops the snapshot loop the first time a window ends with
      nothing pending. Log the ack floor and the count of messages sent as
      snapshots at `Sync <desc>:` level like the other session messages.
- [x] The snapshot loop never holds up the history pass. On an error, it logs
      and stops; the history pass carries on alone. Its consumer is left to its
      inactivity threshold (30 s) rather than deleted, since a device is not
      granted consumer deletion on its upstream.
- [x] Stop the snapshot for the session when the receiving server refuses the
      `snap.` subject. Instead of an async error handler, the first snapshot
      message is followed by a flush: nats.go records a permissions violation on
      the connection before it processes the server's PONG, so the refusal is
      known before a second message goes out, and an older upstream logs one
      violation per session.
- [x] Tests (`client/sync_snapshot_test.go`, two embedded JetStream servers,
      with a test-only delay between windows to keep a backlog in flight): after
      an earlier session, the current values of the changed subjects arrive
      before the history catches up, and a subject unchanged since that session
      is not sent; a point written during catch-up is sent before the history
      reaches it; the snapshot consumer goes away once the history has caught
      up, and later points are not sent as snapshots. First sync of a new device
      snapshots every subject. Against a server whose grant lacks `snap.>`, the
      snapshot stops after the first message and the history still catches up.
      Merging into the upstream's cache is covered by Phase 1, and end to end by
      the existing `TestSyncOfflineCatchup`.
- [x] `go test -race ./client/... ./store/...`, `golangci-lint run`. Commit.

### Phase 3: Permissions and docs

- [x] `server/auth.go` `devicePermissions`: grant publish on `snap.X.X.>` beside
      `inst.X.X.>`. The `jetstream` package names an ephemeral consumer itself,
      so `$JS.API.CONSUMER.CREATE.inst_X_R.>` and `MSG.NEXT.inst_X_R.*` already
      cover it. `CONSUMER.DELETE` is not granted; the consumer is left to its
      inactivity threshold.
- [x] `server/auth_spike_test.go`: a device credential can publish `snap.X.X.>`
      and read `inst_X_R` through an ephemeral last-per-subject consumer, and
      its `snap.Y.Y.>` and `snap.X.R.>` publishes land nowhere.
      `TestDevicePermissions` lists the new subject.
- [x] `docs/ref/sync.md`: add the snapshot pass to the pump description and the
      "Offline catch-up" section. `docs/ref/security.md`: add the subject to the
      device grant list. `docs/adr/7-jetstream-store.md`: a short note under the
      sync section that current values are sent ahead of history and why the
      streams are not split.
- [x] Adapt this plan's diagrams for `docs/ref/sync.md` ("Offline catch-up"),
      saved in `docs/ref/images/`:
  - `snapshot-catchup.svg` becomes `sync-catchup.svg`: keep the device stream
    and the snapshot lane with the broadcast marks, and drop the "today" lane,
    since the docs describe the current behavior.
  - `snapshot-flow.svg` becomes `sync-snapshot-flow.svg`: drop the "new ·" tags
    and the "as now" and "unchanged" notes.
  - `sync-connection.svg`: show the snapshot running alongside the history in
    the catch-up part of the sequence.
  - Add the new diagrams to the gallery in `docs/ref/diagrams.md`, following its
    style guide.
- [x] `CHANGELOG.md`: current values arrive first after a reconnect, and rules
      on the upstream see them instead of a replay of the backlog. Upgrade the
      upstream first. Older and newer devices and servers work together, and
      stored data is unchanged. A new device with an older upstream works but
      gets no benefit for its own data, and the upstream logs a permissions
      violation each time the device reconnects. With the shared token, an older
      upstream drops the snapshot messages silently, so they only add traffic
      during catch-up until it is upgraded.
- [x] `siot_test`. Commit. Close #836 with a link to this plan (left for the
      maintainer, after the branch is pushed).

## Not in this plan

- A current-values stream on disk on the upstream, so a restart during a long
  catch-up keeps the tips. Revisit if restarts during catch-up turn out to
  matter.
