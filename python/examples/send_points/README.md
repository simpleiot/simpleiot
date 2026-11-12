# Send points

Creates a variable node under the root, then writes a value to it once a second,
the way a script reading a sensor would. Stop it with Ctrl-C.

```
SIOT_AUTH_TOKEN=... python examples/send_points/send_points.py
```

```
writing to node bff924jzv12sf98jkbnkk1terg
value 17.28
value 17.72
```

The node is created with `create_node()`, which takes the initial points and a
description. A later run finds it again with `get_children()` and reuses it.
Each value is written with `send_point()`, which waits for the store to accept
it. Open the node in the web UI to see the value change.
