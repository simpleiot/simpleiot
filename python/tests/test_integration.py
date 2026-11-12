"""Tests against a running siot. They are skipped unless SIOT_BIN names a siot
binary, which is started with a fresh data directory on scratch ports.
`siot_test_python` in envsetup.sh builds one and sets it."""

import asyncio
import os
import socket
import subprocess
import tempfile
import time

import nats.errors
import pytest

import simpleiot
from simpleiot import NotFoundError, Point, SiotError

SIOT_BIN = os.environ.get("SIOT_BIN")
TOKEN = "python-test-token"

pytestmark = pytest.mark.skipif(not SIOT_BIN, reason="set SIOT_BIN to run against siot")


def free_port():
    # siot takes the NATS port and the two above it (HTTP and monitoring)
    for port in range(15300, 16000, 10):
        try:
            for p in (port, port + 1, port + 2):
                with socket.socket() as s:
                    s.bind(("127.0.0.1", p))
            return port
        except OSError:
            continue
    raise RuntimeError("no free ports for siot")


@pytest.fixture(scope="session")
def server():
    with tempfile.TemporaryDirectory() as d:
        port = free_port()
        env = {k: v for k, v in os.environ.items() if not k.startswith("SIOT_")}
        env.update(
            SIOT_DATA=os.path.join(d, "data"),
            SIOT_NATS_PORT=str(port),
            SIOT_AUTH_TOKEN=TOKEN,
        )
        log = open(os.path.join(d, "siot.log"), "w")
        proc = subprocess.Popen([SIOT_BIN, "serve"], env=env, stdout=log, stderr=log)
        url = f"nats://127.0.0.1:{port}"
        deadline = time.time() + 30
        while True:
            if proc.poll() is not None or time.time() > deadline:
                log.close()
                pytest.fail("siot did not start:\n" + open(log.name).read()[-2000:])
            try:
                socket.create_connection(("127.0.0.1", port), 0.2).close()
                break
            except OSError:
                time.sleep(0.2)
        try:
            yield url
        finally:
            proc.terminate()
            proc.wait(10)
            log.close()


@pytest.fixture
async def client(server):
    c = await simpleiot.connect(server, TOKEN)
    # the store may still be starting
    for _ in range(100):
        try:
            await c.get_root()
            break
        except Exception:
            await asyncio.sleep(0.1)
    yield c
    await c.close()


async def test_refuses_a_wrong_token(server):
    with pytest.raises(nats.errors.Error, match="Authorization"):
        await simpleiot.connect(server, "wrong")


async def test_fails_when_no_server():
    with pytest.raises(nats.errors.NoServersError):
        await asyncio.wait_for(simpleiot.connect("nats://127.0.0.1:1", TOKEN), 5)


async def test_create_read_update_delete(client):
    root = await client.get_root()
    assert root.type == "device"

    id = await client.create_node(root.id, "variable", [Point("value", 1.5)], description="tank")
    n = await client.get_node(id, root.id)
    assert n.type == "variable"
    assert n.description == "tank"
    assert n.value("value") == 1.5

    await client.send_point(id, "value", 2.5)
    assert (await client.get_node(id)).value("value") == 2.5

    children = await client.get_children(root.id, type="variable")
    assert id in [c.id for c in children]

    found = await client.search(root.id, "tank")
    assert id in [c.id for c in found]

    await client.delete_node(id, root.id)
    assert id not in [c.id for c in await client.get_children(root.id)]
    deleted = await client.get_nodes(root.id, id, include_deleted=True)
    assert deleted and deleted[0].deleted


async def test_missing_node(client):
    with pytest.raises(NotFoundError):
        await client.get_node(simpleiot.new_id())


async def test_store_refuses_a_bad_point(client):
    root = await client.get_root()
    with pytest.raises((SiotError, ValueError)):
        await client.send_points(root.id, [Point("bad.type", 1)])


async def test_subscribe(client):
    root = await client.get_root()
    id = await client.create_node(root.id, "variable", description="watched")

    async with await client.subscribe_tree(root.id) as sub:
        await client.send_point(id, "value", 42)
        async with asyncio.timeout(5):
            async for m in sub:
                if m.node_id == id and m.type == "value":
                    assert m.points[0].value == 42
                    assert m.upstream_id == root.id
                    assert not m.is_edge
                    break

    async with await client.subscribe_node(id) as sub:
        await client.send_point(id, "value", 43)
        async with asyncio.timeout(5):
            async for m in sub:
                assert m.node_id == id
                assert m.points[0].value == 43
                break

    await client.delete_node(id, root.id)


async def test_provision(client):
    root = await client.get_root()

    def names():
        return client.get_children(root.id, type="group")

    yaml = "nodes:\n  - group:\n      description: Provisioned\n"
    id = await client.provision("test.yaml", yaml)
    assert "Provisioned" in [g.description for g in await names()]

    # unchanged contents are already applied, so this returns at once
    assert await client.provision("test.yaml", yaml) == id

    # changed contents are applied again, to the same file node
    yaml2 = yaml + (
        "      children:\n        - variable:\n            description: Provisioned value\n"
    )
    assert await client.provision("test.yaml", yaml2) == id
    g = [g for g in await names() if g.description == "Provisioned"][0]
    assert "Provisioned value" in [n.description for n in await client.get_children(g.id)]

    with pytest.raises(SiotError, match="bad.yaml"):
        await client.provision("bad.yaml", "nodes: [not, a, node", timeout=10)
