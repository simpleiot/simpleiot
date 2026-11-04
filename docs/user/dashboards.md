# Dashboards

A dashboard shows live values from any nodes on one page: a tank level as a
gauge, a pump as a status pill, a temperature as a trend. It is built from
nodes, so it syncs between instances, follows the same permissions as the nodes
it shows, and can be created by a script or an agent through the same API as
everything else.

![The demo's Overview: four canvases shown as tiles](images/dashboard-overview.png)

Open **Dashboards** in the header to see the dashboards you can reach, and open
one to view it. On a phone the grid becomes a single column. Tap a widget to
open the node behind it: a dialog on a wide screen, the node's page in the tree
on a narrow one.

## Building one

1. In **Nodes**, add a **Dashboard** under a device or group and give it a
   description. Open its details for an **open** link to the dashboard page.
2. On the dashboard page, press **edit**. **add widget** places a new widget
   below the others and opens its settings beside the grid. Pick the node it
   reads by typing its description in the node field, then the point type and
   key if the default `value` is not what you want. Drag a widget by its body to
   move it and by its corner to resize it; the place is saved when you let go.
   Press **done** to leave edit mode. The address follows the selected widget,
   `/dash/<dashboard>/<widget>`, so a link opens that widget's settings. A
   widget's settings also offer **duplicate**, which adds a copy with the same
   bindings and options and opens it, and **copy**, which puts the widget on the
   clipboard; **paste** then adds a copy of it to any dashboard in edit mode,
   this one or another.
3. Leave the component on **auto** to show the node the usual way for its type,
   or pick one:

   | Component | Shows                       | Options                          |
   | --------- | --------------------------- | -------------------------------- |
   | value     | the number or text, large   | `decimals`, `unit`, `confirm`    |
   | gauge     | a radial gauge over a range | `min`, `max`, `unit`, `decimals` |
   | status    | on or off                   | `on`, `off`, `confirm`           |
   | trend     | the last values as a line   | `window`, `min`, `max`, `unit`   |

4. A dashboard has 12 columns unless its **Columns** setting says otherwise. The
   column, row, width, and height fields in the widget's settings set its place
   in grid units when dragging is not convenient.
5. Add options under **Options**, keyed by the names in the table. Every widget
   also takes `scale`: `2` draws it at twice the size, for a reading that should
   carry across a room, and `0.5` at half.

Widgets are nodes under the dashboard, so everything above can also be done in
the node tree, by a script, or by an agent.

## Canvas layouts

Set a dashboard's **Layout** to **canvas** for a process mimic: a plane of
**Canvas width** by **Canvas height** units (900 by 500 unless set), scaled to
fit the page. Drag the background to pan, and zoom with the buttons in the
corner or with ctrl and the mouse wheel. A canvas stays a canvas on a phone,
scaled to fit the width; zoom in for a closer look.

Equipment components draw on the canvas:

| Component | Shows                                      | Bindings           |
| --------- | ------------------------------------------ | ------------------ |
| tank      | liquid at the level, tinted by temperature | `level`, `temp`    |
| pump      | an impeller that turns while running       | `running`, `speed` |
| valve     | a disc that turns with the position        | `open`             |

The grid widgets work on a canvas as well. **X**, **Y**, and **Width** are in
canvas units there. In edit mode, drag the corner of a widget to set its width,
which is the size the equipment draws at.

Links join two widgets at their **ports**, the named points on a widget's edge
that show as dots in edit mode. Drag from one port to another to draw a link. A
link's **From** and **To** are `widget.port`, where the widget is named by its
ID or by its description. Select a link to drag its corner points and to set its
routing, direct or orthogonal, in the panel. A link's `flowing` and `rate`
bindings animate it, and a link whose bound node is missing is drawn dashed and
dim. Links are drawn, not run: a pipe from a pump to a tank describes the plant
and implies nothing about where data goes.

A widget's ports can be changed under **Ports**: `x,y` or `x,y,dir` as fractions
of the widget's box, where `dir` is `left`, `right`, `up`, or `down`.

A widget whose node cannot be found draws with a dashed border and shows no
value. A dashboard synced to a device that lacks some of the nodes still shows
its shape.

## Controls

A widget bound to a node that accepts a new value is a control. A status pill on
an on/off variable is a switch: click it to flip the state. A value tile on a
number or text variable is a setting: click the value for an edit box, and Enter
sends it. A click anywhere else on a dashboard does nothing; the node behind a
widget is reached through edit mode, where the widget's settings link to it.

Which nodes accept writes, and on which point, is part of each node type: a
variable takes the value itself, and an output such as a Modbus IO or a GPIO
takes a set request. Set the `control` option to `off` on a widget that should
only show, and `confirm` to `on` for one whose change must be confirmed before
it is sent.

## Relative bindings

A binding usually names one node. It can instead name a path from the
dashboard's parent: `type/description` segments, such as `modbusIo/pump1` or
`device/Station A/modbusIo/pump1`, each picking a child of the node before it by
type and description. A dashboard with relative bindings placed under several
devices, by mirroring it, shows each device's own nodes. In a widget's settings,
after picking a node that sits below the dashboard's parent, press **make
relative to** that parent; **absolute** goes back to the node ID. A path that
reaches nothing draws the widget dead.

Opening a mirrored dashboard from the tree starts the paths from the parent it
was opened under; `?via=<node ID>` on the dashboard's address picks the parent
explicitly.

## Nested dashboards

The **nested dashboard** component shows another dashboard inside a widget: bind
its `dashboard` to the dashboard node. A dashboard that contains itself, at any
depth, stops there with a note.

## Home dashboard

A user's **Home dashboard**, set on the user's node, is where signing in and the
logo link land. A user without one lands on the dashboard list when any
dashboard is visible to them, and on the node tree otherwise.

## What a widget does not do

Widgets display; they never compute. Scaling, unit conversion, aggregation, and
events belong in the node that produces the point, or in a [rule](rules.md), so
the result is also available to notifications, the database, and other
dashboards, and keeps working when no browser is open. A widget may format what
it shows: decimals, units, a gauge range.

## Kiosk

Add `?kiosk` to a dashboard's address to hide the header and title for a wall
display.

## A demo

`testdata/dashboard-demo.yaml` in the source tree builds a small water system
under one group: a supply pump, a storage tank, a heating loop, and an alarm
panel, each on its own canvas, and an Overview that shows the four canvases as
tiles. Rules make the switches do something: the pump switch starts the flow and
pressure signals and opens the outlet valve, the inlet switch opens the inlet
valve and runs the tank level, the heater switch runs the temperatures, and a
high tank level raises an alarm that asks to be acknowledged. Load it into a
running instance with:

```
siot import < testdata/dashboard-demo.yaml
```

The file is also a worked example of writing dashboards by hand: widget bindings
name nodes by description, and link ends name widgets by description.

## Coming later

Trends over stored history.
