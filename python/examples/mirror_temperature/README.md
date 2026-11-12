# React to a point

Watches the `value` point of one node, converts it from Celsius to Fahrenheit,
and writes the result to a new node beside it. This is the shape of a small
control script: subscribe to one point, compute, and write another. Stop it with
Ctrl-C.

```
SIOT_AUTH_TOKEN=... python examples/mirror_temperature/mirror_temperature.py <node-id>
```

```
writing Fahrenheit to 2wer551eezqp8a683j3bbg7kn8
20.45 °C -> 68.81 °F
20.94 °C -> 69.69 °F
```

Pass the ID of the node [send_points](../send_points/) writes to, or of any node
with a `value` point. The script reads the source with `get_node()`, creates the
target under the same parent with `create_node()`, and writes to it from a
`subscribe_node()` loop.
