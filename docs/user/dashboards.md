# Dashboards

A dashboard shows live values from any nodes on one page: a tank level as a
gauge, a pump as a status pill, a temperature as a trend. It is built from
nodes, so it syncs between instances, follows the same permissions as the nodes
it shows, and can be created by a script or an agent through the same API as
everything else.

Open **Dashboards** in the header to see the dashboards you can reach, and open
one to view it. On a phone the grid becomes a single column.

## Building one

1. In **Nodes**, add a **Dashboard** under a device or group and give it a
   description. Open its details for an **open** link to the dashboard page.
2. Under the dashboard, add a **Widget** for each thing to show. Pick the node
   it reads by typing its description in the node field, then the point type
   and key if the default `value` is not what you want.
3. Leave the component on **auto** to show the node the usual way for its type,
   or pick one:

   | Component | Shows                       | Options                            |
   | --------- | --------------------------- | ---------------------------------- |
   | value     | the number or text, large   | `decimals`, `unit`                 |
   | gauge     | a radial gauge over a range | `min`, `max`, `unit`, `decimals`   |
   | status    | on or off                   | `on`, `off` (the labels)           |
   | trend     | the last values as a line   | `window`, `min`, `max`, `unit`     |

4. Set the widget's column, row, width, and height in grid units. A dashboard
   has 12 columns unless its **Columns** setting says otherwise.
5. Add options under **Options**, keyed by the names in the table.

A widget whose node cannot be found draws with a dashed border and shows no
value. A dashboard synced to a device that lacks some of the nodes still shows
its shape.

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

Editing widgets in place on the dashboard, a home dashboard per user, a canvas
layout for process mimics with links between equipment, and trends over stored
history.
