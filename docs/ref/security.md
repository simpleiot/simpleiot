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
- Change the `admin` password, on every instance. A device's users replicate to
  its upstream and can sign in there, so a default account left on one device is
  a default account on the upstream.
- Give each device a [credential](../user/sync.md#device-credentials) and set
  `SIOT_DEVICE_AUTH=required` on the upstream. Prefer enrollment tokens that
  leave credentials pending over ones that approve automatically.
- Expose only the HTTP port, behind a reverse proxy that terminates TLS.
  Firewall the NATS (4222), NATS WebSocket (9222), and NATS monitoring (8222)
  ports unless devices connect to 4222 directly, in which case configure
  `SIOT_NATS_TLS_CERT` and `SIOT_NATS_TLS_KEY`. All listeners bind every
  interface.
- On an edge device, firewall all incoming ports unless the local UI is needed.
  Anyone who can write a point can configure the update, rule, and network
  clients, so write access to an instance is equivalent to control of the host.
- Keep a Modbus TCP server node on a trusted network. Modbus has no
  authentication, and the listener binds every interface.
- Set `SIOT_NATS_WS_ORIGINS` to the origin the UI is served from.
- Leave `-debugHttp` off in production; it logs sign-in requests and replies.
- Run a release built with a current Go toolchain, and keep it updated with
  `siot update`.

## Server

For cloud/server deployments, we recommend installing a web server like Caddy in
front of Simple IoT. See the [Installation page](../user/installation.md) for
more information.

## Edge

Simple IoT Edge instances initiate all connections to upstream instances;
therefore, no incoming connections are required on edge instances and all
incoming ports can be firewalled. By default the HTTP, NATS, NATS WebSocket, and
NATS monitoring listeners bind every interface, and an edge instance often runs
with no auth token, so the firewall is what keeps the local network out.
`SIOT_NATS_WS_HOST` and `SIOT_NATS_HTTP_HOST` bind the WebSocket and monitoring
listeners to loopback, which `siot install` does, and the installed service is
given a generated token; see [installation](../user/installation.md).

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
  token from a remote address is refused before it is forwarded. A connection
  arriving through a reverse proxy on the same host still looks local, on the
  HTTP routes and the WebSocket alike, so a reverse proxy has to pass the
  client's address through or the token has to stay off that path. Treat the
  shared token as a secret under either setting.

An **enrollment token** is a third, narrower credential: a connection presenting
one may publish to `enroll.request` and subscribe to its reply inbox, and
nothing else. The token is presented together with a key, which signs the
connection nonce, so the upstream gives the connection an inbox of its own. The
request has to name that same key, since its reply can only be delivered to that
key's inbox; a request naming another key is refused. The device ID has to be
usable as a subject token (no period, wildcard, or whitespace). A device that
already has a live credential gets any further key as pending, whatever the
token says, so a token holder cannot take over a known device; and at most 100
devices may wait for approval at once. It exists so a device with no credential
can ask for one; see
[Devices that enroll themselves](../user/sync.md#2-devices-that-enroll-themselves).
Only a hash of the token is stored, in an `enrollToken` node.

### What a device credential allows

A device with root ID `X`, connecting to an upstream with root ID `R`, is
granted these subjects and nothing else. Permissions are derived from the device
ID at connect time; nothing about them is stored or configurable.

| Purpose                              | Subjects                                                                                                                                                               |
| ------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Find the upstream root               | `nodes.root.all`                                                                                                                                                       |
| Check whether it is adopted          | `nodes.all.X`                                                                                                                                                          |
| Announce itself under the root       | `ep.X.R`                                                                                                                                                               |
| Push its origin stream               | `inst.X.X.>`, `$JS.API.STREAM.INFO.inst_X_X`, `$JS.API.STREAM.CREATE.inst_X_X`                                                                                         |
| Discover streams for its boundary    | `$JS.API.STREAM.NAMES`                                                                                                                                                 |
| Pull each origin `o` writing into it | `$JS.API.STREAM.INFO.inst_X_o`, `$JS.API.CONSUMER.CREATE.inst_X_o.>`, `$JS.API.CONSUMER.INFO.inst_X_o.*`, `$JS.API.CONSUMER.MSG.NEXT.inst_X_o.*`, `$JS.ACK.inst_X_o.>` |
| Receive replies                      | subscribe `_INBOX_<key>.>`, the connection's own inbox                                                                                                                 |

A device never needs `p.>`, `up.>`, `auth.*`, `admin.*`, or another instance's
streams, and the permission set refuses them. Replies arrive on an inbox named
for the device's public key rather than the `_INBOX.>` every client on a server
shares, so one device cannot read another's replies; the device sets the same
prefix on its connection. Stream names are one subject token and cannot be
matched by prefix, so the origins a device may pull from (the upstream itself,
and any higher upstream writing configuration for the device) are enumerated
when it connects. When a new origin stream appears for a device's boundary, the
upstream closes the device's connection and it reconnects with the new stream
included.

Two things to know about the boundary of this model:

- `$JS.API.STREAM.NAMES` answers with the names of every stream on the upstream,
  which are instance IDs. A credentialed device can therefore learn which other
  instances exist, but nothing about them.
- An instance with no shared token is open, and accepts a device key it does not
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
refuses new ones; devices already enrolled are unaffected.

`lastConnect` and `connected` on each credential are maintained by the upstream.

### What the store checks

JetStream does not record who published a message, so the permission set is the
enforcement point and the store cannot tell a device's write from anyone else's.
What it does check: when it finds a replica stream for a boundary that is not a
node in its tree, it logs a warning naming the stream. That is what a write that
got past the permissions looks like, and also what a device deleted from the
tree while its stream remains looks like, so the stream is still consumed.

### Browser

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

A user signs in on the instance that holds the user's password. Sync replicates
a device's user nodes to its upstream with the rest of the device's tree, and
those users are not accepted at the upstream's sign-in: the password was set by
whoever administers the device, and a device left with its default account would
otherwise be a default account on the upstream.

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
Most of what it found was closed in the release that followed; the items below
are still open. The numbers refer to the
[security cleanup plan](https://github.com/simpleiot/simpleiot/blob/master/plans/2026-08-24-security-cleanup.md),
which has the detail and the proposed change for each.

| Area     | Limitation                                                                                                                                                                    | Plan item |
| -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- |
| Users    | Any member of a group can write any node under it, including another member's password.                                                                                       | 20        |
| Users    | The sign-in token lasts seven days; it is refused once the user is gone, but not on a password change until the NATS connection is closed.                                    | 8         |
| Defaults | The first account is `admin`/`admin` and nothing forces a change; the NATS monitoring port has no authentication. Both are left to the deployment checklist and the firewall. | 4, 6      |
| Releases | Releases carry checksums and no signature, so `siot update` trusts the release host.                                                                                          | 20        |
| Clients  | Update payloads are not signed; a point can still start a download and restart, from the release host, over HTTPS.                                                            | excluded  |

What the audit found sound: the binary point and node decoders bound every
length and count; no client runs a command through a shell, disables TLS
verification, or builds a database query from strings; the metrics scraper
bounds its reads; a credentialed device cannot publish outside its own boundary
or read another connection's replies; point types and keys cannot carry NATS
wildcards; the JWT check rejects other signing algorithms; the web UI renders
node values as text, keeps its token out of URLs and logs, and loads no
third-party script; no secrets are committed to the repository; and
`govulncheck` reports no called vulnerability in the current source.
