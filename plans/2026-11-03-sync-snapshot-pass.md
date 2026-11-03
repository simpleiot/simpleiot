# Plan: Send Current Values First on Sync Catch-up

**Branch:** `cbrake/master` **Branched from:** `e8d0390b`
**Issue:** [simpleiot#836](https://github.com/simpleiot/simpleiot/issues/836)

## Context

A pump (`runPump` in `client/sync.go`) copies a boundary-origin stream upstream
in stream order from a durable consumer. After a device has been offline, the
upstream learns the current value of a subject only when the backlog reaches
it: a day of 1 Hz data on a slow link can leave the upstream showing stale
values for hours.

Issue #836 proposes separate current and history streams. Writing every point
to two streams would send every point twice over the link. This plan keeps one
stream per boundary/origin on disk and splits the _sync_ into two passes
instead: a snapshot of the current value of each subject that changed while
offline, then the history as now.

The cost is one extra message per changed subject per reconnect. The newest
point of a changed subject crosses the link in the snapshot and again when the
history reaches it. Unchanged subjects cost nothing. The second copy is a no-op
on the receiving side: `tipWins` (`store/merge.go`) does not replace a tip with
an equal time from the same origin.

Snapshot messages do not enter the replica stream, so the stream keeps the
ADR-7 invariant that the last message per subject is the current value, and
the ordered `DeliverLastPerSubject` consumer in `consumeReplica`
(`store/replica.go`) is unchanged. They are merged into the receiving store's
cache and broadcast locally, like a live replica message. If the receiving
instance restarts before the history catches up, its cache rebuilds from the
stream and the tips arrive again with the history; this is accepted for now.

## Approach

Each pump runs a snapshot pass before it starts copying history:

1. Create (or find) the durable consumer as now and read its ack floor
   (`ConsumerInfo.AckFloor.Stream`).
2. Open an ephemeral consumer on the source stream with
   `DeliverLastPerSubjectPolicy`, filtered to the stream's capture subject.
   Read until nothing is pending.
3. For each tip whose stream sequence is above the ack floor, publish it on the
   receiving side as a core NATS message on `snap.<boundary>.<origin>.<rest>`:
   the storage subject with `snap` in place of `inst`, same payload, same
   `OriginHeader`. No acknowledgement; the history pass carries the same
   message durably.
4. Delete the ephemeral consumer and start the history pass as now.

A new durable (first sync, ack floor 0) snapshots every subject, so a newly
adopted device shows its full current state on the upstream before any history
is copied.

The receiving store subscribes to `snap.>`, rewrites the first token to `inst`,
and calls `mergeReplicaMsg`; when that returns true it calls
`broadcastReplica`. A snapshot whose origin is this instance's own root ID is
dropped: `replicaWait` trusts streams this instance wrote, and a snapshot must
not be able to claim that trust.

Both directions use the same code. The pull pump snapshots `inst_X_R` into the
device, which matters less (configuration streams are small) but keeps one
pump implementation.

## Phases

### Phase 1: Store accepts snapshot messages

- [ ] `store/replica.go`: subscribe to `snap.>` on the store's connection.
      Handler: reject `origin == db.meta.RootID`, rewrite the subject prefix,
      `mergeReplicaMsg(subject, payload, origin, true)`, broadcast if changed.
- [ ] Test (`store/replica_test.go` or `store_test.go`): a snapshot for a
      boundary this instance replicates updates the cache and is broadcast on
      the `p.` subject; a later, older history message for the same subject
      does not change the tip; a snapshot claiming this instance's own origin
      is ignored.
- [ ] `go test -race ./store/...`, `golangci-lint run`. Commit.

### Phase 2: Pump snapshot pass

- [ ] `client/sync.go`: in `runPump`, after `CreateOrUpdateConsumer`, read the
      ack floor, run the snapshot pass (`snapshotTips`), then start the window
      loop. Log the count of tips sent and the ack floor at `Sync <desc>:`
      level like the other session messages.
- [ ] The snapshot pass must not block the session on an unreachable
      receiver longer than one pull expiry; on error, log and go on to the
      history pass.
- [ ] Tests (`client/sync_test.go`): with the upstream stopped, write N points
      to one subject and one point to a second subject on the device; on
      reconnect the upstream's cache shows both tips before the history
      consumer's pending count reaches zero. A subject unchanged since the
      last session is not in the snapshot (count tips on the `snap.` subject).
      First sync of a new device snapshots every subject.
- [ ] `go test -race ./client/... ./store/...`, `golangci-lint run`. Commit.

### Phase 3: Permissions and docs

- [ ] `server/auth.go` `devicePermissions`: grant publish on `snap.X.X.>`
      beside `inst.X.X.>`. `$JS.API.CONSUMER.CREATE`/`INFO`/`MSG.NEXT` on
      `pull` already cover the ephemeral consumer; confirm an ephemeral
      consumer needs no further subject, and that `CONSUMER.DELETE` is granted
      or the consumer is left to its inactivity threshold.
- [ ] `server/auth_spike_test.go` or `sync_cred_test.go`: a device credential
      can publish `snap.X.X.>` and cannot publish `snap.Y.Y.>` or
      `snap.X.R.>`.
- [ ] `docs/ref/sync.md`: add the snapshot pass to the pump description and
      the "Offline catch-up" section. `docs/ref/security.md`: add the subject
      to the device grant list. `docs/adr/7-jetstream-store.md`: a short note
      under the sync section that current values are sent ahead of history
      and why the streams are not split.
- [ ] `CHANGELOG.md`: current values arrive first after a reconnect. Both
      sides need this version: an older upstream denies the `snap.` publish
      and the device falls back to history order only, with a permissions
      violation in the upstream log.
- [ ] `siot_test`. Commit. Close #836 with a link to this plan.

## Not in this plan

- A current-values stream on disk on the upstream, so a restart during a long
  catch-up keeps the tips. Revisit if restarts during catch-up turn out to
  matter.
- Snapshot priority during the history pass (sending new tips ahead of the
  backlog while catch-up is still running). The history pass's window loop
  would need a second consumer interleaved with it.
