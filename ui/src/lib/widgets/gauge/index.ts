import { defineWidget } from "../registry.ts"
import Widget from "./Widget.svelte"

export default defineWidget({
	name: "gauge",
	label: "Gauge",
	host: "both",
	bindings: ["value"],
	options: { min: "0", max: "100", unit: "", decimals: "1" },
	component: Widget,
})
