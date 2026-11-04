import { defineWidget } from "../registry.ts"
import Widget from "./Widget.svelte"

export default defineWidget({
	name: "valve",
	label: "Valve",
	host: "canvas",
	bindings: ["open"],
	options: {},
	width: 90,
	// at the tips of the two triangles, 12 and 78 of 90 across, 44 of 80 down
	ports: [
		{ name: "in", x: 0.13, y: 0.55, dir: "left" },
		{ name: "out", x: 0.87, y: 0.55, dir: "right" },
	],
	component: Widget,
})
