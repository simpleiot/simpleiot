---
name: siot-build
description:
  Use when building out a Simple IoT instance together with the user, step by
  step — a dashboard demo, a site mimic, a set of devices with their dashboards,
  or a sample tree for a talk or a customer. Triggers on requests like "let's
  build a demo dashboard", "add a tank and a pump to the canvas", "make the
  switches do something", "put a drawing of the panel on it", "show me how it
  looks", or "set up an instance that shows X". Keeps one node file as the
  source of truth, applies it with `siot import` after each step, screenshots
  the result, and asks for the next step. Use `siot-add-node` for a single node
  and `siot-debug` for a running instance that misbehaves.
---

# Building out an instance interactively

The instance is built from one YAML node file that grows a step at a time. Each
step: change the file, apply it, look at the result, show the user, ask what is
next. The file stays the source of truth, so the whole demo can be rebuilt on a
fresh instance with one import and kept in the repo or handed to someone else.

The file format and the node schemas are in the `siot-add-node` skill; read it
first. This skill adds the loop, the dashboard schema, and the pitfalls that
only show up when building over several rounds.

## 1. Settle three things before the first node

Ask when not given, in one message:

- **Which instance.** The user's dev instance is usually on the default ports,
  NATS 4222 and HTTP 4223. Confirm with `ss -ltnp | grep siot`; the HTTP port
  is one above the NATS port unless `SIOT_HTTP_PORT` says otherwise. Pass
  `-natsServer nats://127.0.0.1:<port>` to `siot import` for any other port, and
  `-token` or `SIOT_AUTH_TOKEN` when the instance requires it. Never start a
  second instance on the default ports; a scratch instance for trying things
  goes on its own ports with its own `SIOT_DATA` directory.
- **Where the file lives.** `testdata/<name>.yaml` for a demo meant to stay
  with the repo, such as the two there already; the scratchpad for a throwaway;
  the user's own directory when they say so. One file, named for what it builds.
- **What it is for.** A demo to show a feature, a mimic of a real plant, a
  wall display. This decides the layout and how much should move on its own.

## 2. Build in rounds

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
   acknowledged.
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

## 3. The screenshot helper

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

## 4. Dashboard schema

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
  `data-siot-state`. `testdata/custom-widget-demo.yaml` has a lamp, a dial, a
  vessel, a start and stop panel, and a pump to copy from.
- **Home dashboard**: `homeDashboard: Overview` on the admin user, matched by
  `email: admin`, lands sign-in on the demo.
- **Kiosk**: `?kiosk` on the address hides the header for a wall display.

## 5. Pitfalls of building over several rounds

- **The UI and the file both write positions.** A widget dragged in the browser
  gets new `x`, `y`, `w`, `h` points; the next import of the file puts it back.
  Before a round that follows the user moving things by hand, pull the layout
  into the file with `./siot export` (grep for the dashboard) and copy the
  numbers, or agree that layout is edited in one place only.
- **A renamed node is a new node.** Descriptions are the match key, so
  renaming in the file creates a second node beside the first. Rename with a
  `delete:` entry for the old description in the same file, or rename in the
  UI and then in the file.
- **A widget's description must differ from the node it shows**, since a
  binding is resolved by description across the tree. "Pump" the widget and
  "Pump enable" the variable, not "Pump" twice. The label a widget shows is the
  bound node's description unless the widget's own says otherwise.
- **Nothing moves without a generator or a rule.** A variable holds its value;
  a reading that should drift needs a `signalGenerator`, and a switch that
  should do something needs a rule.
- **Quote option values and text points.** `prop` values are text, so `min:
  "40"` not `min: 40`; a bare number becomes a numeric point the widget reads
  as empty.
- **A dead widget is a binding that found nothing**: a description that does
  not match, a path that reaches nothing, or a node that is not there yet
  because it comes later in the file. `shot.mjs` prints them.
- **Canvas sizes.** A canvas of 600 by 270 fills a tile six columns by four
  rows on a 12 column overview; keep the same ratio when nesting.

## 6. Finishing

When the user is happy: make sure the file applies cleanly to a fresh instance
(`-dryRun` shows only `create` lines against an empty tree), put a comment at
the top saying what it builds and how to load it, and, for a file in
`testdata/`, mention it in `docs/user/dashboards.md` under "A demo". A demo
that shows a new feature gets a changelog line; one that only exercises
existing ones does not.
