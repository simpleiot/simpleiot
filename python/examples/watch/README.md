# Watch for changes

Prints every point change at or below a node, the root by default. Stop it with
Ctrl-C.

```
SIOT_AUTH_TOKEN=... python examples/watch/watch.py [node-id]
```

```
13:50:15 bff924jzv12sf98jkbnkk1terg value.0 = 17.28
13:50:16 bff924jzv12sf98jkbnkk1terg value.0 = 17.72
```

`subscribe_tree()` delivers node and edge points for the node and everything
below it, after the store accepts them. Run it next to
[send_points](../send_points/) or change a value in the web UI to see output.
Use `subscribe_node()` instead to watch one node.
