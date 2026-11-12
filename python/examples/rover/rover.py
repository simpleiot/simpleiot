"""An autonomous rover that reports to a dashboard and takes commands from it.

    SIOT_AUTH_TOKEN=... python examples/rover/rover.py

At start it uploads rover.yaml as a provisioning file, which creates the Rover
group and its dashboard, or brings them up to date. See README.md for what the
dashboard shows.
"""

import asyncio
import sys
from pathlib import Path

from sim import FIELD_H, FIELD_W, HOME, Commands, Rover

import simpleiot

SIM_PERIOD = 0.1  # s between simulation steps
REPORT_PERIOD = 0.5  # s between status reports

# command variables, by description, and the Commands field each one sets
COMMANDS = {
    "Mode": "mode",
    "Front drive": "front",
    "Rear drive": "rear",
    "Speed limit": "speed_limit",
    "Steering command": "steering",
    "E-stop": "estop",
}

# status variables, by description, and how to read each from the rover
STATUS = {
    "X": lambda r: round(r.x, 2),
    "Y": lambda r: round(r.y, 2),
    "Heading": lambda r: round(r.heading, 1),
    "Speed": lambda r: round(r.speed, 2),
    "Steering": lambda r: round(r.steering, 1),
    "Battery": lambda r: round(r.battery, 1),
    "Odometer": lambda r: round(r.odometer),
    "Waypoint": lambda r: r.waypoint + 1,
    "State": lambda r: r.state,
}


async def find_nodes(c: simpleiot.Client) -> dict[str, str]:
    """Returns the IDs of the rover's nodes, by description."""
    root = await c.get_root()
    groups = [g for g in await c.get_children(root.id, type="group") if g.description == "Rover"]
    if not groups:
        sys.exit("no Rover group under the root after provisioning rover.yaml")
    group_id = groups[0].id
    ids = {n.description: n.id for n in await c.get_children(group_id)}
    missing = [d for d in [*COMMANDS, *STATUS, "Map drawing"] if d not in ids]
    if missing:
        sys.exit(f"the Rover group is missing {', '.join(missing)}")
    return ids | {"group": group_id}


def apply_command(cmd: Commands, field: str, p: simpleiot.Point) -> None:
    if field == "mode":
        cmd.mode = p.text.strip().lower()
    elif field in ("front", "rear", "estop"):
        setattr(cmd, field, p.bool)
    else:
        setattr(cmd, field, p.number)


async def read_commands(c: simpleiot.Client, ids: dict[str, str], cmd: Commands) -> None:
    """Loads the current commands, then follows changes from the dashboard.

    A command that has never been set gets the default from Commands, so the
    dashboard shows what the rover is using.
    """
    fields = {ids[d]: f for d, f in COMMANDS.items()}
    # subscribe first, so a change made while loading is not missed
    async with await c.subscribe_tree(ids["group"]) as sub:
        for id, f in fields.items():
            p = (await c.get_node(id)).point("value")
            if p and p.value is not None:
                apply_command(cmd, f, p)
            else:
                await c.send_point(id, "value", getattr(cmd, f))
        print("commands:", cmd)
        async for m in sub:
            f = fields.get(m.node_id)
            if f and not m.is_edge and m.type == "value":
                apply_command(cmd, f, m.points[-1])
                print("command:", f, "=", getattr(cmd, f))


async def simulate(rover: Rover, cmd: Commands) -> None:
    while True:
        rover.step(cmd, SIM_PERIOD)
        await asyncio.sleep(SIM_PERIOD)


async def report(c: simpleiot.Client, ids: dict[str, str], rover: Rover) -> None:
    """Writes the rover's status, sending only values that changed."""
    last: dict[str, object] = {}
    while True:
        for desc, read in STATUS.items():
            v = read(rover)
            if last.get(desc) != v:
                await c.send_point(ids[desc], "value", v)
                last[desc] = v
        await asyncio.sleep(REPORT_PERIOD)


def map_svg(mission: list[tuple[float, float]]) -> str:
    """Draws the field, home, and mission, with a marker that follows the rover.

    The field is 10 drawing units to the meter. The marker is drawn at the
    field's origin and moved by the x and y bindings (data-siot-translate),
    and turned by heading; the drawing's CSS eases it between reports.
    """
    s, pad = 10, 20
    w, h = FIELD_W * s, FIELD_H * s

    def at(p):
        return pad + p[0] * s, pad + h - p[1] * s

    grid = ""
    for i in range(10, int(FIELD_W), 10):
        x, _ = at((i, 0))
        grid += f'<line class="rv-grid" x1="{x}" y1="{pad}" x2="{x}" y2="{pad + h}"/>'
        grid += f'<text class="rv-axis" x="{x}" y="{pad + h + 14}">{i} m</text>'
    for j in range(10, int(FIELD_H), 10):
        _, y = at((0, j))
        grid += f'<line class="rv-grid" x1="{pad}" y1="{y}" x2="{pad + w}" y2="{y}"/>'
    route = " ".join(f"{x:g},{y:g}" for x, y in map(at, [*mission, mission[0]]))
    points = "".join(
        f'<circle class="rv-wp" cx="{x:g}" cy="{y:g}" r="9"/>'
        f'<text class="rv-wpn" x="{x:g}" y="{y + 4:g}">{i}</text>'
        for i, (x, y) in enumerate(map(at, mission), 1)
    )
    hx, hy = at(HOME)
    size = f"{w + 2 * pad:g} {h + 2 * pad:g}"
    return f"""<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {size}">
  <style>
    .rv-field {{ fill: #d9e8c8; stroke: currentColor; stroke-width: 2; }}
    .rv-grid {{ stroke: #9ab48a; stroke-width: 1; }}
    .rv-axis {{ font: 10px sans-serif; fill: currentColor; text-anchor: middle; opacity: 0.7; }}
    .rv-route {{ fill: none; stroke: #2563eb; stroke-width: 2; stroke-dasharray: 6 4; }}
    .rv-wp {{ fill: #fff; stroke: #2563eb; stroke-width: 2; }}
    .rv-wpn {{ font: 600 11px sans-serif; fill: #1e3a8a; text-anchor: middle; }}
    .rv-home {{ fill: #fde68a; stroke: #b45309; stroke-width: 2; }}
    .rv-homet {{ font: 600 11px sans-serif; fill: #78350f; text-anchor: middle; }}
    .rv-move {{ transition: transform 0.5s linear; }}
    .rv-marker {{ fill: #dc2626; stroke: #fff; stroke-width: 2; }}
    .rv-marker[data-siot-state="dead"], .rv-marker[data-siot-state="stale"] {{ opacity: 0.3; }}
  </style>
  <rect class="rv-field" x="{pad}" y="{pad}" width="{w:g}" height="{h:g}"/>
  {grid}
  <rect class="rv-home" x="{hx - 16:g}" y="{hy - 16:g}" width="32" height="32" rx="4"/>
  <text class="rv-homet" x="{hx:g}" y="{hy + 30:g}">HOME</text>
  <polyline class="rv-route" points="{route}"/>
  {points}
  <g transform="translate({pad} {pad + h:g})">
    <g class="rv-move" data-siot-bind="x"
      data-siot-translate="{w:g},0" data-siot-range="0,{FIELD_W:g}">
      <g class="rv-move" data-siot-bind="y"
        data-siot-translate="0,{-h:g}" data-siot-range="0,{FIELD_H:g}">
        <path class="rv-marker" d="M-11,-9 L13,0 L-11,9 L-5,0 z" data-siot-bind="heading"
          data-siot-rotate="0,-360" data-siot-range="0,360"/>
      </g>
    </g>
  </g>
</svg>
"""


async def main() -> None:
    async with await simpleiot.connect() as c:
        # the server applies rover.yaml as siot import would; uploading the
        # same contents again does nothing, so this is safe on every start
        yaml = Path(__file__).with_name("rover.yaml").read_text()
        await c.provision("rover.yaml", yaml)
        ids = await find_nodes(c)
        rover = Rover()
        cmd = Commands()
        await c.send_point(ids["Map drawing"], "data", map_svg(rover.mission))
        print("rover running; open the Rover operations dashboard")
        await asyncio.gather(
            read_commands(c, ids, cmd),
            simulate(rover, cmd),
            report(c, ids, rover),
        )


try:
    asyncio.run(main())
except KeyboardInterrupt:
    pass
