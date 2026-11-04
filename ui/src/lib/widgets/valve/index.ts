import { defineWidget } from "../registry.ts"
import Widget from "./Widget.svelte"

export default defineWidget({
	name: "valve",
	label: "Valve",
	host: "canvas",
	bindings: ["open"],
	options: {},
	width: 90,
	ports: [
		{ name: "in", x: 0, y: 0.55, dir: "left" },
		{ name: "out", x: 1, y: 0.55, dir: "right" },
	],
	component: Widget,
})
