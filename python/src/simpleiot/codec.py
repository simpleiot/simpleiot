"""Points, nodes, and their binary wire encoding.

This mirrors ``data/point.go`` and ``data/node.go`` in the Go code. Integers
are little endian, strings and byte fields carry a two-byte length, time is
int64 nanoseconds since the epoch, and a node reply starts with a version
byte and an error string.

The tests check this module against the fixtures ``data/point_fixture_test.go``
writes, so the Python and Go encoders cannot drift without a test failing.
"""

from __future__ import annotations

import json
import struct
from dataclasses import dataclass, field
from datetime import datetime, timedelta, timezone
from enum import IntEnum
from typing import Any

__all__ = [
    "DataType",
    "Node",
    "Point",
    "SiotError",
    "NotFoundError",
    "decode_nodes",
    "decode_points",
    "encode_points",
]

NODE_FRAME_VERSION = 1
MAX_POINTS_PER_MESSAGE = 10000
MAX_NODES_PER_FRAME = 10000

# the error string the server sends when a node does not exist
# (data.ErrDocumentNotFound)
_NOT_FOUND = "document not found"

_EPOCH = datetime(1970, 1, 1, tzinfo=timezone.utc)


class SiotError(Exception):
    """An error reported by the Simple IoT server."""


class NotFoundError(SiotError):
    """The requested node does not exist."""


class DataType(IntEnum):
    """How a point's value is encoded, as ``data.PointDataType``."""

    UNKNOWN = 0
    FLOAT = 1
    INT = 2
    STRING = 3
    JSON = 4


Value = float | int | str | bool | dict | list | bytes | None


@dataclass
class Point:
    """A single value on a node or an edge.

    ``value`` sets both the data and its type when sending: a ``str`` is a
    string point, a ``dict`` or ``list`` is a JSON point, and a number or
    ``bool`` is a float point. Pass ``data_type=DataType.INT`` to send an
    integer as an int point, or ``DataType.JSON`` to send a string that
    already holds JSON. ``None`` sends a point with no value.

    A missing ``time`` is filled in with the current time when the point is
    sent. ``key`` separates points of the same type, such as the channels of
    an input; most points use ``"0"``.
    """

    type: str
    value: Value = None
    key: str = "0"
    time: datetime | None = None
    tombstone: int = 0
    origin: str = ""
    data_type: DataType | None = None

    @property
    def number(self) -> float:
        """The numeric value, or 0 for a point that is not a number.

        This matches ``Point.Val()`` in Go. A ``bool`` value reads as 1 or 0.
        """
        v = self.value
        if isinstance(v, bool):
            return 1.0 if v else 0.0
        if isinstance(v, (int, float)):
            return float(v)
        return 0.0

    @property
    def text(self) -> str:
        """The string value, or ``""`` for a point that is not a string.

        This matches ``Point.Txt()`` in Go. A JSON point returns its JSON text.
        """
        dt, data = _point_data(self)
        if dt in (DataType.STRING, DataType.JSON):
            return data.decode()
        return ""

    @property
    def bool(self) -> bool:
        """True when the numeric value is 1, as ``Point.Bool()`` in Go."""
        return self.number == 1

    @property
    def deleted(self) -> bool:
        """True when the point has been deleted (its tombstone is set)."""
        return self.tombstone != 0


@dataclass
class Node:
    """A node and the edge points that place it under ``parent``."""

    id: str
    type: str
    parent: str = ""
    points: list[Point] = field(default_factory=list)
    edge_points: list[Point] = field(default_factory=list)

    def point(self, type: str, key: str = "0") -> Point | None:
        """Returns the node point with this type and key, or None."""
        return _find(self.points, type, key)

    def edge_point(self, type: str, key: str = "0") -> Point | None:
        """Returns the edge point with this type and key, or None."""
        return _find(self.edge_points, type, key)

    def value(self, type: str, key: str = "0", default: Any = None) -> Any:
        """Returns the value of a node point, or ``default`` when it is missing."""
        p = self.point(type, key)
        if p is None or p.deleted:
            return default
        return p.value

    @property
    def description(self) -> str:
        """The node's description, or ``""``."""
        p = self.point("description")
        return p.text if p else ""

    @property
    def deleted(self) -> bool:
        """True when the edge to ``parent`` has been deleted."""
        p = self.edge_point("tombstone")
        return p is not None and p.bool


def _norm_key(key: str) -> str:
    # the store keeps an empty key as "0"
    return key or "0"


def _find(points: list[Point], type: str, key: str) -> Point | None:
    key = _norm_key(key)
    for p in points:
        if p.type == type and _norm_key(p.key) == key:
            return p
    return None


def _int_bytes(v: int) -> bytes:
    # the smallest width that holds the value, as data.Point.PutInt
    if -(1 << 7) <= v < (1 << 7):
        return struct.pack("<b", v)
    if -(1 << 15) <= v < (1 << 15):
        return struct.pack("<h", v)
    if -(1 << 31) <= v < (1 << 31):
        return struct.pack("<i", v)
    return struct.pack("<q", v)


def _point_data(p: Point) -> tuple[DataType, bytes]:
    v = p.value
    dt = p.data_type
    if v is None:
        return DataType.UNKNOWN, b""
    if isinstance(v, bytes):
        return (dt if dt is not None else DataType.UNKNOWN), v
    if isinstance(v, str):
        return (DataType.JSON if dt == DataType.JSON else DataType.STRING), v.encode()
    if isinstance(v, (dict, list)):
        return DataType.JSON, json.dumps(v, separators=(",", ":")).encode()
    if isinstance(v, bool):
        v = 1 if v else 0
    if isinstance(v, (int, float)):
        if dt == DataType.INT:
            if isinstance(v, float) and not v.is_integer():
                raise ValueError(f"point {p.type}: {v} is not an integer")
            return DataType.INT, _int_bytes(int(v))
        return DataType.FLOAT, struct.pack("<d", float(v))
    raise TypeError(f"point {p.type}: unsupported value type {type(v).__name__}")


def _to_nanos(t: datetime | None) -> int:
    if t is None:
        t = datetime.now(timezone.utc)
    elif t.tzinfo is None:
        t = t.astimezone()
    d = t - _EPOCH
    return (d.days * 86400 + d.seconds) * 1_000_000_000 + d.microseconds * 1000


def _from_nanos(ns: int) -> datetime:
    return _EPOCH + timedelta(microseconds=ns // 1000)


def _decode_value(dt: int, data: bytes) -> Value:
    if dt == DataType.FLOAT:
        if len(data) == 8:
            return struct.unpack("<d", data)[0]
        if len(data) == 4:
            return struct.unpack("<f", data)[0]
        return 0.0
    if dt == DataType.INT:
        fmt = {1: "<b", 2: "<h", 4: "<i", 8: "<q"}.get(len(data))
        return struct.unpack(fmt, data)[0] if fmt else 0
    if dt in (DataType.STRING, DataType.JSON):
        return data.decode()
    if not data:
        return None
    return data


class _Writer:
    def __init__(self) -> None:
        self.buf = bytearray()

    def u8(self, v: int) -> None:
        self.buf.append(v & 0xFF)

    def u32(self, v: int) -> None:
        self.buf += struct.pack("<I", v & 0xFFFFFFFF)

    def i64(self, v: int) -> None:
        self.buf += struct.pack("<q", v)

    def bytes(self, b: bytes) -> None:
        if len(b) > 0xFFFF:
            raise ValueError(f"field of {len(b)} bytes is too long to encode")
        self.buf += struct.pack("<H", len(b))
        self.buf += b

    def string(self, s: str) -> None:
        self.bytes(s.encode())


class _Reader:
    def __init__(self, data: bytes) -> None:
        self.data = memoryview(data)
        self.off = 0

    def _take(self, n: int, what: str) -> memoryview:
        if self.off + n > len(self.data):
            raise ValueError(f"decode: not enough data for {what} at offset {self.off}")
        b = self.data[self.off : self.off + n]
        self.off += n
        return b

    def u8(self) -> int:
        return self._take(1, "byte")[0]

    def u32(self) -> int:
        return struct.unpack("<I", self._take(4, "count"))[0]

    def i32(self) -> int:
        return struct.unpack("<i", self._take(4, "int32"))[0]

    def i64(self) -> int:
        return struct.unpack("<q", self._take(8, "time"))[0]

    def bytes(self) -> bytes:
        n = struct.unpack("<H", self._take(2, "length"))[0]
        return bytes(self._take(n, "bytes"))

    def string(self) -> str:
        return self.bytes().decode()


def _encode_points_to(w: _Writer, points: list[Point]) -> None:
    w.u32(len(points))
    for p in points:
        dt, data = _point_data(p)
        w.string(p.type)
        w.string(p.key)
        w.i64(_to_nanos(p.time))
        w.u8(dt)
        w.bytes(data)
        w.u32(p.tombstone)
        w.string(p.origin)


def _decode_points_from(r: _Reader) -> list[Point]:
    count = r.u32()
    if count > MAX_POINTS_PER_MESSAGE:
        raise ValueError(f"decode: point count {count} exceeds maximum")
    points = []
    for _ in range(count):
        typ = r.string()
        key = r.string() or "0"
        t = _from_nanos(r.i64())
        dt = r.u8()
        data = r.bytes()
        tombstone = r.i32()
        origin = r.string()
        points.append(
            Point(
                type=typ,
                key=key,
                time=t,
                value=_decode_value(dt, data),
                data_type=DataType(dt) if dt in DataType._value2member_map_ else None,
                tombstone=tombstone,
                origin=origin,
            )
        )
    return points


def encode_points(points: list[Point]) -> bytes:
    """Returns the wire encoding of a list of points."""
    w = _Writer()
    _encode_points_to(w, points)
    return bytes(w.buf)


def decode_points(data: bytes) -> list[Point]:
    """Returns the points in a wire payload."""
    return _decode_points_from(_Reader(data))


def decode_nodes(data: bytes) -> list[Node]:
    """Returns the nodes in a node reply.

    An error the server put in the reply is raised as :class:`SiotError`, or
    :class:`NotFoundError` when the node does not exist. An empty payload is
    no nodes.
    """
    if not data:
        return []
    r = _Reader(data)
    version = r.u8()
    if version != NODE_FRAME_VERSION:
        raise ValueError(f"decode: unsupported node frame version {version}")
    err = r.string()
    if err == _NOT_FOUND:
        raise NotFoundError(err)
    if err:
        raise SiotError(err)
    count = r.u32()
    if count > MAX_NODES_PER_FRAME:
        raise ValueError(f"decode: node count {count} exceeds maximum")
    nodes = []
    for _ in range(count):
        nid = r.string()
        typ = r.string()
        parent = r.string()
        points = _decode_points_from(r)
        edge_points = _decode_points_from(r)
        nodes.append(Node(nid, typ, parent, points, edge_points))
    return nodes
