# Plan: Stop Polling the Upstream for Streams

**Branch:** `cbrake/master` **Branched from:** `4c0e082d`

## Context

An idle edge device (`edge-bw-test`: one sync node, a user, nothing else)
exchanges about 385 B/s with its upstream, roughly 33 MB a day, while it stores
about six points a minute. Measured over 60 s on the upstream TCP connection:
233 B/s received, 152 B/s sent, matching the `rxBytes`/`txBytes` totals on the
sync node.

Most of that is `scanPulls` (`client/sync.go`). Every `syncPullScanPeriod` (2 s)
it sends `$JS.API.STREAM.NAMES` with the filter `inst.<X>.>` and receives a
265-byte JSON reply naming both streams. That is about 70% of the bytes received
and half of the bytes sent.

The scan exists because the upstream creates `inst_X_R` lazily, on its first
configuration write into the device's subtree, so the stream may not exist when
a session starts. With a single upstream, a device boundary has exactly two
streams, `inst_X_X` and `inst_X_R`, and the device learns `R` from `GetRootNode`
at session start.

Any other origin can only come from multi-hop or nested-boundary sync
([follow-ups](2026-09-19-jetstream-sync-followups.md) items 1 and 2, not
started), so the device can pull `inst_X_R` by name and needs no listing at all.
Pulling one stream by name also lets the upstream drop its per-device origin
tracking (`loadOrigins`, `originsFor`, and the grant check in `enforce` in
`server/auth.go`). It also lets the upstream stop granting
`$JS.API.STREAM.NAMES`, which today tells a credentialed device the ID of every
instance on the upstream.

## Approach

1. The upstream creates `inst_X_R` when it adopts device `X`, so the stream is
   always there when a device connects.
2. The sync client starts one pull pump on `inst_X_R` per session, naming the
   stream directly. It does not list or rescan streams.
3. A device credential grants exactly `inst_X_X` (push) and `inst_X_R` (pull).
   Finding other origins is left to the multi-hop follow-up.

## Phases

### Phase 1: Upstream creates `inst_X_R` at adoption

- [x] When the store persists a new, non-tombstoned edge to a `device` node,
      create `inst_X_R` (`ensureDeviceStreams` in `store/jetstream.go`).
- [x] At store startup, ensure `inst_X_R` for every device boundary already in
      the edge cache (`store/edge_cache.go`), so hubs upgraded with existing
      devices have the stream before those devices reconnect.
- [x] Test (`TestDbJetStreamDeviceStream`): after a device announces itself,
      `inst_X_R` exists on the upstream with no configuration written, and it
      holds no messages.
- [x] Check that the empty stream does not change the grant fingerprint in
      `originsFor`. `R` is always in the set, so it should not reconnect
      devices.
- [x] `go test -race ./store/... ./client/...`, `golangci-lint run`. Commit.

### Phase 2: Pull `inst_X_R` by name

- [x] In `runSession`, after the push pump starts, start the pull pump with
      `runPump(ctx, jsRemote, jsLocal, X, rootRemote.ID, X)`. Then wait on
      `ctx.Done()`.
- [x] Remove `scanPulls`, the `pulls` map, the ticker, and `syncPullScanPeriod`.
- [x] `runPump` creates a missing destination stream but expects the source to
      exist. If `inst_X_R` is missing upstream, the upstream is older than Phase
      1: log it and keep the push pump running, rather than returning an error
      that would end the session. Configuration then arrives after the next
      reconnect. Do not add a polling fallback (no compatibility shims).
- [x] Tests (`client/sync_test.go`):
  - [x] A configuration write on the hub after the device connects reaches the
        device without a reconnect. `TestSync` covers it: its first hub write
        comes after the device connects.
  - [x] Pre-provisioned configuration arrives on first connect
        (`TestSyncPreProvisioned`).
- [x] Docs: `docs/ref/sync.md`, Life of a connection, steps 2 and 4;
      `CHANGELOG.md`, Changed.
- [x] `go test -race ./client/...`, `golangci-lint run`. Commit.

### Phase 3: Narrow the device credential

- [x] `devicePermissions` (`server/auth.go`): grant the pull subjects for
      `inst_X_R` only, and drop `$JS.API.STREAM.NAMES`.
- [x] Remove the per-device origin tracking that only fed the grant: the
      `origins` field, `loadOrigins` and its enqueue on stream changes,
      `originsFor`, and the `grant` comparison in `enforce`.
- [x] Remove `client.StreamBoundaryOrigin`, whose only caller was `loadOrigins`.
- [x] Update `server/auth_test.go` and `server/auth_spike_test.go` for the
      narrower grant. A device asking for `$JS.API.STREAM.NAMES` or another
      origin's stream is refused.
- [x] Docs:
  - [x] `docs/ref/security.md`, What a device credential allows: remove the
        "Discover streams" row and the `STREAM.NAMES` limitation, list only
        `inst_X_R` for pull, and replace the paragraph on enumerating origins
        and reconnecting.
  - [x] `CHANGELOG.md`: the Changed entry is drafted. Add a Security entry
        saying a device can no longer list the instances on its upstream.
- [x] Add a note to follow-ups items 1 and 2: multi-hop and nested boundaries
      need their own way for a device to learn of, and be granted, origins
      beyond its upstream.
- [x] `go test -race ./client/... ./server/...`, `golangci-lint run`. Commit.

### Phase 4: Measure

- [x] Measured on a scratch upstream and device on this machine rather than the
      edge in `/scratch/siot/edge-test2`, whose upstream is the production
      server. Same configuration both times (one sync node, bytes reported every
      60 s), 180 s sampled with `ss -tin` after 30 s to settle:

      | Build                 | Sent    | Received | Per day  |
      | --------------------- | ------- | -------- | -------- |
      | `4c0e082d` (before)   | 128 B/s | 165 B/s  | 25.3 MB  |
      | Phase 3 (after)       | 38 B/s  | 31 B/s   | 6.0 MB   |

- [x] Replace the estimate in the changelog with the measured saving. Commit.
- [ ] Confirm on the `edge-sync` Grafana dashboard for `edge-bw-test` once the
      edge and its upstream run this release.
- [ ] If the remainder still matters on cellular, list the next contributors
      (pull consumer idle heartbeats, long `inst.*` publish subjects,
      once-a-minute metrics and byte reports) as follow-ups rather than
      expanding this plan.

## Incompatibilities for the changelog

- A device on this release that connects to an older upstream with no
  configuration yet written for it has no `inst_X_R` to pull. It receives that
  configuration only after it next reconnects. Upgrade the upstream along with
  its devices.
- A credentialed device on an older release receives no configuration from an
  upstream on this release. It finds `inst_X_R` with `$JS.API.STREAM.NAMES`,
  which the upstream no longer grants; its own data still reaches the upstream.
  Upgrade devices along with their upstream.
