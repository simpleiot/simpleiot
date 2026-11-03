import { defineWidget } from "../registry.ts"
import Widget from "./Widget.svelte"

export default defineWidget({
	name: "status",
	label: "Status",
	host: "both",
	bindings: ["value"],
	options: { on: "On", off: "Off" },
	component: Widget,
})
