# Dashboards

A dashboard shows live values from any nodes on one page: a tank level as a
gauge, a pump as a status pill, a temperature as a trend. It is built from
nodes, so it syncs between instances, follows the same permissions as the nodes
it shows, and can be created by a script or an agent through the same API as
everything else.

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
   key if the default `value` is not what you want. Drag a widget by its body
   to move it and by its corner to resize it; the place is saved when you let
   go. Press **done** to leave edit mode.
3. Leave the component on **auto** to show the node the usual way for its type,
   or pick one:

   | Component | Shows                       | Options                            |
   | --------- | --------------------------- | ---------------------------------- |
   | value     | the number or text, large   | `decimals`, `unit`                 |
   | gauge     | a radial gauge over a range | `min`, `max`, `unit`, `decimals`   |
   | status    | on or off                   | `on`, `off` (the labels)           |
   | trend     | the last values as a line   | `window`, `min`, `max`, `unit`     |

4. A dashboard has 12 columns unless its **Columns** setting says otherwise.
   The column, row, width, and height fields in the widget's settings set its
   place in grid units when dragging is not convenient.
5. Add options under **Options**, keyed by the names in the table.

Widgets are nodes under the dashboard, so everything above can also be done in
the node tree, by a script, or by an agent.

A widget whose node cannot be found draws with a dashed border and shows no
value. A dashboard synced to a device that lacks some of the nodes still shows
its shape.

## Home dashboard

A user's **Home dashboard**, set on the user's node, is where signing in and
the logo link land. A user without one lands on the dashboard list when any
dashboard is visible to them, and on the node tree otherwise.

## What a widget does not do

Widgets display; they never compute. Scaling, unit conversion, aggregation, and
events belong in the node that produces the point, or in a [rule](rules.md),
so the result is also available to notifications, the database, and other
dashboards, and keeps working when no browser is open. A widget may format what
it shows: decimals, units, a gauge range.

## Kiosk

Add `?kiosk` to a dashboard's address to hide the header and title for a wall
display.

## Coming later

A canvas layout for process mimics with links between equipment, and trends
over stored history.
