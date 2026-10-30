# Large Files and OTA Updates over NATS

- Author: Cliff Brake, last updated: 2026-10-19
- PR/Discussion:
- Status: discussion

## Problem

Edge devices need software updates, and update images are large: tens to
hundreds of megabytes for a Yoe image, sometimes approaching the 1 GiB cap the
[update client](../user/update.md) enforces. Many devices sit on cellular,
satellite, or radio links that are slow, metered, and drop for seconds to hours
at a time.

Update images are one case of a general need: delivering a file larger than a
NATS message to the devices that use it. MCU firmware for the
[serial client](../user/mcu.md) and CAN databases are others, and the
[file node](../user/file.md) that holds them today is limited to 1 MB.

We want a large file to reach a device over the connection it already has to its
upstream, and to finish reliably on a poor link: a dropped connection or a
reboot resumes where it left off, and no byte crosses the link twice.

## Context/Discussion

### What exists today

The **update client** (`client/update.go`) is the only working way to move an
image to a device. It polls an HTTPS server for `files.txt` and downloads
`<prefix>_<version>.upd` with a single `GET`. The download has a one-hour
timeout and no resume, so an interruption restarts it from zero. It needs a
separate update server that every device can reach directly, with its own TLS
and access control outside SIOT.

The **file node** keeps its contents in a `data` point, base64 encoded when
binary. The point replicates through sync like any other, which is what makes a
file uploaded upstream appear on the device, and it is also what limits a file
to the 1 MB NATS payload. The store keeps each upload as history, so replacing a
large file repeatedly accumulates copies. Readers call `File.GetContents()` and
receive a `[]byte`. File nodes are tree scoped (`data/edge_role.go`), so one
file node can be mirrored under several parents.

An older NATS transfer, **`SendFile`/`ListenForFile`**
(`client/file-transfer.go`), pushes a file as 50 KB request/reply chunks on
`device.<id>.file`. Nothing in `siot` uses it any more: its only sender,
`StartUpdate` in `client/update.go`, is commented out, and its only receiver is
the `cmd/edge` example. Device credentials cannot subscribe to
`device.<id>.file`, so it would not reach a credentialed device in any case. It
also holds the whole file in memory and has no resume.

### What the sync connection already provides

Every device keeps one authenticated NATS connection to its upstream, over
`nats`, `tls`, `ws`, `wss`, or the compressed `natsz`/`tlsz` transports. It
works behind NAT and firewalls, carries a per-device credential with a scoped
set of subjects (`devicePermissions` in `server/auth.go`), and already resumes
JetStream replication at a stream sequence after an outage (see the
[synchronization reference](../ref/sync.md)). A transfer that rides this
connection inherits all of that, with no second server to run or secure.

### NATS Object Store

The
[JetStream Object Store](https://docs.nats.io/nats-concepts/jetstream/obj_store)
stores large files in a stream named `OBJ_<bucket>`:

- Each object is split into chunks, published in order on
  `$O.<bucket>.C.<nuid>`, where the NUID is unique to that version of the
  object. The default chunk size is 128 KiB and the writer can choose another.
- A metadata message on `$O.<bucket>.M.<encoded-name>` records the name, size,
  chunk count, NUID, and a SHA-256 digest of the whole object. The stream keeps
  only the latest metadata per name, and replacing an object removes the old
  chunks.
- The stream has direct get enabled, so a chunk or the metadata can be read with
  a single request.

The nats.go `Get` reads an object through an ordered consumer and fails if the
connection drops, so it does not resume on its own. The stream layout does
support resuming: an object's chunks are the messages on one subject, in order,
so a read that asks for the next message on that subject after a given stream
sequence picks up exactly where an earlier read stopped.

### Options considered

1. **Add HTTP range requests to the update client.** This gives resume with a
   small change, and is worth doing for the HTTPS path regardless. It keeps the
   separate update server, and it does nothing for other large files.
2. **Revive `SendFile`.** Add resume, stream to disk, and grant device
   credentials its subject. The upstream would have to track every device's
   progress and push to each one, and a push design does not let the device pace
   the transfer to its own link.
3. **JetStream sourcing.** Have the device's JetStream source the object stream
   from upstream, which resumes natively. Sourcing across servers needs a leaf
   node connection and JetStream domains, which SIOT does not use.
4. **An update-only bucket.** The upstream keeps images in a `siot-updates`
   bucket, and every device may read the whole bucket. This suits public images,
   but chunk subjects change with every version, so subject permissions cannot
   limit a device to its own files, and the mechanism stays specific to updates.
5. **Large file nodes.** A file node above a size threshold keeps its contents
   in an Object Store bucket instead of a point. The node's points replicate as
   they do today and announce a new version; each instance that holds the node
   fetches the contents from its upstream through a request that the store
   authorizes against the device's tree. OTA updates are a mode of the existing
   update node that installs images from its child file nodes.

## Decision

Use option 5. The update node keeps its HTTPS mode (with range requests added)
for devices that have no upstream and gains a mode that installs from file
nodes.

### File node storage

Each instance has an Object Store bucket named `files`, with a configurable size
limit. The object name is the file node's ID.

- Contents up to **256 KiB** stay in the `data` point, as today, so small files,
  provisioning, and YAML export and import work unchanged.
- Larger contents go to the bucket, written with a **32 KiB chunk size**. At 2
  KB/s, a 128 KiB chunk takes over a minute, and a drop partway through discards
  it. A 32 KiB chunk bounds the loss to about 16 seconds at that rate, and NATS
  message overhead stays under 1%.
- `hash` becomes the SHA-256 of the contents for every file node, matching the
  Object Store digest. A large file also has an `object` point with the NUID of
  the current version, and an empty `data` point.
- `File` gains `Open() (io.ReadCloser, error)`, which reads from either place.
  `GetContents()` remains for readers that want the whole file in memory.

Large contents are written on the instance at the top of the tree that will use
them, typically the cloud, through an HTTP endpoint that streams an upload into
the bucket (used by the UI) or a `siot` command line upload. Contents travel
down the tree only. A large file uploaded on a device stays on that device, and
its node says so in its `error` point.

### Fetching contents

The file client on each instance watches the `hash` and `object` points. A large
file whose object is not complete in the local bucket is fetched:

- **Eagerly** by default, so a CAN database or MCU firmware is ready when its
  reader needs it.
- **On demand** when the file node is a child of an update node, so a device
  pulls only the image it will install. The update client asks for the fetch
  once a version is chosen, and a gateway fetches when one of its downstream
  devices requests the file.

A fetch proceeds as follows:

1. It sends `files.<deviceID>.<nodeID>` with the NUID, the stream sequence of
   the last chunk it holds (zero to start), and a batch size.
2. The upstream store checks that the node is in that device's tree, as
   `handleUserRequest` does for users, and that the NUID is the current one. It
   then reads up to the batch size of chunks with direct get, starting after the
   given sequence, and sends each to the reply inbox tagged with its sequence,
   followed by an end marker. An upstream that does not have the object yet,
   such as a gateway still fetching, replies so, and the device retries later.
3. The device publishes each chunk into its own bucket on the same subject,
   `$O.files.C.<nuid>`, and records nothing else. The last message on that
   subject is the resume point, so a dropped connection or a reboot continues
   from the next chunk with no state file.
4. When the byte count reaches the size, the device checks the SHA-256 and
   writes the metadata message, which makes the object complete and readable
   with the standard Object Store API.

The upstream keeps no per-device state, and the device controls the pace by how
many chunks it asks for at a time. If the `object` point changes during a fetch,
the device purges the partial chunks and starts the new version. When a file
node is deleted or its contents replaced, each instance removes the old object
from its bucket.

Because the device stores the object under the same NUID, a gateway serves its
own devices from its copy, and a file crosses each link once however many
devices below it use it.

Progress is reported in a `fetched` point on the file node, keyed by instance
ID, sent at most once a minute or per 5%, since points replicate upstream and
spend the same link.

### OTA updates

The existing update node gains a `source` point with two modes. The client's
flow is the same in both: list the available versions in `osUpdate`, choose one
(`downloadOS`, or automatically with `autoDownload`), place the image in
`directory`, report `osDownloaded`, and with `autoReboot` install and reboot.
The modes differ only in how versions are listed and images obtained, which
`updateFetcher` already separates from the rest of the client.

| Step          | `https` (today, the default)                    | `files`                                                                        |
| ------------- | ----------------------------------------------- | ------------------------------------------------------------------------------ |
| List versions | `files.txt` on `uri`, polled every `pollPeriod` | `name` of each child file node, `<prefix>_<version>.upd`, as its points arrive |
| Obtain image  | HTTPS `GET` into `directory`                    | fetch the child's contents, check free space, and copy them to `directory`     |

`prefix`, `directory`, `osUpdate`, `downloadOS`, `osDownloaded`,
`discardDownload`, `versionOS`, `autoDownload`, `autoReboot`, and the UI around
them apply to both modes. `uri`, `pollPeriod`, and `refresh` apply only to
`https`.

- To update a device, add or replace a file node under its update node. Several
  child file nodes list several versions, so choosing an older one manually
  still works.
- To update a fleet, upload one file node and mirror it under each device's
  update node. The update node itself holds per-device state (`versionOS`,
  `osDownloaded`), so it is not mirrored.
- The update node is already a primary type in `data/edge_role.go`, so it runs
  only on the device it updates.
- Release workflows that publish to an HTTPS update server can keep doing so: an
  update node on the cloud in `https` mode can create a file node from each new
  image it finds.

### Permissions

Device credentials gain one publish subject, `files.<deviceID>.*`. The store
authorizes each request against the device's tree, so a device can read only
files in its own tree and cannot write, delete, or list anything. The
`auth_spike_test.go` canary gains cases for a file in the device's tree and one
outside it.

### Concerns

- **Authenticity.** The digest proves the contents arrived intact, and TLS
  proves they came from the upstream. Neither proves who built an image.
  Signature verification belongs in the installer (the Yoe updater or RAUC) and
  is out of scope here, but a device running a plain `nats` or `ws` transport
  relies entirely on that check.
- **Space on the device.** An image sits in the bucket and in the installer's
  directory until it is installed, so a device briefly needs twice its size. The
  update client could remove the bucket copy after the copy out, which would
  leave the device unable to serve the file to a downstream instance.
- **Link sharing.** A large fetch competes with telemetry on a narrow link.
  Small batches let TCP share the link fairly; a rate limit on the sync node is
  an option if that proves insufficient.
- **Compression.** Images are already compressed, so the `natsz` transport
  spends some CPU on them for no saving. The cost is small compared with the
  link time.

## Consequences

- Large files use the connection, credential, and transport a device already
  has. No separate update server needs to be reachable from the field.
- A fetch on a link that drops many times, or a device that reboots during one,
  still finishes, and each byte crosses each link once.
- The file node size limit rises from 1 MB to the bucket limit, and replacing a
  large file no longer accumulates copies in the store's history.
- OTA updates reuse the tree and the existing update node: a device's update is
  a file node under its update node, and a fleet update is a mirrored file node.
- The upstream does not track per-device progress.
- Device permissions grow by one subject, authorized per request by the store.
- `SendFile`, `ListenForFile`, `NatsSendFileFromHTTP`, the commented-out
  `StartUpdate`, the `cmd/edge` example with its `siot_edge_run` helper, and the
  `FileChunk` protocol buffer are removed, leaving Sparkplug B as the only use
  of protocol buffers.

For the changelog: file node `hash` changes from MD5 to SHA-256, so a stored
`hash` changes once after the upgrade. File nodes gain `object` and `fetched`
points. The update node gains `source`; a node without it uses `https` and
behaves as it does today.

## Additional Notes/Reference

- [NATS Object Store](https://docs.nats.io/nats-concepts/jetstream/obj_store)
- [Object Store ADR (ADR-20)](https://github.com/nats-io/nats-architecture-and-design/blob/main/adr/ADR-20.md)
- [Yoe Updater](https://github.com/YoeDistro/yoe-distro/blob/master/docs/updater.md)
- Delta updates (casync, zchunk, bsdiff) would reduce the bytes sent and are
  compatible with this transport, since a delta is just a smaller file. They are
  a separate decision.
