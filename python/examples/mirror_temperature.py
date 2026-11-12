"""Watches a node's value and writes it to another node in Fahrenheit.

This is the shape of a small control script: subscribe to one point, compute,
and write another.

    SIOT_AUTH_TOKEN=... python examples/mirror_temperature.py <celsius-node-id>
"""

import asyncio
import sys

import simpleiot


async def main(source: str):
    async with await simpleiot.connect() as c:
        src = await c.get_node(source)
        target = await c.create_node(
            src.parent, "variable", description=f"{src.description or source} (°F)"
        )
        print("writing Fahrenheit to", target)

        async with await c.subscribe_node(source) as sub:
            async for m in sub:
                for p in m.points:
                    if p.type == "value":
                        f = p.number * 9 / 5 + 32
                        await c.send_point(target, "value", round(f, 2))
                        print(f"{p.number} °C -> {f:.2f} °F")


if len(sys.argv) != 2:
    sys.exit(__doc__)

try:
    asyncio.run(main(sys.argv[1]))
except KeyboardInterrupt:
    pass
