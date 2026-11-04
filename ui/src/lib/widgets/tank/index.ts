import { defineWidget } from "../registry.ts"
import Widget from "./Widget.svelte"

export default defineWidget({
	name: "tank",
	label: "Tank",
	host: "canvas",
	bindings: ["level", "temp"],
	options: {},
	width: 220,
	// the inlet at the top, the outlet at the bottom, as nautilus places them
	ports: [
		{ name: "in", x: 0.5, y: 0, dir: "up" },
		{ name: "out", x: 0.5, y: 1, dir: "down" },
		{ name: "left", x: 0, y: 0.5, dir: "left" },
		{ name: "right", x: 1, y: 0.5, dir: "right" },
	],
	component: Widget,
})
