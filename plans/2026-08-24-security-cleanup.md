# Plan: Security Cleanup

**Branch:** `cbrake/master` **Branched from:** `179d06ea`

## Context

The security review recorded in [`security.md`](../../security.md) found a set
of issues that are independent of the per-device credential design in
[2026-08-20-per-device-credentials.md](2026-08-20-per-device-credentials.md).
Each is small, self-contained, and worth landing **before** the credentials
plan: they close holes that credentials do not address, and several of them
(password hashing, secret redaction) change interfaces the credentials work will
build on, so doing them first avoids rework.

Every item below stands alone. They can land as individual commits in any order,
though the listed order puts the highest-impact fixes first.

The [UI over NATS plan](2026-09-01-ui-over-nats.md) (complete, September 2026)
changed the ground under several items: browsers now hold a NATS connection
scoped to the user, `auth.getNatsURI` and `GET /v1/nodes` are gone, and the
store has a subtree check (`isUnder` in `store/jetstream.go`) that the HTTP
routes could reuse. Each affected item says what that leaves.

## Checklist

### 1. Hash user passwords (complete)

- [x] Store user passwords as bcrypt hashes instead of plaintext.

**Problem:** `store/jetstream.go:1050` compares `u.Pass == password` directly;
the plaintext value is stored as an ordinary point (`data/user.go:23`),
replicated upstream by sync, returned to any browser in node responses, and
included in `siot export`.

**Change:**

- On password set (UI, import, provisioning), hash with
  `golang.org/x/crypto/bcrypt` before the point is written. bcrypt rather than
  argon2id for consistency with the `mqttUser` sketch in the MQTT plan.
- `userCheck` verifies with `bcrypt.CompareHashAndPassword` (constant-time by
  construction).
- Migration: on successful login where the stored value is not a bcrypt hash (no
  `$2` prefix), rehash and rewrite the point. Existing plaintext values keep
  working until each user's next login, and no bulk migration pass is needed.
- The UI password field becomes set-only: it writes a new value but never
  displays the stored one (`frontend/src/Components/NodeUser.elm`).

**Verify:** unit test covering set → login → point contains a bcrypt hash; login
with a legacy plaintext point succeeds once and leaves a hash behind; wrong
password fails.

### 2. Redact secret points from read paths

- [ ] Strip secret-valued points (`pass`, `authToken`, and any future seed
      point) from node read responses.

**Problem:** the `nodes.*.*` NATS subjects, and the `u.<anchor>.<user>.nodes.*`
subjects the browser reads through, return all points, so the tree a browser
fetches carries every user's password hash and the sync node's upstream token.
Hashing (item 1) reduces the damage for `pass` but the sync `authToken` is still
a live credential. `auth.me` already strips `pass` from its reply; nothing else
does. (`GET /v1/nodes`, which shipped the whole tree on every poll, is gone.)

**Change:** a single redaction function applied where nodes are serialized for
clients (HTTP node responses, the `nodes.*.*` reply path, and the `u.*` dispatch
in `store.handleUserRequest`, which calls the same handler), keyed on a small
list of secret point types. Writes are unaffected. `siot export` already omits
`authToken` unless `-secrets` is given.

**Verify:** test that a node response for a user node contains no `pass` point
value and a sync node response contains no `authToken` value; export without
`--secrets` omits them with a comment.

### 3. Remove the empty-token auth bypass on the HTTP API

- [ ] Require a valid JWT on `/v1/nodes` routes whenever the device token check
      does not apply, and never treat an empty token as a match.

**Problem:** `authenticate` in `api/nodes.go` compares
`req.Header.Get("Authorization")` with `h.authToken`. With `SIOT_AUTH_TOKEN`
unset (the default) and no header, both sides are `""`, the JWT check is
skipped, and every write path (`POST /v1/nodes`, `POST .../points`, `DELETE`,
parent moves) proceeds unauthenticated. The browser no longer reads or posts
points over HTTP, so the routes this affects are the node operations (add,
delete, move, mirror, duplicate, notify, key) and direct API use.

**Change:** the device-token branch applies only when a token is configured and
non-empty, compared with `subtle.ConstantTimeCompare`. In all other cases the
JWT must validate or the request is rejected — including `GET`.

**Verify:** regression test: with no auth token configured and no
`Authorization` header, `POST /v1/nodes/{id}/points` returns 401.

### 4. Replace the default `admin`/`admin` account (not doing this for now)

- [ ] Generate a random admin password on first boot instead of `admin`.

**Problem:** `store/jetstream.go:1141` seeds `admin`/`admin` and nothing ever
forces a change.

**Change:** on first start, use `SIOT_ADMIN_PASS` if set; otherwise generate a
random password, store its hash (item 1), and print the plaintext once to the
log with clear framing. Document in `docs/user/configuration.md`.

**Verify:** fresh data dir boot logs a generated password and `admin`/`admin`
fails to log in.

### 5. Stop serving the shared token over NATS (complete)

- [x] Remove the `auth.getNatsURI` token handout.

**Problem:** `store/store.go` published `SIOT_AUTH_TOKEN` in plaintext to any
connected client that asked, converting any foothold into the fleet credential.

**Change:** the subject, `client.GetNatsURI`, and its test were removed with the
UI over NATS plan; the browser authenticates with the user's JWT instead and
nothing else needed the handout.

**Verify:** done; a request to the subject gets no responders.

### 6. Bind the NATS monitoring port to localhost (note doing for now, depend on firewall)

- [ ] Default the NATS monitoring HTTP endpoint to `127.0.0.1`, or off.

**Problem:** `server/nats-server.go:37` enables port 8222 on all interfaces with
no authentication; `/connz` and `/subsz` disclose connection and subject detail.

**Change:** default `HTTPHost` to `127.0.0.1`; add an option to widen it
deliberately. Mention in configuration docs.

**Verify:** default boot: 8222 refuses connections from a non-loopback address.

### 7. Rate-limit and log authentication attempts

- [ ] Throttle `auth.user` (NATS) and `/v1/auth` (HTTP) and log failures.

**Problem:** `store/store.go` answers password checks for any connected client
at full speed with no logging — an unthrottled, invisible oracle. The NATS
authorizer now logs every refused user JWT with the remote address (`checkUser`
in `server/auth.go`); `auth.user` and `/v1/auth` still log nothing.

**Change:** a small in-memory limiter (per email, e.g. exponential backoff after
N failures) shared by both entry points, and a log line on every failed attempt
with the source. The authorizer's refusals should feed the same limiter, so a
JWT guessed over the WebSocket counts like a password guessed over HTTP. Full
audit-trail-as-points is out of scope here.

**Verify:** test that repeated failures are delayed/refused and logged.

### 8. Harden the user JWT check

- [ ] Validate issuer and confirm the user still exists on every check, and
      shorten the token lifetime.

**Problem:** `api/key.go` issues 7-day tokens, never checks the issuer claim,
and `Valid` never confirms the user node still exists — a deleted user keeps
access for up to a week on the HTTP routes. On NATS this is already closed: the
authorizer computes a user's groups at connect time, refuses a user with none,
closes the user's connections when their edges or password change, and the NATS
server closes a connection when its JWT expires (`ConnectionDeadline`).

**Change:** check `Issuer == "simpleiot"`; after signature validation, confirm
the user node exists (and is not tombstoned) — `UserAnchors` on the store
answers that; reduce expiry to 24 hours as an interim value. Shortening the
lifetime signs a live browser tab out once a day until JWT renewal over the NATS
connection exists (a follow-up in the UI over NATS plan), so land renewal with
or before the shorter lifetime. Full revocation semantics (`tokenVersion`, short
tokens with renewal) stay with the later user-authz work per `security.md`
sequencing.

**Verify:** test that a token for a deleted user is rejected; a token with a
wrong issuer is rejected.

### 9. TLS plumbing corrections

- [ ] Make the NATS WebSocket listener able to serve TLS, and let the NATS
      client pin a CA.

**Problem:** `server/nats-server.go:89` hardcodes `Websocket.NoTLS = true` even
when NATS TLS certs are configured; `client/edge.go` sets no `nats.RootCAs`, so
an edge device cannot verify it is talking to its real upstream beyond the
system trust store; the server's client-cert fields (`TLSCaCert`, `TLSVerify`)
are copied from options that are never settable.

**Change:** `NoTLS` follows whether certs are configured; add a CA-file option
to the sync/edge client config wired to `nats.RootCAs`; delete the dead
client-cert plumbing rather than leaving it looking functional (client-cert
auth, if wanted, arrives with the credentials plan).

**Verify:** WS listener serves TLS when certs are set; edge client with a pinned
CA refuses a server presenting a different chain.

### 10. Scope reply inboxes per connection (complete)

- [x] Give device and enrollment connections their own inbox prefix instead of
      the shared `_INBOX.>`.

**Problem:** `devicePermissions` and `enrollPermissions` in `server/auth.go`
each grant exactly one subscribe permission, `_INBOX.>`. That is the whole
shared inbox space rather than one connection's replies, so a credentialed
device can subscribe to the wildcard and receive every other client's replies on
the upstream: another device's `nodes.all.Y`, the server's own client, the CLI.
The publish side is scoped to the device, the read side is not. An enrollment
token is the sharper case, since a connection holding one has not been approved
by anyone yet and can publish nothing but `enroll.request`.

**Change:** the pattern already existed for the browser, and now serves all
three. `client.InboxPrefix` is the one place the prefix is built;
`devicePermissions` and `enrollPermissions` grant `_INBOX_<pubKey>.>`, and
`client/edge.go` and `Enroll` set `nats.CustomInboxPrefix` to match.

An enrolling connection presents only a token, so the authorizer had no identity
to key a prefix on. It now presents the key being enrolled as well, signing the
nonce with it, and `checkNkey` accepts an unknown key that arrives with a live
enrollment token. That also proves the enrolling instance holds the key it is
asking to have enrolled. An enrollment token on its own is no longer accepted,
so the branch for it in `checkToken` is gone, along with the `enroll:` username
marker: an enrolling connection is tracked by `authConn.enrollID` instead, which
is what `enforce` closes on when the token is revoked.

**Verify:** done. `TestDeviceInboxScope` and `TestEnrollTokenScope` cover a
device and an enrolling connection receiving their own replies and being refused
`_INBOX.>`; `TestEnrollTokenScope` also covers a token presented without a key
being refused; the existing `TestSyncCredential` and `TestSyncEnroll` round
trips cover sync and enrollment end to end with the prefix in place.

**Incompatible:** an upstream and its devices have to be upgraded together. The
two inbox spaces share no subject prefix, so an old device against a new
upstream, or the reverse, has every request refused and stops syncing until both
sides move.

## Audit, September 2026

A security audit at `v0.28.0` confirmed items 2, 3, 4, 6, 7, 8, and 9 above are
still open and found the items below. Each finding marked _proven_ was
reproduced with a throwaway test against a test server; the others were read in
the code. The [security reference](../docs/ref/security.md#known-limitations)
carries the operator-facing summary.

Suggested order, by how much each one closes:

1. Item 11 (edge writes escape the anchor), because it bypasses the browser
   scope that shipped in `v0.28.0`.
2. Item 3 (empty-token bypass) and item 12 (HTTP routes scoped to the user),
   because together they are the path from any account to the whole tree.
3. Item 13 (replicated users sign in upstream), which also changes the case for
   item 4: a default `admin`/`admin` on any one synced device is a sign-in on
   the upstream.
4. Items 21 and 22 (one bad point or Modbus frame stops the instance), which are
   small and remove a whole class of failure.
5. Item 23 (client writes outside their subtree), the same escape as item 11 by
   another route.
6. Items 14 and 15 (enrollment binding, `required` through the proxy), which are
   small.
7. The rest in any order.

### 11. Check both ends of an edge write from a browser (proven)

- [ ] Require the child of a `u.<anchor>.<user>.ep.<child>.<parent>` request to
      be under the anchor already, as well as the parent.

**Problem:** `handleUserRequest` in `store/store.go` checks
`isUnder(parent, anchor)` for an `ep` request and never looks at the child. A
user in group G publishes `u.G.U.ep.<any node>.G` with `tombstone=0` and a
`nodeType`, the store adds the edge, and `isUnder` then answers true for that
node because of the edge the user just wrote. In the test a user in G was
refused a node in group H, grafted it under G with one request, and then read
and wrote it. Grafting the instance root under G makes a cycle through the root:
the request never completes and the store publishes on `up.root.>` continuously
until it is restarted.

**Change:** refuse an `ep` request unless the child is under the anchor before
the write, and refuse any edge whose child is an ancestor of its parent. A new
node (no existing edges) is the one case where the child has no scope yet; allow
it only when the child ID has no edges at all. Carry a visited set across levels
in `getNodesDepth` and cap `depth`, so a cycle from any source cannot hold a
request open.

**Verify:** a user in G is refused `u.G.U.ep.<node in H>.G` and
`u.G.U.ep.<root>.G`; creating a new node under G still works; a `nodes` request
with a large depth over a mirrored subtree returns.

### 12. Scope the HTTP node routes to the user (proven)

- [ ] Apply `UserAnchors` and `isUnder` to every `/v1/nodes/{id}` route, for
      both the node and any parent named in the body.

**Problem:** this was listed under "Deliberately excluded" as future work. The
audit shows it gives any account the whole tree: `api/nodes.go` accepts any
valid JWT for any node ID, so any account, however small its group, can post
points to the root, delete or move any node, and generate an enrollment token on
any `enrollToken` node. `Key.Valid` also never confirms the user still exists
(item 8), so a deleted user keeps this access for seven days.

**Change:** resolve the JWT to a user, list the user's anchors, and refuse a
request whose node, old parent, or new parent is not under one of them. The
longer-term plan of moving node operations onto `u.<anchor>.<user>.op.*` and
retiring the routes stands; this closes the gap until then.

**Verify:** a user in G gets 403 for a point write, delete, move, mirror,
duplicate, notification, and key request on a node in H, and 200 for the same on
a node in G.

### 13. Decide where a replicated user may sign in (proven)

- [ ] Stop a user node that arrived from a downstream instance from signing in
      on the upstream, or make that an explicit, documented choice.

**Problem:** sync replicates a device's user nodes upstream, and `userCheck` in
`store/jetstream.go` accepts any user with a path to the root. In the test, a
user created on a credentialed device signed in at the upstream's `/v1/auth`.
With item 12 open that user then wrote a point on the upstream root. The same
test changed the upstream's own admin password and then signed in at the
upstream with `admin`/`admin`, because the device still had its default account.
Every device that keeps the default account is therefore a default account on
its upstream.

**Change:** the simplest rule is that `userCheck` considers only users whose
points originate on this instance (the user's origin stream is this instance's
own). If signing in upstream with a device's account is wanted, keep it, scope
it to the device subtree on every door (items 11 and 12), and say so in
`docs/user/users-groups.md`. Either way, revisit item 4 in this light.

**Verify:** a user created on a downstream instance is refused at the upstream's
`/v1/auth` (or, if kept, is refused everything outside the device subtree).

### 14. Bind an enrollment request to its connection and its device (proven)

- [ ] Refuse an enrollment request for a device that already has a live
      credential, and check the key in the request against the connection.

**Problem:** `enroll` in `server/enroll.go` takes `DeviceID` and `PubKey` from
the request body. A holder of the fleet enrollment token can name an existing
device's ID and its own key; with an auto-approve token the credential is live
at once and is granted that device's boundary (`inst.X.X.>` and the streams
written for it). In the test the new key published into the other device's
stream. With a token that leaves credentials pending, an operator sees one more
pending credential under a known device. The request's key is also never
compared with the key that signed the connection nonce, so the proof of
possession the reference describes is not enforced. Node IDs are not validated;
a device ID of `*` was refused only because JetStream rejected the stream name.

**Change:** when the device node already has a credential that is not pending,
always create the new one pending, whatever the token says, so a second key for
a known device needs an operator. Require `msg.Reply` to start with
`client.InboxPrefix(req.PubKey)`, which only the connection holding that key can
subscribe to. Validate `DeviceID` with the same rule as point subject tokens
(`data.CheckSubjectTokens`). Limit how many pending devices one token may
create.

**Verify:** enrolling a second key onto a device with a live credential returns
`pending` under an auto-approve token; a request naming a key other than the
connection's is refused; a device ID containing `.`, `*`, `>`, or whitespace is
refused before any node is created.

### 15. Make `required` hold on the HTTP port (proven)

- [ ] Stop the built-in WebSocket proxy from making a remote connection look
      local.

**Problem:** under `SIOT_DEVICE_AUTH=required` the shared token is accepted only
from loopback (`checkToken` in `server/auth.go`, `authenticate` in
`api/nodes.go`). The HTTP port proxies WebSocket connections to
`ws://localhost:<ws port>` (`api/server.go`), so a connection from any address
to the HTTP port reaches the authorizer from `127.0.0.1`. In the test the shared
token was refused from a LAN address on the NATS and WebSocket ports and
accepted from the same address through the HTTP port, with full access. A
reverse proxy in front of the HTTP port has the same effect on the HTTP routes.

**Change:** on the WebSocket listener, never accept the shared token when
`required` is set: the server's own client and the CLI use the NATS port, and
browsers present a JWT. `server.ClientAuthentication.Kind()` and the connection
type tell the listeners apart. For the HTTP routes behind a reverse proxy,
document that `required` does not limit the token there, or refuse the shared
token on HTTP entirely under `required`.

**Verify:** with `required`, the shared token is refused through the HTTP port's
WebSocket proxy from a non-loopback address and still accepted on the NATS port
from loopback.

### 16. Keep the device key seed off the bus

- [ ] Remove the seed from the `auth.deviceKey` reply.

**Problem:** `handleGet` in `server/device-key.go` answers `auth.deviceKey` with
the instance's NKey seed as well as its public key. Any full-access connection
can ask: a shared-token holder, or anyone who can reach the NATS or HTTP port of
an instance with no token, which is how an edge device runs by default. The seed
is the device's identity on its upstream, and unlike other access it remains
useful after the reader has gone. This is the same shape as the
`auth.getNatsURI` handout that item 5 removed.

**Change:** the sync client runs in the same process as the server. Pass it a
signing function (or the key pair) through the client manager's options instead
of over NATS, and reply to `auth.deviceKey` with the public key only. `siot`
command line tools that need the seed read `SIOT_DATA/device.nkey` directly,
which file permissions protect.

**Verify:** a request to `auth.deviceKey` returns no seed; sync with a device
credential still connects.

### 17. Harden the HTTP server

- [ ] Add timeouts, body limits, and response headers, and keep credentials out
      of the debug log.

**Problem:** `api/server.go` calls `http.Serve` with no timeouts, so idle or
slow connections are held open without limit. Request bodies are decoded with no
size limit. No response carries `Content-Security-Policy`,
`X-Content-Type-Options`, `X-Frame-Options`, or `Referrer-Policy`. `-debugHttp`
logs the full body of `/v1/auth` requests and replies, which is the password and
the JWT. Token comparisons (`api/nodes.go`, `checkToken`) use `==` rather than a
constant-time compare (already noted in item 3).

**Change:** an `http.Server` with `ReadHeaderTimeout`, `ReadTimeout` for the API
routes, and `IdleTimeout`; `http.MaxBytesReader` on API bodies; a small
middleware that sets the four headers (`frame-ancestors 'none'`,
`connect-src 'self' ws: wss:`, and the font origins the UI uses); skip bodies
for `/v1/auth` in the HTTP logger.

**Verify:** a connection that sends no headers is closed; an oversized body gets
413; responses carry the headers; `-debugHttp` output for a sign-in shows no
password or token.

### 18. Build releases with a current Go toolchain (complete)

- [x] Pin a current toolchain in `go.mod` and check for known vulnerabilities in
      CI.

**Done:** `go.mod` requires 1.27.1; both workflows take the version from
`go.mod`; `go.yml` runs `govulncheck ./...`; `golang.org/x/crypto` is at
v0.57.0; the `golangci-lint` pin moved to v2.13.2, the first with Go 1.27
support. A pinned version was chosen over `stable` so that a new Go release
cannot break the linter or ship untested in a release; bump `go.mod` before each
release.

**Problem:** `release.yml` takes the Go version from `go.mod`, which says
`go 1.25.0` with no `toolchain` line, so release binaries are built with Go
1.25.0 and carry every standard-library fix made since, including several in
`net/http` and `crypto/tls`. `go.yml` tests with 1.25.12, a different version
from the one that ships. `govulncheck ./...` on the source with a current
toolchain reports nothing called; `golang.org/x/crypto` v0.54.0 has three
advisories in `ssh`, which is not imported.

**Change:** add a `toolchain` line for the current stable release (or use
`go-version: stable` in both workflows so tests and releases match), bump
`golang.org/x/crypto`, and add a `govulncheck ./...` step to `go.yml`.

**Verify:** `go version` on a release binary reports the pinned toolchain; CI
fails on a called vulnerability.

### 19. Safer defaults for an installed service

- [ ] Have `siot install` produce a service that is not open to the network.

**Problem:** `install/siot.service` sets `SIOT_AUTH_TOKEN=""` and has no `User=`
or sandboxing directives; every listener (8118, 4222, 8222, 9222) binds all
interfaces and there is no option to change that; the data directory is created
`0755`. With no token, the WebSocket listener also accepts an anonymous
connection from any page origin, so a page open in a browser on the same network
can reach it. Items 4 and 6 cover the default account and the monitoring port.

**Change:** generate a random token at install time into an `EnvironmentFile`
with mode `0600`; add a bind-address option and bind the WebSocket and
monitoring listeners to loopback by default, since the HTTP port already proxies
the WebSocket; create the data directory `0700`; add `NoNewPrivileges`,
`ProtectSystem=strict`, `ProtectHome`, `PrivateTmp`, and `ReadWritePaths` to the
unit, with a documented profile for the hardware clients that need device
access.

**Verify:** a fresh `siot install` refuses an anonymous NATS connection from
another host, and `systemd-analyze security siot` improves.

### 20. Smaller items

- [ ] Redaction (item 2) should also cover `nodes.all.<id>` replies to a
      browser: they list every parent edge of the node, including parents
      outside the user's anchors, which discloses the IDs item 11 needs.
- [ ] Inside an anchor every node is writable, including other users' `pass` and
      `email` points, so one member of a group can take over another. A `role`
      edge point, or limiting writes on `user` nodes to the user and to members
      of a parent group, would close it.
- [ ] `userCheck` runs bcrypt only when the email matches, so response time
      tells whether an account exists. Compare against a fixed hash when it does
      not.
- [ ] Use a 32-byte JWT signing key (`initJwtKey` makes 20 bytes).
- [ ] Sign `checksums.txt` for releases and verify it in `siot update`; the
      checksum today comes from the same place as the binary.
- [ ] Pin GitHub Actions to commit SHAs and give `go.yml` a
      `permissions:     contents: read` block.
- [ ] Serve the UI fonts from the binary instead of `fonts.googleapis.com`,
      which also lets the UI render on an isolated network.
- [ ] Remove `frontend/public/ble.js`, which is embedded and served but unused,
      and refresh or remove `contrib/configs/signal-gen.yaml`.
- [ ] Un-anchor the key patterns in `.gitignore` (`device.nkey`, `*.nkey`,
      `*.pem`, `*.key`) so a data directory elsewhere in the tree is covered.

### 21. Keep one bad point from stopping the instance (proven)

- [ ] Recover from panics in client goroutines and clamp values that come from
      points before they reach a ticker or an allocation.

**Problem:** no goroutine in `client/`, `modbus/`, `server/`, `store/`, or
`api/` recovers from a panic, so a panic in any client ends the whole process:
NATS, the store, HTTP, and every other client. Several panics are reachable from
a stored point, so the instance stops again at every restart until the store is
repaired. Any user who can write a point under their group can cause them.

- `client/manager.go`: when `newClientState` returns an error it is logged and
  the nil result is then used (`cs.run()`, `cs.node.ID`). `data.Decode` fails on
  a non-integer or negative key, or an index above 1000, on any slice field
  (metrics `prefix`, db `tagPointType`, ntp `server`, rule `weekday`, and
  others). A metrics node with a `prefix` point keyed `abc` reproduced it.
- Durations from points reach `time.NewTicker` and `Ticker.Reset` unchecked: a
  zero `pollPeriod` on an update node, a very large `sampleRate` on a signal
  generator, and a `period` on a metrics node large enough to overflow each
  reproduced a panic. Modbus, IIO, OneWire, GPIO, and the GPS simulator multiply
  a point into a duration the same way.
- `maxMessageLength` on a serial node sizes an allocation on every read.

**Change:** `continue` when `newClientState` fails and record the error on the
node; wrap each client's `Run`, the manager loop, and NATS callbacks in a
`recover` that logs, records the error on the node, and restarts the client with
backoff; one helper that clamps a duration from a point into a minimum and
maximum, used everywhere a ticker is built; a ceiling on `maxMessageLength`.

**Verify:** the three point values above leave the instance running with an
error on the node.

### 22. Bound Modbus requests and responses (proven at unit level)

- [ ] Validate counts in the Modbus server and client and close connections over
      the limit.

**Problem:** a Modbus TCP server node listens on every interface with no
authentication, which is how Modbus works, so its parser is exposed to the
network. In `modbus/pdu.go`, a read-registers request with a count of `0x8000`
wraps `1+2*count` to a one-byte buffer, and a read-coils request for 2048 bits
truncates the byte count to zero; both then index past the buffer and the panic
ends the process (item 21). `RespReadBits` trusts the count in a response
without checking the data length, so a server can do the same to a Simple IoT
client. `modbus/tcp.go` never closes a connection accepted over `maxClients`,
and continues with a nil socket after an `Accept` error.

**Change:** enforce the protocol limits (1 to 125 registers, 1 to 2000 coils)
and return an illegal-value exception; check response lengths against the
request; close connections over the limit and back off on `Accept` errors.

**Verify:** unit tests for each malformed frame return an exception or an error;
a sixth connection is closed.

### 23. Keep a client's writes inside its own subtree (proven)

- [ ] Check the target of a rule action, a signal generator destination, and a
      serial high-rate destination against the client's parent before
      publishing.

**Problem:** clients publish on the server's full-access connection. A rule
action's `nodeID`, `pointType`, and value come from points and are published
with no check (`client/rule.go`), and the signal generator destination
(`client/subject.go`) and serial `hrDest` do the same. A user scoped to one
group can add a rule whose action writes any point on any node whose ID they
know, which reaches the same result as item 11 by another route. In the test a
rule in one group rewrote a node in a sibling group. The IDs are placed in
subjects unvalidated, so an ID containing `.` addresses an edge subject.

**Change:** a shared check that the target is the client's parent or below it,
and rejection of node IDs containing `.`, `*`, `>`, or whitespace. A rule that
needs to reach across groups belongs at a level that contains both.

**Verify:** a rule under G is refused an action targeting a node under H, with
the error recorded on the action node.

### 24. Validate what clients take from points and peers

Smaller client items from the audit, each independent:

- [ ] **Update client:** in addition to signing (see "Deliberately excluded"),
      require `https`, check the response status, and add a timeout and a size
      cap to the download and to `files.txt`.
- [ ] **Kiosk browser client:** reject control characters and validate the URL
      scheme before writing `/etc/default/yoe-kiosk-browser`, since a newline
      adds a variable to the unit's environment.
- [ ] **Message service:** a `message` point raised anywhere below the service
      is sent to the phone or email in the point, with no rate limit, so a user
      in a subgroup can send through the operator's Twilio or SMTP account.
      Derive recipients from user nodes and add a limit. Strip CR and LF from
      SMTP headers (`msg/smtp.go`).
- [ ] **Outbound requests:** the metrics scraper, Shelly `ip`, ntfy URL, Modbus
      `uri`, and gpsd address are dialed as given, and errors are written back
      to the node, so a user on a cloud instance can probe its internal network.
      A shared policy that refuses loopback, link-local, and private ranges
      unless allowed, and parses `ip` as an address.
- [ ] **Shelly discovery:** an mDNS answer with a known device name rewrites
      that device's `ip`. Confirm identity before changing a known address, and
      bound the JSON and WebSocket reads.
- [ ] **Serial and Particle:** points arriving from an MCU are merged into the
      serial node's own configuration (and the parent's with `syncParent`). Drop
      the client's configuration point types when they arrive from the wire.
- [ ] **Secrets:** extend item 2 and `dropSecretPoints` to `psk`, `enrollToken`,
      `sid`, and the messaging, database, and Particle tokens. Send the Particle
      token as a header; it is in the URL today and appears in logged errors.
- [ ] **MQTT:** a user can add an `mqtt` node subscribed to `#` and receive
      every tenant's publishes as points. Restrict filters on nodes below the
      root, and cap the nodes Sparkplug may create.
- [ ] **Paths from points:** the IIO `device` and `channel`, the OneWire `id`,
      the rule `playAudio` file, and `ListenForFile` names accept `../`.
      Validate each against its expected form.
- [ ] **NTP client:** filter newlines in `server` and `fallbackServer`, write a
      newline between the two settings, and close the file.

## Deliberately excluded

Tracked elsewhere, not forgotten:

- **Per-node authorization on the HTTP API** (`api/nodes.go`): moved into this
  plan as item 12 after the September 2026 audit. What stays with the later
  user-authz work is retiring the HTTP node routes in favor of
  `u.<anchor>.<user>.op.*`.
- **MQTT origin/scoping** (`client/mqtt.go:40` vs `store/store.go:311`) — lands
  with the `mqttUser` per-client credential work.
- **Connection-bound replication origin** — addressed by the per-device
  credentials plan's derived subject permissions; add the impersonation test to
  its Phase 0 spike.
- **Signed update/provisioning payloads** (`client/update.go`) — the
  sign-the-payload design in `server-provisioning.md`.
- **Full JWT revocation (`tokenVersion`)** — step 3 of the `security.md`
  sequencing. The websocket UI rework (step 4) shipped as the UI over NATS plan;
  what remains of it is JWT renewal over the live connection and moving node
  operations off HTTP.
