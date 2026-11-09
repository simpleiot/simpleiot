---
name: siot-build
description:
  Use when adding or changing nodes in a Simple IoT instance, one node or a
  whole demo — a Modbus bus, a signal generator, a database client, a rule, a
  user, a dashboard, a site mimic, a set of devices with their dashboards, or a
  sample tree for a talk or a customer. Triggers on requests like "add a modbus
  node", "create a signal generator writing to X", "set up a db node pointing at
  Victoria Metrics", "add a rule that ...", "configure an upstream sync",
  "provision these nodes on the device", "let's build a demo dashboard", "add a
  tank and a pump to the canvas", "make the switches do something", "put a
  drawing of the panel on it", or "show me how it looks". Writes a YAML node
  file, dry runs it, applies it with `siot import`, and, when building out a
  dashboard, screenshots the result and asks for the next step. Use the
  provisioning directory only when the request says provision, and `siot-debug`
  for a running instance that misbehaves.
---

# Adding nodes and building out an instance

Nodes are added by writing a YAML node file and applying it, rather than by
clicking through the UI. The same file format serves `siot import`,
`siot export`, and provisioning, so a file that works one way works the others.
A demo or a site mimic is the same file grown a step at a time: change it, apply
it, screenshot it, show the user, ask what is next. The file stays the source of
truth, so the whole thing can be rebuilt on a fresh instance with one import,
kept in the repo, or handed to someone else.

**One node or a build?** A request for a single node — "add a db node", "add a
rule that ..." — is sections 1 to 4: schema, file, dry run, import, verify.
Building out a dashboard or a tree with the user adds the loop in section 6 and
the dashboard schema in section 8.

**Default to `siot import`.** It applies to a running instance immediately and
is what almost every request means. Reach for the provisioning directory only
when the request says provision, or asks for configuration that survives a
reflash and comes up on its own.

| Path                   | When                                                      | Applied                                  |
| ---------------------- | --------------------------------------------------------- | ---------------------------------------- |
| `siot import`          | The default                                               | Once, immediately, to a running instance |
| Provisioning directory | Asked for by name, or config that must come up on its own | At start-up and whenever a file changes  |

## 1. Find the schema

Every client doc has a `## Schema` section showing an export of that node type
with its point types, value forms, and enumerations. Read it before writing
anything — point types are exact strings, and a wrong one is accepted silently
as a point nothing reads.

| Node type                                       | Doc                             |
| ----------------------------------------------- | ------------------------------- |
| `browser`                                       | `docs/user/browser.md`          |
| `canBus`                                        | `docs/user/can.md`              |
| `dashboard`, `widget`, `link`                   | section 8, `docs/user/dashboards.md` |
| `db`                                            | `docs/user/database.md`         |
| `file`                                          | `docs/user/file.md`             |
| `gps`                                           | `docs/user/gps.md`              |
| `group`, `user`                                 | `docs/user/users-groups.md`     |
| `metrics`                                       | `docs/user/metrics.md`          |
| `modbus`, `modbusIo`                            | `docs/user/modbus.md`           |
| `msgService`                                    | `docs/user/messaging.md`        |
| `oneWire`, `oneWireIO`                          | `docs/user/onewire.md`          |
| `particle`                                      | `docs/user/particle.md`         |
| `rule`, `condition`, `action`, `actionInactive` | `docs/user/rules.md`            |
| `serialDev`                                     | `docs/user/mcu.md`              |
| `shelly`, `shellyIo`                            | `docs/user/shelly.md`           |
| `signalGenerator`                               | `docs/user/signal-generator.md` |
| `sync`                                          | `docs/user/sync.md`             |
| `update`                                        | `docs/user/update.md`           |

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
  numbers, times, and every widget `prop`.
- **Descriptions are the match key.** A node in a file matches an existing node
  when the parent and the description agree, which is what makes applying twice
  the same as applying once. Choose descriptions meant to last: a renamed node
  is a new node beside the old one.
- **Three keys are reserved**: `parent`, `children`, `edgePoints`. Everything
  else is a point type, `id` included — on a Modbus or 1-wire node `id` is a
  device address, not the node's own ID, which never appears in a file.
- **`parent` names a node by description** and only applies to a top level
  entry. Entries apply in order, so a `parent` naming a node the same file
  creates has to come after it.
- **A `nodeID` point names its target by description** too, and resolves after
  the whole file is read, so it may point forward. Widget bindings and link
  ends resolve the same way.
- **Edge points go under `edgePoints`** — a user's `role` is the one in common
  use.
- **An entry with no description matches the single node of its type**, which is
  how a `metrics` or `serial` node is addressed. A user has no description, so
  it is found by `email`, and by name when there is no email.
- **A `root:` section sets points on the instance's own device node**, such as
  its `description` and `tag`. It holds points only, at the top of the file
  beside `nodes:`, and is what to use for "name this device" or "tag this
  instance".
- `apiVersion: 1` is optional. Adding it costs nothing and says which format the
  file is in.

Where the file lives: `testdata/<name>.yaml` for a demo meant to stay with the
repo, such as the ones there already; the scratchpad for a one-off or a
throwaway; the user's own directory or a provisioning directory when they say
so. One file, named for what it builds.

## 3a. Import (the default)

```bash
./siot import -dryRun < config.yaml     # prints the plan, applies nothing
./siot import < config.yaml
```

Read the dry run before applying. It prints one line per node —
`create modbus Sensor bus (7 point(s))`, `update ...`, `delete ...` — so
`create` where you expected `update` means the description did not match an
existing node and a second node is about to appear beside it.

`import` reads STDIN and gives up after two seconds, so always redirect a file
into it. The user's dev instance is usually on the default ports, NATS 4222 and
HTTP 4223; confirm with `ss -ltnp | grep siot`, where the HTTP port is one above
the NATS port unless `SIOT_HTTP_PORT` says otherwise. Add
`-natsServer nats://127.0.0.1:<port>` for any other port and `-token` (or
`SIOT_AUTH_TOKEN`) when the instance requires authentication. Never start a
second instance on the default ports; a scratch instance for trying things goes
on its own ports with its own `SIOT_DATA` directory.

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
  `10-`, `20-` prefixes to express which goes first. Dotfiles and subdirectories
  are skipped.
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

Export the tree and confirm the node reads back the way the file described it. A
point that came out empty where text was expected is the quoting rule; a point
that is missing entirely is a point type that does not match what the client
reads. A dashboard is verified with a screenshot (section 7).

## 5. Removing nodes

Applying a file never removes anything for going unmentioned. A `delete` list
removes nodes, matched the way `nodes` entries are:

```yaml
delete:
  - modbus:
      parent: Tank farm
      description: Old sensors
```

Deleting what is already gone does nothing. Renaming a node in a file is a
`delete:` entry for the old description beside the entry with the new one.

## 6. Building in rounds

Before the first node, settle in one message anything the request does not say:
which instance, where the file lives (section 2), and what the build is for — a
demo of a feature, a mimic of a real plant, a wall display. The purpose decides
the layout and how much should move on its own.

Order the rounds so each one shows something:

1. **A group and the live values**: variables for switches and settings, signal
   generators for anything that should move. Nothing on a dashboard moves
   without a point that changes, so give every reading a generator from the
   start (`signalType` sine, triangle, or random walk; `sampleRate` 1 is one
   sample a second).
2. **One dashboard with a few widgets**, applied and screenshotted, so the
   user sees the shape before it is filled in.
3. **The rest of the widgets, links, and drawings**, a dashboard at a time.
4. **Rules that make the switches do something**: a rule whose condition is
   the switch and whose action starts a generator or sets a variable.
   `testdata/dashboard-demo.yaml` has the pattern for a pump switch that starts
   the flow signal and opens a valve, and for an alarm that asks to be
   acknowledged. All of a rule's conditions must hold, so "any switch on" is a
   rule on "every switch off" with the work in its `actionInactive`.
5. **An overview** of nested dashboards and the admin's home dashboard, last,
   once the parts are right.

After every round:

```bash
./siot import -dryRun < demo.yaml     # read it: create where update was expected means a duplicate
./siot import < demo.yaml
node .claude/skills/siot-build/scripts/shot.mjs --dash "Overview" --out /path/shot.png
```

Look at the image with the Read tool, and describe it to the user in a line or
two: what is drawn, what moved, what is dead. Then ask what is next, with one
suggestion. Keep each round small enough that a screenshot tells whether it
worked; five widgets, not fifty.

## 7. The screenshot helper

`scripts/shot.mjs` signs in, opens a dashboard by its description from the
dashboard list, waits for values, and saves a PNG. It also prints the widgets
drawn dead and the state of every bound element in a drawing, so a binding
that found nothing is in the output as well as in the picture.

```bash
node .claude/skills/siot-build/scripts/shot.mjs --dash "Supply pump" --out shot.png
node .claude/skills/siot-build/scripts/shot.mjs --dash "Overview" --edit --out edit.png   # ports and frames
node .claude/skills/siot-build/scripts/shot.mjs --out list.png                            # the dashboard list
```

`--url` picks the instance, `--user` and `--pass` the sign-in, `--width` and
`--height` the viewport, `--kiosk` hides the header, `--dark` uses the dark
theme, and `--wait` the milliseconds given to arriving values. It drives the
system Chrome through the Playwright in `ui/node_modules`, so `siot_setup` has
to have run. Run it from the repository root.

For a closer look at one widget or a click on a control, write a short
Playwright script in the scratchpad along the same lines, or use the browser
tools when they are available. The dashboard page reaches a widget's settings at
`/dash/<dashboard id>/<widget id>` in edit mode.

## 8. Dashboard schema

A dashboard, its widgets, and its links are nodes, written like any other.
Bindings and link ends name nodes by description, so the file reads as the
dashboard looks. Keyed points are YAML mappings.

```yaml
- dashboard:
    description: Supply pump
    layout: canvas # grid (the default, 12 columns) or canvas
    canvasWidth: 600 # canvas units; 900 by 500 when unset
    canvasHeight: 270
    cols: 12 # grid only
    children:
      - widget:
          description: Pump switch # must differ from the bound node's description
          component: status # auto when unset: the node type's usual widget
          x: 20 # grid: column, row, width, height in cells
          y: 20 # canvas: x, y, width in canvas units; height from the component
          w: 160
          bindNode:
            value: Pump enable # binding name: node description
          bindType: # the point type a binding reads, value when unset
            value: value
          bindKey: # the point key, "0" when unset
            value: "0"
          prop: # the component's options, all text
            on: Running
            off: Stopped
            confirm: "on"
          port: # override or add a port: "x,y" or "x,y,dir" as fractions
            out: "1,0.5,right"
      - link:
          description: Pump to outlet
          from: Pump.discharge # widget description or ID, then port name
          to: Outlet.in
          routing: orthogonal # or direct
          bindNode:
            flowing: Pump enable # animates while on; rate scales the speed
```

The components, their bindings, and their options:

| Component   | Host   | Bindings           | Options                                            | Ports                           |
| ----------- | ------ | ------------------ | -------------------------------------------------- | ------------------------------- |
| `value`     | both   | `value`            | `decimals`, `unit`, `multiplier`, `confirm`        |                                 |
| `gauge`     | both   | `value`            | `min`, `max`, `unit`, `decimals`, `multiplier`     |                                 |
| `status`    | both   | `value`            | `on`, `off`, `confirm`                             |                                 |
| `trend`     | both   | `value`            | `window`, `min`, `max`, `unit`, `multiplier`       |                                 |
| `tank`      | canvas | `level`, `temp`    |                                                    | `in` up, `out` down, `left`     |
| `pump`      | canvas | `running`, `speed` |                                                    | `suction` left, `discharge` up  |
| `valve`     | canvas | `open`             |                                                    | `in` left, `out` right          |
| `svg`       | both   | `svg` + the drawing's own | `confirm`                                   | from `data-siot-port` elements  |
| `dashboard` | both   | `dashboard`        |                                                    |                                 |

Every widget also takes `scale`, and `control: "off"` makes a control show
only. A `status` on an on/off variable is a switch and a `value` on a number or
text variable is a setting; `confirm: "on"` asks before sending.

Other points worth knowing, all in `docs/user/dashboards.md`:

- **Relative bindings**: `bindPath: { value: variable/Pump enable }` names a
  path from the dashboard's parent, so a dashboard mirrored under several
  devices shows each device's own nodes. A mirror in a file is the `mirror:`
  key under the second parent naming the dashboard by description.
- **Drawings**: an `svg` widget binds `svg` to a `file` node whose `data` is
  the SVG, written inline in the file as a block scalar. The drawing declares
  its bindings with `data-siot-bind`; `data-siot-text`, `rotate`, `level`,
  `toggle`, `set`, and `port` do the rest, and the drawing's own CSS styles
  `data-siot-state`, which is `on` for any value that is not zero, so a numeric
  binding can show an element while a signal runs. A CSS animation on
  `[data-siot-state="on"]` is how a wheel or an impeller spins.
  `testdata/custom-widget-demo.yaml` has a lamp, a dial, a vessel, a start and
  stop panel, and a pump to copy from.
- **A drawing's styles are global.** The widget injects the SVG inline, so a
  `<style>` block inside it applies to the whole page, not just the drawing.
  A class name the UI also uses, such as `.frame`, `.body`, `.content`, or
  `.label`, restyles the dashboard around it, and an inherited property like
  `stroke` or `font-weight` then flows into every other widget's SVG: a
  `.frame { stroke: ... }` rule outlines the text in every gauge on the canvas.
  Prefix every class in a drawing with something of its own (`.tr-frame`,
  `.tr-wheel`) and keep the rules to those classes.
- **Home dashboard**: `homeDashboard: Overview` on the admin user, matched by
  `email: admin`, lands sign-in on the demo.
- **Kiosk**: `?kiosk` on the address hides the header for a wall display.

## 9. Pitfalls

| Symptom                                              | Cause                                                                                        |
| ---------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| A second node appears beside the first               | The description no longer matches, usually because one side was renamed                      |
| `2 nodes here match "..."`                           | Two nodes share a parent and a description; make them distinct                               |
| `is a group node here and a modbus node in the file` | The description names a node of another type; delete it first if the type is meant to change |
| A text setting reads as empty                        | The value was written bare and became numeric — quote it                                     |
| `this looks like the old export format`              | The file names its type in a `type:` field; re-export to get the current one                 |
| A user is not found                                  | A user is matched by `email`, or by name when there is no email                              |
| A provisioning file was edited but nothing changed   | Provisioning applies on a content change; check the checksum on the `provisioningFile` node  |
| A widget is drawn dead                               | Its binding found nothing: a description that does not match, a path that reaches nothing, or a node later in the file; `shot.mjs` prints them |
| A widget shows nothing or the wrong label            | Its description equals the bound node's; bindings resolve by description across the tree, so name the widget "Pump" and the variable "Pump enable" |
| A reading never changes                              | A variable holds its value; a reading that should drift needs a `signalGenerator`, and a switch that should do something needs a rule |
| A widget jumps back after the user dragged it        | The UI and the file both write positions; pull the layout into the file with `./siot export` before the next round, or edit layout in one place only |
| Text in every gauge or widget looks smeared or outlined | A drawing's `<style>` is global, and one of its class names matches a UI class; prefix the drawing's classes (section 8) |
| `2 nodes have the description "..."` on a binding or `nodeID` | A widget or other node shares its description with the node it binds to; bindings resolve by description across the whole tree, so rename one of them |

Canvas sizes: a canvas of 600 by 270 fills a tile six columns by four rows on a
12 column overview; keep the same ratio when nesting.

Files carry secrets in the clear: `authToken` on `db`, `sync`, `particle`, and
`msgService` nodes, and `pass` on a `user`. Treat a node file the way you would
treat the credentials inside it, and keep exports of a live tree out of the
repo.

## 10. Finishing

When the user is happy: make sure the file applies cleanly to a fresh instance
(`-dryRun` shows only `create` lines against an empty tree), put a comment at
the top saying what it builds and how to load it, and, for a file in
`testdata/`, mention it in `docs/user/dashboards.md` under "A demo". A demo
that shows a new feature gets a changelog line; one that only exercises
existing ones does not.
