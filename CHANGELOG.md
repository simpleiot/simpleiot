# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

For more details or to discuss releases, please visit the
[Simple IoT community forum](https://community.tmpdir.org/c/simple-iot/5)

## [Unreleased]

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
- fix bug in client manager where Stop() hangs if Start() has already exitted
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
- Addeed Signal generator -- can be used to generate arbitrary signals
  (currently, high rate Sine waves only)
- add NATS subjects for high rate data (see [API](docs/ref/api.md))
- add [test app](cmd/point-size/main.go) to determine point protobuf sizes
- fix syncronization problem on shutdown -- need to wait for clients to close
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
[documenation](https://docs.simpleiot.org/docs/user/ui.html#deleting-moving-mirroring-and-duplicating-nodes)
or a [demo video](https://youtu.be/ZII9pzx9akY) for more information.

## [[0.0.43] - 2022-03-11](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.43)

- improvement in UI to fix collapsing nodes #259
- implemented functionality to duplicate nodes and refactored
  copy/move/mirror/duplicate UI (#312)
- update nats-server from v2.6.6 -> v2.7.4 (and associated dependencies)

## [[0.0.42] - 2022-02-22](https://github.com/simpleiot/simpleiot/releases/tag/v0.0.42)

- move HTTP API to get nodes for user to use NATS instead of direct call into
  database (#327)
- **BREAKING API CHANGE**: the Nats `node.<id>` subject now returns an array of
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
- update frontend dependencies to satisify github security checks

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
  - don't sort nodes while editting, only on fetch
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
  - fixed issue with backoff algorith not adhearing to max
- backend:
  - switched data structure name from device -> node -- see
  - this issue for dicussion
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
    [this issue](https://github.com/simpleiot/simpleiot/issues/91) for dicussion
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
  - add list of guiding principles to the [README](README.md)

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
