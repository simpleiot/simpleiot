# Python Client

The `simpleiot` package is a Python client for the [NATS API](api.md#nats). It
reads and writes nodes and points and subscribes to changes, connecting with the
instance's shared token (`SIOT_AUTH_TOKEN`).

```
pip install simpleiot
```

```python
import asyncio
import simpleiot

async def main():
    async with await simpleiot.connect() as c:
        root = await c.get_root()
        id = await c.create_node(root.id, "variable", description="tank level")
        await c.send_point(id, "value", 42.5)

asyncio.run(main())
```

`provision(name, yaml)` sets up nodes from a YAML file through
[configuration provisioning](../user/configuration.md#configuration-provisioning):
the server applies the file as `siot import` does, and the call waits for the
result.

`connect()` reads `SIOT_NATS_SERVER` and `SIOT_AUTH_TOKEN` from the environment
when they are not passed. The shared token reaches the whole tree, so the client
suits scripts and services that run alongside an instance you operate.

The source, API summary, and examples are in
[`python/`](https://github.com/simpleiot/simpleiot/tree/master/python) in the
repository.

## Development

The codec tests check the Python encoder against the fixtures
`data/point_fixture_test.go` writes, the same ones the JavaScript client is
checked against. `siot_test_python` builds `siot`, then runs the codec tests and
the integration tests against a scratch instance. It needs
[uv](https://docs.astral.sh/uv/).

To publish a release, set the version in `python/src/simpleiot/__init__.py`, add
a `python/CHANGELOG.md` entry, and push a `py-v<version>` tag. The
`Python release` workflow builds the package and publishes it to PyPI.
