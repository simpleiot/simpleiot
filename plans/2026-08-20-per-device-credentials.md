# Plan: Per-Device Credentials

**Branch:** `cbrake/master` **Branched from:** `cbc07a67`

## Context

Every device that syncs to an upstream today presents the same shared token
(`SIOT_AUTH_TOKEN`), and the token grants full access to the upstream's NATS
server. The consequences:

- Revoking one device means rotating the token on the server and on every other
  device.
- A device (or anyone holding its token) can publish as any other device, read
  any instance's configuration, and use the admin subjects.
- An exported sync node carries the fleet-wide secret.

The stream-per-boundary store layout (ADR-7) was designed so that a device's
entire upstream footprint is a small, enumerable set of subjects keyed by its
own root ID `X`. That makes a one-rule-per-device grant possible without any
change to the data model. This plan adds the credential, the authorizer that
issues the grant, the workflows for getting a credential onto a device, and
revocation.

The user-facing behavior this plan implements is already described in
[Per-device credentials (planned)](../docs/user/sync.md#per-device-credentials-planned)
and the [security reference](../docs/ref/security.md).

### What a device touches on the upstream today

From `client/sync.go` (`runSession`, `scanPulls`, `runPump`) and
`client/node.go`, a device with root `X` syncing to an upstream with root `R`
uses:

| Purpose                           | Subject(s)                                                                          |
| --------------------------------- | ----------------------------------------------------------------------------------- |
| Find upstream root                | request `nodes.root.all`                                                            |
| Check whether it is adopted       | request `nodes.all.X`                                                               |
| Announce itself (adoption)        | publish `ep.X.R`                                                                    |
| Push its origin stream            | publish `inst.X.X.>` (captured by `inst_X_X`)                                       |
| Create / inspect replica stream   | `$JS.API.STREAM.INFO.inst_X_X`, `$JS.API.STREAM.CREATE.inst_X_X`                    |
| Discover streams for its boundary | `$JS.API.STREAM.LIST` / `$JS.API.STREAM.NAMES` (filtered by `inst.X.>`)             |
| Pull upstream-origin streams      | `$JS.API.STREAM.INFO.inst_X_*`, `$JS.API.CONSUMER.*.inst_X_*`, `$JS.ACK.inst_X_*.>` |
| Reply inboxes                     | subscribe `_INBOX.>`                                                                |

Nothing else. In particular a device never needs `p.>`, `up.>`, `auth.*`,
`admin.*`, or any `inst.<other>.>` subject. A credential scoped to the rows
above is sufficient for sync to work and prevents a device from touching any
other instance's data.

## Design Decisions

**The credential is an NKey user keypair, not a password.** The device holds the
seed; the upstream holds only the public key. A compromised upstream store or
export leaks nothing that lets anyone impersonate a device, and revocation is
deleting or disabling the public key. nats.go (`nats.Nkey`) and nats-server
already implement the nonce challenge, and `github.com/nats-io/nkeys` is already
in `go.mod`. The shared token remains available for the transition and for
deployments that do not need isolation; see "Compatibility" below.

**Authentication is done in-process with
`server.Options.CustomClientAuthentication`.** The embedded server calls our
`Check(ClientAuthentication) bool` for every connection on every listener (TCP,
WebSocket, and later MQTT). This lets the authorizer look the public key up in
the upstream's own node tree and return permissions computed from the device ID,
with no accounts file, no NKey operator hierarchy, and no auth-callout service
account to manage. Auth callout remains an option for an external NATS
deployment later; the permission set is the same either way.

One consequence shapes the implementation: once `CustomClientAuthentication` is
set, the built-in token, user/password, and NKey checks are bypassed for all
listeners, including `Websocket.Token`. The authorizer must therefore implement
the shared-token check itself, and the server's own client, `siot` CLI
connections, and existing devices must keep working unchanged.

**Credentials are nodes in the tree.** A `deviceCred` node lives under the
device node it authorizes on the upstream, with points `pubKey`, `disabled`,
`description`, and status points the authorizer maintains (`lastConnect`,
`connected`). This means issuing, disabling, and deleting credentials works
through every existing path: UI, `siot import`, provisioning files, export, and
sync to a higher upstream. The authorizer keeps an in-memory index
(`pubKey → {deviceID, disabled}`) that it rebuilds from the store at start-up
and maintains from `up.root.>` point messages, so `Check` never does a NATS
request (it runs inside the server's connection handling).

**Permissions are derived from the device ID, not stored.** A credential under
device `X` authorizes exactly the subject set in the table above with `X`
substituted. There is nothing to configure and nothing that can drift.

**Revocation disconnects live sessions.** Setting `disabled` or tombstoning a
credential removes it from the index and the authorizer disconnects any
connection authenticated with it (`Server.Connz` with `User: true` to find the
connection ID, then `Server.DisconnectClientByID`). The device's sync client
sees a disconnect, retries, and is refused; it keeps running standalone and logs
the refusal once per backoff period rather than on every attempt.

**Rotation is two live credentials.** A device node may have more than one
`deviceCred`. Rotation is: add a new one, deliver the seed to the device, wait
for `lastConnect` on the new credential, disable the old one. No special
rotation state machine.

**Two provisioning workflows, sharing one credential format.**

1. _Upstream-issued._ The operator creates the credential on the upstream (UI
   button or `siot cred add`), receives the seed once, and delivers it to the
   device's sync node (UI, `siot import`, or a provisioning file). Suited to
   small fleets and bench setup.
2. _Device-initiated enrollment._ The device generates its own keypair on first
   boot and connects with a fleet-wide _enrollment token_ whose only permission
   is to publish an enrollment request. The upstream records a pending
   credential; an operator approves it (or the enrollment token is marked
   auto-approve). Suited to images: every unit ships the same enrollment token
   and ends up with a unique key, and the enrollment token can be revoked
   without affecting enrolled devices. This is Phase 6 and can ship after the
   rest.

**User passwords are out of scope but noted.** `data.User.Pass` is stored in
plaintext today. Hashing it is a separate change that this plan does not depend
on; it is listed under follow-ups so it is not forgotten.

## Compatibility

- `SIOT_AUTH_TOKEN` continues to work exactly as today and grants full access. A
  deployment that sets nothing still runs open, as today, with the existing
  warning.
- A sync node with only `authToken` set behaves as before. A sync node with
  `nkeySeed` set uses the NKey and ignores `authToken` for the NATS connection.
- A new server flag/env `--deviceAuth=required` (`SIOT_DEVICE_AUTH`) refuses the
  shared token from non-local clients so an operator can finish a migration by
  turning the token off for devices while keeping it for the server's own client
  and the CLI. Default is `optional`.
- The wire format, stream layout, and node schema of existing nodes do not
  change. `deviceCred` is a new node type; `nkeySeed` is a new point on sync
  nodes.

## Phases

Commit after each phase. Each phase updates `CHANGELOG.md` and the docs it
affects.

### Phase 0: Spike (no product code)

Goal: prove the four things the design depends on before building on them. Lives
in `server/auth_spike_test.go`, kept as a regression test once green.

1. An embedded server with `CustomClientAuthentication` accepts a token client
   with full permissions and an NKey client with a scoped permission set, and
   refuses an unknown NKey. Verify the nonce signature check using
   `ClientAuthentication.GetOpts()` (`Nkey`, `Sig`) and `GetNonce()`.
2. Under the scoped permission set, the existing sync pumps work end to end:
   `runPump` push into a replica stream, pull consumer on `inst_X_R`, stream
   discovery with a subject filter. Confirm exactly which `$JS.API.*` subjects
   are needed (the table above is the expected answer; the spike is the
   authority).
3. A scoped client that publishes to `inst.Y.Y.>` or requests
   `$JS.API.STREAM.INFO.inst_Y_Y` gets a permissions violation and the message
   does not land.
4. `Connz(&ConnzOptions{User: true})` reports the NKey public key per connection
   and `DisconnectClientByID` closes it.

Also confirm the WebSocket listener routes through the custom authenticator and
that the MQTT listener (when the MQTT plan lands) does as well.

### Phase 1: Authorizer and credential nodes

- `data/schema.go`: `NodeTypeDeviceCred = "deviceCred"`, point types `pubKey`,
  `lastConnect`, `connected`, `nkeySeed`, `deviceAuth`.
- `server/auth.go`: `type authorizer struct` implementing
  `server.Authentication`:
  - `Check`: local/loopback server client and shared token → full permissions;
    NKey in index and not disabled → `devicePermissions(deviceID)`; otherwise
    refuse. Honors `--deviceAuth=required`.
  - `devicePermissions(X string) *server.Permissions` returning the publish and
    subscribe allow lists from the table, plus `allow_responses` so request
    replies work.
  - Index maintenance: load all `deviceCred` nodes at start-up (including the
    device they sit under — the parent edge gives `deviceID`), then subscribe
    `up.root.>` and apply `pubKey`, `disabled`, and tombstone changes. Moving a
    credential to another parent re-keys it.
  - Revocation hook: when an entry is removed or disabled, find and disconnect
    matching connections.
  - Status: on successful `Check`, queue a `lastConnect`/`connected` point
    update for the credential node (sent from a goroutine, not from `Check`).
- `server/nats-server.go`: set `opts.CustomClientAuthentication`; drop
  `Authorization` and `Websocket.Token` since the authorizer now owns both. Keep
  the start-up log line reporting whether auth is enabled and add the
  device-auth mode.
- The authorizer needs the store before the NATS server accepts clients, but the
  store needs a NATS connection. Resolve by starting the server with the
  authorizer in "token only" mode, loading the index once the store is up, then
  flipping a flag that enables NKey acceptance. Device connections that arrive
  in between are refused and retry.
- `client/sync.go` and `client/edge.go`: `NkeySeed` field on `Sync`,
  `EdgeOptions.NkeySeed`; `EdgeConnect` uses `nats.Nkey(pub, sign)` when set.
  The seed is parsed once and zeroed from the config struct copy after the
  signer is built.
- `siot key gen`: prints a new user seed and public key (wraps
  `nkeys.CreateUser`). Used by tests, scripts, and the workflows below.

Tests (`server/auth_test.go`, extend `client/sync_test.go`):

- Table test for `devicePermissions` against the subject table.
- Two-instance sync test where the downstream uses an NKey credential: adoption,
  push, pull, and offline catch-up all pass (reuse the three existing sync tests
  with a credential-aware variant of the setup).
- Negative: a downstream using device Y's seed under device X's tree is refused;
  a downstream whose credential is `disabled` is refused; a credential under the
  wrong parent authorizes nothing useful.
- Revocation: disable the credential mid-session, assert the remote connection
  closes within a bounded time, re-enable, assert it reconnects and resumes.
- Shared token still works for the server client, CLI, and a token-only
  downstream; `--deviceAuth=required` refuses a token-only downstream.

Docs: `docs/ref/security.md` (move the planned section to present tense, add the
permission table), `docs/user/configuration.md` (`SIOT_DEVICE_AUTH`),
`docs/user/sync.md` schema section (`nkeySeed`).

### Phase 2: Upstream-issued credential workflow

- UI (`frontend/src/Components/NodeDeviceCred.elm`, and an "Add credential"
  entry on device nodes): shows `pubKey`, `disabled`, `lastConnect`, and
  `connected`. A "Generate" action on a new credential node asks the backend to
  create a keypair; the seed is returned once in the HTTP response and shown
  with a copy button and a notice that it will not be shown again. The backend
  stores only the public key.
  - HTTP: `POST /v1/nodes/{deviceID}/credentials` in `api/`, JWT-protected like
    the rest of the node API, returning `{id, pubKey, seed}`.
- CLI: `siot cred add --device <id-or-description> [--description ...]` prints
  the seed and public key; `siot cred list`, `siot cred disable <id>`,
  `siot cred rm <id>`. These are thin wrappers over node operations plus
  `siot key gen`, so they work against a remote upstream with the usual
  `--natsServer`/`--token` flags.
- `client/sync.go`: `NodeSync.elm` gains an `nkeySeed` field that is write-only
  in the UI (shown as set / not set, never echoed back).
- Provisioning and import: a device's provisioning file carries `nkeySeed` on
  its sync node like any other point. Document that such a file is a secret and
  that per-unit files (`provisioning/<unit>.yaml`) are the intended layout.
- Export: `deviceCred` nodes export their public key (harmless). The device-side
  sync node exports `nkeySeed` only with a new `siot export --secrets` flag;
  without it the point is omitted and a comment notes the omission, so a casual
  export no longer carries a credential.

Tests:

- API test for the credential endpoint (seed returned once, store has only the
  public key).
- Import a provisioning file containing a sync node with `nkeySeed` and confirm
  the downstream connects.
- Export without `--secrets` omits `nkeySeed` and `authToken`; with it includes
  both.

Docs: `docs/user/sync.md` (replace the planned section with the workflow),
`docs/user/configuration.md` provisioning section, `docs/ref/api.md`.

### Phase 3: Revocation and rotation polish

- Device-side behavior on refusal: `EdgeConnect` error handler distinguishes
  `nats: Authorization Violation` from network errors; the sync node shows a
  `lastError` of "credential refused by upstream" rather than a generic
  reconnect count, and backoff is longer (the key is not going to start working
  by retrying fast).
- Upstream: when a device node is tombstoned (detach, already supported), its
  credentials are disconnected too, since a detached device must not keep
  writing. Undelete restores them.
- Rotation: document the two-credential procedure and add a test that rotates
  without losing messages (publish continuously on the device while swapping
  seeds; assert the upstream replica has every message).
- `siot cred rotate --device <id>` as a convenience: creates the new credential,
  prints the seed, and leaves the old one enabled for the operator to disable
  after `lastConnect` moves.

### Phase 4: Scope the HTTP device API

`api/` accepts the shared token for device HTTP access (`docs/ref/security.md`
"HTTP"). Devices that sync use NATS, so this path is secondary, but it is a
full-access token on the wire. Accept a signed NKey JWT
(`Authorization: Bearer <jwt>` signed with the device seed, issuer = public key,
short expiry) through the same authorizer index and apply the same device
scoping to the node endpoints (only nodes under `X`). Leave the shared token
working under `--deviceAuth=optional`.

This phase is independent of Phases 2 and 3 and can be reordered or dropped if
the HTTP device API is retired instead.

### Phase 5: Stream-side validation

Defense in depth from the Stage 3 plan (section 9): the upstream store, when
consuming a replica stream `inst_X_X`, already only persists what arrives in
that stream. Add a check that the connection that created or publishes into
replica stream `inst_X_X` is authorized for boundary `X` is not possible at the
store level (JetStream does not carry the publisher identity), so the permission
set is the enforcement point. What the store can do: refuse to create a replica
stream whose boundary is not a known device, and log at warning when a replica
stream's message subject does not match its boundary. Small phase; mostly tests
and a log line.

### Phase 6: Device-initiated enrollment

- `enrollToken` node under the upstream root: `token` (random, shown once,
  stored hashed), `disabled`, `autoApprove`, `expires`, optional `parent` (where
  approved devices are placed; default root).
- Authorizer: a connection presenting an enrollment token gets a permission set
  of exactly publish `enroll.request` and subscribe `_INBOX.>`.
- Device side: a sync node with neither `authToken` nor `nkeySeed` but with
  `enrollToken` generates a keypair on first run, stores the seed as `nkeySeed`
  on its own sync node (local store, so it survives restart), connects with the
  enrollment token, publishes `{deviceID, pubKey, description}` to
  `enroll.request`, and then polls (reconnects with the NKey every backoff
  period) until the credential is approved.
- Upstream: the enrollment handler creates the device node (tombstoned, i.e.
  pending, so it does not appear as live) with a `deviceCred` carrying the
  public key and `pending: 1`. With `autoApprove` the node is created live
  immediately. The UI lists pending enrollments on the upstream root with
  Approve / Reject. `siot cred approve <deviceID>` for scripts.
- Revoking the enrollment token does not affect already-approved devices.

Tests: full enrollment round trip with and without auto-approve; a revoked
enrollment token is refused; an enrollment-token connection cannot publish to
anything but `enroll.request`; a second enrollment from the same device ID with
a different key is held as pending and does not replace the approved key.

Docs: `docs/user/sync.md` enrollment section, `docs/user/installation.md`
image-build notes.

### Phase 7: Wrap-up

- `plans/plans.md` status, `CLAUDE.md` note on the authorizer, changelog review.
- Mark the docs planned sections as current.
- Open follow-ups as issues: hash user passwords; auth-callout variant for
  external NATS; MQTT credentials share the authorizer (tracked in
  `2026-08-20-mqtt.md`).

## Testing Strategy

- All new server behavior is covered by two-instance tests using
  `server.TestServer` / `TestServerOpts`, the same harness the sync tests use,
  so credential tests exercise the real authorizer, real JetStream, and real
  pumps. Add `TestServerOptions` fields for `DeviceAuth` and a helper that
  returns a pre-enrolled device (seed, public key, credential node) to keep each
  test short.
- Negative tests assert on the NATS error (`nats.ErrAuthorization`,
  permissions-violation async error) and on the absence of the message in the
  target stream, not only on the error, so a permission that fails open is
  caught.
- The spike test from Phase 0 stays in the suite as the canary for nats-server
  upgrades changing custom-auth behavior.
- `go test -race ./...` and `golangci-lint run` after each phase; `siot_test`
  before the wrap-up commit.

## Open Questions

- Should `$JS.API.STREAM.LIST` be allowed at all? It discloses every stream name
  (instance IDs) on the upstream even with a subject filter. Alternative: the
  device asks the upstream for its stream list through a small request handler
  (`sync.streams.X`) that the store answers, and the permission set drops
  `STREAM.LIST`. Decide in Phase 0 once the spike shows whether `STREAM.NAMES`
  with a filter leaks the same way.
- Should the upstream pre-create `inst_X_X` when a credential is issued so the
  device never needs `STREAM.CREATE`? Cleaner permission set, one more step in
  the issue path. Leaning yes; decide in Phase 1.
- Where does a device store its seed when it enrolls itself: as a point on its
  sync node (exported with `--secrets`, survives via the store) or as a file in
  `SIOT_DATA`? The point keeps one mechanism; the file keeps the seed out of the
  store entirely. Leaning point, for consistency with upstream-issued seeds.

## Key Files

- `server/nats-server.go`, `server/server.go`, `server/args.go`: wire the
  authorizer and the new flag.
- `server/auth.go` (new), `server/auth_test.go` (new),
  `server/auth_spike_test.go` (new).
- `client/sync.go`, `client/edge.go`, `client/sync_test.go`: NKey connection and
  credential-aware tests.
- `client/device-cred.go` (new): credential node type and index loading helpers.
- `data/schema.go`: node and point types.
- `cmd/siot/main.go`: `key gen`, `cred *` subcommands, `export --secrets`.
- `api/`: credential endpoint; Phase 4 device JWT.
- `frontend/src/Components/NodeDeviceCred.elm` (new), `NodeSync.elm`,
  `NodeDevice.elm`.
- `docs/user/sync.md`, `docs/ref/security.md`, `docs/user/configuration.md`,
  `docs/ref/api.md`, `CHANGELOG.md`.
