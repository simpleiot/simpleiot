"""Prints every point change at or below a node, the root by default.

SIOT_AUTH_TOKEN=... python examples/watch/watch.py [node-id]
"""

import asyncio
import sys

import simpleiot


async def main():
    async with await simpleiot.connect() as c:
        id = sys.argv[1] if len(sys.argv) > 1 else (await c.get_root()).id
        print("watching", id)

        async with await c.subscribe_tree(id) as sub:
            async for m in sub:
                where = f"{m.node_id} (edge to {m.parent_id})" if m.is_edge else m.node_id
                for p in m.points:
                    print(f"{p.time:%H:%M:%S} {where} {p.type}.{p.key} = {p.value!r}")


try:
    asyncio.run(main())
except KeyboardInterrupt:
    pass
