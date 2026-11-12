"""Prints the node tree of a Simple IoT instance.

SIOT_AUTH_TOKEN=... python examples/tree.py
"""

import asyncio
from collections import defaultdict

import simpleiot


async def main():
    async with await simpleiot.connect() as c:
        root = await c.get_root()
        # one request returns the tree several levels down, placed by parent
        nodes = await c.get_nodes(root.id, "all", depth=10)

        children = defaultdict(list)
        for n in nodes:
            children[n.parent].append(n)

        def show(n, indent=0):
            print(f"{'  ' * indent}{n.type}: {n.description or '(no description)'}  [{n.id}]")
            for child in sorted(children[n.id], key=lambda x: (x.type, x.description)):
                show(child, indent + 1)

        show(root)


asyncio.run(main())
