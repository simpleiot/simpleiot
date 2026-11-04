import { defineWidget } from "../registry.ts"
import Widget from "./Widget.svelte"

export default defineWidget({
	name: "tank",
	label: "Tank",
	host: "canvas",
	bindings: ["level", "temp"],
	options: {},
	width: 220,
	// on the shell the drawing paints, which sits inside the box: 34..186
	// of 220 across and 20..222 of 260 down
	ports: [
		{ name: "in", x: 0.5, y: 0.077, dir: "up" },
		{ name: "out", x: 0.5, y: 0.854, dir: "down" },
		{ name: "left", x: 0.155, y: 0.47, dir: "left" },
		{ name: "right", x: 0.845, y: 0.47, dir: "right" },
	],
	fit: "width",
	component: Widget,
})
