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
safely do; the rest are hardening.

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
