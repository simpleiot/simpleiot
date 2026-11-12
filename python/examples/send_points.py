"""Creates a variable node under the root and writes a value to it once a
second, the way a script reading a sensor would.

    SIOT_AUTH_TOKEN=... python examples/send_points.py
"""

import asyncio
import math
import time

import simpleiot
from simpleiot import Point


async def main():
    async with await simpleiot.connect() as c:
        root = await c.get_root()

        # reuse the node from an earlier run if there is one
        existing = [
            n
            for n in await c.get_children(root.id, type="variable")
            if n.description == "python example"
        ]
        if existing:
            id = existing[0].id
        else:
            id = await c.create_node(
                root.id, "variable", [Point("units", "°C")], description="python example"
            )
        print("writing to node", id)

        while True:
            value = round(20 + 5 * math.sin(time.time() / 10), 2)
            await c.send_point(id, "value", value)
            print("value", value)
            await asyncio.sleep(1)


try:
    asyncio.run(main())
except KeyboardInterrupt:
    pass
