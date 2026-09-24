# Plan: Security Follow-ups

**Branch:** `cbrake/master` **Branched from:** `d4855615`

## Context

The [security cleanup plan](2026-08-24-security-cleanup.md) is complete: every
item the September 2026 audit found that was a bug or a missing check is closed
and tested. What it left behind are the items that need a design decision or a
piece of infrastructure rather than a fix, collected here so the old plan can be
read as a record of what shipped. The
[known limitations](../docs/ref/security.md#known-limitations) table in the
security reference points at the items below.

Each item stands alone. Item 1 is the one that changes what a deployment can
safely do; items 2 to 7 are hardening. Items 8 to 12 come from a second audit
(September 2026, below); 8, 9, and 10 each let an account or a device take over
the instance and come first, in that order.

## Checklist

### 1. Honor host-changing node types only under the root

- [ ] Run the update, NTP, network manager, and kiosk browser clients only on
      nodes directly under the root, so an account in a group cannot change the
      host.

**Problem:** the scope checks keep a user inside their groups but not away from
node types that act on the host. Every client runs wherever its node type
appears, on the server's connection, so a user in any group can add an `update`
node with any `https` URL and have the server replace its own binary and reboot;
`ntp`, `networkManager`, and `browser` nodes rewrite system configuration the
same way. An account is therefore control of the host, which is fine for an
operator's own team and not for anyone else. Documented under
[What an account can do](../docs/ref/security.md#what-an-account-can-do).

**Change:** give `client.NewManager` a way to restrict a client to nodes whose
parent is the root (the manager already takes a list of parent types), and use
it for these four; a node of one of these types elsewhere in the tree gets an
`error` point saying so and is not run. Consider the same for `modbus` server
nodes and `file` nodes.

**Verify:** an update node created by a user under a group is not acted on and
carries an error; one under the root still updates.

### 2. Keep one group member from taking over another

- [ ] Limit writes to a `user` node's `pass` and `email` points to the user
      themself and to members of a parent group.

**Problem:** inside an anchor every node is writable, including other users'
`pass` and `email` points, so one member of a group can take over another. A
user can also add user nodes to the group.

**Change:** a `role` edge point (`admin` or `user`, the constants already exist
in `data/schema.go`) checked in `handleUserRequest` and the HTTP routes: a write
to a `user` node's credential points, or a new `user` edge, needs the writer to
be that user or to hold `admin` on the anchor or an ancestor of it. The UI needs
a way to set the role.

**Verify:** a member without the role is refused a write to another member's
`pass`; an admin of the parent group is not.

### 3. Shorten the sign-in token and renew it over the live connection

- [ ] Reduce the token lifetime to 24 hours, renew it over the NATS connection
      before it expires, and invalidate tokens on a password change.

**Problem:** the token lasts seven days. It is refused once the user is gone
from the tree, but a password change does not invalidate it on the HTTP routes
until the NATS connection is closed. The audit's proposed 24-hour lifetime was
not applied because it would sign a browser out once a day until renewal exists
(a follow-up the [UI over NATS plan](2026-09-01-ui-over-nats.md) already lists).

**Change:** an `auth.renew` request over the user's connection that answers with
a fresh token, the JS client and the Elm app storing it; then the shorter
lifetime. For password changes, include a short hash of the stored password hash
in the claims and compare it in `TokenClaims`, which the store can do from its
point cache without a second lookup.

**Verify:** a browser open for more than a day stays signed in; a token issued
before a password change is refused on HTTP and on NATS.

### 4. Rate limit sign-in by source as well as by account

- [ ] Add a per-address limit alongside the per-account one, and read the client
      address from a trusted reverse proxy.

**Problem:** the limiter keys on the account presented, so anyone who can reach
the sign-in page can lock a known email out for up to five minutes at a time.
Behind a reverse proxy every connection arrives from loopback, so a per-address
limit, and the loopback rule for the shared token under
`SIOT_DEVICE_AUTH=required`, both need the real address. Simple IoT does not
read `X-Forwarded-For`, and must not without knowing which proxy to trust.

**Change:** a `SIOT_TRUSTED_PROXIES` setting (addresses or ranges); when a
request or WebSocket arrives from one, take the client address from the last
untrusted hop in `X-Forwarded-For`. Use it for the loopback rule on the HTTP
routes and in the WebSocket proxy, and add a second limiter key on the address
with a higher threshold than the account one, so a lockout needs the account's
own failures.

**Verify:** with a trusted proxy configured, the shared token from a remote
address through the proxy is refused under `required`; failures from one address
against many accounts are refused after the address threshold.

### 5. Sign releases and update payloads

- [ ] Sign `checksums.txt` for releases, verify it in `siot update`, and sign
      update payloads for the update client.

**Problem:** releases carry checksums and no signature, so `siot update` and the
update client trust whatever the release host serves. The update client already
requires `https` and checks the status and size; the remaining trust is in the
host.

**Change:** a signing key held as a release secret (a maintainer decision),
goreleaser signing `checksums.txt`, the public key embedded in the binary, and
`siot update` and the update client refusing an unsigned or mis-signed manifest.
The sign-the-payload design in `server-provisioning.md` covers the update node.

**Verify:** `siot update` refuses a release whose checksums file does not
verify.

### 6. Replace the default `admin`/`admin` account (deferred by choice)

- [ ] Generate a random admin password on first boot instead of `admin`.

Deferred: the deployment checklist asks for the password to be changed before
the sign-in page is reachable, and the limiter slows a guess. The proposed
change, from the earlier plan: use `SIOT_ADMIN_PASS` if set on first start,
otherwise generate a password, store its hash, and print it once to the log.

### 7. Bind the NATS monitoring port to localhost

- [x] Default the monitoring listener to `127.0.0.1`.

Done 2026-09-19 with the port simplification: the monitoring listener always
binds to loopback, and `SIOT_NATS_HTTP_HOST` is gone.

## Audit, September 2026 (second pass)

A second audit at `1b37e2ab` reviewed the authorizer, the HTTP API, enrollment,
the store, the clients, the frontend, dependencies, and packaging, for a public
instance with the shared token set and the HTTP and NATS ports open. Findings
marked _proven_ were reproduced with a throwaway test against a server on its
own ports. Suggested order: 8, 9, 10, 11, then 12.

### 8. Stop devices from choosing their replica stream's configuration

- [x] Create `inst_X_X` on the upstream with a fixed configuration, and take
      `$JS.API.STREAM.CREATE` out of the device's permissions.

**Done:** `refreshCred` calls `ensureDeviceStream` (`server/auth.go`) before a
credential is marked live, so the stream is in place before the device can
connect; a stream whose configuration has other subjects, a republish, a mirror,
sources, or a subject transform is deleted and recreated. A credential whose
stream cannot be made authorizes nothing. `TestDeviceStreamConfig` and
`TestAuthSpike` cover it.

**Problem (_proven_):** `devicePermissions` in `server/auth.go` lets a device
publish `$JS.API.STREAM.CREATE.inst_X_X`, and JetStream accepts any
configuration in the request. A new credential (one whose stream does not exist
yet) created `inst_X_X` with the subjects `inst.X.X.>`, `auth.>`, `_INBOX.>`,
and `p.>` and a `republish` to its own inbox. From then on it received every
`auth.user` request, with the email and plaintext password, every reply on
`_INBOX.>` including sign-in JWTs, and every point write on the instance. Since
the stream answers requests on those subjects with a publish acknowledgement,
sign-in also stopped working. A stream could equally set `sources` or unlimited
`max_bytes`.

**Change:** the authorizer creates `inst_X_X` with the store's own configuration
(one subject, `inst.X.X.>`, no republish, sources, or mirror, bounded size) when
a credential becomes live, and the device keeps only
`$JS.API.STREAM.INFO.inst_X_X`. On a stream advisory for `inst.>`, refuse and
log any stream whose configuration does not match, so a stream created before
the change is caught.

**Verify:** a credentialed device's `STREAM.CREATE` is refused; its sync still
pushes; a stream with extra subjects or a republish is reported at start-up.

### 9. Keep a device's replicated writes inside its boundary

- [ ] Drop any point or edge in `inst.X.X.>` whose node, or edge parent, is not
      in boundary `X`.

**Problem (_proven_):** `mergeReplicaMsg` in `store/replica.go` merges every
`<node>.p.*` and `<parent>.ep.<child>` from a device's stream into the upstream
without checking that the node is in the device's boundary. A credentialed
device wrote the upstream root's description, and created a user node with an
email and password it chose and an edge directly under the upstream root. That
user is not under a device boundary, so `userCheck` let it sign in, its anchor
was the root, and it could read and write the whole tree. From the code, a
future-dated `pass` point on an existing user should win the same way, and a
replicated password is not hashed; not tested. "A credentialed device cannot
publish outside its own boundary" in the security reference holds for NATS
subjects but not for what the store does with them.

**Change:** in `mergeReplicaMsg` and in the start-up load of replica streams,
accept a point only when its node is `X` or under `X` in the edge cache, and an
edge only when its parent is. Log and count what is dropped.

**Verify:** a device that publishes a point for the upstream root, or an edge
under it, into `inst.X.X.>` changes nothing and is logged; its own subtree still
syncs.

### 10. Limit which node types an account may create

- [ ] Refuse `deviceCred`, `enrollToken`, and `sync` nodes (and the host types
      in item 1) from a principal that is not the shared token or anchored at
      the root.

**Problem (_proven_, three ways):** `handleUserRequest` in `store/store.go` and
`insertNode` in `api/nodes.go` accept a node of any type inside the user's
group, and the authorizer and the clients honor these types wherever they sit.

- A user created a `device` node and a `deviceCred` under it with a key it held,
  and connected as that device. With item 8 or 9, that is the whole instance.
- A user created an `enrollToken` with `autoApprove` in its group, took the
  token from `POST /v1/nodes/<id>/key`, and enrolled a device, which the
  enroller places under the root. The device was approved at once and created
  the stream in item 8.
- A user created a `sync` node pointing at a server it runs. The instance
  replicated its whole origin stream there, including users in other groups and
  their password hashes (replication does not redact), and accepted writes back
  from it: the user's server moved the user directly under the instance's root.

**Change:** one list of privileged node types in `data/`, checked when a node is
created and when a `nodeType` point is written, in `handleUserRequest` and the
HTTP routes. Add `deviceCred`, `enrollToken`, and `sync` to the types item 1
restricts, or land both together. The authorizer can also ignore credentials and
enrollment tokens that are not under the root or a root-level device.

**Verify:** each of the three cases above is refused for a user in a group and
still works with the shared token.

### 11. Make `required` hold on the WebSocket proxy (reopens cleanup item 15)

- [ ] Refuse the shared token on the WebSocket proxy under `required` however
      the `CONNECT` is framed.

**Problem (_proven_):** `connectHasToken` in `api/ws-proxy.go` inspects only the
first WebSocket frame and requires `CONNECT` followed by a space and nothing
after the JSON. From a LAN address under `SIOT_DEVICE_AUTH=required`, the shared
token was accepted with full access when `CONNECT` was split across two frames,
followed by a tab, or followed by `PING` in the same frame, which is what the
nats.ws client sends. `TestAPIRequiredThroughProxy` passes because nats.go sends
`CONNECT` alone.

**Change:** buffer client frames until the first `\r\n`, require a `CONNECT`
verb followed by any whitespace, decode the object with a `json.Decoder` (which
allows trailing data), and close the connection when it carries a token or
cannot be parsed. The NATS server reports WebSocket connections with the same
kind as TCP ones, so the check has to stay in the proxy.

**Verify:** the three framings above are refused from a non-loopback address; a
browser with a JWT still connects.

### 12. Smaller items

- [ ] `CheckPassword` in `data/password.go` accepts an empty candidate against
      an empty stored password, so a user with no password signs in with none.
      Refuse an empty stored or candidate password. (Read in code.)
- [ ] `-debugHttp` reads the `/auth` body and does not restore it
      (`api/http-logger.go`), so sign-in fails with it on; it also logs
      unredacted replies to shared-token requests. (Read in code.)
- [ ] `install/siot.service` has no `User=`, so `sudo siot install` runs the
      service as root, which with item 1 makes an account root on the host. Run
      it as a dedicated user.
- [ ] `contrib/siot.service` and `contrib/siot.init` set `SIOT_PORT`, which
      nothing reads, run as root with no token, and use the working directory
      for data. Update or remove them.
- [ ] `SIOT_OUTBOUND_DENY_PRIVATE` is off by default, so a user's `metrics`,
      `mqtt`, `shelly`, or `modbus` node can probe and read services on the
      host's loopback and private networks. Consider defaulting it on when
      `SIOT_AUTH_TOKEN` is set.
- [ ] `docs/user/installation.md` passes the token with `-token` on the command
      line, where other local users can read it; recommend `SIOT_AUTH_TOKEN`.
- [ ] Drop `ws: wss:` from `connect-src` in the content security policy, since
      `'self'` covers the UI's own WebSocket, and add HSTS to the reverse proxy
      guidance.
- [ ] Document that `SIOT_DATA` defaults to the working directory.
- [ ] Consider a vulnerability reporting contact; the repository has none.
