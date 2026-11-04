# Vendored from nautilus

These files come from the [nautilus](https://github.com/joyautomation/nautilus)
HMI kit (`hmi/src/lib`), Apache License 2.0, at commit `d9c750a`. They are
copied rather than taken as a dependency because the kit is pre-1.0, carries a
realtime client Simple IoT does not use, and the pieces needed are small.

- `Gauge.svelte`: the radial gauge faceplate.
- `Tank.svelte`, `Pump.svelte`, `Valve.svelte`, `color.ts`: the equipment
  faceplates and the temperature color the tank uses.
- `Pipe.svelte`: a pipe run with its three states (flowing, still, dead) and
  layered painting for junctions.
- `mimic.ts`: ports, anchoring, exit stubs, and orthogonal routing, without the
  editor-only attach and connect helpers.
- `motion.ts`: the reduced-motion check the pump reads, from the system setting
  alone.
- `Sparkline.svelte`, `sparkline.ts`: the single-series line and its geometry.
- `tokens.css`: maps the kit's theme tokens (`--s1`, `--ink`, `--grid`, and so
  on) onto the Simple IoT theme, so the components follow light and dark mode.

Changes from the source are kept small: imports point at this directory, and
non-null assertions are replaced, since the lint here forbids them. Fixes worth
sharing go upstream first. See `LICENSE` beside this file.
