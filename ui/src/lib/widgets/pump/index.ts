import { defineWidget } from "../registry.ts"
import Widget from "./Widget.svelte"

export default defineWidget({
	name: "pump",
	label: "Pump",
	host: "canvas",
	bindings: ["running", "speed"],
	options: {},
	width: 120,
	// the suction stub on the left and the discharge on top, where the
	// nautilus drawing puts them
	ports: [
		{ name: "suction", x: 0, y: 0.53, dir: "left" },
		{ name: "discharge", x: 0.48, y: 0.05, dir: "up" },
	],
	component: Widget,
})
