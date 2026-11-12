# simpleiot

A Python client for [Simple IoT](https://simpleiot.org). Read and write nodes
and points, and watch for changes, over the NATS API.

The client connects with the instance's shared token (`SIOT_AUTH_TOKEN`), which
reaches the whole tree. It suits scripts and services running alongside an
instance you operate.

## Install

```
pip install simpleiot
```

Requires Python 3.10 or newer. The client is built on
[nats-py](https://github.com/nats-io/nats.py) and uses `asyncio`.

## Usage

```python
import asyncio
import simpleiot
from simpleiot import Point


async def main():
    async with await simpleiot.connect() as c:
        root = await c.get_root()
        print("root:", root.description)

        id = await c.create_node(root.id, "variable", description="tank level")
        await c.send_point(id, "value", 42.5)

        async with await c.subscribe_node(id) as sub:
            async for m in sub:
                for p in m.points:
                    print(p.type, p.value)


asyncio.run(main())
```

`connect()` reads `SIOT_NATS_SERVER` (default `nats://localhost:4222`) and
`SIOT_AUTH_TOKEN` from the environment, or takes them as arguments:

```python
c = await simpleiot.connect("nats://10.0.0.5:4222", token="...")
```

## Nodes and points

Data in Simple IoT is a tree of **nodes**, each holding **points**. A point has
a `type`, a `key` (`"0"` unless a node holds several of a type), a `value`, and
a `time`. The edge from a node to its parent carries its own points, such as
`tombstone` for a deleted node. See the
[data reference](https://docs.simpleiot.org/docs/ref/data.html).

`Point.value` sets the encoding when sending:

| Value                  | Sent as               |
| ---------------------- | --------------------- |
| `float`, `int`, `bool` | a float point         |
| `str`                  | a string point        |
| `dict`, `list`         | a JSON point          |
| `None`                 | a point with no value |

Pass `data_type=DataType.INT` to send an integer as an int point. A received
point's `value` is a `float`, `int`, or `str` by its data type; `number` and
`text` read it as a number or string either way.

## API

All methods are coroutines on `Client`.

- `get_root()`, `get_node(id, parent="all")`, `get_children(parent, type=...)`
- `get_nodes(parent, id="all", type=..., include_deleted=..., depth=...)`, the
  `nodes.<parent>.<id>` request
- `search(below, query, limit=..., type=...)`
- `send_points(id, points)`, `send_point(id, type, value, key="0")`
- `send_edge_points(id, parent, points)`
- `create_node(parent, type, points, id=..., description=..., primary=...)`
- `delete_node(id, parent)`, `move_node(id, old_parent, new_parent)`
- `subscribe_node(id)`, `subscribe_tree(id)`, `subscribe(subject)`: async
  iterators of `PointsMessage`
- `close()`

A server error raises `SiotError`; a missing node raises `NotFoundError`.

## Examples

The [examples](examples/) directory has short scripts:

- `tree.py` prints the node tree
- `send_points.py` creates a node and writes a value to it once a second
- `watch.py` prints every point change below a node
- `mirror_temperature.py` reacts to one point by writing another

Run them against a local instance:

```
SIOT_AUTH_TOKEN=... python examples/tree.py
```

## Development

The client lives in the
[Simple IoT repository](https://github.com/simpleiot/simpleiot) under `python/`,
so it changes together with the server.

```
cd python
uv sync
uv run pytest
```

The codec tests check the encoder against fixtures the Go tests write. The
integration tests start a scratch `siot` when `SIOT_BIN` names one;
`siot_test_python` in `envsetup.sh` builds it and runs them.
