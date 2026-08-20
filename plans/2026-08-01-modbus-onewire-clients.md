# Plan: Move Modbus and 1-Wire to Proper Clients

**Branch:** `feat/modbus-onewire-clients` **Branched from:** `54169492`

## Context

Modbus and 1-Wire are the last two subsystems still living in the `node/`
package, which predates the client architecture. `node.Manager` is started
directly from `server/server.go` and runs a 20-second loop that polls the store
for `modbus` and `oneWire` nodes, constructing and tearing down bus objects by
hand:

```
node/node.go             191   Manager: version reporting + 20s poll loop
node/modbus-manager.go    63   scans for modbus nodes, creates/destroys busses
node/modbus.go          1050   bus goroutine, NATS subscription, point dispatch
node/modbus-node.go      134   decodes a modbus node into a struct, by hand
node/modbus-io-node.go   107   decodes a modbusIo node into a struct, by hand
node/modbus-io.go         53   per-IO NATS subscription
node/onewire-manager.go  114   scans for oneWire nodes, plus bus autodetection
node/onewire.go          301   bus goroutine, NATS subscription, point dispatch
node/onewire-node.go      32   decodes a oneWire node, by hand
node/onewire-io-node.go   53   decodes a oneWireIO node, by hand
node/onewire-io.go       132   per-IO NATS subscription, sysfs read
```

Everything `client.Manager` already does is reimplemented here, less well:

- **Node discovery is a poll, not a subscription.** New busses are noticed up to
  20 seconds late, and new IOs up to 10 seconds late. The comment in
  `node/node.go` says as much:
  `TODO: this will not scale and needs to be made event driven`.
- **Node decoding is hand-written.** `NewModbusNode` and friends read every
  point by type and convert it, roughly 320 lines that `data.Decode` and the
  `point:` struct tags do generically. The same code then hand-writes the
  reverse mapping in a 150-line `switch` in `Modbus.Run` when points arrive.
- **Every bus and every IO opens its own NATS subscription** on `p.<nodeID>` and
  funnels points through a shared channel with a hand-rolled closure to avoid a
  data race. `client.Manager` subscribes once per client to `up.<nodeID>.>` and
  delivers points for the node and all its descendants.
- **The subscription subject is the old one.** `p.<nodeID>` is the point
  subject, not the `up.` propagation subject the rest of the system uses, so
  these nodes do not participate in the same point flow as every other client.
- **Discovery only searches below the root node.**
  `GetNodes(nc, rootNodeID, "all", data.NodeTypeModbus, ...)` finds only direct
  children of root, so a modbus node placed inside a group is never started.
  `client.Manager.scanHelper` recurses through groups and declared parent types.
- **Nothing is testable.** There are no tests for either subsystem. The client
  framework has `server.TestServer` and `client.NodeWatcher`, which the serial,
  GPS, and rule clients use for real end-to-end tests.

There is also a latent bug worth fixing on the way through: `oneWire.CheckIOs`
(`node/onewire.go`) queries for `data.NodeTypeModbusIO` instead of
`data.NodeTypeOneWireIO`. It is dead code, superseded by the lowercase
`checkIOs` immediately below it, but it shows how easily these two copies of the
same logic drift.

Neither node type nor point type changes in this work, so the Elm frontend
(`NodeModbus.elm`, `NodeModbusIO.elm`, `NodeOneWire.elm`, `NodeOneWireIO.elm`)
needs no changes. Configurations in existing databases keep working.

## Design Decisions

**IOs are children of the bus client, not clients of their own.** A modbus IO
cannot act alone: it shares the serial port or TCP socket, the register map, and
the poll cycle with its bus. Making each IO a separate client would require
those to be shared across client boundaries. Instead the bus client declares
`IOs []ModbusIo \`child:"modbusIo"\``, exactly as `CanBus`declares`Databases
[]File`and`Shelly`declares`IOs
[]ShellyIo`. `client.Manager`subscribes to`up.<busID>.>`, which carries points for the bus node and all its descendants, and `data.MergePoints(pts.ID,
...)`routes a point to the right element by matching the`node:"id"`field anywhere in the struct tree. One`MergePoints`
call handles both bus and IO points.

The same reasoning applies to 1-Wire: the bus owns the poll timer, the device
detection, and the bus-level error count.

**Adding or deleting an IO restarts the bus client.** This is inherent to the
client framework: `client.Manager` watches edge points, and a `tombstone` or
`nodeType` point below the client node stops the client so it can be
reconstructed with fresh config. For modbus this means adding an IO in the UI
reopens the port. That is acceptable — it happens when a person edits
configuration, not during steady-state polling — and it removes the 10-second
`CheckIOs` poll along with the `ModbusIONode.Changed` comparison helper (whose
own comment reads `FIXME, we should not need this once we get NATS wired`).

**Points the client sends to its IO children carry `Origin` set to the bus node
ID.** `client.Manager` filters a point back out when `p.Origin == cs.node.ID`,
which is how a client avoids reacting to its own writes. For points the client
sends to its own node the existing `Origin == ""` filter already applies, so
those stay as they are. Without this, every value the modbus client publishes to
an IO would come straight back in as a new point.

**The Go type for a modbus IO is `ModbusIo`, not `ModbusIO`.** Node type strings
are derived from the Go type name by `data.ToCamelCase`, and the existing node
type in `data/schema.go` and in the frontend is `modbusIo`. `ToCamelCase`
("ModbusIO") produces `modbusIO`, which would not match. `ToCamelCase`
("ModbusIo") produces `modbusIo`, which does. The 1-Wire types work out the
other way: the existing node types are `oneWire` and `oneWireIO`, and
`ToCamelCase("OneWireIO")` produces `oneWireIO`, so `OneWire` and `OneWireIO`
are the right names. This only matters directly if a type is ever passed to
`client.NewManager`, since the `child:` tag names the type explicitly, but
keeping the derivation honest avoids a confusing trap later.

**The modbus node has both a node ID and an `id` point.** The `id` point is the
modbus server address (unit ID), unrelated to the SIOT node ID. `data.Decode`
reads the `node:` and `point:` tags independently, so two fields can both be
tagged `id` as long as the Go field names differ. The config uses
`ID string \`node:"id"\``and`ServerID int
\`point:"id"\``, and the IO config uses `ID string \`node:"id"\``and`Address int
\`point:"address"\``alongside`ServerID int \`point:"id"\``.

**Runtime state stays out of the config struct.** The config struct mirrors the
node's points and is rewritten by `MergePoints` on every update. Values that
exist only while the client runs — the `lastSent` timestamp that forces a value
point at least every ten minutes, the per-IO subscription state, the register
map — live in a separate map on the client keyed by IO node ID, rebuilt when
`Run` starts.

**1-Wire buses are no longer auto-created at the root node; devices on a bus
still are.** Today `oneWireManager.update` globs
`/sys/bus/w1/devices/ w1_bus_master*` and creates a `oneWire` node under root
for each controller it finds. Nothing in the client framework runs when no node
exists, so preserving this would mean keeping a hardware-scanning goroutine
alive in `server/` purely for 1-Wire, which is the thing this plan sets out to
remove. Every other bus in Simple IoT — modbus, CAN, serial — is added by the
person configuring the system, who is also the only one who knows which group or
device node it belongs under. So a person adds a 1-Wire node and sets its bus
index, and the client detects the sensors on that bus and creates `oneWireIO`
children, which is the part that actually saves work. `docs/user/onewire.md` is
updated to describe this.

**1-Wire device detection is scoped to the bus.** The current `detect()` globs
`/sys/bus/w1/devices/28-*`, which lists every DS18B20 on every controller, so
with two controllers each bus node would claim all the sensors. Detection moves
to `/sys/bus/w1/devices/w1_bus_master<index>/28-*`, and the read path keeps
using the flat `/sys/bus/w1/devices/<id>/temperature` path, which is stable
regardless of controller.

**Version reporting moves to `server/`.** Beyond the two bus managers,
`node.Manager` writes the app version and OS version to the root node at
startup. That is about forty lines with no relationship to modbus or 1-Wire. It
becomes a small function in `server/server.go` that runs once after the store
starts, rather than a long-lived actor in the run group.

## Phase 1 — Modbus Client

Add `client/modbus.go` and `client/modbus-io.go`; remove `node/modbus*.go`.

### Config types

```go
// Modbus describes a modbus bus. The name matches the frontend node type
// "modbus" so that when a modbus node is created the client manager knows to
// start a Modbus client.
type Modbus struct {
	ID                 string     `node:"id"`
	Parent             string     `node:"parent"`
	Description        string     `point:"description"`
	ClientServer       string     `point:"clientServer"`
	Protocol           string     `point:"protocol"`
	URI                string     `point:"uri"`
	Port               string     `point:"port"`
	Baud               string     `point:"baud"`
	ServerID           int        `point:"id"`
	PollPeriod         int        `point:"pollPeriod"`
	Timeout            int        `point:"timeout"`
	Debug              int        `point:"debug"`
	Disabled           bool       `point:"disabled"`
	ErrorCount         int        `point:"errorCount"`
	ErrorCountCRC      int        `point:"errorCountCRC"`
	ErrorCountEOF      int        `point:"errorCountEOF"`
	ErrorCountReset    bool       `point:"errorCountReset"`
	ErrorCountCRCReset bool       `point:"errorCountCRCReset"`
	ErrorCountEOFReset bool       `point:"errorCountEOFReset"`
	IOs                []ModbusIo `child:"modbusIo"`
}

// ModbusIo describes a modbus IO, a child of a Modbus node
type ModbusIo struct {
	ID                 string  `node:"id"`
	Parent             string  `node:"parent"`
	Description        string  `point:"description"`
	ServerID           int     `point:"id"`
	Address            int     `point:"address"`
	ModbusIOType       string  `point:"modbusIoType"`
	DataFormat         string  `point:"dataFormat"`
	ReadOnly           bool    `point:"readOnly"`
	Scale              float64 `point:"scale"`
	Offset             float64 `point:"offset"`
	Value              float64 `point:"value"`
	ValueSet           float64 `point:"valueSet"`
	Disabled           bool    `point:"disabled"`
	ErrorCount         int     `point:"errorCount"`
	ErrorCountCRC      int     `point:"errorCountCRC"`
	ErrorCountEOF      int     `point:"errorCountEOF"`
	ErrorCountReset    bool    `point:"errorCountReset"`
	ErrorCountCRCReset bool    `point:"errorCountCRCReset"`
	ErrorCountEOFReset bool    `point:"errorCountEOFReset"`
}
```

`Baud` is a string to match how the frontend sends it and how `SerialDev`
already stores it.

### Validation

`NewModbusNode` currently returns an error when a required point is missing,
which the old manager logged and then retried on the next 20-second scan. A
client cannot refuse to start that way. Instead `Run` validates the config and,
when it is incomplete, logs once and idles until a point arrives that completes
it — the same shape `SerialDevClient` uses for an unset port and baud. The
timeout correction already in `NewModbusNodeWithCorrections` (a non-positive
timeout becomes 100 ms and the corrected value is published back) carries over
into that validation step, so the `ModbusNodeResult`/`TimeoutCorrected` pair is
no longer needed.

### Client structure

`ModbusClient` keeps the existing bus logic largely intact — `SetupPort`,
`ClosePort`, `ClientIO`, `ServerIO`, `ReadBusReg`, `ReadBusBit`,
`WriteBusHoldingReg`, `ReadReg`, `WriteReg`, `InitRegs`, `LogError`, and
`regCount` move over close to as-is, retyped against the new config structs. The
work is in the surrounding plumbing:

- `Run` keeps the `select` over the poll timer, the port-check timer, and the
  register-change channel, and replaces `chPoint`/`chDone` with the standard
  `newPoints`/`newEdgePoints`/`stop` channels.
- The 150-line point `switch` collapses into
  `data.MergePoints(pts.ID, pts.Points, &c.config)` followed by a much smaller
  switch that reacts only to the points with side effects: `clientServer`, `id`,
  `debug`, `port`, `baud`, `uri`, and `timeout` re-open the port; `pollPeriod`
  resets the scan timer; the three `errorCount*Reset` points publish zeros; a
  `value` point on an IO drives `ServerIO`; a `valueSet` point on an IO drives
  `ClientIO`.
- `CheckIOs` and the 10-second IO scan go away. The port-check timer stays,
  since it detects a serial port that was unplugged and plugged back in.
- Points sent to IO nodes set `Origin` to `c.config.ID`.
- Runtime state (`lastSent` per IO, `ioErrorCount`, `regs`, `client`, `server`,
  `serialPort`) lives on `ModbusClient`.

Register `NewModbusClient` in `client.DefaultClients`.

### Tests

The modbus client is the first subsystem in this move that can be tested end to
end without hardware: a modbus **server** bus and a modbus **client** bus, both
running as `ModbusClient` instances inside one `server.TestServer`, talking to
each other over a TCP loopback socket. Every register type and data format goes
over a real wire, through the real transport encode/decode, and back out as a
point on a SIOT node. `modbus/rtu-end-to-end_test.go` already does this at the
protocol layer with an in-memory serial pair; these tests do it a layer up,
through nodes and points.

`client/modbus_test.go` follows the shape of `client/serial_test.go`.

#### Topology

```
root
├── modbus (server)  protocol=TCP  port=<free port>  id=1
│   ├── modbusIo  discreteInput  bit 0
│   ├── modbusIo  coil           bit 16
│   ├── modbusIo  inputRegister  uint16   address 100
│   └── ...                               one IO per case below
└── modbus (client)  protocol=TCP  uri=127.0.0.1:<free port>  id=1  pollPeriod=50
    ├── modbusIo  discreteInput  bit 0
    ├── modbusIo  coil           bit 16
    ├── modbusIo  inputRegister  uint16   address 100
    └── ...                               matching IO for each server IO
```

Both busses are children of root, so `client.Manager` starts one `ModbusClient`
for each. The `id` point (the modbus unit ID) matches on the bus nodes and on
every IO. The client bus needs a short `pollPeriod` (50 ms) so tests converge
quickly; the server bus stops its scan timer and reacts to register-change
callbacks and incoming points, so its poll period does not matter.

Two address-space constraints shape the layout, both of which are easy to get
wrong:

- **The TCP server binds a real port.** Grab one by listening on `127.0.0.1:0`,
  reading the assigned port, and closing the listener before creating the nodes.
  A fixed high port would collide with a developer's machine or a parallel
  `go test` run.
- **Coils and registers share one address space.** `Regs.AddCoil(num)` maps a
  coil to register `num/16` (`modbus/reg.go`), so bit addresses 0–31 occupy
  registers 0 and 1. The test keeps bits below 32 and words at 100 and above so
  a coil write cannot silently corrupt a holding register.

#### Register type matrix

Data flows in a different direction for each register type, which follows from
`ClientIO` and `ServerIO` — a discrete input and an input register are written
by the server and read by the client, while a coil and a holding register are
written by the client and observed by the server. Each case gets its own IO pair
at its own address, and each is asserted in both the direction it supports and,
where it applies, on the read-back path:

| IO type           | Direction       | Stimulus                           | Assertion                                                     |
| ----------------- | --------------- | ---------------------------------- | ------------------------------------------------------------- |
| `discreteInput`   | server → client | `value` point on server IO node    | client IO node `value` matches                                |
| `inputRegister`   | server → client | `value` point on server IO node    | client IO node `value` matches, scaled                        |
| `coil`            | client → server | `valueSet` point on client IO node | server IO node `value` matches, and client IO `value` follows |
| `holdingRegister` | client → server | `valueSet` point on client IO node | server IO node `value` matches, and client IO `value` follows |

`InitRegs` seeds each server register from the node's stored `value` when the
port opens, so a fresh coil or holding register starts at the server's value and
the client read-back sees it before any write. Asserting that initial read-back
first, then the write, covers both halves of the path.

#### Data format matrix

Input and holding registers are exercised across every supported format, since
each one takes a different branch through `ReadBusReg`, `ReadReg`, and
`WriteReg`, and the 32-bit formats span two registers:

| Format    | Registers | Test value | Notes                                  |
| --------- | --------- | ---------- | -------------------------------------- |
| `uint16`  | 1         | 65535      | top of range                           |
| `int16`   | 1         | -1234      | sign preserved through the uint16 wire |
| `uint32`  | 2         | 4000000000 | above int32 range                      |
| `int32`   | 2         | -2000000   | sign across the register pair          |
| `float32` | 2         | 3.25       | exact in binary, so equality is safe   |

Addresses advance by the register count of the case before it, so a two-register
format cannot overlap its neighbor. One case also sets `scale` and `offset` to
non-default values (for example scale 0.1, offset -40, a common temperature
transmitter scaling) and asserts the scaled value on the client and the unscaled
value in the server's register, which is the one place the two sides
deliberately disagree.

A table-driven test builds one IO pair per row and asserts them all against a
single pair of busses, so the whole matrix costs one server startup and one port
open rather than one per case.

#### Additional cases

1. **Adding an IO to a running bus.** Create a bus with one IO, wait for a
   value, then add a second IO and assert it starts polling. This exercises the
   client-restart-on-child-change path described above, and is the one case that
   must not be folded into the matrix test.
2. **Error counts.** Point a client bus at a port with nothing listening and
   assert `errorCount` rises on both the bus node and the IO node, then start a
   server on that port and assert the counts stop rising and values arrive.
   Separately, assert that an `errorCountReset` point zeros the count.
3. **Read-only.** Set `readOnly` on a client holding-register IO, send a
   `valueSet`, and assert the server register does not change.
4. **Disabled.** Set `disabled` on an IO and assert it is not polled; set
   `disabled` on the bus and assert the port closes.

#### Helpers

Two small helpers keep the assertions readable and are worth writing first:

- `waitFor(t, timeout, func() bool)` — polls a condition at 10 ms intervals and
  fails with a message on timeout, replacing the hand-rolled deadline loops in
  `client/serial_test.go`.
- `newModbusTestBusses(t)` — starts `server.TestServer`, allocates a port,
  creates the server and client bus nodes with a supplied set of IO pairs, sets
  up a `client.NodeWatcher[client.ModbusIo]` for each IO node, and returns the
  watchers plus a cleanup function.

Unit tests for register scaling and the data-format conversions can go in the
same file without a server, since `ReadReg`/`WriteReg` operate on a
`modbus.Regs` directly. These cover the conversion math in isolation, which
makes a failure in the end-to-end matrix easier to place: if the unit tests pass
and the matrix fails, the problem is in the plumbing rather than the arithmetic.

### Cleanup and docs

Delete `node/modbus.go`, `node/modbus-manager.go`, `node/modbus-node.go`,
`node/modbus-io.go`, `node/modbus-io-node.go`, and remove `modbusManager` from
`node.Manager`. Update `docs/user/modbus.md` if any behavior described there
changes, and add a changelog entry.

## Phase 2 — 1-Wire Client

Add `client/onewire.go`; remove `node/onewire*.go`.

```go
// OneWire describes a 1-wire bus
type OneWire struct {
	ID              string      `node:"id"`
	Parent          string      `node:"parent"`
	Description     string      `point:"description"`
	Index           int         `point:"index"`
	PollPeriod      int         `point:"pollPeriod"`
	Debug           int         `point:"debug"`
	Disabled        bool        `point:"disabled"`
	ErrorCount      int         `point:"errorCount"`
	ErrorCountReset bool        `point:"errorCountReset"`
	IOs             []OneWireIO `child:"oneWireIO"`
}

// OneWireIO describes a device on a 1-wire bus
type OneWireIO struct {
	ID              string  `node:"id"`
	Parent          string  `node:"parent"`
	Description     string  `point:"description"`
	DeviceID        string  `point:"id"`
	Units           string  `point:"units"`
	Value           float64 `point:"value"`
	Disabled        bool    `point:"disabled"`
	ErrorCount      int     `point:"errorCount"`
	ErrorCountReset bool    `point:"errorCountReset"`
}
```

`Run` keeps one poll timer (defaulting to 3 s when `PollPeriod` is unset). Each
tick detects new devices under `/sys/bus/w1/devices/w1_bus_master<Index>/28-*`
and creates a `oneWireIO` child for any not already in `config.IOs`, then reads
`/sys/bus/w1/devices/<DeviceID>/temperature` for each enabled IO, converting to
Fahrenheit when `Units` is `F`, and publishes a `value` point when the reading
changed or ten minutes have passed. Read failures increment the bus and IO error
counts. Point handling reduces to `MergePoints` plus a small switch for
`pollPeriod` and `errorCountReset`.

Sysfs access is behind a small interface (or a package-level variable holding
the device root) so tests can point it at a fixture directory. Register
`NewOneWireClient` in `client.DefaultClients`.

Tests in `client/onewire_test.go` use a temporary directory laid out like the w1
sysfs tree: assert that a `28-*` directory produces a `oneWireIO` child node,
that a `temperature` file of `23456` produces a `value` of `23.456`, that
`Units` of `F` converts, and that a missing or empty file increments both error
counts.

Delete `node/onewire.go`, `node/onewire-manager.go`, `node/onewire-node.go`,
`node/onewire-io.go`, `node/onewire-io-node.go`, and remove `oneWireManager`
from `node.Manager`. Update `docs/user/onewire.md` to say that a 1-Wire node is
added by hand with the bus index set, and that sensors on that bus are detected
automatically. Add a changelog entry that calls out the change in bus creation,
since it affects existing installs on their next start.

## Phase 3 — Retire the `node` Package

With both managers gone, `node.Manager` does nothing but write the app and OS
version to the root node, and `node/node.go` also holds `renderNotifyTemplate`,
which nothing outside `node/node_test.go` calls.

- Move the version reporting into `server/server.go` as a function that runs
  once after `siotStore.WaitStart`, replacing the `nodeManager` entry in the run
  group. It needs `o.AppVersion`, `o.OSVersionField`, and
  `system.ReadOSVersion`, all already reachable from there.
- Delete `renderNotifyTemplate`, `nodeTemplateData`, and `node/node_test.go`.
  Notification templating lives in the rule client now; if any of this is worth
  keeping it belongs there, and nothing currently calls it.
- Delete the `node/` package and its import in `server/server.go`.

Verify with `golangci-lint run` and `go test -race ./...`.

## Phase 4 — Documentation and Verification

- `docs/ref/architecture-app.md` and `docs/ref/client.md`: check for references
  to the node manager or to modbus/1-Wire as non-client code and update them.
  `client/doc.go` already describes Modbus and 1-Wire as example clients, which
  becomes true with this work.
- `docs/user/clients.md`: confirm modbus and 1-Wire are listed with the other
  clients.
- Run the full suite: `siot_test`.
- Manual check against hardware where available: an RTU client against a real
  device, and a DS18B20 on a Raspberry Pi GPIO bus. Confirm that a configuration
  exported from the current release imports and runs unchanged.

## Risks

**Client restart on IO changes.** A modbus server holding TCP connections drops
them when an IO is added or removed. Worth watching during the hardware check;
if it turns out to matter, the fix is to keep the socket open across restarts,
but there is no reason to build that before seeing the problem.

**1-Wire bus nodes are no longer created automatically.** Existing installs keep
their bus nodes, since node data is unchanged, so this affects only new setups
and any install that was relying on rediscovery after deleting a bus node. The
changelog entry and `docs/user/onewire.md` need to be clear about it.

**Behavior differences from generic decoding.** The hand-written decoders apply
defaults and validation in ways `data.Decode` does not — most notably the
timeout correction and the required-point errors. Phase 1's validation step has
to cover these deliberately rather than by accident; a modbus node that silently
polls with a zero timeout would be a regression.

**No test coverage exists today**, so there is no safety net for the move
itself. The tests written in phases 1 and 2 are the first coverage either
subsystem has had, which argues for writing them against the new client early in
each phase rather than at the end.
