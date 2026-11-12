# Print the node tree

Prints every node in an instance, indented under its parent, with its type,
description, and ID.

```
SIOT_AUTH_TOKEN=... python examples/tree/tree.py
```

```
device: (no description)  [e23ez4m98xn4cyy92hry0q9dgw]
  user: (no description)  [62dtq0d9je9g26rs2qz53d2b6w]
  variable: python example  [bff924jzv12sf98jkbnkk1terg]
```

It fetches the root with `get_root()`, then the rest of the tree in one request
with `get_nodes(root.id, "all", depth=10)`. The reply is a flat list, and each
node's `parent` says where it goes.
