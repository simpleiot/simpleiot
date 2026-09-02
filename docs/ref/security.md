# Security

Users and downstream devices will need access to a Simple IoT instance. Simple
IoT currently provides access via HTTP and NATS.

## Server

For cloud/server deployments, we recommend installing a web server like Caddy in
front of Simple IoT. See the [Installation page](../user/installation.md) for
more information.

## Edge

Simple IoT Edge instances initiate all connections to upstream instances;
therefore, no incoming connections are required on edge instances and all
incoming ports can be firewalled.

## HTTP

The Web UI uses JWT (JSON web tokens) issued at login.

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

NOTE, it is important to set an auth token or use device credentials; otherwise
there is no restriction on accessing the device API.

## NATS

The embedded NATS server authenticates every connection on every listener (NATS,
WebSocket, and MQTT) through one authorizer inside Simple IoT, so there is no
NATS accounts file to manage. Two kinds of credential are accepted:

- **The shared token** (`SIOT_AUTH_TOKEN`) grants full access. The server's own
  client, the `siot` command line tools, and MQTT clients (which send it as the
  password) use it. When no token is configured the instance is open, as it
  always has been.
- **A device credential** is an NKey pair. The device keeps the seed in
  `SIOT_DATA/device.nkey` and signs the connection challenge with it; the
  upstream keeps only the public key, in a `deviceCred` node under the device's
  node, and grants the connection exactly the subjects that device needs to
  sync. See [Device credentials](../user/sync.md#device-credentials) for the
  workflow.

`SIOT_DEVICE_AUTH` (or `--deviceAuth`) selects how the two combine:

- `optional` (the default) accepts the shared token from anywhere.
- `required` accepts the shared token only from loopback connections, so every
  remote connection has to present a device credential. This is the setting for
  a fleet on the public internet once every device has a credential. A
  connection arriving through a reverse proxy on the same host looks local, so
  `required` limits the token only on ports that are reached directly.

An **enrollment token** is a third, narrower credential: a connection presenting
one may publish to `enroll.request` and subscribe to its reply inbox, and
nothing else. It exists so a device with no credential can ask for one; see
[Devices that enroll themselves](../user/sync.md#3-devices-that-enroll-themselves).
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
| Receive replies                      | subscribe `_INBOX.>`                                                                                                                                                   |

A device never needs `p.>`, `up.>`, `auth.*`, `admin.*`, or another instance's
streams, and the permission set refuses them. Stream names are one subject token
and cannot be matched by prefix, so the origins a device may pull from (the
upstream itself, and any higher upstream writing configuration for the device)
are enumerated when it connects. When a new origin stream appears for a device's
boundary, the upstream closes the device's connection and it reconnects with the
new stream included.

Two things to know about the boundary of this model:

- `$JS.API.STREAM.NAMES` answers with the names of every stream on the upstream,
  which are instance IDs. A credentialed device can therefore learn which other
  instances exist, but nothing about them.
- An instance with no shared token is open, and accepts a device key it does not
  know the way it accepts a connection with no credentials at all: with full
  access. A key it does know is scoped as above.

### Revocation

The upstream keeps an index of credentials in memory, rebuilt from its tree and
kept current as the tree changes. Disabling a credential, deleting it, moving it
under another node, or deleting the device it sits under removes it from the
index, and the upstream closes every connection authenticated with it. The
device's sync client sees the refusal, keeps running standalone, and tries again
every minute. Enabling the credential again lets it back in with what it queued.

`lastConnect` and `connected` on each credential are maintained by the upstream.

### What the store checks

JetStream does not record who published a message, so the permission set is the
enforcement point and the store cannot tell a device's write from anyone else's.
What it does check: when it finds a replica stream for a boundary that is not a
node in its tree, it logs a warning naming the stream. That is what a write that
got past the permissions looks like, and also what a device deleted from the
tree while its stream remains looks like, so the stream is still consumed.

### External NATS servers

The authorizer is part of the embedded server. An instance started with
`-natsDisableServer` against an external NATS server relies on that server's own
configuration for both tokens and device credentials.

Long term we plan to leverage more of the NATS
[security model](https://docs.nats.io/nats-concepts/security) for user
authentication:

- [NATS authentication](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/auth_intro)
- [NATS authorization](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/authorization)
