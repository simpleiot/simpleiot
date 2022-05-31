# User Interface

**Contents**

<!-- toc -->

## Basic Navigation

After Simple IoT is started, a web application is available on port `:8080`
(typically [http://localhost:8080](http://localhost:8080)). After logging in
(default user/pass is `admin@admin.com`/`admin`), you will be presented with a
tree of nodes.

![nodes](images/nodes.png)

The `Node` is the base unit of configuration. Each node contains `Points` which
describe various attributes of a node. When you expand a node, the information
you see is a rendering of the point data in the node.

You can expand/collapse child nodes by clicking on the arrow
![arrow](images/icon-arrow.png) to the left of a node.

You can expand/edit node details by clicking on the dot
![dot](images/icon-dot.png) to the left of a node.

![node edit](images/node-edit.png)

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

- **move** - moves a node to new location
- **mirror** - is useful if you want a user or device to be a member of multiple
  groups. If you change a node, all of the mirror copies of the node update as
  well.
- **duplicate** - recursively duplicates the copied node plus all its
  descendants. This is useful for scenarios where you have a device or site
  configuration (perhaps a complex Modbus setup) that you want to duplicate at a
  new site.
