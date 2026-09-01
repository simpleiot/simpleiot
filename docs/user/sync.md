# Synchronization

Simple IoT provides for synchronized upstream connections via NATS or NATS over
WebSocket.

![upstream](images/multiple-upstream.png)

To create an upstream sync, add a sync node to the root node on the downstream
instance. If your upstream server has a name of `myserver.com`, then you can use
the following connections URIs:

- `nats://myserver.com:4222` (4222 is the default NATS port)
- `ws://myserver.com` (WebSocket unencrypted connection)
- `wss://myserver.com` (WebSocket encrypted connection)

IP addresses can also be used for the server name.

Auth token is optional and needs to be
[configured in an environment variable](configuration.md) for the upstream
server. If your upstream is on the public internet, you should use an auth
token, or better, a [device credential](#device-credentials), which limits each
device to its own data and can be revoked on its own. If both devices are on an
internal network, then you may not need either.

Typically, `wss` are simplest for servers that are fronted by a web server like
Caddy that has TLS certs. For internal connections, `nats` or `ws` connections
are typically used.

Occasionally, you might also have edge devices on networks where NATS outgoing
connections on port 4222 are blocked. In this case, it's handy to be able to use
the `wss` connection, which just uses standard HTTP(S) ports.

![sync](images/upstream.png)

## How synchronization behaves

Synchronization works by replicating the JetStream streams that store each
instance's data — see the [synchronization reference](../ref/sync.md) for how
this works. The behavior you will observe:

- **First connect:** the device announces itself and appears under the upstream
  root node; its full tree (structure, configuration, and history) then arrives
  through replication. Configuration written on the upstream for a device that
  has not connected yet is delivered on first connect.
- **Offline changes catch up.** Changes made on either side while the connection
  is down are delivered when it comes back — replication resumes exactly where
  it left off, and only missed data is sent. See
  [Queuing while offline](#queuing-while-offline) below.
- **Both sides can edit.** Configuration can be changed on either instance; the
  newest change wins everywhere.
- **Deleting a device on the upstream detaches it.** The device keeps running
  standalone and does not add itself back; undelete the device node on the
  upstream to resume synchronization.

## Queuing while offline

An edge instance does not need its upstream to keep working. It writes every
point to its own local store first, and the sync client replicates that store
upstream. When the connection drops, the instance keeps collecting data, running
rules, and accepting local configuration changes, all of which queue on disk.

On reconnect:

- The backlog is sent in order, with the original timestamps, so history
  upstream has no gap.
- Only the missed messages are sent. Replication resumes at the position it
  reached before the outage, which keeps the recovery cheap on a metered or low
  bandwidth link.
- Clients that act on current values (rules, protocol clients, the UI) see one
  update per changed value once the backlog drains rather than a replay of every
  intermediate reading, so a device coming back online does not re-trigger rules
  on stale data.
- History consumers still receive every point. A Db client feeding a time-series
  database reads the stream with its own durable consumer, so the backlog
  reaches the database as well.

Configuration written upstream while a device is offline, or before it has ever
connected, waits and is delivered on the next connect.

How long a device can be offline and still catch up in full depends on how much
history the store keeps. The default is 20,000 points per value, which is
adjustable per instance. See [Store](../ref/store.md#retention-and-durability)
for the setting, and the [synchronization reference](../ref/sync.md) for how the
queuing works.

## Schema

The configuration of a sync node:

```yaml
nodes:
  - sync:
      authToken: your-auth-token
      description: Cloud
      disabled: 0
      uri: wss://myserver.com
```

`uri` is the upstream connection, written as one of the forms described above.
`authToken` matches `SIOT_AUTH_TOKEN` on the upstream server. Leave it out to
connect with the instance's [device key](#device-credentials) instead; the
client then writes the public key on the node as `pubKey`, which is why an
export of a running node carries one.

A sync node belongs on the root node of the downstream instance, so a file that
carries one leaves `parent` out and it attaches to the device node this instance
runs as.

An export carries `authToken` as it was entered, so treat a file that contains
sync nodes the way you would treat the token itself.

The count of synchronizations is a point the client maintains, so an export of a
running node carries it as well.

## Device credentials

Every instance has a device key, generated the first time it starts and kept in
`device.nkey` under `SIOT_DATA`. The key is the instance's identity when it
connects to an upstream: a sync node with no `authToken` signs the upstream's
connection challenge with it, so the secret never leaves the device. The public
half is shown on the sync node as `pubKey`, and `siot key show` prints it.

An upstream accepts a device key when a `deviceCred` node under the device's
node carries the matching `pubKey`. The credential limits the connection to that
one device: it can push its own data and pull the configuration written for it,
and nothing else. A device cannot publish as another device or read another
device's configuration, and the upstream holds only public keys, so an export or
a copy of its store gives away nothing that could impersonate a device. The
[security reference](../ref/security.md#nats) lists exactly what a credential
allows.

Revoking access is one action: disable the credential (or delete it, or delete
the device node) and the upstream closes the device's connection and refuses it
from then on. Nothing else in the fleet is affected. The device keeps running on
its own and tries again every minute, so re-enabling the credential brings it
back with everything it queued while it was out.

The upstream records `lastConnect` and `connected` on each credential, which is
how to tell whether a device has picked up a new key.

### Enrolling a device

1. Let the device sync once with the shared `authToken`, which creates its
   device node on the upstream, or create the device node some other way.
2. On the upstream, add a credential under the device's node with the device's
   public key. Read the key off the device's sync node, or run `siot key show`
   on the device. Any of these does it:
   - In the UI, add a **Device credential** node under the device node and paste
     the key into **Public key**.
   - `siot cred add -device "Gateway 42" -pubKey UBXF...`, where `-device` is
     the device node's ID or description.
   - A provisioning or import file:

     ```yaml
     nodes:
       - deviceCred:
           parent: Gateway 42
           description: gateway 42
           pubKey: UBXF3PJZPNL5CW35ECS2U3XG5EODFUUQ6XRNEZTJA3EFNP33K7Z54UCM
     ```

3. Clear `authToken` on the device's sync node. The device reconnects with its
   key.
4. Once every device has a credential, set `SIOT_DEVICE_AUTH=required` on the
   upstream so the shared token is accepted only from the upstream's own host.
   See [configuration](configuration.md#environment-variables).

`siot cred list` shows every credential with its device, state, and whether it
is connected; `siot cred disable ID`, `siot cred enable ID`, and
`siot cred rm ID` change one. All of them take the usual `-natsServer` and
`-token` options, so they work against a remote upstream.

### Keys issued by the upstream

A key can also be made on the upstream and delivered to the device, which suits
building images and bench setup. The seed is shown once and never stored:

- In the UI, add a **Device credential** under the device node and press
  **Generate key**. The public key is stored on the credential and the seed is
  shown until you leave the page.
- `siot cred add -device "Gateway 42"` with no `-pubKey` does the same and
  prints the seed.
- `siot key gen` prints a seed and public key without touching any instance, for
  scripts that create the credential some other way.

Give the device the seed one of these ways:

- On the device's sync node in the UI, paste the seed into **Install key**. It
  goes straight to the key file through the API, never through a point.
- `siot key install SEED` on the device.
- A provisioning or import file with a top level `deviceKey` entry, which
  applying the file installs. A file that carries one is a secret; keep one file
  per unit (`provisioning/<unit>.yaml`) rather than a shared one.

  ```yaml
  deviceKey: SUAB...
  nodes:
    - sync:
        description: Cloud
        uri: wss://myserver.com
  ```

- Ship it as `SIOT_DATA/device.nkey` in the image.

A running instance switches to the installed key straight away. Treat the seed
as you would the shared token: it is the device's identity.

`siot export` leaves `authToken` points and the device key out and says so at
the top of the file; `siot export -secrets` includes both, for backing up or
cloning an instance in full.

### Rotating a key

A device node may carry more than one credential, so a key is rotated without
downtime: generate a new key, add a second `deviceCred` with its public key,
install the seed on the device, wait for `lastConnect` on the new credential,
then disable the old one.

## Videos

There are also several videos that demonstrate upstream connections:

### [Simple IoT upstream synchronization support](https://youtu.be/6xB-gXUynQc)

<iframe width="791" height="445" src="https://www.youtube.com/embed/6xB-gXUynQc" title="Simple IoT upstream synchronization support" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>

### [Simple IoT Integration with PLC Using Modbus](https://youtu.be/-1PuBoTAzPE)

<iframe width="791" height="445" src="https://www.youtube.com/embed/-1PuBoTAzPE" title="Simple IoT Integration with PLC Using Modbus" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>
