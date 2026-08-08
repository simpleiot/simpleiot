---
name: siot-add-node
description:
  Use when adding or changing nodes in a Simple IoT instance — a Modbus bus, a
  signal generator, a database client, a rule, a user, a group, or any other
  node type. Triggers on requests like "add a modbus node", "create a signal
  generator writing to X", "set up a db node pointing at Victoria Metrics",
  "add a rule that ...", "configure an upstream sync", or "provision these
  nodes on the device". Writes the node file, dry runs it, and applies it with
  `siot import`; use the provisioning directory only when the request says
  provision.
---

# Adding nodes to a SIOT instance

Nodes are added by writing a YAML node file and applying it, rather than by
clicking through the UI. The same file format serves `siot import`, `siot
export`, and provisioning, so a file that works one way works the others.

**Default to `siot import`.** It applies to a running instance immediately and
is what almost every request means. Reach for the provisioning directory only
when the request says provision, or asks for configuration that survives a
reflash and comes up on its own.

| Path | When | Applied |
| --- | --- | --- |
| `siot import` | The default | Once, immediately, to a running instance |
| Provisioning directory | Asked for by name, or config that must come up on its own | At start-up and whenever a file changes |

## 1. Find the schema

Every client doc has a `## Schema` section showing an export of that node type
with its point types, value forms, and enumerations. Read it before writing
anything — point types are exact strings, and a wrong one is accepted silently
as a point nothing reads.

| Node type | Doc |
| --- | --- |
| `browser` | `docs/user/browser.md` |
| `canBus` | `docs/user/can.md` |
| `db` | `docs/user/database.md` |
| `file` | `docs/user/file.md` |
| `gps` | `docs/user/gps.md` |
| `group`, `user` | `docs/user/users-groups.md` |
| `metrics` | `docs/user/metrics.md` |
| `modbus`, `modbusIo` | `docs/user/modbus.md` |
| `msgService` | `docs/user/messaging.md` |
| `oneWire`, `oneWireIO` | `docs/user/onewire.md` |
| `particle` | `docs/user/particle.md` |
| `rule`, `condition`, `action`, `actionInactive` | `docs/user/rules.md` |
| `serialDev` | `docs/user/mcu.md` |
| `shelly`, `shellyIo` | `docs/user/shelly.md` |
| `signalGenerator` | `docs/user/signal-generator.md` |
| `sync` | `docs/user/sync.md` |
| `update` | `docs/user/update.md` |

`docs/user/configuration.md` is the reference for the format itself.

Node types with no user doc — `variable`, `ntp`, `networkManager`,
`networkManagerConn`, `networkManagerDevice` — take their point types from the
`point:"..."` struct tags in the matching `client/*.go`. Reading an existing
node of that type with `siot export` also works and shows real values.

## 2. Write the file

The node type is the key and each point type is a key under it:

```yaml
nodes:
  - group:
      description: Tank farm
  - modbus:
      parent: Tank farm
      description: Sensor bus
      clientServer: client
      protocol: RTU
      port: /dev/ttyUSB0
      baud: "9600"
      pollPeriod: 500
      children:
        - modbusIo:
            description: Tank level
            id: 1
            address: 3
            modbusIoType: modbusHoldingRegister
            dataFormat: uint16
            scale: 0.1
```

What decides the outcome:

- **How a value is written decides what it becomes.** `9600` is numeric,
  `"9600"` is text. A client expecting text reads an empty value when given a
  number, so quote anything the schema shows quoted — `port`, `baud`, phone
  numbers, times.
- **Descriptions are the match key.** A node in a file matches an existing node
  when the parent and the description agree, which is what makes applying twice
  the same as applying once. Choose descriptions meant to last.
- **Three keys are reserved**: `parent`, `children`, `edgePoints`. Everything
  else is a point type, `id` included — on a Modbus or 1-wire node `id` is a
  device address, not the node's own ID, which never appears in a file.
- **`parent` names a node by description** and only applies to a top level
  entry. Entries apply in order, so a `parent` naming a node the same file
  creates has to come after it.
- **A `nodeID` point names its target by description** too, and resolves after
  the whole file is read, so it may point forward.
- **Edge points go under `edgePoints`** — a user's `role` is the one in common
  use.
- **An entry with no description matches the single node of its type**, which
  is how a `metrics` or `serial` node is addressed. A user has no description,
  so it is found by `email`, and by name when there is no email.
- `apiVersion: 1` is optional. Adding it costs nothing and says which format
  the file is in.

Write files under the scratchpad directory unless they are going into the repo
or a provisioning directory.

## 3a. Import (the default)

```bash
./siot import -dryRun < config.yaml     # prints the plan, applies nothing
./siot import < config.yaml
```

Read the dry run before applying. It prints one line per node — `create modbus
Sensor bus (7 point(s))`, `update ...`, `delete ...` — so `create` where you
expected `update` means the description did not match an existing node and a
second node is about to appear beside it.

`import` reads STDIN and gives up after two seconds, so always redirect a file
into it. Add `-natsServer nats://127.0.0.1:4222` for an instance that is not on
the default port and `-token` (or `SIOT_AUTH_TOKEN`) when it requires
authentication.

## 3b. Provisioning (only when asked)

Provisioning applies the same files at start-up and whenever they change, so a
unit built from an image comes up configured with no import step.

```bash
./siot provision -dir ./provisioning -check   # parse only, no instance needed
./siot provision -dir ./provisioning          # what it would do, applying nothing
```

Where the files go:

- `-provisioningDir` or `SIOT_PROVISIONING_DIR`, falling back to
  `<SIOT_DATA>/provisioning` when that directory exists.
- Files are `.yaml` or `.yml`, applied in lexical order, so use the familiar
  `10-`, `20-` prefixes to express which goes first. Dotfiles and
  subdirectories are skipped.
- Files uploaded through the UI are `file` nodes under the `provisioning` node
  and are layered on top of the ones on disk.

A file is applied when its contents change, which is what leaves a value edited
in the UI alone until the file describing it changes. Check the `provisioning`
node afterwards: each file gets a `provisioningFile` child carrying its name,
the checksum applied, and the last error.

## 4. Verify

```bash
./siot export | grep -A20 'Sensor bus'
```

Export the tree and confirm the node reads back the way the file described it.
A point that came out empty where text was expected is the quoting rule; a
point that is missing entirely is a point type that does not match what the
client reads.

## Removing nodes

Applying a file never removes anything for going unmentioned. A `delete` list
removes nodes, matched the way `nodes` entries are:

```yaml
delete:
  - modbus:
      parent: Tank farm
      description: Old sensors
```

Deleting what is already gone does nothing.

## Pitfalls

| Symptom | Cause |
| --- | --- |
| A second node appears beside the first | The description no longer matches, usually because one side was renamed |
| `2 nodes here match "..."` | Two nodes share a parent and a description; make them distinct |
| `is a group node here and a modbus node in the file` | The description names a node of another type; delete it first if the type is meant to change |
| A text setting reads as empty | The value was written bare and became numeric — quote it |
| `this looks like the old export format` | The file names its type in a `type:` field; re-export to get the current one |
| A user is not found | A user is matched by `email`, or by name when there is no email |
| A provisioning file was edited but nothing changed | Provisioning applies on a content change; check the checksum on the `provisioningFile` node |

Files carry secrets in the clear: `authToken` on `db`, `sync`, `particle`, and
`msgService` nodes, and `pass` on a `user`. Treat a node file the way you would
treat the credentials inside it, and keep exports of a live tree out of the
repo.
