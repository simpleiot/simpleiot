# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

For more details or to discuss releases, please visit the
[Simple IoT community forum](https://community.tmpdir.org/c/simple-iot/5)

## [Unreleased]

## [0.26.2] - 2026-08-27

### Changed

- **Shelly devices are read from what they report, not from a list of models.**
  A Gen2 or later device answers with its own component list, so any such device
  works, including ones released after this release and ones with an add-on
  module attached. Support for cover, energy meter, temperature, humidity, and
  battery components comes with this. Existing device nodes pick up the change
  on restart: `type` becomes the model the device reports, such as
  `SNPL-00116US` rather than `PlugUS`, and a new `gen` point records the
  generation. See the [Shelly documentation](docs/user/shelly.md).
- **Shelly status arrives by push instead of by polling.** Simple IoT holds a
  WebSocket open to each Gen2 or later device and receives changes as they
  happen, so a relay switched at the wall shows up right away, and a device
  going offline is noticed when the connection drops. The whole device is still
  read once a minute as a backstop, down from every two seconds per component.
  Gen1 devices continue to be polled every two seconds.

### Fixed

- **Mirroring a hardware node created before this release now marks the roles.**
  A GPIO line, Modbus bus, or other node that owns hardware kept no record of
  where it lived until edge roles arrived in 0.26.1, so mirroring one onto an
  upstream instance left both edges unmarked and a second client started on the
  upstream. Mirroring such a node now marks the edge it was mirrored from as the
  primary and the new edge as a mirror. A mirror made before this release still
  carries no role; remove it and mirror again to have both edges marked. See
  [Primary and mirror edges](docs/ref/data.md#primary-and-mirror-edges).
- **Shelly status updates arrive right away again.** A Gen2 or later device
  binds its pushed status to the name a connection registers under, and answers
  a later connection that reuses a name still held without ever notifying it.
  Every connection used the same name, so a connection the device had not yet
  released, such as one left behind by a previous run, silenced the one after
  it: control took effect immediately while the status it produced waited for
  the once-a-minute read. Each connection now registers under a name of its own.
- **Shelly discovery no longer misses devices on a busy network.** A scan handed
  each mDNS answer straight to the code that reads the device, and the scan
  discarded any answer that arrived while the previous one was still being read.
  On a network with a dozen responders roughly half of them were lost each
  minute, so a device could go unfound indefinitely. A scan now collects every
  answer before reading any device.
- **Shelly Plus 1PM, Plus i4, and RGBW2 devices now report correctly.** The Plus
  1PM read as unsupported, only the first of the Plus i4's four inputs appeared,
  and the RGBW2 reported nothing. Gen1 relay control now uses the right address
  and works.

## [0.26.1] - 2026-08-26

### Fixed

- **Mirrored hardware nodes no longer run a second client.** Mirroring a Modbus
  IO, Shelly IO, GPIO line, MQTT connection, or other hardware node into a group
  now creates a view of it, so only the instance where the node actually lives
  talks to the device. For MQTT this also stops a mirrored connection from
  building a second copy of the node tree its topic schema creates. Nodes
  mirrored before this release keep the old behavior until the mirror is removed
  and created again. See the
  [data reference](docs/ref/data.md#primary-and-mirror-edges).
- **Setting a value on a mirrored node reaches the device.** A node mirrored
  from a device into a group on an upstream instance now stays owned by that
  device, so a `valueSet` written on the mirror, by a rule or from the UI,
  travels down and is acted on where the hardware is. Previously such a write
  was stored on the upstream and never arrived. See
  [Across sync boundaries](docs/ref/data.md#across-sync-boundaries).

### Changed

- **Deleting a node removes its mirrors.** Mirrors of a deleted sensor or bus no
  longer linger in the groups they were mirrored into. Removing a mirror still
  leaves the node itself alone.
- **Nodes that belong to a parent can only be mirrored, not moved.** A Modbus
  IO, Shelly IO, rule condition, and similar nodes are found through their
  parent, so moving one elsewhere left it inert. The UI now offers `mirror` for
  these instead, and says where the node belongs.
- **Mirrored nodes are labeled in the tree**, so a node that shows values but
  runs nothing is no longer mistaken for one that has stopped reporting.

## [0.26.0] - 2026-08-24

### Added

- **Read a switch or drive a relay from a GPIO line.** A new `gpio` node reads
  or drives one line on a Linux GPIO character device, publishing `value` on
  every change and taking `valueSet` to drive an output, so a rule can watch a
  float switch and run a pump. Set the chip to `sim` to develop rules before the
  hardware exists. See the [GPIO client](docs/user/gpio.md).

### Changed

## [0.25.1] - 2026-08-24

- **User passwords are now stored hashed, not in plaintext.** The store, sync
  streams, and exports carry only a bcrypt hash, so a copy of any of them no
  longer reveals passwords. Existing passwords convert automatically the next
  time each user signs in, and the password field in the UI now shows blank and
  sets a new password when typed into. See
  [Users/Groups](docs/user/users-groups.md).

## [0.25.0] - 2026-08-20

### Added

- **Simple IoT now serves MQTT.** Set `SIOT_NATS_MQTT_PORT=1883` and gateways
  and sensors publish straight to Simple IoT, with no separate broker to deploy.
  The port is disabled by default, the auth token authenticates clients in the
  connect packet password field, and TLS settings serve the MQTT listener too.
  See the [MQTT page](docs/user/mqtt.md).
- **MQTT messages become points.** An `mqtt` node and its `mqttSub` children map
  topics into points, with a JSON path such as `$.value` selecting a single
  value and a blank path turning each field of an object into its own point.
  Units, scale, and offset apply the way they do on a Modbus IO. See the
  [MQTT page](docs/user/mqtt.md#subscriptions).
- **Sparkplug B builds the node tree for you.** Set `sparkplug: true` on an
  `mqtt` node and the groups, edge nodes, and devices a gateway announces appear
  as nodes, one point per metric, each carrying a tag naming its Sparkplug
  identity. Nodes go offline on a death certificate rather than disappearing,
  and tags or descriptions you set on them survive a rebirth. See the
  [MQTT page](docs/user/mqtt.md#sparkplug-b).
- **A topic schema creates nodes as data arrives.** Set
  `topicSchema: "{site}/{gateway}/{device}"` on an `mqtt` node and matching
  topics build that node path themselves, each level carrying a tag named by its
  schema label. Explicit `mqttSub` children still take precedence, and
  `maxNodes` (1000 by default) bounds what a topic level carrying an unbounded
  value can create. See the
  [MQTT page](docs/user/mqtt.md#automatic-nodes-with-a-topic-schema).
- **MQTT device nodes show the values they hold.** Expanding a node a topic
  schema created lists every point the device publishes next to its current
  value, the way a serial device node does.

### Changed

- **The Simple IoT title in the documentation header links to simpleiot.org.**
  Selecting the name at the top of any documentation page now returns you to the
  main site.

## [0.24.1] - 2026-08-20

### Added

- **Notify actions take a repeat interval.** Set in minutes on a rule's notify
  action, it re-sends the notification while the rule stays active and keeps
  that action from notifying more often than the interval no matter how often
  the rule transitions. Leaving it unset keeps one notification per transition
  with no rate limit. See the
  [rules documentation](docs/user/rules.md#notifications).
- **Rule conditions can hold themselves active with `minInactive`.** Set in
  minutes on a condition, it keeps the condition met until its input has been
  clear for that long, so a value oscillating around a threshold is one incident
  and one notification rather than one per cycle. See the
  [rules documentation](docs/user/rules.md#conditions).

### Changed

- **`minActive` on a rule condition is now enforced.** A condition with
  `minActive` set has to hold continuously for that many minutes before it is
  considered met, so a brief spike no longer activates the rule. The field has
  been stored and shown in the UI for some time without doing anything, so a
  rule that already carries a non-zero `minActive` changes behavior on upgrade.
  See the [rules documentation](docs/user/rules.md#conditions).

### Fixed

- **Rule actions run only when the rule changes state.** Editing a rule, a
  condition, or an action while the rule was active used to re-run the actions,
  which re-sent the notification and rewrote any `setValue` output. Disabling an
  active rule now publishes the rule inactive and runs its inactive actions
  once. A restart resumes in the rule's persisted state without re-running
  anything.
- **Notify actions can find the node that fired the rule again.** The lookup
  used a form of the node request the JetStream store rejects, so every notify
  action logged an error instead of sending its notification.

## [0.24.0] - 2026-08-20

### Added

_The message service client has not been thoroughly tested yet, testing feedback
welcome._

- **Notifications work again, with SMS, email, and ntfy push delivery.** A rule
  notify action or the web UI's message function now reaches users through
  Twilio SMS, email (SMTP), or an [ntfy](https://ntfy.sh) topic, configured on a
  Messaging Service node. ntfy needs no user nodes — every notification in the
  service's scope is pushed to the topic. Delivery is deduplicated, so a user
  mirrored into several groups gets one message. Notifications now travel as
  ordinary points, so they are stored, synchronized, and visible in history; the
  old `node.<id>.not` and `node.<id>.msg` NATS subjects are gone. See the
  [notifications](docs/user/notifications.md) and
  [messaging](docs/user/messaging.md) documentation.
- **The database page describes TimescaleDB support as a planned addition.** The
  [database page](docs/user/database.md) outlines how points would map to a
  hypertable, how graphing and configuration would differ from the existing
  options, and what TimescaleDB offers that the current ones do not, including
  storing text.
- **A PLC page describes the options for reading data from controllers.** The
  [PLC page](docs/user/plc.md) covers which client to use for which protocol,
  what Modbus supports today for Allen-Bradley Logix and other controllers, and
  the planned MQTT, Sparkplug B, OPC UA, and EtherNet/IP support. It also covers
  Opto 22 groov, Phoenix Contact PLCnext, and Ignition, how PLC data maps to
  VictoriaMetrics labels for graphing, and notes that the NATS server embedded
  in Simple IoT includes an MQTT server, so an MQTT deployment would not need a
  separate broker.
- **A high availability reference describes the options and their constraints.**
  The [high availability reference](docs/ref/high-availability.md) covers what
  the store and synchronization design already protect, what running against an
  external or hosted NATS cluster would take, and what happens to points
  published while the application is stopped.

## [0.23.5] - 2026-08-18

### Fixed

- **Logging in to an upstream instance now signs you in as that instance's own
  user.** Sync copies a downstream instance's users upstream, so an upstream can
  hold more than one user with the same email and password. Login previously
  picked one of them at random, which could show the downstream device's tree
  instead of the upstream's. The user closest to the root now wins.

## [0.23.4] - 2026-08-13

### Added

- **`siot dump` describes a running instance for troubleshooting.** It reports
  the instance root ID, the tree with node IDs and every parent of each node,
  deleted nodes, and anything holding a second root. `-points` adds each point's
  origin and timestamp, `-streams` adds the replication stream inventory, and
  `-all` turns on both. See the
  [configuration reference](docs/user/configuration.md).

### Changed

- **Sync replicates in windows instead of one message at a time.** A pump now
  sends up to 256 messages before waiting for the receiving instance to confirm
  them, so the round trips overlap. A first sync of a large stream, such as the
  one that follows an upstream store reset, finishes in a fraction of the time.

### Fixed

- **A failed replication window is resent rather than skipped.** A message the
  receiving instance did not accept used to be left for redelivery while later
  messages kept flowing, which could store an older point after a newer one on a
  subject. A window that fails is now retried as a unit with none of it
  acknowledged, so each subject arrives in source order.
- **An upstream no longer adopts a downstream instance's root as its own.**
  Every instance anchors its tree with a virtual `root` parent, and the store
  was loading that edge out of replica streams, so an upstream ended up with two
  root nodes and could serve the downstream's as its own. The upstream then
  exported the downstream's tree and started clients for its hardware. An
  upstream already in this state recovers on restart.

## [0.23.3] - 2026-08-12

- **Prometheus scrape in the metrics client.** A `prometheus` metrics type
  scrapes an application's `/metrics` endpoint and publishes the samples as
  points, so a service can keep its endpoint on loopback and still report
  upstream — no open port, VPN, or second agent. Metric names become point types
  and labels become point keys, and counters also publish the change since the
  previous scrape so a rule can act on them. See the
  [metrics documentation](docs/user/metrics.md).
- **Key label expansion in the db client.** On by default, each label in a point
  key written as a label set becomes its own database label, so a scraped series
  queries the way the Prometheus series it came from did, `histogram_quantile`
  included. Keys written by other clients are left alone. See the
  [database documentation](docs/user/database.md).
- **JetStream streams are compressed with S2, on by default.** A store of
  100,000 scraped points went from 33.4 MB to 11.7 MB in testing.
  `--storeCompression` (or `SIOT_STORE_COMPRESSION`) accepts `s2` or `none`.
  Existing instances pick this up on the next start and their messages stay
  readable. See the [store reference](docs/ref/store.md).
- **Default store retention raised from 5000 to 20,000 points per subject.**
  About four months of 10-minute data, or two weeks of per-minute data.
  Compression absorbs most of the extra disk. Existing instances pick this up on
  the next start; `--storeMaxMsgsPerSubject` (or
  `SIOT_STORE_MAX_MSGS_PER_SUBJECT`) lowers it on a device with little flash.
- **The store logs its effective retention and compression at startup.** Both
  are resolved from a flag, an environment variable, and a default, and neither
  was visible once an instance was running.
- **The metrics UI lists readings only.** The node's own configuration points
  are shown as inputs above the table and no longer repeated inside it.

### Fixed

## [0.23.2] - 2026-08-11

- store: reject points whose type or key contains a period, whitespace, or a
  NATS wildcard. A point travels on a subject ending in its type and key, and
  listeners read the node ID and parent ID from fixed positions in that subject,
  so a period added a token and the point was delivered as an edge point to the
  wrong handler. This showed up as repeated
  `error merging new points: no matching struct with a node:"id" field matching <id>`
  messages, and the affected points never reached the client that owned the
  node. Rejected points are logged with their type and key, and an `error` point
  is set on the node so the sender is visible in the UI.

  **This affects existing devices.** A device sending a key such as
  `PCIe_bridge_0.95V` will have those points rejected until the sender is
  updated to use a name without a period. Points already stored keep their keys
  and continue to display.

### Changed

- metrics: keys generated from names the client does not control -- kernel
  device names, mount points, and network interface names -- now have characters
  that are invalid in a point key replaced with underscores. A cooling device
  the kernel calls `devfreq-17000000.gpu` is now reported under
  `devfreq-17000000_gpu`, and a VLAN interface `eth0.100` under `eth0_100`.
- docs: the documentation site now uses the simpleiot.org palette and typography
  — the near-black surfaces, orange accent, Inter, and JetBrains Mono. The dark
  theme is the default; the built-in mdBook themes remain available in the theme
  picker.

## [0.23.1] - 2026-08-10

### Changed

- db: tag points are now inherited from ancestor nodes. A tag set on a group,
  device, or site node is applied to every point written from nodes beneath it,
  with the value nearest the emitting node winning when the same tag is defined
  at more than one level. Resolution stops at the Database client's parent node
  (inclusive). This changes output for existing users of `tagPointType`: points
  now also carry tags set on ancestor nodes. See the Database client
  documentation for the resolution rules.
- update NATS dependencies (nats.go v1.52.0, nats-server v2.14.4). Sourcing
  across a leaf connection now relies on the periodic source health check to
  rebuild a source consumer after the origin server restarts, so a hub replica
  can take roughly 20 seconds to catch up following a device restart.
- docs: describe the VictoriaMetrics `-search.latencyOffset` flag in the
  [database documentation](docs/user/database.md). Its 30 second default delays
  when freshly written points become visible to queries, and the documentation
  covers setting it on the command line, per request, and through a systemd
  environment file.
- docs: give the [database documentation](docs/user/database.md) a section on
  what happens while the database is unavailable, and restate the custom tag
  instructions as the two steps they take: add a tag point to the node, then
  list its point type on the Database node.

### Fixed

- db: points sent while the time-series database is down are now written when it
  comes back instead of being lost. Stream deliveries are batched and written
  synchronously, and are acknowledged to JetStream only after the database
  accepts them, so an unreachable database holds its points in the stream and
  retries with a backoff that grows to one minute. Previously the points were
  acknowledged as soon as they were handed to the InfluxDB client's background
  writer, whose retry buffer discards batches after three minutes, which left a
  gap in the recorded history. High-rate points are still best-effort, since
  they are not stored in streams.
- db: tag and description edits are reflected in the tags written with
  subsequent points again. Since the boundary-origin stream rework, cached tag
  values were not refreshed until the client restarted or its `tagPointType`
  list was changed.

## [0.23.0] - 2026-08-08

### Breaking changes

- store: existing SQLite databases must be migrated via `siot export` using
  v0.22.0 and then `siot import`
- sync: deleting a device node on the upstream now detaches it — the device no
  longer re-creates itself; only the upstream can restore it
- sync: the Sync Period setting is gone from the sync node config and UI. Stage
  3 replication is continuous, so there is no periodic sync interval left to
  configure
- db: the `history.<nodeId>` NATS API and the `data.HistoryQuery`,
  `data.HistoryResults`, and `data.TagFilters` types that supported it are
  removed. The API built Flux queries, so it only ever worked with InfluxDB, and
  nothing used it: the user interface has no graph views, and the query was
  never reachable from the HTTP API. If you call this subject from an external
  client, query InfluxDB directly instead. Grafana remains the way to graph time
  series data, as described in the
  [graphing documentation](docs/user/graphing.md)

### Store

- replace the SQLite store and hash tree with JetStream boundary-origin streams,
  `inst_<boundaryID>_<originID>` (ADR-7 Stage 2). Streams are created per
  sync/AuthZ boundary rather than per node, subjects carry both routing tokens,
  and current state is the merge of subject tips with a deterministic origin
  tie-break
- add in-memory point and edge caches for fast lookups. The point cache is
  pre-populated at startup and loads on a cache miss, fixing config points being
  lost after the first write following a restart
- add boundary resolution to the edge cache (`IsBoundary`, `OwningBoundary`)
- migrate a node's subjects, and its owned subtree, to the new boundary stream
  when a move or undelete changes its owning boundary, preserving original point
  timestamps
- consume replica streams from other instances, merge them into the caches with
  the ADR-7 tip rule, and re-broadcast changed tips locally with a `Siot-Origin`
  header. Remote-origin data is never written to local origin streams, so sync
  cannot echo
- validate that new edges carry a `nodeType` point before writing to JetStream,
  so the stream and edge cache cannot diverge
- an edge now holds one point per type and key. The store appended a repeated
  point in the same message rather than replacing what it had
- reply once and stop processing when an edge point DB write fails

### Store configuration

- add `--storeMaxMsgsPerSubject` / `SIOT_STORE_MAX_MSGS_PER_SUBJECT` to bound
  per-subject history (current state is always preserved). It defaults to 5000
  messages, about a month of 10-minute data; configuration subjects are
  effectively unlimited, and `-1` means unlimited. Each instance applies its own
  retention to replica streams it discovers, so hub and device retain
  independently
- add `--storeSyncInterval` / `SIOT_STORE_SYNC_INTERVAL` to tune the JetStream
  fsync interval (`always` fsyncs every write) for power-loss durability on edge
  devices

### Sync

- rewrite upstream sync on boundary-origin stream replication (ADR-7 Stage 3):
  durable-consumer replication in both directions with resumable offline
  catch-up, adoption announcement on first connect, and no hash tree walks

### Database client

- read points from the store streams with durable consumers instead of the
  `up.>` wire subjects, so external history is gap-free across sync outages,
  instance restarts, and the client's own downtime (high-rate points still
  arrive over `phrup.>`)
- add a database type selector to the Database node so you can choose between
  InfluxDB 2.x and Victoria Metrics. Existing nodes keep their current behavior,
  as an unset type is treated as InfluxDB
- when Victoria Metrics is selected, write only the numeric `value` field and
  skip string points entirely. Victoria Metrics stores numeric samples only and
  converts any other field value to 0, so the previous `text` field produced a
  `points_text` series that was always 0. Grafana with a Victoria Metrics data
  source is the way to graph this data
- discard points until the Database node has a valid URI. An unset or malformed
  URI produced a write API that failed on every point and filled the log with
  errors, so the client now skips the write entirely and acknowledges stream
  messages, then connects as soon as a usable URI is set

### Points and encoding (#742)

- replace Point `Value` / `Text` with a unified `DataType` / `Data` encoding
  (ADR-7 Stage 1), replace protobuf point encoding with a compact binary format,
  and add the `dataType` field to the Elm frontend Point type
- move edge point NATS subjects to the `ep.` prefix and add type and key to node
  point subjects
- add `MarshalYAML` for human-readable point export, and fix JSON/YAML unmarshal
  when `dataType` is set but `data` is empty
- add OOM protection to `DecodePoints`

### Rules

- a play audio action now reads the `device`, `channel`, and `filePath` points
  the UI writes. The rule client had been reading `pointDevice`, `pointChannel`,
  and `pointFilePath`, so an action configured in the UI played nothing and
  tried to open an empty path
- a play audio action that cannot open its wave file, or whose file has an
  unusable sample rate, now records the error on the action node instead of
  exiting the application

### Frontend

- the signal generator summary line now reads the value from the configured
  destination point type and key instead of always reading `value`, so
  generators writing to a custom point type (for example `volt`) show a live
  reading rather than a constant `0`

### Import and export

- a node created from a file now keeps a single `nodeType` edge point. A file
  carries the node type, and sending a node added a second one, so an export of
  an imported tree did not match the file it came from
- an export no longer writes the `nodeType` edge point. It repeated the key the
  node is already written under, so every node in a file carried an `edgePoints`
  block that said nothing. Import derives the type from the key, and a file that
  still carries one is read as before

### Other fixes

- fix a data race between a node watcher and its caller when a node has a keyed
  point that decodes into a map. Merging an update wrote into the map the caller
  was already holding, so decoding now replaces the map with a copy
- fix a Shelly mDNS data race by creating fresh params per scan (#742)

### Documentation, tests, and dependencies

- document the configuration schema of each client in the user documentation,
  written in the format `siot export`, `siot import`, and provisioning share, so
  the points a node is configured with can be read in one place and copied into
  a file
- update docs to reflect the new point encoding and NATS subjects (#742)
- the db client test starts a throwaway VictoriaMetrics instance itself (skipped
  when the binary is not installed), replacing the external-InfluxDB
  requirement, and verifies the point path end to end
- update NATS dependencies (nats.go v1.49.0, nats-server v2.12.5)

## [0.22.0] - 2026-08-04

- config files: one YAML format now describes a tree of nodes, and
  `siot export`, `siot import`, and provisioning all use it. The node type is
  the key and each point type is a key of its own, so a file reads as
  configuration rather than as a serialized structure:

  ```yaml
  nodes:
    - group:
        description: Sensors
        children:
          - modbus:
              description: Modbus sensors
              port: /dev/ttyS1
              baud: 9600
  ```

  How a value is written decides what it becomes: `9600` is numeric, `"9600"` is
  text, `1` and `1.5` are an integer and a float, a mapping is a set of keyed
  points, and a sequence is an array. Files in the previous format are not read;
  re-export the tree to get one in the current format. See
  [user/configuration](docs/user/configuration.md).

- config files: a file describes configuration and nothing else. It carries no
  node IDs, no origins, and no points without a value. A `nodeID` point names
  the node it refers to by description, so a rule can point at a variable
  without either of them knowing its UUID, in a file or across files.

- import: applying a file is now idempotent. Nodes are matched by description
  rather than by ID, so importing a file creates what is missing, sends only the
  points that differ, and does nothing when the tree already agrees. Running the
  same import twice does what running it once did. A `delete` list removes
  nodes.

  A description is how a file finds a node, so renaming one in the UI detaches
  it from the file that describes it and the next apply creates a second node
  beside it. Rename in two steps: delete the old description in the same file
  that introduces the new one.

- import: `-parentID` and `-preserveIDs` are gone. A file says where its nodes
  attach with `parent`, naming a node by description, and every node a file
  creates gets a fresh ID. `-dryRun` prints what a file would do without
  applying it. The `(import)` suffix that was appended to descriptions is gone,
  since it would have broken idempotence.

- import: importing at the root no longer replaces the root node. A file never
  describes the root, which is this instance rather than configuration, so the
  restart that an import at `root` used to require is gone along with the root
  watcher that performed it. `siot serve` still exits non-zero on error, so
  supervisors that only restart on failure will bring it back up.

- export: an export is now a provisioning file. The root node is left out, as
  are node IDs, valueless points, and origins. An export refuses to write a tree
  whose siblings share a description, since no file could say which node it
  meant.

- provisioning: an instance can configure itself from files. A directory given
  by `-provisioningDir` or `SIOT_PROVISIONING_DIR`, defaulting to
  `<SIOT_DATA>/provisioning` when it exists, is applied at start-up and whenever
  a file changes, so a unit built from an image comes up configured with no
  import step. `SIOT_PROVISIONING_INTERVAL` sets how often to look for changes
  the watch might have missed.

  Files can also be uploaded through the UI as `file` nodes under the
  provisioning node, which is how a unit whose filesystem you cannot reach gets
  configured. Files on disk apply first and uploads layer on top, oldest first.

  A `provisioning` node under the root records what was applied, with a checksum
  per file and the last error if one failed. Removing a file removes its status;
  the nodes it created stay, since provisioning describes what should exist
  rather than owning what it made.

- provisioning: `siot provision -dir <dir>` prints what a directory of files
  would do to a running instance without applying it, and `-check` only parses
  them, which needs no instance and is what a build can use to fail on a bad
  file.

## [0.21.0] - 2026-08-03

- modbus: move Modbus to the client architecture. Modbus buses are now started
  and stopped by the client manager like every other client, rather than by a
  20-second poll of the store, so a new bus or IO starts collecting data right
  away instead of up to 20 seconds later. Node and point types are unchanged, so
  existing configurations keep working.
- modbus, 1-wire: start a bus that lives inside a group. Discovery looked only
  at the direct children of the root node, so a bus placed anywhere else never
  ran.
- modbus: adding or removing an IO now restarts its bus, which reopens the port.
  A Modbus server drops the TCP connections it holds when this happens. This
  applies when a person edits the configuration, not during normal polling.
- modbus: read `int16` values as signed. An `int16` input or holding register
  was decoded as an unsigned value, so a negative reading appeared as a large
  positive number on both the client and the server.
- modbus: write a `valueSet` to the device as soon as it arrives. The check that
  decided whether a value could be written compared the data format against the
  IO type, which never matched, so a write waited for the next poll.
- modbus: treat a scale of zero as one. An IO created without a scale read every
  register as zero; the old code declined to start such an IO at all, which was
  just as hard to diagnose.
- modbus: publish the corrected response timeout when a bus has none configured
  or has a non-positive one, so what the bus uses is what the configuration
  shows.
- modbus: closing a Modbus TCP server no longer waits on a connection listener
  that has already exited, which could happen when the far end dropped the
  connection at the moment the server was being closed.
- modbus: the first tests for the Modbus subsystem, covering every register type
  and data format end to end through a server bus and a client bus talking over
  a TCP socket.
- 1-wire: **1-Wire bus nodes are no longer created automatically.** Add a 1-Wire
  node where you want it and set its `Index` to the bus controller number, which
  matches the `w1_bus_master<index>` directory in `/sys/bus/w1/devices`. The
  sensors on that bus are still detected and added for you. Existing bus nodes
  keep working unchanged; this affects new setups, and any install that relied
  on a deleted bus node reappearing.
- 1-wire: move 1-Wire to the client architecture. A new bus or sensor is picked
  up as soon as it is configured rather than on the next 20-second scan. This is
  also the first test coverage the subsystem has had.
- 1-wire: detect sensors below the configured bus controller rather than across
  all of them. With two controllers, every bus node claimed every sensor.
- server: retire the `node` package. With Modbus and 1-Wire moved to clients,
  all that remained of the node manager was writing the app and OS versions to
  the root node, which is now a small function that runs once after the store
  starts. The 20-second scan that drove the old bus managers is gone.
- import: restart automatically after importing with `-parentID=root`. Such an
  import replaces the root node, which leaves everything that resolved a node
  relative to the old root working against a tree that no longer exists, so the
  instance had to be restarted by hand before it resumed collecting data. The
  server now exits once the import settles and expects the service manager to
  start it again; the service file installed by `siot install` sets
  `Restart=always` for this reason. If you run Simple IoT some other way, please
  make sure the process is restarted. `siot serve` also exits non-zero on error
  now, so supervisors that only restart on failure will bring it back up.
- client manager: refresh the root node on every scan. The manager resolved the
  root once at startup, so it kept scanning below a root that an import had
  replaced and never started clients for the imported nodes. This also covers
  instances that are not configured to restart.
- client manager: stop clients when the last node of a type is deleted. The scan
  returned early when it found no nodes, which skipped the step that shuts down
  clients whose nodes are gone.

## [0.20.6] - 2026-08-01

- metrics client: collect fan speed and drive level from the hwmon interface,
  reported as `metricSysFanSpeed` in RPM and `metricSysFanPWM` on the 0-255
  scale the kernel uses. Drivers that report a plain `rpm` file rather than the
  usual `fan1_input`, such as the Tegra tachometer, are read as well.
- metrics client: collect the state of each thermal cooling device as
  `metricSysCoolingState`, keyed by device type. A state above zero means the
  thermal governor is limiting the system, so a rising `cpufreq` or `devfreq`
  state shows performance being given up to stay cool, which temperature alone
  does not reveal. The scale each device is measured against is published once
  at startup as `metricSysCoolingStateMax`.
- metrics client: collect rail voltage, current, and power from the hwmon power
  monitors, published as the existing `voltage`, `current`, and `power` types in
  volts, amps, and watts. A channel is published when its driver labels it,
  which is how a board names the rail a channel measures, so a Jetson reports
  `VDD_GPU_SOC` and friends by name. Monitors that do not report power directly,
  including the INA3221, still give voltage and current, and the product stands
  in for the missing reading.
- metrics client: collect the current clock of each CPU as `metricSysCPUFreq`,
  in MHz and keyed by `cpu0`, `cpu1`, and so on. Read alongside the cooling
  device states, it shows where the clocks settled once the thermal governor
  pulled them back.
- documentation: describe the thermal, power, and clock readings the system
  metrics collect, in [metrics](docs/user/metrics.md)

## [0.20.5] - 2026-08-01

- metrics client: read the Linux thermal zones directly so SoC temperatures are
  collected. The sensor library only consults the zones on systems that have no
  hwmon temperature inputs, so boards that expose both, such as the Jetson AGX
  Orin, reported only their board sensors while the CPU, SoC, and junction
  readings went uncollected. Zones whose rail is powered down are skipped
  individually, sensors that fail to read no longer discard the ones that
  succeeded, and repeated sensor names are numbered (`tmp451`, `tmp451_2`) so
  two readings no longer overwrite each other.

## [0.20.4] - 2026-08-01

- serial node UI: keep the remaining configuration fields out of the point and
  value table, so it shows the data the MCU reports rather than the node's own
  settings. `protocol`, `timeout`, `logConsole`, `syncParent`, `download`, and
  `progress` were listed there alongside MCU points. `protocol` and `timeout`
  are Modbus settings as well, so that node is tidier too.

## [0.20.3] - 2026-08-01

- serial client: exit the port reader when the port is closed. It previously
  fell through to its retry path, and because a closed port fails every read
  immediately, the goroutine spun at full speed for the life of the process. One
  was left behind on every close, including each disable/enable of a serial
  node.
- serial client: make the connected state self-correcting. It was published only
  when it changed, so a single update that was lost or overwritten left a node
  reported as not connected even while data kept arriving. The state is now
  recorded locally only after a successful publish, and in shell mode the
  watchdog republishes it on a slow cadence so the reported state converges.

## [0.20.2] - 2026-08-01

- update frontend assets

## [0.20.1] - 2026-07-31

- serial client: don't display log field in UI when in shell mode

## [0.20.0] - 2026-07-31

- serial: add a Zephyr shell protocol mode to the MCU serial client. Points are
  exchanged as lines of ASCII with an MCU's console shell rather than
  COBS-framed binary packets, so the link a developer reads is the link Simple
  IoT uses for data. Select it with the Protocol setting on a serial node; an
  empty value means binary, so existing nodes are unaffected. See the
  [MCU documentation](docs/user/mcu.md) and the
  [serial reference](docs/ref/serial.md).
- serial: add a "Log console output" option that mirrors the MCU console to the
  Simple IoT server log. Shell protocol only.

## [0.19.0] - 2026-07-31

- add GPS client that reads position data from a serial NMEA receiver, the gpsd
  daemon, or an internal simulator, and publishes latitude, longitude, altitude,
  speed, heading, fix status, and satellite information. See the
  [GPS documentation](docs/user/gps.md), which includes instructions for
  plotting a track on a Grafana geomap.
- the simulated GPS source now continues from the node's last published
  position, so a configuration change or an application restart picks the track
  up where it left off. A `Reset location` button, or editing the start
  latitude, longitude, or heading, moves the track back to the start.
- the simulated GPS source now logs the points it generates at debug level 4,
  matching the raw data the serial and gpsd sources log at that level. The
  [GPS documentation](docs/user/gps.md#debug-levels) now describes what each
  debug level covers.
- changing the GPS debug level now takes effect right away. The level was read
  once when a source started, so a change was ignored until something else
  restarted the source.
- the GPS `Rx count` and `Error count` reset checkboxes now clear after the
  counter is zeroed. They stayed set, so a reset only took effect on every
  second click.
- document what storing text points in Victoria Metrics actually does; the value
  is converted to 0 rather than the line being rejected
- (BREAKING) remove the legacy `gps` package, which predated the client
  architecture and is superseded by the GPS client. `data.GpsPos` is unchanged
  and still used by the modem code.
- add `siot update`, which updates Simple IoT to the latest release published on
  GitHub. It downloads the release for the platform it is running on, verifies
  it against the published checksums, and replaces the binary in place.
  `siot update -check` reports what is available without installing it. See the
  [installation documentation](docs/user/installation.md#updating).
- releases now publish the `siot` executable directly instead of wrapping it in
  a `.tar.gz` or `.zip` archive, so downloading and running it takes one less
  step. Asset names are unchanged apart from the archive extension, and Windows
  binaries now end in `.exe`.
- releases are now built and published by a GitHub Actions workflow when a
  version tag is pushed, and the release notes come from the matching
  `CHANGELOG.md` section

## [[0.18.5] - 2025-09-05](https://github.com/simpleiot/simpleiot/releases/tag/v0.18.5)

- add configurable response timeout parameter for Modbus clients with 100ms
  default

## [[0.18.4] - 2025-08-04](https://github.com/simpleiot/simpleiot/releases/tag/v0.18.4)

- don't return power information for Shelly devices that don't have power
  measurement (like Plus 1)
- support status for Shelly gen1 relays
- update Go packages to
  [latest versions](https://github.com/simpleiot/simpleiot/pull/752/commits/6f6a00ed4b27d0809ef28e96216631a2e8da9559)
- fix bug with importing a list of nodes
- add browser client allowing control and configuration of Yoe Kiosk Browser

## [[0.18.3] - 2025-03-20](https://github.com/simpleiot/simpleiot/releases/tag/v0.18.3)

- add favicon to frontend so icon displays in browser tabs (#756)

## [[0.18.2] - 2025-03-19](https://github.com/simpleiot/simpleiot/releases/tag/v0.18.2)

- fix bug where export/import displayed no nodes (#749). Fixed in (#753)

## [[0.18.1] - 2024-11-19](https://github.com/simpleiot/simpleiot/releases/tag/v0.18.1)

- fix bug where Shelly IO Enable Control option was not working (#739)

## [[0.18.0] - 2024-11-07](https://github.com/simpleiot/simpleiot/releases/tag/v0.18.0)

- change default login to `admin`/`admin` (used to be `admin@admin.com`, but
  there was is reason to have a bogus email address). (#730)
- file client/node
  - option to store binary files
  - display filename, file size, and stored size
  - create file client backend code that runs for file nodes
  - calculate and populate md5sum when file contents change
  - display md5sum in file node UI
- serial client/node
  - add serial file download -- can be used for MCU updates
  - fix issues with populate node ID for high rate data
- db client
  - fix crash if node ID is not populated correctly in data
- client (BREAKING API CHANGE)
  - renamed client.Group -> client.RunGroup. This is so we don't conflict with
    the client that manages group nodes
- fix issue with user login when moving users to different groups (#713)

## [[0.17.0] - 2024-08-05](https://github.com/simpleiot/simpleiot/releases/tag/v0.17.0)

- add rule/condition/action disable flag (#352)
- rule action: add point key field (#714)

## [[0.16.2] - 2024-06-03](https://github.com/simpleiot/simpleiot/releases/tag/v0.16.2)

- db client: Improve Influx history query functionality
  - If history query response fails, try responding again with ErrorMessage
  - TagFilters values can now be empty string or a slice of strings

## [[0.16.1] - 2024-05-22](https://github.com/simpleiot/simpleiot/releases/tag/v0.16.1)

- Modbus API: add an option to validate the input when a client writes to a
  register.
- Update client:
  - improve autodownload logic
  - check for updates when URI is changed
  - improve error handling and reporting
  - fix bug when reducing update list
- expand documentation on
  [creating a client](https://docs.simpleiot.org/docs/ref/client.html#creating-new-clients).

## [[0.16.0] - 2024-05-11](https://github.com/simpleiot/simpleiot/releases/tag/v0.16.0)

- add Update client -- currently supports system updates
  [docs](https://docs.simpleiot.org/docs/user/update.html).
- update elm-tooling
- api: Added `history.<nodeId>` NATS endpoint to send Influx history queries to
  an Influx DB client node.

## [[0.15.3] - 2024-03-19](https://github.com/simpleiot/simpleiot/releases/tag/v0.15.3)

- UI: add tag UI to metrics client UI

## [[0.15.0] - 2024-03-19](https://github.com/simpleiot/simpleiot/releases/tag/v0.15.0)

- NTP client: Do not set configuration if servers are not specified. This allows
  timesyncd to use the default configuration if no servers are specified.
- server: Args now accepts a `*FlagSet` to allow flags to be extended
- Influx client when writing points from a given node also adds additional tags
  based on the node that emitted the point. Previously, `nodeID` tag was added,
  but this has been renamed to `node.id`. Also added is `node.type` and
  `node.description` (populated with the value of a point of type
  "description").
- For each Influx DB client, the user can specify an array of tag point types
  (via point type "tagPointType"). These point types are also copied as tags for
  each point emitted by the node. For example, if node A has two points tag:city
  (i.e. Point.Type is "tag" and Point.Key is "city") and tag:state, then these
  point values are appended to every single point emitted by node A. In Influx,
  each point would have a tag `node.tag.city` and `node.tag.state` with their
  respective Point.Text values.
- BREAKING CHANGE: Influx DB tag `nodeID` is now `node.id`
- update frontend dependencies and fix various build issues
- UI: add tag UI most clients so that custom tags can be added to each node.

## [[0.14.10] - 2024-02-05](https://github.com/simpleiot/simpleiot/releases/tag/v0.14.10)

- store: Improved performance when loading many nodes and edges
- serial: Fixed bug: do not write points over closed serial port

## [[0.14.9] - 2024-01-18](https://github.com/simpleiot/simpleiot/releases/tag/v0.14.9)

- require custom UI assets to be rooted and not be public directory
- add `-UIAssetsDebug` cmdline flag. This will dump all the UI assets file and
  is useful in debugging to make sure your assets files are correct -- it can
  get a little tricky with embedding, etc.

## [[0.14.8] - 2024-01-16](https://github.com/simpleiot/simpleiot/releases/tag/v0.14.8)

- support passing in a custom UI (fs.FS or directory name) to the SIOT server.
  This allows you to replace the SIOT UI with a custom version.

## [[0.14.7] - 2024-01-09](https://github.com/simpleiot/simpleiot/releases/tag/v0.14.7)

- add modbus swap words for Int32/Uint32 writes

## [[0.14.6] - 2024-01-09](https://github.com/simpleiot/simpleiot/releases/tag/v0.14.6)

- verb -> adjective changes in several types. This is more consistent and
  accurate with how things are done in this industry (HTML, etc). This is a
  breaking change in that nodes with disable or control flag set will need to be
  reconfigured.
  - disable -> disabled
  - control -> controlled
- add modbus Float32ToRegsSwapWords()

## [[0.14.5] - 2024-01-02](https://github.com/simpleiot/simpleiot/releases/tag/v0.14.5)

- simpleiot-js: Fixed bugs and improved README
- Replace deprecated `io/ioutil` functions (#680)
- fixed frontend bug where only custom node types could be added

## [[0.14.4] - 2023-12-19](https://github.com/simpleiot/simpleiot/releases/tag/v0.14.4)

- UI: in node raw view, you can now edit/add/delete points (#676)
- UI: add custom node types

## [[0.14.3] - 2023-12-05](https://github.com/simpleiot/simpleiot/releases/tag/v0.14.3)

- UI: display unknown nodes as raw type and points
- UI: add raw view button to node expanded view. This allows us to view the raw
  points in any node which is useful for debugging and development. (see
  [docs](https://docs.simpleiot.org/docs/user/ui.html#raw-node-view) for more
  information)

## [[0.14.2] - 2023-11-29](https://github.com/simpleiot/simpleiot/releases/tag/v0.14.2)

- Signal generator client: replaced "Sync Parent" option with "Destination" to
  indicate the destination node and point type for generated points
- update gonetworkmanager to v2.1.0 and fix sync bugs
- network-manager client: Now supports better connection sync via connection
  `Managed` flag; fixed a few bugs; WiFiConfig sync now works

## [[0.14.1] - 2023-11-15](https://github.com/simpleiot/simpleiot/releases/tag/v0.14.1)

- update frontend assets (missed that in v0.14.0)

## [[0.14.0] - 2023-11-14](https://github.com/simpleiot/simpleiot/releases/tag/v0.14.0)

- update to nats-server to v2.10.4
- update to nats client package to v1.31.0
- development: `envsetup.sh` sources `local.sh` if it exists
- Go client API for export/import nodes to/from YAML
- `siot` CLI export and import commands
- simpleiot-js improvements
- Network Manager Client (WIP)
- NTP Client
- serial client: allow configuration of HR point destination
- serial client: add "Sync Parent" option
- Signal generator client: add support for square, triangle, and random walk
  patterns
- fix issue with batched points of the same type/key (#658)

## [[0.13.1] - 2023-10-03](https://github.com/simpleiot/simpleiot/releases/tag/v0.13.1)

- update client manager API to include list of parent node types
- fix issue with duplicating nodes where there were two copies of Description
  points
- display decode error count for high-rate serial packets
- display rate for high-rate serial packets

## [[0.13.0] - 2023-09-20](https://github.com/simpleiot/simpleiot/releases/tag/v0.13.0)

- implement `siot install` command (#527)
- update frontend poll rate from 3s to 4s
- fix `siot store` (was crashing due to Opened not being defined)

## [[0.12.7] - 2023-09-14](https://github.com/simpleiot/simpleiot/releases/tag/v0.12.7)

- serial client decoding improvements

## [[0.12.6] - 2023-09-13](https://github.com/simpleiot/simpleiot/releases/tag/v0.12.6)

- fix issue with email in user node UI (#609)

## [[0.12.5] - 2023-08-25](https://github.com/simpleiot/simpleiot/releases/tag/v0.12.5)

- add supported for Linux temp sensors (#607)

## [[0.12.4] - 2023-08-25](https://github.com/simpleiot/simpleiot/releases/tag/v0.12.4)

- Reworked and simplified decode and merge routines (#589). See
  [documentation](https://docs.simpleiot.org/docs/ref/data.html?#converting-nodes-to-other-data-structures)
- UI: fixed issue with with paste node rule condition/actions (#600)
- Can client: fixed various issues (#498)
- Rule client: fix issue with error reporting (#599)
- switch to forked mdns package to get rid of closing messages (#558)
- update nats.go package from v1.20.0 => v1.28.0
- update nats-server package from v2.9.6 => v2.9.21
- default NATS server to 127.0.0.1 instead of localhost

## [[0.12.3] - 2023-08-03](https://github.com/simpleiot/simpleiot/releases/tag/v0.12.3)

- switch to elm-tooling to enable building on Linux and MacOS ARM machines
- enable riscv builds in release

## [[0.12.2] - 2023-08-01](https://github.com/simpleiot/simpleiot/releases/tag/v0.12.2)

- fix login

## [[0.12.1] - 2023-07-27](https://github.com/simpleiot/simpleiot/releases/tag/v0.12.1)

- fix control of Shelly lights
- required that `Point:Key` field always be set (#580)
- improvements in point decode and merge with arrays (not finished)

## [[0.12.0] - 2023-07-21](https://github.com/simpleiot/simpleiot/releases/tag/v0.12.0)

- support Dates in Rule schedule conditions
- Rules are re-run if any rule configuration changes
- Display error conditions in Rule nodes
- hide schedule weekday entry when dates are active
- hide schedule date entry when weekdays are active
- support deleting (tombstone points) in NodeDecode and NodeMerge functions

## [[0.11.4] - 2023-06-08](https://github.com/simpleiot/simpleiot/releases/tag/v0.11.4)

- remove index field from Point data structure. See #565
- add support for Shelly Plus2PM
- change Shelly client to use Shelly API
  [Component model](https://shelly-api-docs.shelly.cloud/gen2/ComponentsAndServices/Introduction)

## [[0.11.3] - 2023-06-08](https://github.com/simpleiot/simpleiot/releases/tag/v0.11.3)

- serial client: add high rate rx count for debugging

## [[0.11.2] - 2023-06-05](https://github.com/simpleiot/simpleiot/releases/tag/v0.11.2)

- fix race condition in Client Manager client startup (#552). This fixes a crash
  when detecting Shelly devices.

## [[0.11.1] - 2023-05-30](https://github.com/simpleiot/simpleiot/releases/tag/v0.11.1)

- update point merge code to handle complex types
- more fixes for rule condition schedule processing (#547)
- fix issue with Shelly device discovery duplicating devices (#552)
- client manager: fix race condition with subscriptions and deleting client
  states

## [[0.11.0] - 2023-05-23](https://github.com/simpleiot/simpleiot/releases/tag/v0.11.0)

- fix rule condition schedule processing (#547)
- support high rate serial MCU data (#517)

## [[0.10.3] - 2023-05-16](https://github.com/simpleiot/simpleiot/releases/tag/v0.10.3)

- use mDNS responses to set shelly IO back online
- Client Manager: improve filtering of points -- see
  [Message echo](https://docs.simpleiot.org/docs/ref/client.html#message-echo)

## [[0.10.2] - 2023-05-15](https://github.com/simpleiot/simpleiot/releases/tag/v0.10.2)

- default to control being disabled for shelly devices and add UI to enable
  control (#544)

## [[0.10.1] - 2023-05-13](https://github.com/simpleiot/simpleiot/releases/tag/v0.10.1)

- fix issues with Shelly devices appearing offline when first discovered
- disable IPv6 in Shelly mDNS (does not seem to fix all issues on some machines)

## [[0.10.0] - 2023-04-28](https://github.com/simpleiot/simpleiot/releases/tag/v0.10.0)

- support for Shelly Home Automation devices (#189) (see
  [docs](https://docs.simpleiot.org/docs/user/shelly.html))
- switch Linting/CI to use golangci-lint and fix issues in codebase
- point encode/decode functions now support arrays and maps. Thanks @bminer!

## [[0.9.0] - 2023-02-28](https://github.com/simpleiot/simpleiot/releases/tag/v0.9.0)

- change default HTTP port from 8080 to 8118. This should reduce conflicts with
  other apps and require us to configure the HTTP port less often. (#495)
- BREAKING CHANGE: change protobuf point.value encoding from float to double
  (#291) This change introduces a protocol change so all instances in a system
  will need to be updated. If this is a problem, let us know and we can work out
  a migration.
- sqlite schema: change time storage from two fields (time_s, time_ns) to single
  time that contains NS since Unix epoch.
- documentation cleanup (#509)
- move particle code to client and add UI (#503). See
  [Particle client docs](https://docs.simpleiot.org/docs/user/particle.html).
- simplify serial MCU encoding (#517)
- improve serial MCU UI point display
- use Go crypto/rand API instead of /dev/random. May fix windows issues (#517)

## [[0.8.0] - 2023-01-23](https://github.com/simpleiot/simpleiot/releases/tag/v0.8.0)

- update elm-watch to v1.1.2
- add system, application, and process
  [metrics](https://docs.simpleiot.org/docs/user/metrics.html) (#256, #255)

## [[0.7.2] - 2023-01-02](https://github.com/simpleiot/simpleiot/releases/tag/v0.7.2)

- fix race condition with clients that have multi-level nodes (ex Rule client)
  #487

## [[0.7.1] - 2023-01-02](https://github.com/simpleiot/simpleiot/releases/tag/v0.7.1)

(DO NOT USE, THIS VERSION HAS PROBLEMS WITH FRONTEND ASSETS)

- upgrade frontend to elm-spa 6 (#197)
- apply elm-review rules to frontend code and integrate with CI (#222)
- changes so user does not have to log in if backend or browser is restarted
  (#474)
  - frontend: store JWT Auth token in browser storage
  - frontend: store JWT key in db
- use [air](github.com/cosmtrek/air) instead of entr for watching Go files
  during development. This allows `siot_watch` to work on MacOS, and should also
  be useful in a Windows dev setup.

See the [Hot reloading the Simple IoT UI](https://youtu.be/_Nrs2_l62_Q) for a
demo of these changes.

## [[0.7.0] - 2022-12-09](https://github.com/simpleiot/simpleiot/releases/tag/v0.7.0)

- add [CAN bus client](https://docs.simpleiot.org/docs/user/can.html)

## [[0.6.2] - 2022-12-07](https://github.com/simpleiot/simpleiot/releases/tag/v0.6.2)

- moved the node type from node point to edge field. This allows us to index
  this and make queries that search the node tree more efficient.
- support for processing clients in groups. Previously, client nodes had to be a
  child of the root device node.
- fix issue with `siot log` due to previous NATS API change

## [[0.6.1] - 2022-12-01](https://github.com/simpleiot/simpleiot/releases/tag/v0.6.1)

- fix bug in influx db client due to recent API changes
- fix bug in client manager where Stop() hangs if Start() has already exited
- don't allow deleting of root node
- allow configuring of root node ID, otherwise UUID is used
- sync:
  - add option to configure sync period (defaults to 20s).
  - if upstream node is deleted on the upstream, it is restored
  - don't include edge points of root node in hash calculation. This allows node
    to be moved around in the upstream instance and other changes.

## [[0.6.0] - 2022-11-15](https://github.com/simpleiot/simpleiot/releases/tag/v0.6.0)

- improve error handling in serial client cobs decoder
- rename upstream -> sync
  - re-implement node hash using CRC-32 and XOR hash
  - re-implement upstream sync using new hash mechanism
  - write tests for sync
- implement `siot log` subcommand -- this dumps SIOT messages
- implement `siot store` subcommand -- used to check and fix store
- simpleiot-js frontend library changes
  - re-worked to use updated NATS API
  - added `sendEdgePoints` API function
  - added unit tests, linting, etc.

Note, there have been some database changes. To update, do the following:

- `sqlite3 siot.sqlite`
  - `update set up="root" from edges where up="none";`
- start simpleiot
  - in another terminal, run: `siot store -fix`. Do this several times until the
    original siot process does not show any fixes.

## [[0.5.5] - 2022-10-31](https://github.com/simpleiot/simpleiot/releases/tag/v0.5.5)

- fix population of AppVersion in server
- serial client
  - add configuration of max message length
  - improve error handling and port resets

## [[0.5.4] - 2022-10-28](https://github.com/simpleiot/simpleiot/releases/tag/v0.5.4)

- clean up SIOT main to allow callers to have their own set of flags at the top
  level before calling SIOT server.

NOTE, to run siot with flags, you must do something like:

`siot serve -debugHttp`

The server flags are now part of the serve subcommand.

## [[0.5.3] - 2022-10-27](https://github.com/simpleiot/simpleiot/releases/tag/v0.5.3)

- add serial client debug level 9 to dump raw serial data before COBS processing

## [[0.5.2] - 2022-10-26](https://github.com/simpleiot/simpleiot/releases/tag/v0.5.2)

- **Breaking change**: the node hash type has changed from a string to an int,
  which requires deleting the database and starting over.
- switch from Genesis to go-embed for embedding frontend assets
- add embedded assets FS wrapper to allow embedding compressed assets and we
  decompress them on the fly if requested.
- add `elm.js.gz` to repo. This will allow us to run SIOT without building the
  frontend first. Should enable stuff like
  `go run github.com/simpleiot/simpleiot/cmd/siot` and allow using SIOT server
  as a Go package in other projects.
- add server API to add clients. This will allow customization of what clients
  are used in the system, as well as easily adding custom ones.
- fix version in SIOT app to be Git version (was always printing development)

You can now do things like:
`go run github.com/simpleiot/simpleiot/cmd/siot@latest`

## [[0.5.1] - 2022-10-12](https://github.com/simpleiot/simpleiot/releases/tag/v0.5.1)

- handle config changes in influx db client
- lifecycle improvements
  - fix race condition in http api shutdown
  - shutdown nats client after the rest of the apps
  - store: close nats subscriptions on shutdown
- Added Signal generator -- can be used to generate arbitrary signals
  (currently, high rate Sine waves only)
- add NATS subjects for high rate data (see [API](docs/ref/api.md))
- add [test app](cmd/point-size/main.go) to determine point protobuf sizes
- fix synchronization problem on shutdown -- need to wait for clients to close
  before closing store, otherwise we can experience delays on node fetch
  timeouts.
- fix issue when updating multiple points in one NATS message (only the first
  got written) (introduced in v0.5.0)
- Serial MCU Client:
  - added debug level for logging points and
    [updated what logging levels mean](https://docs.simpleiot.org/docs/user/mcu.html).
  - don't send rx/tx stats reset points to MCU
  - support high-rate MCU data (set message subject to `phr`).

## [[0.5.0] - 2022-09-20](https://github.com/simpleiot/simpleiot/releases/tag/v0.5.0)

**NOTE, this is a testing release where we are still in the middle of reworking
the store and various clients. Upstream functionality does not work in this
release. If you need upstream support, use a 0.4.x release.**

The big news for this release is switching the store to SQLite and moving rule
and db functionality out of the store and into clients.

- switch store to sqlite (#320)
- rebroadcast messages at each upstream node (#390)
- extensive work on client manager. It is now much easier to keep your local
  client config synchronized with ongoing point changes. Client manager also now
  supports client configurations with two levels of nodes, such as is used in
  rules where you have a rule node and child condition/action nodes.
- fix bug with fast changes in UI do not always stick (#414)
- move rules engine from store to siot client (#409)
- move influxdb code from store to client package (#410)
- replace all NatsRequest payloads with array of points (#406)

## [[0.4.5] - 2022-09-02](https://github.com/simpleiot/simpleiot/releases/tag/v0.4.5)

- set time on points received from serial MCU if not set
- display key in points if set

## [[0.4.4] - 2022-09-01](https://github.com/simpleiot/simpleiot/releases/tag/v0.4.4)

- switch serial CRC algorithm to CRC-16/KERMIT

## [[0.4.3] - 2022-08-29](https://github.com/simpleiot/simpleiot/releases/tag/v0.4.3)

- serial MCU: display rx/tx stats and any extra points in UI

## [[0.4.1] - 2022-08-24](https://github.com/simpleiot/simpleiot/releases/tag/v0.4.1)

- docs: add
  [Modbus user documentation](https://docs.simpleiot.org/docs/user/modbus.html).
- docs: add
  [Notification user documentation](https://docs.simpleiot.org/docs/user/notifications.html)
- data/merge.go: fix bug if text and value are both 0
- support Debug levels in serial MCU client: 0=no messages, 1=ascii log, 2=dump
  rx data
- serial MCU client: fix issue where reset error count was not working

## [[0.4.0] - 2022-07-29](https://github.com/simpleiot/simpleiot/releases/tag/v0.4.0)

- serial [MCU client](https://docs.simpleiot.org/docs/ref/serial.html) support
  (#380)
- add
  [origin field](https://docs.simpleiot.org/docs/ref/data.html#tracking-who-made-changes)
  to point type (#391).

## [[0.3.0] - 2022-07-22](https://github.com/simpleiot/simpleiot/releases/tag/v0.3.0)

This release has a few bug fixes and contains new client code that will make
creating new functionality easier.

- Fix invalid users causes panic in Go code #365
- implement data.Decode/Encode for converting nodes to user structs #384
- improve startup/shutdown lifecycle #389
- implemented struct <-> type
  [encode/decode](https://github.com/simpleiot/simpleiot/blob/master/data/encode_decode_test.go)
  functions.
- improved the lifecycle management of the application so we can cleanly shut it
  down. This allows us to test the application more easily (spin up version for
  test, shutdown, repeat).
- implemented a test.Server() function to create a test server to be used in
  tests.
- Go API Change: the `nats` package has been renamed to `client`.
- defined a new Client interface and a client Manager that watches for node
  changes and creates/updates clients and sends any points changes.

## [[0.2.0] - 2022-05-31](https://github.com/simpleiot/simpleiot/releases/tag/v0.2.0)

(implemented in PR #362)

- UI: fix sorting of Rule child nodes
- highlight rule actions when active #266
- better linking of nodes for rules #251
- display clipboard contents at top of screen
- update elm/virtual-dom to 1.0.3 (helps
  [prevent xss attacks](https://jfmengels.net/virtual-dom-security-patch/))

This release improves the process of linking nodes to rule actions or
conditions. In the past, the system clipboard was used and you had to paste the
system clipboard contents into the Node ID field of rule conditions and actions.
Now, when you a copy a node, the SIOT frontend has its own clipboard and a past
button is displayed below the Node ID fields for easy pasting the node ID.
Additionally, the node description is displayed below the Node ID field so you
can easily tell which node the ID is referring to.

A [video is available](https://youtu.be/tqbLZ9CSzRU) which illustrates how node
IDs can now be copied and pasted.
[docs](https://docs.simpleiot.org/docs/user/rules.html) are also updated.

## [[0.1.0] - 2022-05-13](https://github.com/simpleiot/simpleiot/releases/tag/v0.1.0)

- docs: add list of supported devices to install
- docs: add upstream documentation
- add support for 1-wire buses, and DS18B20 temp sensor #230 #361

## [[0.0.45] - 2022-04-30](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.45)

- add DeleteNode, MoveNode, and MirrorNode to
  [nats package](https://pkg.go.dev/github.com/simpleiot/simpleiot@v0.0.44/nats).
  #344, #347
- store and display App Version in root node (see screenshot below). This value
  is extracted by the SIOT build using the `git describe` command. See
  `envsetup.sh`. #192, #349
- store and display OS version in root node (see screenshot below). On Linux,
  this value is extracted from the `VERSION` field in `/etc/os-release`. The
  field can be
  [changed](https://docs.simpleiot.org/docs/user/configuration.html) using the
  OS_VERSION_FIELD environment variable. #324, #353
- update go.bug.st/serial from v1.1.3 -> v1.3.5
- sort nodes in UI a little nicer, conditions before actions, move more often
  used nodes to the top, etc. #355, #337
- add NATS user auth API and change HTTP auth to use that. #326, #356
- fix bug where deleted nodes where still considered for user auth
- add SIOT JS library using NATS over WebSockets (#357)

![os/app version](https://user-images.githubusercontent.com/402813/163829093-14c0d644-243d-49e0-9c83-acc3c642c9ab.png)

## [[0.0.44] - 2022-04-05](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.44)

- UI: fix bug where copy node crashes UI if no on secure URL or localhost (#341)
- support clone/duplicate node (as well as mirror) operation (#312). Now when
  you copy and paste a node, you will be presented with a list of options as
  shown below. The new duplicate option allows you to easily replicate complex
  setups (for instance a bunch of modbus points) from an existing site to a new
  site.

![copy options](https://user-images.githubusercontent.com/402813/153455487-66bc2699-1026-40de-9ca6-4f30f91aeff9.png)

See
[documentation](https://docs.simpleiot.org/docs/user/ui.html#deleting-moving-mirroring-and-duplicating-nodes)
or a [demo video](https://youtu.be/ZII9pzx9akY) for more information.

## [[0.0.43] - 2022-03-11](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.43)

- improvement in UI to fix collapsing nodes #259
- implemented functionality to duplicate nodes and refactored
  copy/move/mirror/duplicate UI (#312)
- update nats-server from v2.6.6 -> v2.7.4 (and associated dependencies)

## [[0.0.42] - 2022-02-22](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.42)

- move HTTP API to get nodes for user to use NATS instead of direct call into
  database (#327)
- **BREAKING API CHANGE**: the Nats `inst.<id>` subject now returns an array of
  `data.NodeEdge` structs instead of a single node. Both instances of an
  upstream connection must be upgraded.
- don't send deleted nodes to frontend -- may fix #259
- default to nats/websocket being enabled on port 9222

## [[0.0.41] - 2022-01-05](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.41)

- with v0.0.40, if upstream URI was specified as ws://myserver.com without the
  port being specified, the NATS Go client assumed the port was 4222. If this
  port is not specified for ws or wss protocols, SIOT now sets the port to :80
  or :443. This makes the behavior more predictable, as these kinds of problems
  are very hard to debug. #315
- if upstream config changes, restart upstream connection. #258

## [[0.0.40] - 2022-01-03](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.40)

- support for NATS over WS connections to upstream. This is handy for cases
  where the edge network may block outgoing connections on the port NATS is
  using. HTTP(s) almost always works. In the upstream config, simply change the
  URL to something like: `ws://my.service.com`.

## [[0.0.39] - 2021-12-17](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.39)

- fix issue where app exits if upstream auth is incorrect (#298)
- fix issues with orphaned device nodes in upstreams. We now make sure devices
  in upstream have upstream edges or are not deleted if the device is still
  receiving points. (#299)
- only report nats stats every 1m. This makes upstream work better as these
  currently are run in sync.

## [[0.0.38] - 2021-11-17](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.38)

- fix population of version when building with envsetup.sh
- changes to point data structure to make it more flexible
  ([ADR-1](https://github.com/simpleiot/simpleiot/pull/279))

## [[0.0.37] - 2021-10-26](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.37)

- fix issue with setup where you sometimes get error: elm: Text file busy
- cleanup simpleiot.Start() so it actually returns

## [[0.0.36] - 2021-10-26](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.36)

- rename `db` package to `store`
- factor out siot server startup code into simpleiot package
- change `siot_run` in `envsetup.sh` to `go build` instead of `go run`

## [[0.0.35] - 2021-10-04](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.35)

- add placeholders for some UI forms
- add disable for Modbus and Modbus client nodes (#250)
- clean up locking issues and simplify DB code

## [[0.0.34] - 2021-09-08](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.34)

- handle audio playback where file bitrate does not match default of audio
  device (#240)
- support rule actions that trigger when rule goes inactive (instead of active).
  This allows a rule to do something with the run goes active as well as
  inactive and in some cases saves us from writing two rules (#241).
- re-enable indexes on edge up/down fields (#219)
- add point min/max to NATS packets
- add NATS api metrics (as points to root device node) (#244)
- don't color root node grey for now
- update influxdb client to 2.5.0
- switch to async influx DB API (batches data, retries, etc)
- implement caching of nodes and edges to speed up read access
- add point processing cycle time and nats client pending messages metrics
- modbus loglevel 1 only prints errors, 2 now prints transactions
- web UI auth expires in 24hr instead of 30m -- still not ideal, but one step at
  a time (#249)
- update front tar package to fix security warnings

## [[0.0.33] - 2021-08-12](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.33)

- fix frontend build issue with last two releases
- add rule audio playback action functionality for Linux (requires alsa-utils)
- fix various bugs with rule schedule condition functionality
- all using rule active in rule conditions (allows chaining rules)
- improve rule condition processing to process all conditions/points rather than
  just first match
- implement schedule conditions for rules
- switch from github.com/dgrijalva/jwt-go to github.com/golang-jwt/jwt/v4
- update frontend dependencies to satisfy github security checks

## [[0.0.32] - 2021-08-11](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.32)

- DO NOT USE, FRONTEND BUILD ISSUE

## [[0.0.31] - 2021-08-10](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.31)

- DO NOT USE, FRONTEND BUILD ISSUE

## [[0.0.30] - 2021-07-22](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.30)

- fix using SIOT_AUTH_TOKEN for -logNats command line option
- upgrade to NATS 2.2.2. Increases SIOT binary by about 2MB (uncompressed), 1MB
  (compressed)
- disable badger for now -- can be re-enabled in db/genji.go. Bolt seems to work
  better for the current SIOT use cases and Badger just adds bloat to the
  binary.
- implement upstream synronization support
  [#109](https://github.com/simpleiot/simpleiot/issues/109)
- update to Genji v0.13.0

## [[0.0.29] - 2021-04-22](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.29)

- fix sending notifications to a single user through UI

## [[0.0.28] - 2021-04-22](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.28)

- modbus: don't require poll period to be set for modbus server
- modbus: fix issue with reg values being sent every poll period, even if not
  changing
- modbus: add timestamp to points being sent out
- support storing Point data in Influxdb 2.0

## [[0.0.27] - 2021-04-15](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.27)

- slow down manual scanning to reduce background CPU usage

## [[0.0.26] - 2021-04-15](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.26)

- make description in nats logger and notification messages smarter
- allow modbus busses to be added to groups as well as devices
- UI:
  - don't show node + operation for nodes that can't have child-nodes
  - force email entry to always be lowercase

## [[0.0.25] - 2021-04-12](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.25)

- track user parent when messaging. This eliminates duplicate messages if a user
  is part of different groups with different messaging services -- we only want
  to message the group the user is a part of.

## [[0.0.24] - 2021-04-12](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.24)

- UI
  - display copy/move node messages for 2-3s when clicking copy/move node button
  - support multiple top level nodes -- for instance a user that is a member of
    multiple groups but not the root node
  - automatically expand node children when moving/copying a node
- Implement rule notifications

## [[0.0.23] - 2021-04-10](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.23)

- Modbus: TCP listen on all interfaces instead of just localhost
- UI
  - add dot for nodes that don't have children
  - don't sort nodes while editing, only on fetch
  - sort nodes by group, then desc, then firstname, then lastname
  - move/copy node can use node ID or description
  - add node icons to add node descriptions
  - replace edit/collapse with dot and color exp nodes
- support copying nodes
- remove remnants of Sample types (we now use Point)
- create notification and message data types and NATS/Db support
- implement node messaging (notifies all node and upstream users)
- BUILD: simplify protobuf generation
- implement Twilio SMS messaging

## [[0.0.22] - 2021-03-17](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.22)

- UI: change node min/max button to edit/close
- Modbus: suppress TCP conn/disc messages at debug level 0
- siot: add cmdline option (-logNats) to trace all node points. This can be run
  in parallel to the siot application to trace points flowing through the system
- genji db: update to v0.11.0 release
- rules: can now write rules that set nodes based on other nodes

## [[0.0.21] - 2021-03-17](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.21)

- modbus: fix server issue with requests not free resources

## [[0.0.20] - 2021-03-17](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.20)

- UI: add Form.onEnter utility function for adding enter handling
- UI: enter can now be used to enter sign-in form
- support for Modbus TCP, both client and server

## [[0.0.19] - 2021-02-27](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.19)

- revert Genji update as there are problems saving nodes

## [[0.0.18] - 2021-02-26](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.18)

- update go.bug.st/serial to support RiscV
- update Genji dependencies

## [[0.0.17] - 2021-02-12](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.17)

- modbus
  - fix bug in setting modbus baud rate
  - include ID in modbus logging messages
  - support for read-only coils and holding regs
- UI
  - add nodeCheckboxInput widget
  - round numbers in places
  - color digital values blue when ON
  - sort nodes by description
- fix windows build

## [[0.0.16] - 2021-02-08](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.16)

- UI
  - expand child nodes and add default description when adding a new node
- modbus improvements
  - send all writes to DB over NATS -- this allows system to be more responsive,
    as well as simplifies code
  - lots of cleanup and error handling

## [[0.0.15] - 2020-12-09](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.15)

- Implementation of tree based UI -- see demo: https://youtu.be/0ktVCPU74mw

## [[0.0.14] - 2020-11-20](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.14)

- fix 32bit binary build

## [[0.0.13] - 2020-11-03](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.13)

- edge:
  - fixed issue with backoff algorithm not adhering to max
- backend:
  - switched data structure name from device -> node -- see
  - [this issue](https://github.com/simpleiot/simpleiot/issues/91) for
    discussion
  - add page to message (currently SMS only) all users
  - UI simplification and cleanup
  - sort users on users page
  - port frontend to elm-spa.dev v5 (this really cleans up the frontend code and
    makes it more idomatic Elm)
  - changing backing store from bolthold to genji (this gives us the flexibility
    to use memory, bbolt, or badger backing stores as well as robust indexing)
  - fix bug with not support Point::Text field in Nats/Protobuf
  - fix up examples for sending device version info to portal
- frontend:
  - only show version information if available
  - don't display special points (description, version, etc) in general node
    points.
  - add -importDb command line option

Note, the database format has changed. To migrate, dump the database with the
old version of SIOT and them import with the new version.

## [[0.0.12] - 2020-11-03](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.12)

- backend:
  - switched data structure name from device -> node -- see
    [this issue](https://github.com/simpleiot/simpleiot/issues/91) for
    discussion
  - add page to message (currently SMS only) all users
  - UI simplification and cleanup
  - sort users on users page
  - port frontend to elm-spa.dev v5 (this really cleans up the frontend code and
    makes it more idomatic Elm)
  - changing backing store from bolthold to genji (this gives us the flexibility
    to use memory, bbolt, or badger backing stores as well as robust indexing)
  - fix bug with not support Point::Text field in Nats/Protobuf
  - fix up examples for sending device version info to portal
- frontend:
  - only show version information if available
  - don't display special points (description, version, etc) in general node
    points.
  - add -importDb command line option

Note, the database format has changed. To migrate, dump the database with the
old version of SIOT and them import with the new version.

## [[0.0.11] - 2020-09-09](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.11)

### Changed

- switched data storage to
  [points](https://github.com/simpleiot/simpleiot/blob/master/docs/development.md#flexible-data-structures)
  vs sensor data and config
- add token auth for device HTTP communication
- documentation improvements

## [[0.0.10] - 2020-08-20](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.10)

### Changed

- documentation improvements
- specify TLS certs using variables instead of embedding
- code cleanup around NATS integration
- NATS don't force TLS 1.2 in client
- remove siotutil functionality and fold into siot exe

## [[0.0.9] - 2020-08-15](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.9)

### Added

- NATS integration for device communication

### Changed

- documentation improvements
  - moved API documentation to simple Markdown
  - better organization
  - add list of guiding principles to the [README](./README.md)

## [[0.0.8] - 2020-08-11](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.8)

### Added

- moved influxDb operations to db package so they are common for all samples
- added env variable to specify Influx database SIOT_INFLUX_DB
- added device ID tag to sample data stored in influx
- add rules engine
- add SMS notifications using Twilio

### Changed

- clean up documentation organization

## [[0.0.7] - 2020-07-04](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.7)

### Added

- display device last update time
- display time since last update

## [[0.0.6] - 2020-06-26](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.6)

### Added

- add modbus API to change debug level at runtime
- add cloud/cloud off icon to indicate connection status of devices
- grey out devices that are not currently connected
- added background process to determine if devices are offline

### Fixed

- workaround for issue where group key in database does not match ID in struct

## [[0.0.5] - 2020-06-16](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.5)

### Fixed

- fixed critical bug where new devices were not showing up in UI

### Added

- add support in modbus pkg for decoding 32-bit int and floating point values
- started general command line modbus utility (cmd/modbus) to interactively read
  modbus devices
