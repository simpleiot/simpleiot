"""Codec tests against the fixtures data/point_fixture_test.go writes, so the
Python and Go encoders are checked against the same bytes."""

import json
from datetime import datetime, timezone
from pathlib import Path

import pytest

from simpleiot import (
    DataType,
    Node,
    NotFoundError,
    Point,
    SiotError,
    decode_nodes,
    decode_points,
    encode_points,
    new_id,
    valid_id,
)

FIXTURES = Path(__file__).resolve().parents[2] / "js" / "simpleiot-js" / "testdata"

pytestmark = pytest.mark.skipif(not FIXTURES.is_dir(), reason="fixtures are in the repository")


def fixture(name):
    return (FIXTURES / name).read_bytes()


def fixture_json(name):
    return json.loads(fixture(name))


def iso(t: datetime) -> str:
    return (
        t.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%S.") + f"{t.microsecond // 1000:03d}Z"
    )


def as_fixture(p: Point) -> dict:
    return {
        "type": p.type,
        "key": p.key,
        "time": iso(p.time),
        "dataType": int(p.data_type),
        "value": p.number,
        "text": p.text,
        "tombstone": p.tombstone,
        "origin": p.origin,
    }


def from_fixture(f: dict) -> Point:
    dt = DataType(f["dataType"])
    if dt in (DataType.STRING, DataType.JSON):
        value = f["text"]
    elif dt == DataType.UNKNOWN:
        value = None
    else:
        value = f["value"]
    return Point(
        type=f["type"],
        key=f["key"],
        value=value,
        time=datetime.fromisoformat(f["time"].replace("Z", "+00:00")),
        tombstone=f["tombstone"],
        origin=f["origin"],
        data_type=dt,
    )


def test_decodes_points_the_way_go_reads_them():
    got = [as_fixture(p) for p in decode_points(fixture("points.bin"))]
    assert got == fixture_json("points.json")


def test_encodes_points_to_the_bytes_go_wrote():
    pts = [from_fixture(f) for f in fixture_json("points.json")]
    assert encode_points(pts) == fixture("points.bin")


def test_decodes_a_node_reply():
    nodes = decode_nodes(fixture("nodes.bin"))
    got = [
        {
            "id": n.id,
            "type": n.type,
            "parent": n.parent,
            "points": [as_fixture(p) for p in n.points],
            "edgePoints": [as_fixture(p) for p in n.edge_points],
        }
        for n in nodes
    ]
    assert got == fixture_json("nodes.json")


def test_raises_the_error_in_a_node_reply():
    with pytest.raises(SiotError, match="not in scope"):
        decode_nodes(fixture("nodes-error.bin"))
    assert decode_nodes(b"") == []


def test_not_found_is_its_own_error():
    msg = b"document not found"
    frame = bytes([1]) + len(msg).to_bytes(2, "little") + msg
    with pytest.raises(NotFoundError):
        decode_nodes(frame)


@pytest.mark.parametrize(
    "value",
    [0, 1, -1, 127, -128, 128, -129, 32767, -32768, 40000, -40000, 2**31, -(2**31) - 1, 2**62],
)
def test_ints_round_trip_signed(value):
    [p] = decode_points(encode_points([Point("count", value, data_type=DataType.INT)]))
    assert p.data_type == DataType.INT
    assert p.value == value


def test_value_types():
    pts = decode_points(
        encode_points(
            [
                Point("value", 2.5),
                Point("value", 3),
                Point("enabled", True),
                Point("description", "pump"),
                Point("config", {"a": 1}),
                Point("empty"),
            ]
        )
    )
    assert [p.data_type for p in pts] == [
        DataType.FLOAT,
        DataType.FLOAT,
        DataType.FLOAT,
        DataType.STRING,
        DataType.JSON,
        DataType.UNKNOWN,
    ]
    assert [p.value for p in pts] == [2.5, 3.0, 1.0, "pump", '{"a":1}', None]
    assert pts[2].bool
    assert pts[4].text == '{"a":1}'


def test_reads_an_empty_key_as_0():
    [p] = decode_points(encode_points([Point("tombstone", 1, key="")]))
    assert p.key == "0"


def test_fills_in_a_missing_time():
    before = datetime.now(timezone.utc)
    [p] = decode_points(encode_points([Point("value", 1)]))
    assert before <= p.time <= datetime.now(timezone.utc)


def test_node_helpers():
    n = Node(
        "n1",
        "device",
        "root",
        points=[Point("description", "pump"), Point("value", 4.0, key="")],
        edge_points=[Point("tombstone", 1)],
    )
    assert n.description == "pump"
    assert n.value("value") == 4.0
    assert n.value("missing", default=7) == 7
    assert n.deleted


def test_ids():
    a = new_id()
    assert len(a) == 26 and valid_id(a) and a != new_id()
    assert a[-1] in "048cgmrw"
    assert not valid_id("") and not valid_id("-a") and not valid_id("a.b")
    assert not valid_id("a" * 65) and valid_id("a" * 64)
