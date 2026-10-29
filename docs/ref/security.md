# Security

Users and downstream devices will need access to a Simple IoT instance. Simple
IoT currently provides access via HTTP and NATS.

Simple IoT is pre-1.0 and its defaults favor getting started quickly: an
instance starts with no auth token and an `admin`/`admin` account. The
[deployment checklist](#deployment-checklist) lists what to change before an
instance is reachable by anyone else, and
[known limitations](#known-limitations) records what the September 2026 audit
found and where each item is tracked.

## Deployment checklist

For any instance that other people or devices can reach, cloud or edge:

- Set `SIOT_AUTH_TOKEN`. With no token, every listener accepts anonymous
  connections with full access, and the HTTP API accepts requests with no
  `Authorization` header.
- Change the `admin` password, on every instance, before the sign-in page is
  reachable. Nothing forces a change, and the limiter on sign-in attempts only
  slows a guess against a known password.
- Give accounts only to people you would trust with the host. Any account can
  add nodes that change the machine the instance runs on; see
  [What an account can do](#what-an-account-can-do).
- Give each device a [credential](../user/sync.md#device-credentials) and set
  `SIOT_DEVICE_AUTH=required` on the upstream. Prefer enrollment tokens that
  leave credentials pending over ones that approve automatically. A device
  credential reaches only its own subtree, though a device that knows the ID of
  a node in another device's subtree can attach it under its own; see
  [What the store checks](#what-the-store-checks). Until item 10 of the
  [security follow-ups plan](https://github.com/simpleiot/simpleiot/blob/master/plans/2026-09-19-security-followups.md)
  lands, any account can create a credential or an auto-approve token in its own
  group, so approve credentials only for devices you control, and do not use
  auto-approve tokens on an instance that holds other people's accounts.
- Expose only the HTTP port (4223), behind a reverse proxy that terminates TLS.
  Firewall the NATS port (4222) unless devices connect to it directly, in which
  case configure `SIOT_NATS_TLS_CERT` and `SIOT_NATS_TLS_KEY`. The NATS
  WebSocket and monitoring listeners bind to loopback only.
- Behind a reverse proxy, every connection reaches Simple IoT from loopback, and
  Simple IoT does not read `X-Forwarded-For`. `SIOT_DEVICE_AUTH=required`
  therefore does not limit the shared token on the HTTP port; keep the token off
  that path and treat it as a secret. Rate limit `POST /v1/auth` per address in
  the proxy, since the built-in limiter keys on the account.
- On an edge device, firewall all incoming ports unless the local UI is needed.
  An edge instance usually runs with no token, and write access to an instance
  is control of the host; see [What an account can do](#what-an-account-can-do).
- Keep a Modbus TCP server node on a trusted network. Modbus has no
  authentication, and the listener binds every interface.
- Set `SIOT_NATS_WS_ORIGINS` to the origin the UI is served from.
- Leave `-debugHttp` off in production; it logs sign-in requests and replies.
- Run a release built with a current Go toolchain, and keep it updated with
  `siot update`.

## What an account can do

The scope checks keep a signed-in user inside the groups the user belongs to:
reads, point writes, edge writes, the HTTP node routes, and the clients' own
writes all stop at the group. What they do not do is limit _which kinds of node_
a user may create there. Every client in `client.DefaultClients` runs on its
node type wherever that node sits in the tree, on the server's own full-access
connection, so a user in any group can add:

- an `update` node with an `https` URL of their choosing, which downloads an
  image, replaces the instance's binary, and with `autoReboot` runs `reboot`.
  This is code execution on the host as the service user;
- an `ntp` or `networkManager` node, which rewrite the host's time and network
  configuration;
- a `browser` node, which rewrites the kiosk browser's configuration;
- a `deviceCred` under a `device` node, or an `enrollToken` that approves
  automatically, which gives the user a device credential (see
  [What a device credential allows](#what-a-device-credential-allows));
- a `sync` node pointing at a server the user runs, which replicates the
  instance's whole tree there, unredacted, and accepts writes back from it;
- a `modbus` server node, which opens a listener on every interface, or an
  `mqtt`, `metrics`, `shelly`, or `db` node, which dials out from the host
  (`SIOT_OUTBOUND_DENY_PRIVATE` keeps those off private addresses).

The trust boundary is therefore the instance, not the group: **an account is
authority over the machine the instance runs on.** For an operator and their own
team that is the intended model. For a deployment that gives accounts to people
who should not administer the host, the host-changing node types need to be
honored only directly under the root, which is tracked as item 1 of the
[security follow-ups plan](https://github.com/simpleiot/simpleiot/blob/master/plans/2026-09-19-security-followups.md);
until then, do not give such people accounts.

Inside a group, every member can write every node, including another member's
`pass` and `email` points. A user with an account can also add user nodes to the
group, and a message service delivers to the users in its scope.

## Server

For cloud/server deployments, we recommend installing a web server like Caddy in
front of Simple IoT. See the [Installation page](../user/installation.md) for
more information, and the reverse proxy note in the
[deployment checklist](#deployment-checklist).

## Edge

Simple IoT Edge instances initiate all connections to upstream instances;
therefore, no incoming connections are required on edge instances and all
incoming ports can be firewalled. The HTTP and NATS listeners bind every
interface (the NATS WebSocket and monitoring listeners bind to loopback), and an
edge instance often runs with no auth token, so the firewall is what keeps the
local network out. The installed service is given a generated token; see
[installation](../user/installation.md).

## HTTP

The web UI signs in with `POST /v1/auth` and receives a JWT (JSON web token). It
uses the JWT for the node operations that stay on HTTP (add, delete, move,
mirror, duplicate, notify) and as its NATS credential for everything else; see
[Browser](#browser) below. Every node route resolves the request to a principal
and refuses a node outside what that principal may reach: a user is limited to
the groups the user belongs to, on the node named in the path and on any parent
named in the body, so a node cannot be read, written, moved, mirrored, or
attached from outside those groups. A user's token is refused as soon as the
user is gone from the tree, and a token that names another issuer or was signed
another way is refused. The token is valid for seven days.

Replies to a user leave the values of secret points out and list only the
parents the user can see; see [Secrets in node reads](#secrets-in-node-reads).

Devices can also reach the node API over HTTP, with either credential the NATS
side accepts:

- The shared token, sent as the `Authorization` header, grants full access.
  Under `SIOT_DEVICE_AUTH=required` it is accepted only from loopback, as on the
  NATS side.
- A token signed with the device key, sent as `Authorization: Bearer <jwt>`. The
  token is a NATS-style JWT whose issuer is the device's public key and which
  expires within five minutes; the upstream verifies the signature, looks the
  key up among its credentials, and limits the request to the device's own
  subtree: reading nodes, posting points, and posting notifications, on the
  device node or anything below it. `client.DeviceJWT` builds one from a seed.

A request with no credentials is refused on every node route, whether or not a
token is configured. An instance with no token is still open on NATS, so set
one; see the [deployment checklist](#deployment-checklist).

Sign-in attempts are limited per account: after five failures in a row the
account is refused for a second, doubling with each further failure up to five
minutes, and every failure is logged with its source. The limit is shared by
`POST /v1/auth`, the `auth.user` subject, and the NATS authorizer's check of a
browser's token.

The HTTP server closes a connection that sends no request within ten seconds or
no complete request within thirty, limits a request body to 4 MB, and sends
`Content-Security-Policy`, `X-Content-Type-Options`, `X-Frame-Options`, and
`Referrer-Policy` on every response. The policy allows the UI's own scripts,
styles, and fonts (served from the binary) and a WebSocket back to the server,
and refuses framing. The HTTP port has no TLS of its own; a reverse proxy in
front of it supplies that. With `-debugHttp`, the bodies of sign-in requests and
replies are not logged.

## NATS

The embedded NATS server authenticates every connection on every listener (NATS,
WebSocket, and MQTT) through one authorizer inside Simple IoT, so there is no
NATS accounts file to manage. Three kinds of credential are accepted:

- **The shared token** (`SIOT_AUTH_TOKEN`) grants full access. The server's own
  client, the `siot` command line tools, and MQTT clients (which send it as the
  password) use it. When no token is configured the instance is open, as it
  always has been.
- **A device credential** is an NKey pair. The device keeps the seed in
  `SIOT_DATA/device.nkey`, which only the server process reads: the sync client
  asks the server to sign the connection challenge on `auth.deviceSign`, and
  `auth.deviceKey` answers with the public key alone. The upstream keeps only
  the public key, in a `deviceCred` node under the device's node, and grants the
  connection exactly the subjects that device needs to sync. The credential
  authorizes the one device node it sits under: one under the upstream's own
  root node, or under any node that is not a device, authorizes nothing, and a
  credential marked `pending` (one a device enrolled itself with) authorizes
  nothing until an operator clears it. See
  [Device credentials](../user/sync.md#device-credentials) for the workflow.
- **A user's sign-in JWT**, presented with the user's node ID as the NATS user
  and password, grants the groups that user belongs to and nothing else. This is
  how the web UI connects; see [Browser](#browser).

`SIOT_DEVICE_AUTH` (or `--deviceAuth`) selects how the two combine:

- `optional` (the default) accepts the shared token from anywhere.
- `required` accepts the shared token only from loopback connections, so every
  remote connection has to present a device credential. This is the setting for
  a fleet on the public internet once every device has a credential. The HTTP
  port's WebSocket proxy applies the same rule itself, since the NATS server
  sees every proxied connection arrive from `127.0.0.1`: a `CONNECT` carrying a
  token from a remote address is meant to be refused before it is forwarded. The
  check reads only the first frame and can be passed by splitting or
  reformatting the `CONNECT`, so today `required` does not limit the token on
  the HTTP port (follow-ups item 11). A connection arriving through a reverse
  proxy on the same host still looks local, on the HTTP routes and the WebSocket
  alike, so a reverse proxy has to pass the client's address through or the
  token has to stay off that path. Treat the shared token as a secret under
  either setting.

An **enrollment token** is a third, narrower credential: a connection presenting
one may publish to `enroll.request` and subscribe to its reply inbox, and
nothing else. The token is presented together with a key, which signs the
connection nonce, so the upstream gives the connection an inbox of its own. The
request has to name that same key, since its reply can only be delivered to that
key's inbox; a request naming another key is refused. The device ID has to be
usable as a subject token (no period, wildcard, or whitespace). It exists so a
device with no credential can ask for one; see
[Devices that enroll themselves](../user/sync.md#2-devices-that-enroll-themselves).
Only a hash of the token is stored, in an `enrollToken` node.

A token is honored only under the root or a live `group` node, and in one place.
A token under a device or a deleted group, or one mirrored into two groups,
enrolls nothing, and the enroller logs why. A device the enrollment creates is
placed under the token's parent, approved at once only when the token says so. A
device node that already exists, with or without a credential, gets any further
key as pending whatever the token says, and its edges are left alone; so a token
holder cannot take over a known device, move one, or undo a credential that was
disabled to revoke it. At most 100 devices may wait for approval at once,
counted when a device is created.

A token holder can still name an existing device in another group and queue a
pending credential there, and learns from the reply whether an ID exists
(`approved` for a new device, `pending` for an existing one). Each credential
records the token that made it as `enrolledBy`, which is how an operator tells a
credential from another group's token before approving it.

### How the authorizer works

The authorizer (`server/auth.go`) is set as `CustomClientAuthentication` in the
`server.Options` the embedded server is started with (`server/nats-server.go`).
The options configure no users, tokens, or accounts of their own, so NATS hands
every client connection on every listener (NATS, WebSocket, and MQTT) to the
authorizer's `Check` method. `AlwaysEnableNonce` makes the server send a nonce
in `INFO` for devices to sign. Once the store is serving, Simple IoT calls the
authorizer's `start` with a client connection, the `*server.Server`, and the
store, which it needs for the steps below.

- **Deciding.** `Check` reads the `CONNECT` options. An NKey is a device: the
  signature over the connection nonce is verified, then the key is looked up
  among the credentials, or the token sent with it among the enrollment tokens.
  A user name with a JWT as the password is a browser. Anything else is compared
  with the shared token. A JWT that is present but invalid is refused outright
  rather than tried as a token.
- **Granting.** On success, `Check` calls `RegisterUser` on the connection with
  the subject permissions for that credential, derived at that moment, and for a
  browser a `ConnectionDeadline` at the JWT's expiry, so the server closes the
  connection itself when the token expires. The authorizer records each
  connection's ID with what it was granted.
- **Answering from memory.** Credentials, enrollment tokens, and users' places
  in the tree are held in an in-memory index, so `Check` never waits on a NATS
  request. The index is loaded once the store is serving; until then, device and
  user connections are refused and retry.
- **Keeping grants current.** The authorizer subscribes to `up.root.>` and acts
  on the changes that affect a grant: a credential or enrollment token changed,
  a device or signed-in user moved, a password changed. It closes each affected
  connection with `DisconnectClientByID`, and the client reconnects through
  `Check` and is granted the new set, or refused. Every five seconds it also
  compares the server's live connections with the index and closes any whose
  grant no longer holds.

`server/auth_spike_test.go` checks the server hooks this relies on, so it is the
test to run first after upgrading nats-server.

#### Compared with auth callout

[Auth callout](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/auth_callout)
is how a standalone NATS deployment hands authentication to its own service. The
authorizer does the same job for an embedded server:

|               | Authorizer                                            | Auth callout                                                                                    |
| ------------- | ----------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| Configured in | Go, `server.Options.CustomClientAuthentication`       | The server configuration, `authorization { auth_callout { ... } }`                              |
| Runs          | In the server process; NATS calls `Check` directly    | A separate service; the server sends an authorization request over NATS and waits for the reply |
| Grants with   | `RegisterUser`: permissions and a connection deadline | A user JWT in the reply: permissions and an expiry                                              |
| Revokes with  | `DisconnectClientByID`                                | A short expiry, or the system account's `$SYS.REQ.SERVER.<id>.KICK` request                     |
| Works with    | An embedded server only                               | Standalone servers, clusters, and operator mode                                                 |

In both, the credential a client presents proves who it is, and the service
decides what it may do when it connects, so neither needs permissions written
into the credential. A deployment that outgrows one embedded server, or runs
against an external NATS cluster (`-natsDisableServer`), would move this logic
into an auth callout service; see
[External NATS servers](#external-nats-servers).

### What a device credential allows

A device with root ID `X`, connecting to an upstream with root ID `R`, is
granted these subjects and nothing else. Permissions are derived from the device
ID at connect time; nothing about them is stored or configurable.

| Purpose                        | Subjects                                                                                                                                                               |
| ------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Find the upstream root         | `nodes.root.all`                                                                                                                                                       |
| Check whether it is adopted    | `nodes.all.X`                                                                                                                                                          |
| Announce itself under the root | `ep.X.R`                                                                                                                                                               |
| Push its origin stream         | `inst.X.X.>`, `$JS.API.STREAM.INFO.inst_X_X`                                                                                                                           |
| Pull its configuration         | `$JS.API.STREAM.INFO.inst_X_R`, `$JS.API.CONSUMER.CREATE.inst_X_R.>`, `$JS.API.CONSUMER.INFO.inst_X_R.*`, `$JS.API.CONSUMER.MSG.NEXT.inst_X_R.*`, `$JS.ACK.inst_X_R.>` |
| Receive replies                | subscribe `_INBOX_<key>.>`, the connection's own inbox                                                                                                                 |

A device never needs `p.>`, `up.>`, `auth.*`, `admin.*`, or another instance's
streams, and the permission set refuses them. Replies arrive on an inbox named
for the device's public key rather than the `_INBOX.>` every client on a server
shares, so one device cannot read another's replies; the device sets the same
prefix on its connection.

The upstream creates `inst_X_X` itself before a credential authorizes anything,
since a stream create request carries a whole configuration (extra subjects, a
republish, sources) and not just a name. A stream by that name whose
configuration the upstream did not create is replaced, with what it held. The
upstream also creates `inst_X_R`, the stream it writes the device's
configuration to, when it adopts the device, so the device pulls it by name and
never lists the upstream's streams.

The upstream merges what a device writes to `inst.X.X.>` only where it stays
inside boundary `X`; see [What the store checks](#what-the-store-checks).

An instance with no shared token is open, and accepts a device key it does not
know the way it accepts a connection with no credentials at all: with full
access. A key it does know is scoped as above.

### Revocation

The upstream keeps an index of credentials in memory, rebuilt from its tree and
kept current as the tree changes. Disabling a credential, marking it pending,
deleting it, moving it under another node, or deleting the device it sits under
removes it from the index, and the upstream closes every connection
authenticated with it. The device's sync client sees the refusal, records
`credential refused by upstream` on its sync node, keeps running standalone, and
tries again every minute. Enabling the credential again lets it back in with
what it queued.

Disabling or deleting an enrollment token closes connections made with it and
refuses new ones; devices already enrolled are unaffected. Moving a token
changes where the next new device is created and nothing else.

`lastConnect` and `connected` on each credential, and `online` on each device
node with credentials, are maintained by the upstream.

### What the store checks

The permission set keeps a device's writes in its own stream, `inst_X_X`. The
store then decides what in that stream reaches the tree:

- a point is merged when its node is `X` or under `X`;
- an edge is merged when its parent is `X` or under `X`, and its child is new,
  already under `X`, or in another device's subtree. A node of the upstream's
  own tree, one reached from its root without passing through a device, cannot
  be moved under `X` this way, and neither can an edge close a loop.

A message that does not qualify is held and logged, and merged if a later edge
places it inside the boundary, since a stream delivers messages in the order
they were written rather than tree order. The start-up load applies the same
rule. Streams the instance wrote itself, and those its own upstream writes for
it, are trusted.

When the store finds a replica stream for a boundary that is not a node in its
tree, it logs a warning naming the stream. That is what a device deleted from
the tree while its stream remains looks like, so the stream is still consumed.

### Browser

[How the browser talks to Simple IoT](frontend.md#how-the-browser-talks-to-simple-iot)
describes the whole path and the reasons for it; this section covers what is
granted and enforced.

The web UI connects to NATS over the WebSocket the HTTP port proxies (see
[configuration](../user/configuration.md#environment-variables)), presenting the
user's node ID and sign-in JWT as user and password. The authorizer validates
the JWT against the store's key, confirms it was issued to that user, and grants
the connection exactly the subtrees the user belongs to. An _anchor_ is a node
the user sits directly under; a user in two groups has two.

| Purpose                   | Subjects                                                |
| ------------------------- | ------------------------------------------------------- |
| Fetch nodes, write points | publish `u.<anchor>.<user>.>` for each anchor           |
| Ask who it is             | publish `auth.me`                                       |
| Live points               | subscribe `up.<anchor>.>` for each anchor               |
| Replies                   | subscribe `_INBOX_<user>.>`, the connection's own inbox |

Nothing else: no `p.>`, `nodes.>`, `ep.>`, `$JS.>`, `auth.user`, or `admin.>`.
The server proves the connection may speak for the anchor and user in a `u.*`
subject; the store checks the target of the request against the anchor and sets
the origin of every point to the user, whatever the browser sent. For a node
read or a point write the target is the node. For an edge write both ends are
checked: the parent has to be under the anchor, and the child has to be under it
already (or be a node new to the tree, or one the user deleted from there and is
restoring), so a node from elsewhere cannot be attached into a group and reached
through the new edge. The store also refuses any edge whose parent already sits
below its child, from any writer, and bounds every walk of the tree, so a loop
cannot be made and would not hold a request open if it were. Neither side needs
the other's data structures, and no header can be added or left out to get a
different outcome. Details of the subjects are in the
[API reference](api.md#nats).

What a browser can and cannot do, compared with polling the HTTP API:

| Concern                              | Before                                        | Now                                                                                                                        |
| ------------------------------------ | --------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| Credential on the WebSocket          | Shared token, handed out by `auth.getNatsURI` | The user's JWT; `auth.getNatsURI` is gone                                                                                  |
| Anonymous WebSocket connection       | Accepted when no token is configured          | Refused when a token is configured; an open instance stays open                                                            |
| Read scope                           | Any node in the instance                      | The user's anchors and below                                                                                               |
| Write scope                          | Any subject, any origin                       | Points under the user's anchors, origin forced to the user                                                                 |
| JetStream, admin, auth subjects      | Reachable                                     | Not in the permission set                                                                                                  |
| A page on another origin             | Could connect with the shared token           | Cannot authenticate: the JWT lives in this origin's local storage. `SIOT_NATS_WS_ORIGINS` can refuse the handshake as well |
| User removed from a group or deleted | Access until the JWT expires                  | Disconnected within seconds; reconnecting recomputes or refuses                                                            |
| Password changed                     | Access until the JWT expires                  | Disconnected; the browser returns to sign-in                                                                               |
| JWT expiry                           | Not enforced on a live connection             | The server closes the connection when the token expires                                                                    |

Permissions are computed when the connection is made. The authorizer watches the
tree and closes a user's connections when the user's edges change, so the
browser reconnects and is granted the new set, or is refused if the user is
gone. The HTTP node routes apply the same scope; see [HTTP](#http). A deployment
with no shared token still runs open on every NATS listener, WebSocket included;
the UI presents its JWT either way and is scoped either way.

A user signs in on the instance whose tree holds it, and a user under a device
node belongs to that device. Sync replicates a device's user nodes to its
upstream with the rest of the device's tree, and those users are not accepted at
the upstream's sign-in, whoever set the password: an upstream operator editing a
device user's password is administering the device, and a device left with its
default account would otherwise be a default account on the upstream. The store
decides this from the tree (`userIsLocal`, `store/jetstream.go`), not from which
instance wrote the password point, so a restart or a copy of the point in
another stream cannot change the answer.

### Secrets in node reads

Some points hold credentials: a user's `pass` (a bcrypt hash), the `authToken`
of a sync, message service, database, or Particle node, a sync node's
`enrollToken`, a Twilio `sid`, and a Wi-Fi `psk`. A node reply to a browser
(through `u.<anchor>.<user>.nodes.>`) or to a user or device over HTTP carries
these points with an empty value, so the UI can show that a credential is set
without receiving it; writing the point replaces the stored value as usual. A
reply to a browser asking for a node's parents lists only the parents inside the
user's groups. The server's own clients read on the plain `nodes.>` subject,
which the shared token alone reaches, and see the values. `auth.me` leaves
`pass` out. `siot export` leaves every secret point out unless `-secrets` is
given.

The instance's own NKey seed is kept in `SIOT_DATA/device.nkey` and never as a
point or a reply; see [NATS](#nats).

### External NATS servers

The authorizer is part of the embedded server. An instance started with
`-natsDisableServer` against an external NATS server relies on that server's own
configuration for both tokens and device credentials.

The authorizer covers devices, enrollment, and users in process. If a fleet
grows to where connect rate or multi-tenancy calls for it, the NATS
[security model](https://docs.nats.io/nats-concepts/security) (accounts,
decentralized auth) is the escalation path:

- [NATS authentication](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/auth_intro)
- [NATS authorization](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/authorization)

## Known limitations

An audit at `v0.28.0` (September 2026) reviewed the HTTP API, the NATS
authorizer, the store, the clients, the frontend, dependencies, and packaging.
Everything it found that was a bug or a missing check was closed in the release
that followed. A second audit at `1b37e2ab` found items 8 to 12, listed below;
see the
[security cleanup plan](https://github.com/simpleiot/simpleiot/blob/master/plans/2026-08-24-security-cleanup.md)
for what shipped. The items below are still open. The numbers refer to the
[security follow-ups plan](https://github.com/simpleiot/simpleiot/blob/master/plans/2026-09-19-security-followups.md),
which has the detail and the proposed change for each.

| Area     | Limitation                                                                                                                                                                                                         | Plan item |
| -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------- |
| Users    | Any account can add an `update`, `ntp`, `networkManager`, or `browser` node in its own group, and the server acts on it: an account is control of the host. See [What an account can do](#what-an-account-can-do). | 1         |
| Users    | Any account can create a device credential, an auto-approve enrollment token, or a `sync` node in its own group; a `sync` node replicates the whole tree to a server the user runs and accepts writes back.        | 10        |
| Devices  | `SIOT_DEVICE_AUTH=required` does not limit the shared token through the HTTP port's WebSocket proxy, since the check reads only the first frame.                                                                   | 11        |
| Users    | Any member of a group can write any node under it, including another member's password.                                                                                                                            | 2         |
| Users    | The sign-in limiter keys on the account, so a known email can be locked out for up to five minutes at a time by anyone who can reach the sign-in page.                                                             | 4         |
| Devices  | Behind a reverse proxy every connection arrives from loopback, so `SIOT_DEVICE_AUTH=required` does not limit the shared token on the HTTP port.                                                                    | 4         |
| Users    | The sign-in token lasts seven days; it is refused once the user is gone, but not on a password change until the NATS connection is closed.                                                                         | 3         |
| Defaults | The first account is `admin`/`admin` and nothing forces a change. It is left to the deployment checklist.                                                                                                          | 6         |
| Releases | Releases carry checksums and no signature, so `siot update` trusts the release host.                                                                                                                               | 5         |
| Clients  | Update payloads are not signed; a point can still start a download and restart, from the release host, over HTTPS.                                                                                                 | excluded  |

What the audit found sound: the binary point and node decoders bound every
length and count; no client runs a command through a shell, disables TLS
verification, or builds a database query from strings; the metrics scraper
bounds its reads; a credentialed device cannot publish to NATS subjects outside
its own boundary, and the store merges only what stays inside it; point types
and keys cannot carry NATS wildcards; the JWT check rejects other signing
algorithms; the web UI renders node values as text, keeps its token out of URLs
and logs, and loads no third-party script; no secrets are committed to the
repository; and `govulncheck` reports no called vulnerability in the current
source.
