"""A Simple IoT client over NATS.

The client connects with the shared token (``SIOT_AUTH_TOKEN``), which gives
it the full subject space described in the API reference:
https://docs.simpleiot.org/docs/ref/api.html
"""

from __future__ import annotations

import asyncio
import hashlib
import logging
import os
import time
from collections.abc import AsyncIterator, Iterable
from dataclasses import dataclass

import nats
from nats.aio.client import Client as NATS
from nats.aio.subscription import Subscription as NatsSubscription

from .codec import (
    Node,
    NotFoundError,
    Point,
    SiotError,
    decode_nodes,
    decode_points,
    encode_points,
)
from .ids import new_id, valid_id

__all__ = ["Client", "PointsMessage", "Subscription", "connect"]

log = logging.getLogger("simpleiot")

DEFAULT_SERVER = "nats://localhost:4222"

# The store answers in milliseconds once it is running, but can take
# seconds while it recovers streams at start-up. This matches the Go client.
DEFAULT_TIMEOUT = 20.0

_INVALID_TOKEN_CHARS = set(".*> \t\r\n")


async def connect(
    server: str | None = None,
    token: str | None = None,
    **options,
) -> Client:
    """Connects to a Simple IoT instance.

    ``server`` defaults to ``SIOT_NATS_SERVER``, then ``nats://localhost:4222``.
    ``token`` defaults to ``SIOT_AUTH_TOKEN``; an instance with no token set
    accepts a connection without one. Other options are passed to
    ``nats.connect``.

    The first connection attempt raises if the server cannot be reached or
    refuses the token. Once connected, the client reconnects without limit
    by default. Connection errors are logged to the ``simpleiot`` logger.
    """
    server = server or os.environ.get("SIOT_NATS_SERVER") or DEFAULT_SERVER
    if token is None:
        token = os.environ.get("SIOT_AUTH_TOKEN") or None
    allow_reconnect = options.pop("allow_reconnect", True)
    max_reconnect = options.pop("max_reconnect_attempts", -1)
    wait = options.pop("reconnect_time_wait", 2)
    options.setdefault("error_cb", _log_error)
    # nats-py retries the first connection under the reconnect settings, so
    # with no limit a server that is down would hang here. Try twice, then
    # apply the caller's settings once connected.
    nc = await nats.connect(
        servers=[server],
        token=token,
        allow_reconnect=False,
        max_reconnect_attempts=1,
        reconnect_time_wait=0.2,
        **options,
    )
    nc.options["allow_reconnect"] = allow_reconnect
    nc.options["max_reconnect_attempts"] = max_reconnect
    nc.options["reconnect_time_wait"] = wait
    return Client(nc)


async def _log_error(e: Exception) -> None:
    log.warning("nats: %s", e)


@dataclass
class PointsMessage:
    """Points received on a subscription.

    ``parent_id`` is set for edge points and empty for node points. For an
    ``up`` subject, ``upstream_id`` is the ancestor the points were fanned
    out to.
    """

    subject: str
    node_id: str
    parent_id: str
    type: str
    key: str
    points: list[Point]
    upstream_id: str = ""

    @property
    def is_edge(self) -> bool:
        return self.parent_id != ""


def parse_subject(subject: str) -> dict[str, str]:
    """Splits a point subject into its parts.

    ::

        p.<node>.<type>.<key>
        ep.<node>.<parent>.<type>.<key>
        up.<upstream>.<node>.<type>.<key>
        up.<upstream>.<node>.<parent>.<type>.<key>
    """
    t = subject.split(".")
    ret = {"upstream_id": "", "node_id": "", "parent_id": "", "type": "", "key": ""}
    if t[0] == "up" and len(t) == 6:
        ret.update(upstream_id=t[1], node_id=t[2], parent_id=t[3], type=t[4], key=t[5])
    elif t[0] == "up" and len(t) == 5:
        ret.update(upstream_id=t[1], node_id=t[2], type=t[3], key=t[4])
    elif t[0] == "ep" and len(t) >= 3:
        ret.update(node_id=t[1], parent_id=t[2])
        if len(t) == 5:
            ret.update(type=t[3], key=t[4])
    elif t[0] == "p" and len(t) == 4:
        ret.update(node_id=t[1], type=t[2], key=t[3])
    return ret


class Subscription:
    """Decoded point messages from a NATS subscription.

    Iterate with ``async for msg in sub`` and stop with ``await
    sub.unsubscribe()``, or use it as an async context manager.
    """

    def __init__(self, sub: NatsSubscription) -> None:
        self._sub = sub

    @property
    def subject(self) -> str:
        return self._sub.subject

    async def __aiter__(self) -> AsyncIterator[PointsMessage]:
        async for m in self._sub.messages:
            try:
                points = decode_points(m.data)
            except ValueError:
                continue
            yield PointsMessage(subject=m.subject, points=points, **parse_subject(m.subject))

    async def unsubscribe(self) -> None:
        await self._sub.unsubscribe()

    async def __aenter__(self) -> Subscription:
        return self

    async def __aexit__(self, *exc) -> None:
        await self.unsubscribe()


def _check_token(what: str, s: str) -> None:
    if not s or any(c in _INVALID_TOKEN_CHARS for c in s):
        raise ValueError(f"{what} {s!r} may not be empty or contain '.', '*', '>', or whitespace")


def _check_reply(data: bytes) -> None:
    if data:
        raise SiotError(data.decode(errors="replace"))


class Client:
    """A connection to a Simple IoT instance. Create one with :func:`connect`."""

    def __init__(self, nc: NATS, timeout: float = DEFAULT_TIMEOUT) -> None:
        self.nc = nc
        self.timeout = timeout

    async def close(self) -> None:
        """Flushes pending messages and closes the connection."""
        await self.nc.drain()

    async def __aenter__(self) -> Client:
        return self

    async def __aexit__(self, *exc) -> None:
        await self.close()

    # --- reading nodes ---

    async def get_nodes(
        self,
        parent: str,
        id: str = "all",
        *,
        type: str = "",
        include_deleted: bool = False,
        depth: int = 0,
    ) -> list[Node]:
        """Fetches nodes, as the ``nodes.<parent>.<id>`` request.

        - ``parent="all"`` returns every instance of node ``id``, one per edge.
        - ``id="all"`` returns the children of ``parent``.
        - ``parent="root", id="all"`` returns the root node.

        ``type`` keeps only nodes of that type, ``include_deleted`` includes
        deleted ones, and ``depth`` also returns descendants that many levels
        down, in one flat list placed by each node's ``parent``.
        """
        params: list[Point] = []
        if include_deleted:
            params.append(Point("tombstone", 1))
        if type:
            params.append(Point("nodeType", type))
        if depth > 0:
            params.append(Point("depth", depth))
        m = await self.nc.request(
            f"nodes.{parent or 'all'}.{id or 'all'}", encode_points(params), timeout=self.timeout
        )
        try:
            return decode_nodes(m.data)
        except NotFoundError:
            return []

    async def get_node(self, id: str, parent: str = "all") -> Node:
        """Returns one node. Raises :class:`NotFoundError` when it does not exist.

        A node that sits in several places in the tree is returned under the
        first of its parents unless ``parent`` names one.
        """
        nodes = await self.get_nodes(parent, id)
        if not nodes:
            raise NotFoundError(f"node {id} not found")
        return nodes[0]

    async def get_children(
        self, parent: str, *, type: str = "", include_deleted: bool = False
    ) -> list[Node]:
        """Returns the children of a node."""
        return await self.get_nodes(parent, "all", type=type, include_deleted=include_deleted)

    async def get_root(self) -> Node:
        """Returns the root node of the instance."""
        nodes = await self.get_nodes("root", "all")
        if not nodes:
            raise NotFoundError("root node not found")
        return nodes[0]

    async def search(self, below: str, query: str, *, limit: int = 0, type: str = "") -> list[Node]:
        """Finds nodes below ``below`` whose description, ID, or type matches.

        The reply carries each match and its ancestors below ``below``.
        ``limit`` caps the matches (20 by default, 100 at most).
        """
        params = [Point("search", query)]
        if limit > 0:
            params.append(Point("limit", limit))
        if type:
            params.append(Point("nodeType", type))
        m = await self.nc.request(f"nodes.{below}.all", encode_points(params), timeout=self.timeout)
        return decode_nodes(m.data)

    # --- writing points ---

    async def send_points(self, node_id: str, points: Iterable[Point], *, ack: bool = True) -> None:
        """Writes points to a node.

        Each point is its own message on ``p.<node>.<type>.<key>``. With
        ``ack`` the call waits for the store to accept each point and raises
        :class:`SiotError` if it refuses one.
        """
        _check_token("node ID", node_id)
        for p in points:
            _check_token("point type", p.type)
            _check_token("point key", p.key or "0")
            subject = f"p.{node_id}.{p.type}.{p.key or '0'}"
            await self._send(subject, [p], ack)

    async def send_point(self, node_id: str, type: str, value, key: str = "0", **kw) -> None:
        """Writes a single point to a node. Extra arguments go to :class:`Point`."""
        await self.send_points(node_id, [Point(type, value, key=key, **kw)])

    async def send_edge_points(
        self, node_id: str, parent: str, points: Iterable[Point], *, ack: bool = True
    ) -> None:
        """Writes points to the edge between a node and its parent, in one message."""
        _check_token("node ID", node_id)
        _check_token("parent ID", parent)
        await self._send(f"ep.{node_id}.{parent}", list(points), ack)

    async def _send(self, subject: str, points: list[Point], ack: bool) -> None:
        data = encode_points(points)
        if ack:
            m = await self.nc.request(subject, data, timeout=self.timeout)
            _check_reply(m.data)
        else:
            await self.nc.publish(subject, data)

    # --- creating and removing nodes ---

    async def create_node(
        self,
        parent: str,
        type: str,
        points: Iterable[Point] = (),
        *,
        id: str | None = None,
        description: str | None = None,
        primary: bool = False,
    ) -> str:
        """Creates a node under ``parent`` and returns its ID.

        ``id`` defaults to a new random ID. ``description`` is a shortcut for
        a description point. Set ``primary`` for a node type that owns
        hardware or a connection (Modbus, serial, sync, and the like), so its
        client runs on this edge only; see "Primary and mirror edges" in the
        data reference.

        Sending to an existing ID and parent updates that node.
        """
        id = id or new_id()
        if not valid_id(id):
            raise ValueError(f"invalid node ID {id!r}")
        pts = list(points)
        if description is not None:
            pts.append(Point("description", description))
        await self.send_points(id, pts)
        edge = [Point("tombstone", 0), Point("nodeType", type)]
        if primary:
            edge.append(Point("primary", 1))
        await self.send_edge_points(id, parent, edge)
        return id

    async def delete_node(self, id: str, parent: str) -> None:
        """Removes a node from ``parent``.

        Deleting a node's primary edge also deletes its mirrors, as the web
        UI does. The node can be restored by setting the edge tombstone back
        to 0.
        """
        nodes = await self.get_nodes("all", id)
        tomb = [Point("tombstone", 1)]
        here = next((n for n in nodes if n.parent == parent), None)
        if here is not None and _role_set(here, "primary"):
            for n in nodes:
                if n.parent != parent and _role_set(n, "mirror"):
                    await self.send_edge_points(id, n.parent, tomb)
        await self.send_edge_points(id, parent, tomb)

    async def move_node(self, id: str, old_parent: str, new_parent: str) -> None:
        """Moves a node from one parent to another, keeping its edge role."""
        nodes = await self.get_nodes(old_parent, id)
        if not nodes:
            raise NotFoundError(f"node {id} not found under {old_parent}")
        n = nodes[0]
        edge = [Point("tombstone", 0), Point("nodeType", n.type)]
        for role in ("primary", "mirror"):
            if _role_set(n, role):
                edge.append(Point(role, 1))
        await self.send_edge_points(id, new_parent, edge)
        await self.send_edge_points(id, old_parent, [Point("tombstone", 1)])

    # --- provisioning ---

    async def provision(self, name: str, contents: str, *, timeout: float = 30.0) -> str:
        """Uploads a YAML node file for the server to apply, and waits until it has.

        The file is a ``file`` node called ``name`` under the instance's
        ``provisioning`` node, which is created if there is none. The server
        applies it the way ``siot import`` does: nodes are matched by
        description, missing ones are created, and changed points are
        updated. It applies the file again only when its contents change.

        Returns the file node's ID. Raises :class:`SiotError` with the
        server's message if the file fails to apply, and ``TimeoutError`` if
        it is not applied within ``timeout`` seconds. See "Configuration
        provisioning" in the configuration guide.
        """
        want = hashlib.sha256(contents.encode()).hexdigest()
        root = await self.get_root()
        provs = await self.get_children(root.id, type="provisioning")
        if provs:
            prov = provs[0].id
        else:
            prov = await self.create_node(
                root.id, "provisioning", description="Provisioning", primary=True
            )

        file = next(
            (
                f
                for f in await self.get_children(prov, type="file")
                if name in (f.value("name"), f.description)
            ),
            None,
        )
        if file is None:
            id = await self.create_node(
                prov,
                "file",
                [Point("name", name), Point("created", int(time.time())), Point("data", contents)],
                description=name,
            )
        else:
            id = file.id
            if file.value("data") != contents:
                await self.send_point(id, "data", contents)

        # the server records the hash of what it tried, with the error, after
        # each attempt
        deadline = time.monotonic() + timeout
        while True:
            n = await self.get_node(id, prov)
            if n.value("provisionHash") == want:
                err = n.value("error", default="")
                if err:
                    raise SiotError(f"provisioning {name}: {err}")
                return id
            if time.monotonic() > deadline:
                raise TimeoutError(f"provisioning {name} was not applied within {timeout} s")
            await asyncio.sleep(0.2)

    # --- subscriptions ---

    async def subscribe(self, subject: str) -> Subscription:
        """Subscribes to any point subject and returns decoded messages."""
        return Subscription(await self.nc.subscribe(subject))

    async def subscribe_node(self, id: str) -> Subscription:
        """Subscribes to the node points of one node, after the store accepts them."""
        _check_token("node ID", id)
        return await self.subscribe(f"up.{id}.{id}.*.*")

    async def subscribe_tree(self, id: str) -> Subscription:
        """Subscribes to every node and edge point at or below a node."""
        _check_token("node ID", id)
        return await self.subscribe(f"up.{id}.>")


def _role_set(n: Node, role: str) -> bool:
    p = n.edge_point(role)
    return p is not None and p.bool
