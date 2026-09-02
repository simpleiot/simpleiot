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

### 10. Scope reply inboxes per connection

- [ ] Give device and enrollment connections their own inbox prefix instead of
      the shared `_INBOX.>`.

**Problem:** `devicePermissions` and `enrollPermissions` in `server/auth.go`
each grant exactly one subscribe permission, `_INBOX.>`. That is the whole
shared inbox space rather than one connection's replies, so a credentialed
device can subscribe to the wildcard and receive every other client's replies on
the upstream: another device's `nodes.all.Y`, the server's own client, the CLI.
The publish side is scoped to the device, the read side is not. An enrollment
token is the sharper case, since a connection holding one has not been approved
by anyone yet and can publish nothing but `enroll.request`.

**Change:** the pattern already exists for the browser. `userPermissions` grants
`userInboxPrefix(U)` (`_INBOX_<U>`) and the client sets `nats.CustomInboxPrefix`
to match; do the same for devices. `client/edge.go` sets
`nats.CustomInboxPrefix("_INBOX_" + X)` on the sync connection,
`devicePermissions` grants `_INBOX_<X>.>`, and `enrollPermissions` grants the
prefix for the key being enrolled. Factor the prefix helper so all three callers
share it.

**Verify:** a device connection subscribing to `_INBOX.>` gets a permissions
violation; two devices making concurrent requests each receive only their own
replies; an enrollment-token connection cannot read another connection's
replies; sync still completes end to end with the custom prefix in place.

**Docs:** `docs/ref/security.md` — the "What a device credential allows" table
row for replies, and the "Two things to know about the boundary of this model"
list, which today calls out the `$JS.API.STREAM.NAMES` disclosure but not this
one.

## Deliberately excluded

Tracked elsewhere, not forgotten:

- **Per-node authorization on the HTTP API** (`api/nodes.go`) — needs the
  `userCanAccess` primitive; that is step 3 of the `security.md` sequencing. The
  NATS side of it exists: the store's `isUnder(id, anchor)` scopes every browser
  request to a group the user belongs to, and `UserAnchors` lists those groups.
  The HTTP routes can call the same two functions.
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
