# User Interface

**Contents**

<!-- toc -->

## Basic Navigation

After Simple IoT is started, a web application is available on port `:8118`
(typically [http://localhost:8118](http://localhost:8118)). After logging in
(default user/pass is `admin`/`admin`), you will be presented with a tree of
nodes.

![nodes](images/nodes.png)

The `Node` is the base unit of configuration. Each node contains `Points` which
describe various attributes of a node. When you expand a node, the information
you see is a rendering of the point data in the node.

You can expand/collapse child nodes by clicking on the arrow
![arrow](images/icon-arrow.png) to the left of a node.

You can expand/edit node details by clicking on the dot
![dot](images/icon-dot.png) to the left of a node.

![node edit](images/node-edit.png)

## Live updates

The page keeps one connection to the server and shows changes as they happen: a
value a device reports, a node someone else adds, or a setting you change in
another tab appear without a refresh. Expanding a node fetches its children, and
the arrow shows a spinner until they arrive; expanding again after collapsing
fetches them afresh. A `connecting...` badge next to the logo means the
connection is down and the page is trying again; when it is back, what is on
screen is fetched again. Being removed from a group, or a password change,
closes the connection, and an expired sign-in returns you to the sign-in page.

## Adding nodes

Child nodes can be added to a node by clicking on the dot to expand the node,
then clicking on the plus icon. A list of available nodes to add will then be
displayed:

![node add](images/node-add.png)

Some nodes are populated automatically if a new device is discovered, or a
downstream device starts sending data.

## Deleting, Moving, Mirroring, and Duplicating nodes

Simple IoT provides the ability to re-arrange and organize your node structure.

To delete a node, expand it, and then press the delete
![icon delete](images/icon-delete.png) icon.

To move or copy a node, expand it and press the copy
![copy icon](images/icon-copy.png) icon. Then expand the destination node and
press the paste ![paste icon](images/icon-paste.png) icon. You will then be
presented with the following options:

![paste options](images/paste-options.png)

- `move` - moves a node to new location
- `mirror` - is useful if you want a user or device to be a member of multiple
  groups. If you change a node, all the mirror copies of the node update as
  well.
- `duplicate` - recursively duplicates the copied node plus all its descendants.
  This is useful for scenarios where you have a device or site configuration
  (perhaps a complex Modbus setup) that you want to duplicate at a new site.

Mirroring a node that talks to hardware (a Modbus IO, a Shelly IO, a GPIO line,
an MQTT connection) gives you a view of it rather than a second copy that runs.
The instance where the node actually lives keeps talking to the device, and the
mirror displays the same values wherever you put it. This is what makes it safe
to mirror a sensor from inside a device into a group you share with someone, and
mirrors are labeled in the tree so it is clear that nothing runs there. Controls
still work from a mirror: setting a value on one travels to the device that owns
the node, and the result comes back.

Some nodes belong under a particular parent and are found through it: a Modbus
IO under its Modbus node, a rule condition under its rule. For these, `mirror`
is the only option offered, because moving one somewhere else would leave it
where nothing looks for it.

Deleting a node where it lives also removes its mirrors, so a deleted sensor
does not leave entries behind in the groups it was mirrored into. Removing a
mirror leaves the node itself alone.

If you have mirrors that were created before this behavior existed, they keep
working the way they did. To bring one up to date, remove the mirror and mirror
it again from the node where it lives.

## Raw Node View

If a node is expanded, a raw node button is available that allows you to view
the raw type and points for any node in the tree. It is useful at times during
development and debugging to be able to view the raw points for a node.

<img src="assets/image-20231205121809598.png" alt="image-20231205121809598" style="zoom:80%;" />

After the raw button is pressed, the type and points are displayed:

<img src="./assets/image-20231219103344798.png" alt="image-20231219103344798" style="zoom:67%;" />

Unknown nodes will also be displayed as raw nodes.

Points can also be edited, added, or removed in raw mode.

A custom node type can also be added by specifying the node type when adding a
node. This can be useful when developing new clients or external clients that
run outside of the SImple IoT application.

<img src="./assets/image-20231219103622737.png" alt="image-20231219103622737" style="zoom: 50%;" />

## Graphing and advanced dashboards

If you need graphs and more advanced dashboards, consider coupling Simple IoT
[with Grafana](graphing.md). Someday we hope to have dashboard capabilities
built in.

## Custom UIs

See the [frontend reference documentation](../ref/frontend.md).
