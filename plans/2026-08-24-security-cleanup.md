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

**Problem:** `GET /v1/nodes` and the `nodes.*.*` NATS subjects return all
points, so every poll ships every user's password and the sync node's upstream
token to the browser. Hashing (item 1) reduces the damage for `pass` but the
sync `authToken` is still a live credential.

**Change:** a single redaction function applied where nodes are serialized for
clients (HTTP node responses and the `nodes.*.*` reply path), keyed on a small
list of secret point types. Writes are unaffected. `siot export` keeps the
plan's `--secrets` behavior (omit by default, include on request) — implement
the flag here if it is simpler than waiting for credentials Phase 2.

**Verify:** test that a node response for a user node contains no `pass` point
value and a sync node response contains no `authToken` value; export without
`--secrets` omits them with a comment.

### 3. Remove the empty-token auth bypass on the HTTP API

- [ ] Require a valid JWT on `/v1/nodes` routes whenever the device token check
      does not apply, and never treat an empty token as a match.

**Problem:** `api/nodes.go:60-68` compares
`req.Header.Get("Authorization") != h.authToken`. With `SIOT_AUTH_TOKEN` unset
(the default) and no header, both sides are `""`, the JWT check is skipped, and
every write path (`POST /v1/nodes`, `POST .../points`, `DELETE`, parent moves)
proceeds unauthenticated.

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

### 5. Stop serving the shared token over NATS

- [ ] Remove the `auth.getNatsURI` token handout.

**Problem:** `store/store.go:565` publishes `SIOT_AUTH_TOKEN` in plaintext to
any connected client that asks, converting any foothold into the fleet
credential.

**Change:** remove the token from the reply (return URI only), or remove the
subject entirely if nothing depends on it — check `frontend/lib/siot-nats.mjs`
and docs for callers first.

**Verify:** grep confirms no caller expects the token; a request to the subject
no longer returns it.

### 6. Bind the NATS monitoring port to localhost (note doing for now, depend on firewall)

- [ ] Default the NATS monitoring HTTP endpoint to `127.0.0.1`, or off.

**Problem:** `server/nats-server.go:37` enables port 8222 on all interfaces with
no authentication; `/connz` and `/subsz` disclose connection and subject detail.

**Change:** default `HTTPHost` to `127.0.0.1`; add an option to widen it
deliberately. Mention in configuration docs.

**Verify:** default boot: 8222 refuses connections from a non-loopback address.

### 7. Rate-limit and log authentication attempts

- [ ] Throttle `auth.user` (NATS) and `/v1/auth` (HTTP) and log failures.

**Problem:** `store/store.go:489` answers password checks for any connected
client at full speed with no logging — an unthrottled, invisible oracle.

**Change:** a small in-memory limiter (per email, e.g. exponential backoff after
N failures) shared by both entry points, and a log line on every failed attempt
with the source. Full audit-trail-as-points is out of scope here.

**Verify:** test that repeated failures are delayed/refused and logged.

### 8. Harden the user JWT check

- [ ] Validate issuer and confirm the user still exists on every check, and
      shorten the token lifetime.

**Problem:** `api/key.go:43-70` issues 7-day tokens, never checks the issuer
claim, and `Valid` never confirms the user node still exists — a deleted user
keeps access for up to a week.

**Change:** check `Issuer == "simpleiot"`; after signature validation, confirm
the user node exists (and is not tombstoned); reduce expiry to 24 hours as an
interim value. Full revocation semantics (`tokenVersion`, short tokens with
renewal) stay with the later user-authz work per `security.md` sequencing.

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

## Deliberately excluded

Tracked elsewhere, not forgotten:

- **Per-node authorization on the HTTP API** (`api/nodes.go:48`) — needs the
  `userCanAccess` primitive; that is step 3 of the `security.md` sequencing,
  after the credentials plan.
- **MQTT origin/scoping** (`client/mqtt.go:40` vs `store/store.go:311`) — lands
  with the `mqttUser` per-client credential work.
- **Connection-bound replication origin** — addressed by the per-device
  credentials plan's derived subject permissions; add the impersonation test to
  its Phase 0 spike.
- **Signed update/provisioning payloads** (`client/update.go`) — the
  sign-the-payload design in `server-provisioning.md`.
- **Full JWT revocation (`tokenVersion`) and the websocket UI rework** — steps 3
  and 4 of the `security.md` sequencing.
