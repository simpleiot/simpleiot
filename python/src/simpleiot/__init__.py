"""A Python client for Simple IoT.

Connect with the shared token, read and write nodes and points, and watch
for changes::

    import asyncio
    import simpleiot

    async def main():
        async with await simpleiot.connect() as c:
            root = await c.get_root()
            print(root.description)

    asyncio.run(main())
"""

from .client import Client, PointsMessage, Subscription, connect
from .codec import (
    DataType,
    Node,
    NotFoundError,
    Point,
    SiotError,
    decode_nodes,
    decode_points,
    encode_points,
)
from .ids import new_id, valid_id

__version__ = "0.1.0"

__all__ = [
    "Client",
    "DataType",
    "Node",
    "NotFoundError",
    "Point",
    "PointsMessage",
    "SiotError",
    "Subscription",
    "connect",
    "decode_nodes",
    "decode_points",
    "encode_points",
    "new_id",
    "valid_id",
]
